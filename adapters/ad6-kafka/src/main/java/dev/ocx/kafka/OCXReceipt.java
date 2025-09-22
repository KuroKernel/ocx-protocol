package dev.ocx.kafka;

import com.fasterxml.jackson.annotation.JsonProperty;

public class OCXReceipt {
    @JsonProperty("receipt_id")
    private String receiptId;
    
    @JsonProperty("verified")
    private boolean verified;
    
    @JsonProperty("timestamp")
    private String timestamp;
    
    @JsonProperty("artifact_hash")
    private String artifactHash;
    
    @JsonProperty("signature")
    private String signature;
    
    // Constructors
    public OCXReceipt() {}
    
    public OCXReceipt(String receiptId, boolean verified, String timestamp) {
        this.receiptId = receiptId;
        this.verified = verified;
        this.timestamp = timestamp;
    }
    
    // Getters and Setters
    public String getReceiptId() { return receiptId; }
    public void setReceiptId(String receiptId) { this.receiptId = receiptId; }
    
    public boolean isVerified() { return verified; }
    public void setVerified(boolean verified) { this.verified = verified; }
    
    public String getTimestamp() { return timestamp; }
    public void setTimestamp(String timestamp) { this.timestamp = timestamp; }
    
    public String getArtifactHash() { return artifactHash; }
    public void setArtifactHash(String artifactHash) { this.artifactHash = artifactHash; }
    
    public String getSignature() { return signature; }
    public void setSignature(String signature) { this.signature = signature; }
    
    // Utility methods
    public byte[] toCborBytes() {
        // In a real implementation, this would convert to CBOR format
        // For now, return JSON bytes as placeholder
        try {
            return toString().getBytes("UTF-8");
        } catch (Exception e) {
            throw new RuntimeException("Failed to convert receipt to bytes", e);
        }
    }
    
    @Override
    public String toString() {
        return String.format("OCXReceipt{receiptId='%s', verified=%s, timestamp='%s'}", 
                           receiptId, verified, timestamp);
    }
}
