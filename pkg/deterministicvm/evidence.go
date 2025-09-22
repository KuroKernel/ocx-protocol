package deterministicvm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
)

// EvidenceV1 represents the evidence schema for OCX execution
type EvidenceV1 struct {
	Schema         string `json:"schema"`
	ArtifactID     string `json:"artifact_id"`
	ReceiptHash    string `json:"receipt_hash"`
	EnvHash        string `json:"env_hash"`
	Seed           string `json:"seed"`
	Limits         struct {
		CPUMs     int64 `json:"cpu_ms"`
		RSSBytes  int64 `json:"rss_bytes"`
		PidsMax   int   `json:"pids_max"`
	} `json:"limits"`
	Platform struct {
		Kernel          string `json:"kernel"`
		Libc            string `json:"libc"`
		ContainerDigest string `json:"container_digest"`
	} `json:"platform"`
	Rusage struct {
		MaxRSS   int64 `json:"max_rss"`
		Minflt   int64 `json:"minflt"`
		Majflt   int64 `json:"majflt"`
		UtimeMs  int64 `json:"utime_ms"`
		StimeMs  int64 `json:"stime_ms"`
	} `json:"rusage"`
	SeccompProfile string `json:"seccomp_profile"`
	CgroupPath     string `json:"cgroup_path"`
}

// emitEvidence logs the evidence in a structured format
func emitEvidence(ev EvidenceV1) {
	b, err := json.Marshal(ev)
	if err != nil {
		log.Printf("Failed to marshal evidence: %v", err)
		return
	}
	log.Printf("EVIDENCE %s", string(b))
}

// EnvHash calculates a deterministic hash of the execution environment
func EnvHash() string {
	h := sha256.New()
	
	write := func(k, v string) {
		h.Write([]byte(k))
		h.Write([]byte{0})
		h.Write([]byte(v))
		h.Write([]byte{0})
	}
	
	// Add OCX version
	write("OCX_D_MVM_VERSION", "1.0.0")
	
	// Add locale and timezone
	write("LOCALE", os.Getenv("LC_ALL"))
	write("TZ", os.Getenv("TZ"))
	write("PATH", os.Getenv("PATH"))
	
	// Add platform information
	write("GOOS", runtime.GOOS)
	write("GOARCH", runtime.GOARCH)
	
	// Add kernel version (Linux only)
	if runtime.GOOS == "linux" {
		if kernel, err := getKernelVersion(); err == nil {
			write("KERNEL", kernel)
		}
	}
	
	// Add libc version (Linux only)
	if runtime.GOOS == "linux" {
		if libc, err := getLibcVersion(); err == nil {
			write("LIBC", libc)
		}
	}
	
	// Walk a short whitelist of system directories
	whitelistDirs := []string{"/usr/bin", "/lib", "/lib64", "/etc/ocx"}
	for _, dir := range whitelistDirs {
		if _, err := os.Stat(dir); err == nil {
			filepath.WalkDir(dir, func(p string, d fs.DirEntry, _ error) error {
				if d.Type().IsRegular() {
					if sum, ok := fileSha256(p); ok {
						write("bin:"+p, sum)
					}
				}
				return nil
			})
		}
	}
	
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// fileSha256 calculates SHA256 of a file
func fileSha256(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), true
}

// getKernelVersion gets the Linux kernel version
func getKernelVersion() (string, error) {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return "", err
	}
	// Extract just the version number
	version := string(data)
	if len(version) > 100 {
		version = version[:100] // Truncate for consistency
	}
	return version, nil
}

// getLibcVersion gets the libc version
func getLibcVersion() (string, error) {
	// Try common libc paths
	libcPaths := []string{"/lib/x86_64-linux-gnu/libc.so.6", "/lib64/libc.so.6", "/lib/libc.so.6"}
	for _, path := range libcPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "unknown", fmt.Errorf("libc not found")
}

// getRusage gets resource usage information
func getRusage() (maxRSS, minflt, majflt, utimeMs, stimeMs int64) {
	var rusage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &rusage); err == nil {
		maxRSS = rusage.Maxrss
		minflt = rusage.Minflt
		majflt = rusage.Majflt
		utimeMs = rusage.Utime.Sec*1000 + int64(rusage.Utime.Usec/1000)
		stimeMs = rusage.Stime.Sec*1000 + int64(rusage.Stime.Usec/1000)
	}
	return
}

// CreateEvidence creates evidence for an execution
func CreateEvidence(artifactID, receiptHash, seed string, limits VMConfig) EvidenceV1 {
	envHash := EnvHash()
	maxRSS, minflt, majflt, utimeMs, stimeMs := getRusage()
	
	ev := EvidenceV1{
		Schema:      "evidence_v1",
		ArtifactID:  artifactID,
		ReceiptHash: receiptHash,
		EnvHash:     envHash,
		Seed:        seed,
		Limits: struct {
			CPUMs     int64 `json:"cpu_ms"`
			RSSBytes  int64 `json:"rss_bytes"`
			PidsMax   int   `json:"pids_max"`
		}{
			CPUMs:     int64(limits.Timeout.Milliseconds()),
			RSSBytes:  int64(limits.MemoryLimit),
			PidsMax:   64, // From cgroups config
		},
		Platform: struct {
			Kernel          string `json:"kernel"`
			Libc            string `json:"libc"`
			ContainerDigest string `json:"container_digest"`
		}{
			Kernel:          getKernelVersionSafe(),
			Libc:            getLibcVersionSafe(),
			ContainerDigest: "", // Would be set in containerized environments
		},
		Rusage: struct {
			MaxRSS   int64 `json:"max_rss"`
			Minflt   int64 `json:"minflt"`
			Majflt   int64 `json:"majflt"`
			UtimeMs  int64 `json:"utime_ms"`
			StimeMs  int64 `json:"stime_ms"`
		}{
			MaxRSS:  maxRSS,
			Minflt:  minflt,
			Majflt:  majflt,
			UtimeMs: utimeMs,
			StimeMs: stimeMs,
		},
		SeccompProfile: "ocx-seccomp-v1",
		CgroupPath:     fmt.Sprintf("ocx.slice/%d", os.Getpid()),
	}
	
	return ev
}

// getKernelVersionSafe safely gets kernel version
func getKernelVersionSafe() string {
	if version, err := getKernelVersion(); err == nil {
		return version
	}
	return "unknown"
}

// getLibcVersionSafe safely gets libc version
func getLibcVersionSafe() string {
	if version, err := getLibcVersion(); err == nil {
		return version
	}
	return "unknown"
}
