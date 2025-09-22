package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
)

const (
	// Annotation keys
	AnnotationInject     = "ocx-inject"
	AnnotationCycles     = "ocx-cycles"
	AnnotationProfile    = "ocx-profile"
	AnnotationKeystore   = "ocx-keystore"
	AnnotationVerifyOnly = "ocx-verify-only"
	
	// OCX-specific annotations
	AnnotationMutated = "ocx.dev/mutated"
	
	// Label keys for selective mutation
	LabelNamespaceEnforce = "ocx.dev/enforce"
	LabelPodEnable        = "ocx.dev/enable"

	// Default values
	DefaultCycles   = "10000"
	DefaultProfile  = "v1-min"
	DefaultKeystore = "default"

	// OCX image and binary paths
	OCXImageTag   = "ocx-protocol:latest"
	OCXBinaryPath = "/usr/local/bin/ocx"
	OCXSharedPath = "/shared"
	OCXKeysPath   = "/shared/keys"

	// Container names
	InitContainerName = "ocx-setup"
	SidecarName       = "ocx-verifier"

	// Volume names
	OCXSharedVolume = "ocx-shared"
	OCXKeysVolume   = "ocx-keys"
	OCXReceiptsVolume = "ocx-receipts"
)

// WebhookConfig holds the webhook configuration
type WebhookConfig struct {
	Port         int
	CertFile     string
	KeyFile      string
	OCXServerURL string
	MetricsPort  int
	DebugMode    bool
}

// OCXWebhook implements the Kubernetes mutating admission webhook
type OCXWebhook struct {
	config  *WebhookConfig
	server  *http.Server
	metrics *WebhookMetrics
}

// WebhookMetrics defines Prometheus metrics for the webhook
type WebhookMetrics struct {
	admissionRequests *prometheus.CounterVec
	admissionDuration *prometheus.HistogramVec
	injectionRequests *prometheus.CounterVec
	webhookErrors     *prometheus.CounterVec
}

// newWebhookMetrics creates and registers Prometheus metrics
func newWebhookMetrics() *WebhookMetrics {
	metrics := &WebhookMetrics{
		admissionRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ocx_webhook_admission_requests_total",
				Help: "Total number of admission requests processed",
			},
			[]string{"operation", "kind", "result"},
		),
		admissionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "ocx_webhook_admission_duration_seconds",
				Help:    "Duration of admission request processing",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation", "kind"},
		),
		injectionRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ocx_webhook_injection_requests_total",
				Help: "Total number of OCX injection requests",
			},
			[]string{"injection_type", "result"},
		),
		webhookErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ocx_webhook_errors_total",
				Help: "Total number of webhook processing errors",
			},
			[]string{"error_type"},
		),
	}

	prometheus.MustRegister(
		metrics.admissionRequests,
		metrics.admissionDuration,
		metrics.injectionRequests,
		metrics.webhookErrors,
	)

	return metrics
}

// InjectionSpec defines how OCX should be injected
type InjectionSpec struct {
	Type       string // "true", "verify", "sidecar"
	Cycles     string
	Profile    string
	Keystore   string
	VerifyOnly bool
}

// parseAnnotations extracts OCX injection configuration from pod annotations
func parseAnnotations(annotations map[string]string) (*InjectionSpec, error) {
	inject, exists := annotations[AnnotationInject]
	if !exists || inject == "false" {
		return nil, nil
	}

	spec := &InjectionSpec{
		Type:     inject,
		Cycles:   getAnnotationOrDefault(annotations, AnnotationCycles, DefaultCycles),
		Profile:  getAnnotationOrDefault(annotations, AnnotationProfile, DefaultProfile),
		Keystore: getAnnotationOrDefault(annotations, AnnotationKeystore, DefaultKeystore),
	}

	if verifyOnly := annotations[AnnotationVerifyOnly]; verifyOnly == "true" {
		spec.VerifyOnly = true
	}

	// Validate cycles
	if cycles, err := strconv.Atoi(spec.Cycles); err != nil || cycles <= 0 || cycles > 1000000 {
		return nil, fmt.Errorf("invalid ocx-cycles value: %s (must be 1-1000000)", spec.Cycles)
	}

	return spec, nil
}

