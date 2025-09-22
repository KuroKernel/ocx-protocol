package dev.ocx.kafka;

import org.apache.kafka.clients.producer.ProducerRecord;
import org.apache.kafka.common.header.internals.RecordHeaders;

import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.TimeUnit;

public class BenchmarkRunner {
    public static void main(String[] args) {
        System.out.println("OCX Kafka Interceptor Benchmark");
        System.out.println("================================");
        
        // Create mock configuration
        Map<String, String> config = new HashMap<>();
        config.put("ocx.server.url", "http://localhost:8080");
        config.put("ocx.api.key", "test-key");
        config.put("ocx.async.mode", "false");
        config.put("ocx.fail.closed", "false");
        
        // Create interceptor
        OCXProducerInterceptor<String, String> interceptor = new OCXProducerInterceptor<>();
        interceptor.configure(config);
        
        // Create test record
        ProducerRecord<String, String> record = new ProducerRecord<>(
            "test-topic",
            "test-key",
            "test-value"
        );
        
        // Warm up
        System.out.println("Warming up...");
        for (int i = 0; i < 100; i++) {
            try {
                interceptor.onSend(record);
            } catch (Exception e) {
                // Expected for mock setup
            }
        }
        
        // Benchmark
        System.out.println("Running benchmark...");
        int iterations = 10000;
        long startTime = System.nanoTime();
        
        for (int i = 0; i < iterations; i++) {
            try {
                interceptor.onSend(record);
            } catch (Exception e) {
                // Expected for mock setup
            }
        }
        
        long endTime = System.nanoTime();
        long duration = endTime - startTime;
        
        double avgTime = (double) duration / iterations;
        double throughput = iterations / (duration / 1_000_000_000.0);
        
        System.out.printf("Results:\n");
        System.out.printf("  Iterations: %d\n", iterations);
        System.out.printf("  Total time: %.2f ms\n", duration / 1_000_000.0);
        System.out.printf("  Average time: %.2f μs\n", avgTime / 1000.0);
        System.out.printf("  Throughput: %.2f ops/sec\n", throughput);
        
        interceptor.close();
    }
}
