package dev.ocx.kafka;

import com.fasterxml.jackson.annotation.JsonProperty;

public class MessageContext {
    @JsonProperty("topic")
    private String topic;
    
    @JsonProperty("partition")
    private Integer partition;
    
    @JsonProperty("timestamp")
    private Long timestamp;
    
    @JsonProperty("key_size")
    private int keySize;
    
    @JsonProperty("value_size")
    private int valueSize;
    
    @JsonProperty("header_count")
    private int headerCount;
    
    @JsonProperty("producer_client_id")
    private String producerClientId;
    
    // Constructors
    public MessageContext() {}
    
    public MessageContext(String topic, Integer partition, Long timestamp, int keySize, int valueSize, int headerCount, String producerClientId) {
        this.topic = topic;
        this.partition = partition;
        this.timestamp = timestamp;
        this.keySize = keySize;
        this.valueSize = valueSize;
        this.headerCount = headerCount;
        this.producerClientId = producerClientId;
    }
    
    // Builder pattern
    public static Builder builder() {
        return new Builder();
    }
    
    public static class Builder {
        private String topic;
        private Integer partition;
        private Long timestamp;
        private int keySize;
        private int valueSize;
        private int headerCount;
        private String producerClientId;
        
        public Builder topic(String topic) {
            this.topic = topic;
            return this;
        }
        
        public Builder partition(Integer partition) {
            this.partition = partition;
            return this;
        }
        
        public Builder timestamp(Long timestamp) {
            this.timestamp = timestamp;
            return this;
        }
        
        public Builder keySize(int keySize) {
            this.keySize = keySize;
            return this;
        }
        
        public Builder valueSize(int valueSize) {
            this.valueSize = valueSize;
            return this;
        }
        
        public Builder headerCount(int headerCount) {
            this.headerCount = headerCount;
            return this;
        }
        
        public Builder producerClientId(String producerClientId) {
            this.producerClientId = producerClientId;
            return this;
        }
        
        public MessageContext build() {
            return new MessageContext(topic, partition, timestamp, keySize, valueSize, headerCount, producerClientId);
        }
    }
    
    // Getters and Setters
    public String getTopic() { return topic; }
    public void setTopic(String topic) { this.topic = topic; }
    
    public Integer getPartition() { return partition; }
    public void setPartition(Integer partition) { this.partition = partition; }
    
    public Long getTimestamp() { return timestamp; }
    public void setTimestamp(Long timestamp) { this.timestamp = timestamp; }
    
    public int getKeySize() { return keySize; }
    public void setKeySize(int keySize) { this.keySize = keySize; }
    
    public int getValueSize() { return valueSize; }
    public void setValueSize(int valueSize) { this.valueSize = valueSize; }
    
    public int getHeaderCount() { return headerCount; }
    public void setHeaderCount(int headerCount) { this.headerCount = headerCount; }
    
    public String getProducerClientId() { return producerClientId; }
    public void setProducerClientId(String producerClientId) { this.producerClientId = producerClientId; }
    
    // Utility methods
    public String toJson() {
        try {
            return new com.fasterxml.jackson.databind.ObjectMapper().writeValueAsString(this);
        } catch (Exception e) {
            throw new RuntimeException("Failed to serialize MessageContext to JSON", e);
        }
    }
    
    public String getRequestDigest() {
        // In a real implementation, this would generate a proper digest
        return String.valueOf(hashCode());
    }
}