func getAnnotationOrDefault(annotations map[string]string, key, defaultValue string) string {
	if value, exists := annotations[key]; exists {
		return value
	}
	return defaultValue
}

// injectOCXInitContainer adds OCX init container and shared volume
func (w *OCXWebhook) injectOCXInitContainer(pod *corev1.Pod, spec *InjectionSpec) error {
	// Add shared volume for OCX binary and keys
	sharedVolume := corev1.Volume{
		Name: OCXSharedVolume,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, sharedVolume)

	// OCX setup script
	setupScript := fmt.Sprintf(`#!/bin/bash
set -euo pipefail

echo "OCX Webhook: Setting up OCX binary and keystore..."

# Create directories
mkdir -p %s %s

# Copy OCX binary to shared volume
cp %s %s/ocx
chmod +x %s/ocx

# Generate keystore if it doesn't exist
if [ ! -f "%s/key.pem" ]; then
    echo "Generating OCX keystore..."
    %s/ocx keygen --output %s/ --keystore %s
else
    echo "Using existing OCX keystore"
fi

# Verify OCX setup
%s/ocx --version
echo "OCX setup completed successfully"
`, OCXSharedPath, OCXKeysPath, OCXBinaryPath, OCXSharedPath, OCXSharedPath,
		OCXKeysPath, OCXSharedPath, OCXKeysPath, spec.Keystore, OCXSharedPath)

	// Create init container
	initContainer := corev1.Container{
		Name:    InitContainerName,
		Image:   w.getOCXImage(),
		Command: []string{"/bin/bash", "-c"},
		Args:    []string{setupScript},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      OCXSharedVolume,
				MountPath: OCXSharedPath,
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    mustParseQuantity("100m"),
				corev1.ResourceMemory: mustParseQuantity("128Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    mustParseQuantity("200m"),
				corev1.ResourceMemory: mustParseQuantity("256Mi"),
			},
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot:             boolPtr(true),
			RunAsUser:                int64Ptr(65534), // nobody user
			AllowPrivilegeEscalation: boolPtr(false),
			ReadOnlyRootFilesystem:   boolPtr(true),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		},
	}

	pod.Spec.InitContainers = append(pod.Spec.InitContainers, initContainer)

	// Mount OCX binary and keys in all application containers
	ocxVolumeMount := corev1.VolumeMount{
		Name:      OCXSharedVolume,
		MountPath: "/usr/local/bin/ocx",
		SubPath:   "ocx",
		ReadOnly:  true,
	}

	keysVolumeMount := corev1.VolumeMount{
		Name:      OCXSharedVolume,
		MountPath: "/ocx/keys",
		SubPath:   "keys",
		ReadOnly:  true,
	}

	for i := range pod.Spec.Containers {
		pod.Spec.Containers[i].VolumeMounts = append(
			pod.Spec.Containers[i].VolumeMounts,
			ocxVolumeMount,
			keysVolumeMount,
		)

		// Add OCX environment variables
		pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env,
			corev1.EnvVar{Name: "OCX_CYCLES", Value: spec.Cycles},
			corev1.EnvVar{Name: "OCX_PROFILE", Value: spec.Profile},
			corev1.EnvVar{Name: "OCX_KEYSTORE", Value: spec.Keystore},
			corev1.EnvVar{Name: "OCX_SERVER_URL", Value: w.config.OCXServerURL},
			corev1.EnvVar{Name: "OCX_VERIFY_ONLY", Value: strconv.FormatBool(spec.VerifyOnly)},
		)
	}

	return nil
}

