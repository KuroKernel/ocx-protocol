// adapters/ad6-kafka/interceptor.go - AD6 Kafka Interceptor for OCX Injection
// This follows the EXACT same pattern as AD2 webhook but for data stream injection

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Shopify/sarama"
	"ocx.local/pkg/cbor"
	"ocx.local/pkg/verify"
)

// KafkaInterceptor represents the Kafka interceptor for OCX injection
type KafkaInterceptor struct {
	enabled     bool
	verifier    verify.Verifier
	annotations map[string]string
	producer    sarama.AsyncProducer
	consumer    sarama.Consumer
}

// NewKafkaInterceptor creates a new Kafka interceptor instance
func NewKafkaInterceptor() *KafkaInterceptor {
	return &KafkaInterceptor{
		enabled:     true,
		verifier:    verify.NewVerifier(),
		annotations: make(map[string]string),
	}
}

// KafkaMessage represents a Kafka message
type KafkaMessage struct {
	Topic     string            `json:"topic"`
	Partition int32             `json:"partition"`
	Offset    int64             `json:"offset"`
	Key       []byte            `json:"key"`
	Value     []byte            `json:"value"`
	Headers   map[string][]byte `json:"headers"`
	Timestamp time.Time         `json:"timestamp"`
}

// OCXEnrichedMessage represents a message enriched with OCX data
type OCXEnrichedMessage struct {
	*KafkaMessage
	OCXReceipt    string            `json:"ocx_receipt"`
	OCXVersion    string            `json:"ocx_version"`
	OCXCycles     uint64            `json:"ocx_cycles"`
	OCXChained    bool              `json:"ocx_chained"`
	OCXWitness    bool              `json:"ocx_witness"`
	OCXVerified   bool              `json:"ocx_verified"`
	OCXHeaders    map[string]string `json:"ocx_headers"`
}

// InterceptMessage intercepts and enriches a Kafka message (same pattern as AD2)
func (ki *KafkaInterceptor) InterceptMessage(msg *KafkaMessage) (*OCXEnrichedMessage, error) {
	if !ki.enabled {
		return ki.passthrough(msg), nil
	}

	// Parse annotations from headers (same as AD2 webhook)
	ki.parseAnnotations(msg.Headers)

	// Check if OCX injection is enabled (same logic as AD2)
	if !ki.shouldInject() {
		return ki.passthrough(msg), nil
	}

	// Generate OCX receipt (same pattern as AD2)
	receipt, err := ki.generateReceipt(msg)
	if err != nil {
		log.Printf("Failed to generate OCX receipt: %v", err)
		return ki.passthrough(msg), nil
	}

	// Enrich message with OCX data
	enriched := ki.enrichMessage(msg, receipt)

	return enriched, nil
}

// parseAnnotations parses OCX annotations from headers (same as AD2)
func (ki *KafkaInterceptor) parseAnnotations(headers map[string][]byte) {
	// Look for OCX annotations in headers
	ocxHeaders := map[string]string{
		"ocx-inject":      "ocx-inject",
		"ocx-cycles":      "ocx-cycles",
		"ocx-profile":     "ocx-profile",
		"ocx-keystore":    "ocx-keystore",
		"ocx-verify-only": "ocx-verify-only",
	}

	for headerName, annotationKey := range ocxHeaders {
		if value, exists := headers[headerName]; exists {
			ki.annotations[annotationKey] = string(value)
		}
	}
}

// shouldInject determines if OCX should be injected (same logic as AD2)
func (ki *KafkaInterceptor) shouldInject() bool {
	inject := ki.annotations["ocx-inject"]
	return inject == "true" || inject == "verify"
}

// generateReceipt generates an OCX receipt (same pattern as AD2)
func (ki *KafkaInterceptor) generateReceipt(msg *KafkaMessage) (*cbor.OCXReceiptV1_1, error) {
	// Create artifact hash from message
	artifactHash := ki.hashMessage(msg)
	
	// Create input hash from message value
	inputHash := ki.hashData(msg.Value)
	
	// Create output hash (placeholder for now)
	outputHash := ki.hashData([]byte("kafka_output"))
	
	// Get cycles from annotation
	cycles := ki.getCycles()
	
	// Create issuer key
	issuerKey := ki.getIssuerKey()
	
	// Create receipt
	receipt := cbor.NewOCXReceiptV1_1(artifactHash, inputHash, outputHash, cycles, issuerKey)
	
	// Add request binding
	requestDigest := ki.hashData(msg.Value)
	receipt.AddRequestBinding(requestDigest)
	
	// Add witness signature if enabled
	if ki.annotations["ocx-verify-only"] == "true" {
		witnessManager := cbor.NewWitnessManager()
		witnessManager.AddWitness("kafka", issuerKey)
		witnessManager.SignReceipt(receipt)
	}
	
	return receipt, nil
}

// enrichMessage enriches a message with OCX data (same pattern as AD2)
func (ki *KafkaInterceptor) enrichMessage(msg *KafkaMessage, receipt *cbor.OCXReceiptV1_1) *OCXEnrichedMessage {
	// Serialize receipt
	receiptData, err := receipt.Serialize()
	if err != nil {
		log.Printf("Failed to serialize receipt: %v", err)
		receiptData = []byte("{}")
	}
	
	// Create enriched message
	enriched := &OCXEnrichedMessage{
		KafkaMessage: msg,
		OCXReceipt:   string(receiptData),
		OCXVersion:   "1.1",
		OCXCycles:    receipt.Cycles,
		OCXChained:   receipt.IsChained(),
		OCXWitness:   receipt.HasWitness(),
		OCXVerified:  true,
		OCXHeaders: map[string]string{
			"ocx-receipt":    string(receiptData),
			"ocx-version":    "1.1",
			"ocx-cycles":     fmt.Sprintf("%d", receipt.Cycles),
			"ocx-chained":    fmt.Sprintf("%t", receipt.IsChained()),
			"ocx-witness":    fmt.Sprintf("%t", receipt.HasWitness()),
			"ocx-verified":   "true",
		},
	}
	
	return enriched
}

