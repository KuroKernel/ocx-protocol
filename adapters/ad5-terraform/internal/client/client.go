package client

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type OCXClient struct {
    ServerURL string
    APIKey    string
    Timeout   int64
    client    *http.Client
}

func NewOCXClient(serverURL, apiKey string, timeout int64) *OCXClient {
    return &OCXClient{
        ServerURL: serverURL,
        APIKey:    apiKey,
        Timeout:   timeout,
        client: &http.Client{
            Timeout: time.Duration(timeout) * time.Second,
        },
    }
}

func (c *OCXClient) ExecuteVerification(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {
    jsonData, err := json.Marshal(req)
    if err != nil {
        return nil, err
    }

    httpReq, err := http.NewRequestWithContext(ctx, "POST", c.ServerURL+"/api/v1/execute", bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, err
    }

    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

    resp, err := c.client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("OCX server returned status %d", resp.StatusCode)
    }

    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return result, nil
}

func (c *OCXClient) VerifyReceipt(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {
    jsonData, err := json.Marshal(req)
    if err != nil {
        return nil, err
    }

    httpReq, err := http.NewRequestWithContext(ctx, "POST", c.ServerURL+"/api/v1/verify", bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, err
    }

    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

    resp, err := c.client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("OCX server returned status %d", resp.StatusCode)
    }

    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return result, nil
}