// injectOCXSidecar adds OCX verification sidecar container
func (w *OCXWebhook) injectOCXSidecar(pod *corev1.Pod, spec *InjectionSpec) error {
	sidecarContainer := corev1.Container{
		Name:    SidecarName,
		Image:   w.getOCXImage(),
		Command: []string{"/usr/local/bin/ocx", "verify-daemon"},
		Args: []string{
			"--port=8081",
			"--cycles=" + spec.Cycles,
			"--profile=" + spec.Profile,
			"--keystore=" + spec.Keystore,
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "ocx-verify",
				ContainerPort: 8081,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    mustParseQuantity("50m"),
				corev1.ResourceMemory: mustParseQuantity("64Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    mustParseQuantity("100m"),
				corev1.ResourceMemory: mustParseQuantity("128Mi"),
			},
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot:             boolPtr(true),
			RunAsUser:                int64Ptr(65534),
			AllowPrivilegeEscalation: boolPtr(false),
			ReadOnlyRootFilesystem:   boolPtr(true),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/livez",
					Port: intstr.FromInt(8081),
				},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       30,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/readyz",
					Port: intstr.FromInt(8081),
				},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       10,
		},
	}

	pod.Spec.Containers = append(pod.Spec.Containers, sidecarContainer)
	return nil
}

// mutate performs the OCX injection mutation
func (w *OCXWebhook) mutate(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		w.metrics.webhookErrors.WithLabelValues("unmarshal_error").Inc()
		return nil, fmt.Errorf("failed to unmarshal pod: %w", err)
	}

	// Check if already mutated (idempotent)
	if pod.Annotations != nil && pod.Annotations[AnnotationMutated] == "true" {
		w.metrics.injectionRequests.WithLabelValues("skipped", "already_mutated").Inc()
		return &admissionv1.AdmissionResponse{
			UID:     req.UID,
			Allowed: true,
		}, nil
	}

	// Parse OCX annotations
	spec, err := parseAnnotations(pod.Annotations)
	if err != nil {
		w.metrics.webhookErrors.WithLabelValues("annotation_error").Inc()
		return &admissionv1.AdmissionResponse{
			UID:     req.UID,
			Allowed: false,
			Result: &metav1.Status{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Invalid OCX annotation: %v", err),
			},
		}, nil
	}

	// If no OCX injection requested, allow without changes
	if spec == nil {
		w.metrics.injectionRequests.WithLabelValues("skipped", "no_injection").Inc()
		return &admissionv1.AdmissionResponse{
			UID:     req.UID,
			Allowed: true,
		}, nil
	}

	// Create a copy of the pod for mutation
	mutatedPod := pod.DeepCopy()

	// Add mutated annotation for idempotency
	if mutatedPod.Annotations == nil {
		mutatedPod.Annotations = make(map[string]string)
	}
	mutatedPod.Annotations[AnnotationMutated] = "true"

	// Add receipt volume for OCX receipts
	receiptsVolume := corev1.Volume{
		Name: OCXReceiptsVolume,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	mutatedPod.Spec.Volumes = append(mutatedPod.Spec.Volumes, receiptsVolume)

	// Wrap all application containers with OCX and add provenance env vars
	for i := range mutatedPod.Spec.Containers {
		container := &mutatedPod.Spec.Containers[i]
		
		// Add provenance environment variables
		provenanceEnvVars := createProvenanceEnvVars(mutatedPod, container.Name)
		container.Env = append(container.Env, provenanceEnvVars...)
		
		// Wrap command with OCX
		container.Command, container.Args = wrapCommandWithOCX(container.Command, container.Args)
		
		// Add OCX binary volume mount
		ocxVolumeMount := corev1.VolumeMount{
			Name:      OCXSharedVolume,
			MountPath: "/usr/local/bin/ocx",
			SubPath:   "ocx",
			ReadOnly:  true,
		}
		container.VolumeMounts = append(container.VolumeMounts, ocxVolumeMount)
		
		// Add receipts volume mount
		receiptsVolumeMount := corev1.VolumeMount{
			Name:      OCXReceiptsVolume,
			MountPath: "/var/run/ocx",
		}
		container.VolumeMounts = append(container.VolumeMounts, receiptsVolumeMount)
	}

	// Perform injection based on type
	switch spec.Type {
	case "true":
		// Standard init container injection
		if err := w.injectOCXInitContainer(mutatedPod, spec); err != nil {
			w.metrics.injectionRequests.WithLabelValues("init_container", "error").Inc()
			return nil, fmt.Errorf("failed to inject OCX init container: %w", err)
		}
		w.metrics.injectionRequests.WithLabelValues("init_container", "success").Inc()

	case "verify", "sidecar":
		// Sidecar injection for verification-only workloads
		if err := w.injectOCXSidecar(mutatedPod, spec); err != nil {
			w.metrics.injectionRequests.WithLabelValues("sidecar", "error").Inc()
			return nil, fmt.Errorf("failed to inject OCX sidecar: %w", err)
		}
		w.metrics.injectionRequests.WithLabelValues("sidecar", "success").Inc()

	default:
		w.metrics.webhookErrors.WithLabelValues("invalid_injection_type").Inc()
		return &admissionv1.AdmissionResponse{
			UID:     req.UID,
			Allowed: false,
			Result: &metav1.Status{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Invalid OCX injection type: %s", spec.Type),
			},
		}, nil
	}

	// Generate JSON patches
	patches, err := createJSONPatches(&pod, mutatedPod)
	if err != nil {
		w.metrics.webhookErrors.WithLabelValues("patch_generation_error").Inc()
		return nil, fmt.Errorf("failed to create JSON patches: %w", err)
	}

	patchBytes, err := json.Marshal(patches)
	if err != nil {
		w.metrics.webhookErrors.WithLabelValues("patch_marshal_error").Inc()
		return nil, fmt.Errorf("failed to marshal patches: %w", err)
	}

	klog.V(2).InfoS("OCX injection successful",
		"namespace", req.Namespace,
		"name", req.Name,
		"injectionType", spec.Type,
		"cycles", spec.Cycles,
		"profile", spec.Profile,
	)

	patchType := admissionv1.PatchTypeJSONPatch
	return &admissionv1.AdmissionResponse{
		UID:       req.UID,
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &patchType,
	}, nil
}