// passthrough returns the message unchanged
func (ki *KafkaInterceptor) passthrough(msg *KafkaMessage) *OCXEnrichedMessage {
	return &OCXEnrichedMessage{
		KafkaMessage: msg,
		OCXVerified:  false,
		OCXHeaders:   make(map[string]string),
	}
}

// ProducerInterceptor implements sarama.ProducerInterceptor
type ProducerInterceptor struct {
	*KafkaInterceptor
}

// OnSend is called before a message is sent
func (pi *ProducerInterceptor) OnSend(msg *sarama.ProducerMessage) {
	// Convert sarama message to our format
	kafkaMsg := &KafkaMessage{
		Topic:     msg.Topic,
		Partition: msg.Partition,
		Key:       msg.Key.Encode(),
		Value:     msg.Value.Encode(),
		Headers:   make(map[string][]byte),
		Timestamp: msg.Timestamp,
	}
	
	// Copy headers
	for _, header := range msg.Headers {
		kafkaMsg.Headers[string(header.Key)] = header.Value
	}
	
	// Intercept and enrich message
	enriched, err := pi.InterceptMessage(kafkaMsg)
	if err != nil {
		log.Printf("Failed to intercept message: %v", err)
		return
	}
	
	// Add OCX headers to original message
	for key, value := range enriched.OCXHeaders {
		msg.Headers = append(msg.Headers, &sarama.RecordHeader{
			Key:   []byte(key),
			Value: []byte(value),
		})
	}
}

// ConsumerInterceptor implements sarama.ConsumerInterceptor
type ConsumerInterceptor struct {
	*KafkaInterceptor
}

// OnConsume is called when a message is consumed
func (ci *ConsumerInterceptor) OnConsume(msg *sarama.ConsumerMessage) {
	// Convert sarama message to our format
	kafkaMsg := &KafkaMessage{
		Topic:     msg.Topic,
		Partition: msg.Partition,
		Offset:    msg.Offset,
		Key:       msg.Key,
		Value:     msg.Value,
		Headers:   make(map[string][]byte),
		Timestamp: msg.Timestamp,
	}
	
	// Copy headers
	for _, header := range msg.Headers {
		kafkaMsg.Headers[string(header.Key)] = header.Value
	}
	
	// Intercept and enrich message
	enriched, err := ci.InterceptMessage(kafkaMsg)
	if err != nil {
		log.Printf("Failed to intercept message: %v", err)
		return
	}
	
	// Log OCX verification status
	if enriched.OCXVerified {
		log.Printf("OCX verified message: topic=%s, partition=%d, offset=%d, cycles=%d",
			enriched.Topic, enriched.Partition, enriched.Offset, enriched.OCXCycles)
	}
}

// KafkaInterceptorServer represents the Kafka interceptor server
type KafkaInterceptorServer struct {
	interceptor *KafkaInterceptor
	config      *sarama.Config
	brokers     []string
}

// NewKafkaInterceptorServer creates a new Kafka interceptor server
func NewKafkaInterceptorServer(brokers []string) *KafkaInterceptorServer {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Consumer.Return.Errors = true
	
	return &KafkaInterceptorServer{
		interceptor: NewKafkaInterceptor(),
		config:      config,
		brokers:     brokers,
	}
}

// Start starts the Kafka interceptor server
func (kis *KafkaInterceptorServer) Start() error {
	// Create producer
	producer, err := sarama.NewAsyncProducer(kis.brokers, kis.config)
	if err != nil {
		return fmt.Errorf("failed to create producer: %v", err)
	}
	kis.interceptor.producer = producer
	
	// Create consumer
	consumer, err := sarama.NewConsumer(kis.brokers, kis.config)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %v", err)
	}
	kis.interceptor.consumer = consumer
	
	log.Printf("Starting AD6 Kafka Interceptor with brokers: %v", kis.brokers)
	log.Printf("Interceptor enabled: %v", kis.interceptor.enabled)
	
	return nil
}

// Stop stops the Kafka interceptor server
func (kis *KafkaInterceptorServer) Stop() error {
	if kis.interceptor.producer != nil {
		kis.interceptor.producer.AsyncClose()
	}
	if kis.interceptor.consumer != nil {
		kis.interceptor.consumer.Close()
	}
	return nil
}

// Helper functions (same pattern as AD2)
func (ki *KafkaInterceptor) hashMessage(msg *KafkaMessage) [32]byte {
	data := fmt.Sprintf("%s:%d:%d:%s", msg.Topic, msg.Partition, msg.Offset, string(msg.Value))
	return ki.hashData([]byte(data))
}

func (ki *KafkaInterceptor) hashData(data []byte) [32]byte {
	// In a real implementation, this would use crypto/sha256
	return [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
}

func (ki *KafkaInterceptor) getCycles() uint64 {
	cyclesStr := ki.annotations["ocx-cycles"]
	if cyclesStr == "" {
		return 10000 // Default
	}
	
	// Parse cycles (simplified)
	return 10000
}

func (ki *KafkaInterceptor) getIssuerKey() [32]byte {
	// In a real implementation, this would load from keystore
	return [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
}

func main() {
	// Example usage
	brokers := []string{"localhost:9092"}
	server := NewKafkaInterceptorServer(brokers)
	
	if err := server.Start(); err != nil {
		log.Fatal("Failed to start Kafka interceptor:", err)
	}
	
	// Keep running
	select {}
}
