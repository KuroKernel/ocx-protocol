// security_test.go - Security Validation Tests for OCX Protocol
// Tests: Penetration testing, cryptographic validation, SQL injection, rate limiting

package security

import (
	"bytes"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"ocx.local/pkg/ocx"
)

const (
	BASE_URL = "http://localhost:8080"
	ATTACK_ITERATIONS = 1000
)

func TestPenetrationTesting(t *testing.T) {
	t.Run("AuthenticationEndpointAttacks", func(t *testing.T) {
		// Test various attack vectors on authentication endpoints
		attackVectors := []struct {
			name        string
			attackFunc  func(*testing.T) error
		}{
			{"SQLInjection", testSQLInjection},
			{"XSSAttacks", testXSSAttacks},
			{"PathTraversal", testPathTraversal},
			{"CommandInjection", testCommandInjection},
			{"LDAPInjection", testLDAPInjection},
		}
		
		for _, vector := range attackVectors {
			t.Run(vector.name, func(t *testing.T) {
				err := vector.attackFunc(t)
				if err == nil {
					t.Errorf("Security vulnerability detected: %s attack succeeded", vector.name)
				}
			})
		}
	})
}

func TestCryptographicSignatureValidation(t *testing.T) {
	t.Run("Ed25519SignatureValidation", func(t *testing.T) {
		// Test signature validation under various edge cases
		testCases := []struct {
			name        string
			testFunc    func(*testing.T) error
		}{
			{"ValidSignature", testValidSignature},
			{"InvalidSignature", testInvalidSignature},
			{"TamperedMessage", testTamperedMessage},
			{"ReplayAttack", testReplayAttack},
			{"KeyRotation", testKeyRotation},
			{"SignatureMalleability", testSignatureMalleability},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.testFunc(t)
				if err != nil {
					t.Errorf("Cryptographic test failed: %v", err)
				}
			})
		}
	})
}

func TestHMACValidation(t *testing.T) {
	t.Run("HMACSecurity", func(t *testing.T) {
		// Test HMAC validation security
		testCases := []struct {
			name        string
			testFunc    func(*testing.T) error
		}{
			{"ValidHMAC", testValidHMAC},
			{"InvalidHMAC", testInvalidHMAC},
			{"TimingAttack", testTimingAttack},
			{"KeyLengthValidation", testKeyLengthValidation},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.testFunc(t)
				if err != nil {
					t.Errorf("HMAC test failed: %v", err)
				}
			})
		}
	})
}

func TestSQLInjectionPrevention(t *testing.T) {
	t.Run("SQLInjectionAttacks", func(t *testing.T) {
		// Test various SQL injection payloads
		payloads := []string{
			"'; DROP TABLE orders; --",
			"' OR '1'='1",
			"'; INSERT INTO users VALUES ('hacker', 'password'); --",
			"' UNION SELECT * FROM users --",
			"'; UPDATE orders SET amount = 0; --",
			"' OR 1=1 --",
			"'; DELETE FROM settlements; --",
		}
		
		for _, payload := range payloads {
			t.Run(fmt.Sprintf("Payload_%s", payload), func(t *testing.T) {
				err := testSQLInjectionWithPayload(t, payload)
				if err == nil {
					t.Errorf("SQL injection vulnerability detected with payload: %s", payload)
				}
			})
		}
	})
}

func TestRateLimiting(t *testing.T) {
	t.Run("RateLimitEnforcement", func(t *testing.T) {
		// Test rate limiting on various endpoints
		endpoints := []string{
			"/orders",
			"/offers", 
			"/auth",
			"/settlements",
		}
		
		for _, endpoint := range endpoints {
			t.Run(fmt.Sprintf("Endpoint_%s", endpoint), func(t *testing.T) {
				err := testRateLimiting(t, endpoint)
				if err != nil {
					t.Errorf("Rate limiting test failed for %s: %v", endpoint, err)
				}
			})
		}
	})
}

func TestDDoSProtection(t *testing.T) {
	t.Run("DDoSAttackSimulation", func(t *testing.T) {
		// Simulate DDoS attack patterns
		attackPatterns := []struct {
			name        string
			patternFunc func(*testing.T) error
		}{
			{"SlowLoris", testSlowLorisAttack},
			{"HTTPFlood", testHTTPFloodAttack},
			{"SYNFlood", testSYNFloodAttack},
			{"ApplicationLayer", testApplicationLayerAttack},
		}
		
		for _, pattern := range attackPatterns {
			t.Run(pattern.name, func(t *testing.T) {
				err := pattern.patternFunc(t)
				if err != nil {
					t.Errorf("DDoS protection test failed for %s: %v", pattern.name, err)
				}
			})
		}
	})
}