// admit handles admission requests
func (wh *OCXWebhook) admit(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		wh.metrics.admissionDuration.WithLabelValues(r.Method, "Pod").Observe(duration)
	}()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		wh.metrics.webhookErrors.WithLabelValues("read_body_error").Inc()
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var review admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &review); err != nil {
		wh.metrics.webhookErrors.WithLabelValues("unmarshal_review_error").Inc()
		http.Error(w, "Failed to unmarshal admission review", http.StatusBadRequest)
		return
	}

	req := review.Request
	if req == nil {
		wh.metrics.webhookErrors.WithLabelValues("nil_request_error").Inc()
		http.Error(w, "Admission request is nil", http.StatusBadRequest)
		return
	}

	// Only process Pod objects
	if req.Kind.Kind != "Pod" {
		wh.metrics.admissionRequests.WithLabelValues(string(req.Operation), req.Kind.Kind, "skipped").Inc()
		review.Response = &admissionv1.AdmissionResponse{
			UID:     req.UID,
			Allowed: true,
		}
	} else {
		resp, err := wh.mutate(req)
		if err != nil {
			wh.metrics.admissionRequests.WithLabelValues(string(req.Operation), req.Kind.Kind, "error").Inc()
			klog.ErrorS(err, "Failed to mutate pod",
				"namespace", req.Namespace,
				"name", req.Name,
			)
			http.Error(w, fmt.Sprintf("Mutation failed: %v", err), http.StatusInternalServerError)
			return
		}
		review.Response = resp
		wh.metrics.admissionRequests.WithLabelValues(string(req.Operation), req.Kind.Kind, "success").Inc()
	}

	review.Response.UID = req.UID

	respBytes, err := json.Marshal(review)
	if err != nil {
		wh.metrics.webhookErrors.WithLabelValues("marshal_response_error").Inc()
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}

// health endpoints
func (wh *OCXWebhook) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","webhook":"ocx-mutating-webhook"}`))
}

