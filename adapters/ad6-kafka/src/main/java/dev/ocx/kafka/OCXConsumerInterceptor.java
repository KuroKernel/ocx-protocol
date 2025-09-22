package dev.ocx.kafka;

import org.apache.kafka.clients.consumer.ConsumerInterceptor;
import org.apache.kafka.clients.consumer.ConsumerRecord;
import org.apache.kafka.clients.consumer.ConsumerRecords;
import org.apache.kafka.clients.consumer.OffsetAndMetadata;
import org.apache.kafka.common.TopicPartition;
import org.apache.kafka.common.header.Header;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.nio.charset.StandardCharsets;
import java.util.Map;
import java.util.concurrent.atomic.AtomicLong;

public class OCXConsumerInterceptor<K, V> implements ConsumerInterceptor<K, V> {
    private static final Logger log = LoggerFactory.getLogger(OCXConsumerInterceptor.class);
    
    private OCXClient ocxClient;
    private boolean verifyReceipts;
    private boolean failOnInvalidReceipt;
    private final AtomicLong verifiedMessages = new AtomicLong(0);
    private final AtomicLong invalidMessages = new AtomicLong(0);
    
    @Override
    public void configure(Map<String, ?> configs) {
        String ocxServerUrl = (String) configs.get("ocx.server.url");
        String apiKey = (String) configs.get("ocx.api.key");
        Object verifyReceiptsObj = configs.get("ocx.verify.receipts");
        Object failOnInvalidObj = configs.get("ocx.fail.on.invalid");
        verifyReceipts = verifyReceiptsObj != null ? Boolean.parseBoolean(verifyReceiptsObj.toString()) : true;
        failOnInvalidReceipt = failOnInvalidObj != null ? Boolean.parseBoolean(failOnInvalidObj.toString()) : false;
        
        if (verifyReceipts && (ocxServerUrl == null || apiKey == null)) {
            throw new IllegalArgumentException("OCX server URL and API key must be configured for receipt verification");
        }
        
        if (verifyReceipts) {
            ocxClient = new OCXClient(ocxServerUrl, apiKey);
        }
        
        log.info("OCX Kafka consumer interceptor configured: verify={}, failOnInvalid={}", 
                 verifyReceipts, failOnInvalidReceipt);
    }
    
    @Override
    public ConsumerRecords<K, V> onConsume(ConsumerRecords<K, V> records) {
        if (!verifyReceipts) {
            return records;
        }
        
        for (ConsumerRecord<K, V> record : records) {
            try {
                verifyRecord(record);
                verifiedMessages.incrementAndGet();
            } catch (Exception e) {
                invalidMessages.incrementAndGet();
                log.warn("OCX verification failed for message from topic {} partition {} offset {}: {}", 
                        record.topic(), record.partition(), record.offset(), e.getMessage());
                
                if (failOnInvalidReceipt) {
                    throw new OCXVerificationException("Invalid OCX receipt", e);
                }
            }
        }
        
        return records;
    }
    
    private void verifyRecord(ConsumerRecord<K, V> record) throws Exception {
        Header receiptHeader = record.headers().lastHeader("ocx-receipt");
        if (receiptHeader == null) {
            if (failOnInvalidReceipt) {
                throw new OCXVerificationException("No OCX receipt found in message");
            }
            return;
        }
        
        Header verifiedHeader = record.headers().lastHeader("ocx-verified");
        if (verifiedHeader != null) {
            String verified = new String(verifiedHeader.value(), StandardCharsets.UTF_8);
            if ("false".equals(verified)) {
                throw new OCXVerificationException("Message marked as not verified");
            }
        }
        
        // Extract and verify receipt
        byte[] receiptData = receiptHeader.value();
        
        // Create expected message context
        MessageContext expectedContext = MessageContext.builder()
            .topic(record.topic())
            .partition(record.partition())
            .timestamp(record.timestamp())
            .keySize(record.key() != null ? record.key().toString().length() : 0)
            .valueSize(record.value() != null ? record.value().toString().length() : 0)
            .headerCount(record.headers().toArray().length - 1) // Exclude OCX headers
            .build();
        
        // Verify receipt with OCX server
        VerificationRequest request = VerificationRequest.builder()
            .receiptData(receiptData)
            .expectedContext(expectedContext.toJson())
            .build();
        
        boolean isValid = ocxClient.verifyReceipt(request);
        if (!isValid) {
            throw new OCXVerificationException("OCX receipt verification failed");
        }
    }
    
    @Override
    public void onCommit(Map<TopicPartition, OffsetAndMetadata> offsets) {
        log.debug("Committed offsets: verified={}, invalid={}", 
                 verifiedMessages.get(), invalidMessages.get());
    }
    
    @Override
    public void close() {
        if (ocxClient != null) {
            ocxClient.close();
        }
        log.info("OCX Consumer interceptor closed. Final stats: verified={}, invalid={}", 
                verifiedMessages.get(), invalidMessages.get());
    }
}