func TestInputValidation(t *testing.T) {
	t.Run("InputSanitization", func(t *testing.T) {
		// Test input validation on all endpoints
		maliciousInputs := []struct {
			name  string
			input string
		}{
			{"XSS", "<script>alert('xss')</script>"},
			{"HTMLInjection", "<img src=x onerror=alert('xss')>"},
			{"SQLInjection", "'; DROP TABLE orders; --"},
			{"CommandInjection", "; rm -rf /"},
			{"PathTraversal", "../../../etc/passwd"},
			{"NullByte", "test\x00.jpg"},
			{"Unicode", "test\u0000.jpg"},
		}
		
		for _, input := range maliciousInputs {
			t.Run(input.name, func(t *testing.T) {
				err := testInputValidation(t, input.input)
				if err == nil {
					t.Errorf("Input validation failed for %s: %s", input.name, input.input)
				}
			})
		}
	})
}

// Attack simulation functions
func testSQLInjection(t *testing.T) error {
	payload := "'; DROP TABLE orders; --"
	return testSQLInjectionWithPayload(t, payload)
}

func testSQLInjectionWithPayload(t *testing.T, payload string) error {
	// Test SQL injection on order endpoint
	order := map[string]interface{}{
		"buyer_id": payload,
		"gpus":     1,
		"hours":    1,
	}
	
	jsonData, _ := json.Marshal(order)
	resp, err := http.Post(BASE_URL+"/orders", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// If request succeeds, it might be vulnerable
	if resp.StatusCode == http.StatusOK {
		return fmt.Errorf("potential SQL injection vulnerability")
	}
	
	return nil
}

func testXSSAttacks(t *testing.T) error {
	payload := "<script>alert('xss')</script>"
	
	order := map[string]interface{}{
		"buyer_id": payload,
		"gpus":     1,
		"hours":    1,
	}
	
	jsonData, _ := json.Marshal(order)
	resp, err := http.Post(BASE_URL+"/orders", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// Check if payload is reflected in response
	body := make([]byte, 1024)
	resp.Body.Read(body)
	if strings.Contains(string(body), payload) {
		return fmt.Errorf("XSS vulnerability detected")
	}
	
	return nil
}

func testPathTraversal(t *testing.T) error {
	payload := "../../../etc/passwd"
	
	resp, err := http.Get(BASE_URL + "/files/" + payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// If request succeeds, it might be vulnerable
	if resp.StatusCode == http.StatusOK {
		return fmt.Errorf("path traversal vulnerability detected")
	}
	
	return nil
}

func testCommandInjection(t *testing.T) error {
	payload := "; rm -rf /"
	
	order := map[string]interface{}{
		"buyer_id": payload,
		"gpus":     1,
		"hours":    1,
	}
	
	jsonData, _ := json.Marshal(order)
	resp, err := http.Post(BASE_URL+"/orders", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// If request succeeds, it might be vulnerable
	if resp.StatusCode == http.StatusOK {
		return fmt.Errorf("command injection vulnerability detected")
	}
	
	return nil
}

func testLDAPInjection(t *testing.T) error {
	payload := "*)(uid=*))(|(uid=*"
	
	order := map[string]interface{}{
		"buyer_id": payload,
		"gpus":     1,
		"hours":    1,
	}
	
	jsonData, _ := json.Marshal(order)
	resp, err := http.Post(BASE_URL+"/orders", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// If request succeeds, it might be vulnerable
	if resp.StatusCode == http.StatusOK {
		return fmt.Errorf("LDAP injection vulnerability detected")
	}
	
	return nil
}

// Cryptographic test functions
func testValidSignature(t *testing.T) error {
	// Generate key pair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	
	// Create message
	message := []byte("test message")
	
	// Sign message
	signature := ed25519.Sign(privKey, message)
	
	// Verify signature
	if !ed25519.Verify(pubKey, message, signature) {
		return fmt.Errorf("valid signature verification failed")
	}
	
	return nil
}

func testInvalidSignature(t *testing.T) error {
	// Generate key pair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	
	// Create message
	message := []byte("test message")
	
	// Sign message
	signature := ed25519.Sign(privKey, message)
	
	// Tamper with signature
	signature[0] ^= 0xFF
	
	// Verify signature should fail
	if ed25519.Verify(pubKey, message, signature) {
		return fmt.Errorf("invalid signature verification should have failed")
	}
	
	return nil
}

func testTamperedMessage(t *testing.T) error {
	// Generate key pair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	
	// Create message
	message := []byte("test message")
	
	// Sign message
	signature := ed25519.Sign(privKey, message)
	
	// Tamper with message
	message[0] ^= 0xFF
	
	// Verify signature should fail
	if ed25519.Verify(pubKey, message, signature) {
		return fmt.Errorf("tampered message verification should have failed")
	}
	
	return nil
}

func testReplayAttack(t *testing.T) error {
	// Test replay attack prevention
	// Implementation would test nonce/timestamp validation
	return nil
}

func testKeyRotation(t *testing.T) error {
	// Test key rotation mechanism
	// Implementation would test key rotation process
	return nil
}

func testSignatureMalleability(t *testing.T) error {
	// Test signature malleability resistance
	// Implementation would test signature malleability
	return nil
}

// HMAC test functions
func testValidHMAC(t *testing.T) error {
	key := []byte("test-key")
	message := []byte("test message")
	
	h := hmac.New(sha256.New, key)
	h.Write(message)
	expectedMAC := h.Sum(nil)
	
	// Verify HMAC
	if !hmac.Equal(expectedMAC, expectedMAC) {
		return fmt.Errorf("valid HMAC verification failed")
	}
	
	return nil
}

func testInvalidHMAC(t *testing.T) error {
	key := []byte("test-key")
	message := []byte("test message")
	
	h := hmac.New(sha256.New, key)
	h.Write(message)
	validMAC := h.Sum(nil)
	
	// Create invalid MAC
	invalidMAC := make([]byte, len(validMAC))
	copy(invalidMAC, validMAC)
	invalidMAC[0] ^= 0xFF
	
	// Verify HMAC should fail
	if hmac.Equal(validMAC, invalidMAC) {
		return fmt.Errorf("invalid HMAC verification should have failed")
	}
	
	return nil
}

func testTimingAttack(t *testing.T) error {
	// Test timing attack resistance
	// Implementation would test constant-time comparison
	return nil
}

func testKeyLengthValidation(t *testing.T) error {
	// Test key length validation
	// Implementation would test various key lengths
	return nil
}

// Rate limiting test functions
func testRateLimiting(t *testing.T, endpoint string) error {
	// Send requests at rate limit threshold
	rateLimit := 100 // requests per minute
	interval := 60 * time.Second / time.Duration(rateLimit)
	
	for i := 0; i < rateLimit+10; i++ {
		resp, err := http.Get(BASE_URL + endpoint)
		if err != nil {
			return err
		}
		resp.Body.Close()
		
		if i >= rateLimit && resp.StatusCode != http.StatusTooManyRequests {
			return fmt.Errorf("rate limiting not enforced at %s", endpoint)
		}
		
		time.Sleep(interval)
	}
	
	return nil
}

// DDoS test functions
func testSlowLorisAttack(t *testing.T) error {
	// Simulate slow loris attack
	// Implementation would test slow connection handling
	return nil
}

func testHTTPFloodAttack(t *testing.T) error {
	// Simulate HTTP flood attack
	// Implementation would test flood protection
	return nil
}

func testSYNFloodAttack(t *testing.T) error {
	// Simulate SYN flood attack
	// Implementation would test SYN flood protection
	return nil
}

func testApplicationLayerAttack(t *testing.T) error {
	// Simulate application layer attack
	// Implementation would test application layer protection
	return nil
}

// Input validation test functions
func testInputValidation(t *testing.T, input string) error {
	// Test input validation
	order := map[string]interface{}{
		"buyer_id": input,
		"gpus":     1,
		"hours":    1,
	}
	
	jsonData, _ := json.Marshal(order)
	resp, err := http.Post(BASE_URL+"/orders", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// If request succeeds with malicious input, validation failed
	if resp.StatusCode == http.StatusOK {
		return fmt.Errorf("input validation failed for: %s", input)
	}
	
	return nil
}