func (wh *OCXWebhook) ready(w http.ResponseWriter, r *http.Request) {
	// Check if OCX server is reachable
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(wh.config.OCXServerURL + "/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(w, "OCX server not ready", http.StatusServiceUnavailable)
		return
	}
	resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready","webhook":"ocx-mutating-webhook"}`))
}

// getOCXImage returns the OCX container image to use
func (w *OCXWebhook) getOCXImage() string {
	if image := os.Getenv("OCX_IMAGE"); image != "" {
		return image
	}
	return OCXImageTag
}

// NewOCXWebhook creates a new OCX webhook instance
func NewOCXWebhook(config *WebhookConfig) *OCXWebhook {
	webhook := &OCXWebhook{
		config:  config,
		metrics: newWebhookMetrics(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", webhook.admit)
	mux.HandleFunc("/health", webhook.health)
	mux.HandleFunc("/readyz", webhook.ready)
	mux.Handle("/metrics", promhttp.Handler())

	webhook.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return webhook
}

// Start starts the webhook server
func (w *OCXWebhook) Start(ctx context.Context) error {
	klog.InfoS("Starting OCX Kubernetes Mutating Webhook",
		"port", w.config.Port,
		"ocxServerURL", w.config.OCXServerURL,
	)

	// Setup TLS
	cert, err := tls.LoadX509KeyPair(w.config.CertFile, w.config.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to load TLS certificates: %w", err)
	}

	w.server.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	// Start metrics server
	if w.config.MetricsPort > 0 && w.config.MetricsPort != w.config.Port {
		go func() {
			metricsMux := http.NewServeMux()
			metricsMux.Handle("/metrics", promhttp.Handler())
			metricsServer := &http.Server{
				Addr:    fmt.Sprintf(":%d", w.config.MetricsPort),
				Handler: metricsMux,
			}
			klog.InfoS("Starting metrics server", "port", w.config.MetricsPort)
			if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				klog.ErrorS(err, "Metrics server error")
			}
		}()
	}

	// Start HTTPS server
	errChan := make(chan error, 1)
	go func() {
		errChan <- w.server.ListenAndServeTLS("", "")
	}()

	select {
	case err := <-errChan:
		if err != http.ErrServerClosed {
			return fmt.Errorf("webhook server error: %w", err)
		}
		return nil
	case <-ctx.Done():
		klog.InfoS("Shutting down OCX webhook server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return w.server.Shutdown(shutdownCtx)
	}
}

// loadConfig loads webhook configuration from environment variables
func loadConfig() *WebhookConfig {
	config := &WebhookConfig{
		Port:         getEnvInt("WEBHOOK_PORT", 8443),
		CertFile:     getEnvString("TLS_CERT_FILE", "/etc/certs/tls.crt"),
		KeyFile:      getEnvString("TLS_KEY_FILE", "/etc/certs/tls.key"),
		OCXServerURL: getEnvString("OCX_SERVER_URL", "http://ocx-server:8080"),
		MetricsPort:  getEnvInt("METRICS_PORT", 9090),
		DebugMode:    getEnvBool("DEBUG_MODE", false),
	}

	return config
}

func main() {
	config := loadConfig()

	if config.DebugMode {
		klog.InitFlags(nil)
	}

	webhook := NewOCXWebhook(config)

	// Setup graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := webhook.Start(ctx); err != nil {
		klog.ErrorS(err, "Failed to start OCX webhook")
		os.Exit(1)
	}

	klog.InfoS("OCX webhook server stopped")
}

// Utility functions
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func boolPtr(b bool) *bool {
	return &b
}

func int64Ptr(i int64) *int64 {
	return &i
}

func mustParseQuantity(s string) resource.Quantity {
	q, err := resource.ParseQuantity(s)
	if err != nil {
		panic(err)
	}
	return q
}

// JSON Patch generation (simplified implementation)
type JSONPatch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// createProvenanceEnvVars creates OCX provenance environment variables
func createProvenanceEnvVars(pod *corev1.Pod, containerName string) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{
			Name:  "OCX_NAMESPACE",
			Value: pod.Namespace,
		},
		{
			Name:  "OCX_POD_UID",
			Value: string(pod.UID),
		},
		{
			Name:  "OCX_CONTAINER",
			Value: containerName,
		},
		{
			Name:  "OCX_WORKLOAD",
			Value: getWorkloadName(pod),
		},
	}

	// Add OCX_TEAM if namespace has owner annotation
	if pod.Annotations != nil {
		if team := pod.Annotations["ocx.dev/team"]; team != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "OCX_TEAM",
				Value: team,
			})
		}
	}

	// Add OCX_COMMIT_SHA if present on pod labels
	if pod.Labels != nil {
		if commit := pod.Labels["ocx.dev/commit-sha"]; commit != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "OCX_COMMIT_SHA",
				Value: commit,
			})
		}
	}

	return envVars
}

