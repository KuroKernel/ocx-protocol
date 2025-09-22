package dev.ocx.kafka;

public class OCXVerificationException extends RuntimeException {
    public OCXVerificationException(String message) {
        super(message);
    }
    
    public OCXVerificationException(String message, Throwable cause) {
        super(message, cause);
    }
}
