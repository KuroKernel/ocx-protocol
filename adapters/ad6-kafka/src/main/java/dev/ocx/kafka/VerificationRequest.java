package dev.ocx.kafka;

import com.fasterxml.jackson.annotation.JsonProperty;

public class VerificationRequest {
    @JsonProperty("artifact")
    private String artifact;
    
    @JsonProperty("input")
    private String input;
    
    @JsonProperty("cycles")
    private int cycles;
    
    @JsonProperty("request_digest")
    private String requestDigest;
    
    @JsonProperty("receipt_data")
    private byte[] receiptData;
    
    @JsonProperty("expected_context")
    private String expectedContext;
    
    // Constructors
    public VerificationRequest() {}
    
    public VerificationRequest(String artifact, String input, int cycles, String requestDigest) {
        this.artifact = artifact;
        this.input = input;
        this.cycles = cycles;
        this.requestDigest = requestDigest;
    }
    
    // Builder pattern
    public static Builder builder() {
        return new Builder();
    }
    
    public static class Builder {
        private String artifact;
        private String input;
        private int cycles;
        private String requestDigest;
        private byte[] receiptData;
        private String expectedContext;
        
        public Builder artifact(String artifact) {
            this.artifact = artifact;
            return this;
        }
        
        public Builder input(String input) {
            this.input = input;
            return this;
        }
        
        public Builder cycles(int cycles) {
            this.cycles = cycles;
            return this;
        }
        
        public Builder requestDigest(String requestDigest) {
            this.requestDigest = requestDigest;
            return this;
        }
        
        public Builder receiptData(byte[] receiptData) {
            this.receiptData = receiptData;
            return this;
        }
        
        public Builder expectedContext(String expectedContext) {
            this.expectedContext = expectedContext;
            return this;
        }
        
        public VerificationRequest build() {
            VerificationRequest request = new VerificationRequest();
            request.artifact = this.artifact;
            request.input = this.input;
            request.cycles = this.cycles;
            request.requestDigest = this.requestDigest;
            request.receiptData = this.receiptData;
            request.expectedContext = this.expectedContext;
            return request;
        }
    }
    
    // Getters and Setters
    public String getArtifact() { return artifact; }
    public void setArtifact(String artifact) { this.artifact = artifact; }
    
    public String getInput() { return input; }
    public void setInput(String input) { this.input = input; }
    
    public int getCycles() { return cycles; }
    public void setCycles(int cycles) { this.cycles = cycles; }
    
    public String getRequestDigest() { return requestDigest; }
    public void setRequestDigest(String requestDigest) { this.requestDigest = requestDigest; }
    
    public byte[] getReceiptData() { return receiptData; }
    public void setReceiptData(byte[] receiptData) { this.receiptData = receiptData; }
    
    public String getExpectedContext() { return expectedContext; }
    public void setExpectedContext(String expectedContext) { this.expectedContext = expectedContext; }
}