// getWorkloadName extracts the workload name from pod owner references
func getWorkloadName(pod *corev1.Pod) string {
	for _, owner := range pod.OwnerReferences {
		switch owner.Kind {
		case "Deployment", "ReplicaSet", "StatefulSet", "DaemonSet", "Job", "CronJob":
			return owner.Name
		}
	}
	return pod.Name
}

// wrapCommandWithOCX wraps the original command with OCX
func wrapCommandWithOCX(originalCommand, originalArgs []string) ([]string, []string) {
	if len(originalCommand) == 0 {
		// No command specified, use the image's ENTRYPOINT
		return []string{"/usr/local/bin/ocx", "run", "--"}, originalArgs
	}

	// Wrap the existing command
	wrappedCommand := []string{"/usr/local/bin/ocx", "run", "--"}
	wrappedCommand = append(wrappedCommand, originalCommand...)
	
	// Keep original args
	return wrappedCommand, originalArgs
}

func createJSONPatches(original, modified *corev1.Pod) ([]JSONPatch, error) {
	var patches []JSONPatch

	// Add init containers if they were added
	if len(modified.Spec.InitContainers) > len(original.Spec.InitContainers) {
		for i := len(original.Spec.InitContainers); i < len(modified.Spec.InitContainers); i++ {
			patches = append(patches, JSONPatch{
				Op:    "add",
				Path:  fmt.Sprintf("/spec/initContainers/%d", i),
				Value: modified.Spec.InitContainers[i],
			})
		}
	}

	// Add containers if they were added (sidecars)
	if len(modified.Spec.Containers) > len(original.Spec.Containers) {
		for i := len(original.Spec.Containers); i < len(modified.Spec.Containers); i++ {
			patches = append(patches, JSONPatch{
				Op:    "add",
				Path:  fmt.Sprintf("/spec/containers/%d", i),
				Value: modified.Spec.Containers[i],
			})
		}
	}

	// Update existing containers with volume mounts and env vars
	for i := 0; i < len(original.Spec.Containers); i++ {
		if i < len(modified.Spec.Containers) {
			// Update volume mounts
			if len(modified.Spec.Containers[i].VolumeMounts) > len(original.Spec.Containers[i].VolumeMounts) {
				patches = append(patches, JSONPatch{
					Op:    "replace",
					Path:  fmt.Sprintf("/spec/containers/%d/volumeMounts", i),
					Value: modified.Spec.Containers[i].VolumeMounts,
				})
			}

			// Update environment variables
			if len(modified.Spec.Containers[i].Env) > len(original.Spec.Containers[i].Env) {
				patches = append(patches, JSONPatch{
					Op:    "replace",
					Path:  fmt.Sprintf("/spec/containers/%d/env", i),
					Value: modified.Spec.Containers[i].Env,
				})
			}
		}
	}

	// Add volumes if they were added
	if len(modified.Spec.Volumes) > len(original.Spec.Volumes) {
		for i := len(original.Spec.Volumes); i < len(modified.Spec.Volumes); i++ {
			patches = append(patches, JSONPatch{
				Op:    "add",
				Path:  fmt.Sprintf("/spec/volumes/%d", i),
				Value: modified.Spec.Volumes[i],
			})
		}
	}

	return patches, nil
}
