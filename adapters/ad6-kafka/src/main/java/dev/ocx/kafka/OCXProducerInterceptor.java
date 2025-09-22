package dev.ocx.kafka;

import org.apache.kafka.clients.producer.ProducerInterceptor;
import org.apache.kafka.clients.producer.ProducerRecord;
import org.apache.kafka.clients.producer.RecordMetadata;
import org.apache.kafka.common.header.Headers;
import org.apache.kafka.common.header.internals.RecordHeaders;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.time.Instant;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

public class OCXProducerInterceptor<K, V> implements ProducerInterceptor<K, V> {
    private static final Logger log = LoggerFactory.getLogger(OCXProducerInterceptor.class);
    
    private static final String OCX_RECEIPT_HEADER = "ocx-receipt";
    private static final String OCX_VERIFIED_HEADER = "ocx-verified";
    private static final String OCX_TIMESTAMP_HEADER = "ocx-timestamp";
    
    private OCXClient ocxClient;
    private ExecutorService executorService;
    private boolean asyncMode;
    private boolean failClosed;
    private final Map<String, String> config = new ConcurrentHashMap<>();
    
    @Override
    public void configure(Map<String, ?> configs) {
        config.putAll((Map<String, String>) configs);
        
        String ocxServerUrl = config.get("ocx.server.url");
        String apiKey = config.get("ocx.api.key");
        asyncMode = Boolean.parseBoolean(config.getOrDefault("ocx.async.mode", "true"));
        failClosed = Boolean.parseBoolean(config.getOrDefault("ocx.fail.closed", "false"));
        
        if (ocxServerUrl == null || apiKey == null) {
            throw new IllegalArgumentException("OCX server URL and API key must be configured");
        }
        
        ocxClient = new OCXClient(ocxServerUrl, apiKey);
        executorService = Executors.newFixedThreadPool(
            Integer.parseInt(config.getOrDefault("ocx.thread.pool.size", "4"))
        );
        
        log.info("OCX Kafka interceptor configured: server={}, async={}, failClosed={}", 
                 ocxServerUrl, asyncMode, failClosed);
    }
    
    @Override
    public ProducerRecord<K, V> onSend(ProducerRecord<K, V> record) {
        try {
            long startTime = System.nanoTime();
            
            // Create message context for verification
            MessageContext context = createMessageContext(record);
            
            if (asyncMode) {
                // Async mode: generate receipt in background, don't block send
                CompletableFuture.runAsync(() -> {
                    try {
                        generateAndAttachReceipt(record, context);
                    } catch (Exception e) {
                        log.warn("Async OCX receipt generation failed for topic {}: {}", 
                                record.topic(), e.getMessage());
                    }
                }, executorService);
                
                // Add timestamp header immediately
                record.headers().add(OCX_TIMESTAMP_HEADER, 
                    Instant.now().toString().getBytes(StandardCharsets.UTF_8));
                
                return record;
                
            } else {
                // Sync mode: generate receipt before send
                ProducerRecord<K, V> verifiedRecord = generateAndAttachReceipt(record, context);
                
                long duration = System.nanoTime() - startTime;
                log.debug("OCX verification completed in {}μs for topic {}", 
                         duration / 1000, record.topic());
                
                return verifiedRecord;
            }
            
        } catch (Exception e) {
            log.error("OCX verification failed for topic {}: {}", record.topic(), e.getMessage());
            
            if (failClosed) {
                throw new OCXVerificationException("OCX verification failed", e);
            } else {
                // Add error header and continue
                record.headers().add(OCX_VERIFIED_HEADER, "false".getBytes(StandardCharsets.UTF_8));
                record.headers().add("ocx-error", e.getMessage().getBytes(StandardCharsets.UTF_8));
                return record;
            }
        }
    }
    
