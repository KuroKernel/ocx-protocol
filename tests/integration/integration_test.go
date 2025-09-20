package integration

import (
    "bytes"
    "encoding/json"
    "net/http"
    "testing"
)

func TestHealthEndpoint(t *testing.T) {
    resp, err := http.Get("http://localhost:8080/health")
    if err != nil {
        t.Skip("Server not running, skipping integration test")
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        t.Errorf("Expected status 200, got %d", resp.StatusCode)
    }
}

func TestExecuteAPI(t *testing.T) {
    payload := map[string]interface{}{
        "artifact_hex": "00",
        "input_hex":    "01", 
        "max_cycles":   1000,
    }
    
    jsonData, _ := json.Marshal(payload)
    resp, err := http.Post("http://localhost:8080/api/v1/execute", 
        "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        t.Skip("Server not running, skipping integration test")
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        t.Errorf("Expected status 200, got %d", resp.StatusCode)
    }
}
