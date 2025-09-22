package dev.ocx.kafka;

import com.fasterxml.jackson.databind.ObjectMapper;
import okhttp3.*;

import java.io.IOException;
import java.util.Map;
import java.util.concurrent.TimeUnit;

public class OCXClient implements AutoCloseable {
    private final OkHttpClient client;
    private final String serverUrl;
    private final String apiKey;
    private final ObjectMapper mapper;

    public OCXClient(String serverUrl, String apiKey) {
        this.serverUrl = serverUrl;
        this.apiKey = apiKey;
        this.mapper = new ObjectMapper();
        this.client = new OkHttpClient.Builder()
            .connectTimeout(5, TimeUnit.SECONDS)
            .writeTimeout(10, TimeUnit.SECONDS)
            .readTimeout(10, TimeUnit.SECONDS)
            .build();
    }

    public OCXReceipt executeVerification(VerificationRequest request) throws IOException {
        String json = mapper.writeValueAsString(request);
        
        RequestBody body = RequestBody.create(
            json, MediaType.get("application/json; charset=utf-8"));
        
        Request httpRequest = new Request.Builder()
            .url(serverUrl + "/api/v1/execute")
            .header("Authorization", "Bearer " + apiKey)
            .post(body)
            .build();

        try (Response response = client.newCall(httpRequest).execute()) {
            if (!response.isSuccessful()) {
                throw new IOException("Unexpected response: " + response);
            }
            
            return mapper.readValue(response.body().string(), OCXReceipt.class);
        }
    }

    public boolean verifyReceipt(VerificationRequest request) throws IOException {
        String json = mapper.writeValueAsString(request);
        
        RequestBody body = RequestBody.create(
            json, MediaType.get("application/json; charset=utf-8"));
        
        Request httpRequest = new Request.Builder()
            .url(serverUrl + "/api/v1/verify")
            .header("Authorization", "Bearer " + apiKey)
            .post(body)
            .build();

        try (Response response = client.newCall(httpRequest).execute()) {
            if (!response.isSuccessful()) {
                return false;
            }
            
            Map<String, Object> result = mapper.readValue(
                response.body().string(), Map.class);
            return Boolean.TRUE.equals(result.get("valid"));
        }
    }

    @Override
    public void close() {
        client.dispatcher().executorService().shutdown();
        client.connectionPool().evictAll();
    }
}