    private ProducerRecord<K, V> generateAndAttachReceipt(ProducerRecord<K, V> record, 
                                                         MessageContext context) throws Exception {
        // Generate message hash
        String messageHash = generateMessageHash(record);
        
        // Create verification request
        VerificationRequest request = VerificationRequest.builder()
            .artifact(messageHash)
            .input(context.toJson())
            .cycles(10000)
            .requestDigest(context.getRequestDigest())
            .build();
        
        // Get receipt from OCX server
        OCXReceipt receipt = ocxClient.executeVerification(request);
        
        // Create new record with OCX headers
        ProducerRecord<K, V> verifiedRecord = new ProducerRecord<>(
            record.topic(),
            record.partition(),
            record.timestamp(),
            record.key(),
            record.value(),
            copyHeaders(record.headers())
        );
        
        // Add OCX headers
        verifiedRecord.headers().add(OCX_RECEIPT_HEADER, receipt.toCborBytes());
        verifiedRecord.headers().add(OCX_VERIFIED_HEADER, "true".getBytes(StandardCharsets.UTF_8));
        verifiedRecord.headers().add(OCX_TIMESTAMP_HEADER, 
            Instant.now().toString().getBytes(StandardCharsets.UTF_8));
        verifiedRecord.headers().add("ocx-message-hash", 
            messageHash.getBytes(StandardCharsets.UTF_8));
        
        return verifiedRecord;
    }
    
    private MessageContext createMessageContext(ProducerRecord<K, V> record) {
        return MessageContext.builder()
            .topic(record.topic())
            .partition(record.partition())
            .timestamp(record.timestamp() != null ? record.timestamp() : System.currentTimeMillis())
            .keySize(record.key() != null ? record.key().toString().length() : 0)
            .valueSize(record.value() != null ? record.value().toString().length() : 0)
            .headerCount(record.headers().toArray().length)
            .producerClientId(config.get("client.id"))
            .build();
    }
    
    private String generateMessageHash(ProducerRecord<K, V> record) throws Exception {
        MessageDigest digest = MessageDigest.getInstance("SHA-256");
        
        // Hash topic
        digest.update(record.topic().getBytes(StandardCharsets.UTF_8));
        
        // Hash key
        if (record.key() != null) {
            digest.update(record.key().toString().getBytes(StandardCharsets.UTF_8));
        }
        
        // Hash value
        if (record.value() != null) {
            digest.update(record.value().toString().getBytes(StandardCharsets.UTF_8));
        }
        
        // Hash timestamp
        if (record.timestamp() != null) {
            digest.update(Long.toString(record.timestamp()).getBytes(StandardCharsets.UTF_8));
        }
        
        // Hash headers
        record.headers().forEach(header -> {
            digest.update(header.key().getBytes(StandardCharsets.UTF_8));
            if (header.value() != null) {
                digest.update(header.value());
            }
        });
        
        byte[] hash = digest.digest();
        StringBuilder hexString = new StringBuilder();
        for (byte b : hash) {
            String hex = Integer.toHexString(0xff & b);
            if (hex.length() == 1) {
                hexString.append('0');
            }
            hexString.append(hex);
        }
        
        return hexString.toString();
    }
    
    private Headers copyHeaders(Headers original) {
        Headers copy = new RecordHeaders();
        original.forEach(header -> copy.add(header.key(), header.value()));
        return copy;
    }
    
    @Override
    public void onAcknowledgement(RecordMetadata metadata, Exception exception) {
        if (exception != null) {
            log.warn("Message send failed for topic {}: {}", metadata.topic(), exception.getMessage());
        } else {
            log.debug("Message sent successfully to topic {} partition {} offset {}", 
                     metadata.topic(), metadata.partition(), metadata.offset());
        }
    }
    
    @Override
    public void close() {
        if (executorService != null) {
            executorService.shutdown();
        }
        if (ocxClient != null) {
            ocxClient.close();
        }
        log.info("OCX Kafka interceptor closed");
    }
}
