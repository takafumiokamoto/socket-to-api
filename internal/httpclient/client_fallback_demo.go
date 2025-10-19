package httpclient

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

//go:embed certs/broken-ca-cert.pem
var brokenCACert []byte

type HTTPClientFallbackTest struct {
	brokenClient  *http.Client
	workingClient *http.Client
}

// NewHTTPClientFallbackTest creates a test client that demonstrates fallback
// from broken certs to working certs
func NewHTTPClientFallbackTest() (*HTTPClientFallbackTest, error) {
	client := &HTTPClientFallbackTest{}

	// 1. Create client with BROKEN embedded certs (will fail)
	brokenPool := x509.NewCertPool()
	brokenPool.AppendCertsFromPEM(brokenCACert)
	client.brokenClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: brokenPool},
		},
		Timeout: 10 * time.Second,
	}

	// 2. Create client with WORKING embedded certs (will succeed)
	workingPool := x509.NewCertPool()
	if !workingPool.AppendCertsFromPEM(embeddedCACerts) {
		return nil, fmt.Errorf("failed to load working CA certificates")
	}
	client.workingClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: workingPool},
		},
		Timeout: 10 * time.Second,
	}

	return client, nil
}

// TestFallbackMechanism demonstrates the fallback from broken to working certs
func (c *HTTPClientFallbackTest) TestFallbackMechanism(url string) (map[string]interface{}, string, error) {
	fmt.Println("\n  === Testing Fallback Mechanism ===")

	// Step 1: Try with BROKEN certs (should fail)
	fmt.Println("  [1/2] Attempting with BROKEN CA certificates...")
	result, err := c.makeRequest(c.brokenClient, url)
	if err != nil {
		fmt.Printf("  ✗ Expected failure with broken certs: %v\n", err)

		// Check if it's a cert error
		if isCertErrorFallbackTest(err) {
			fmt.Println("  → Certificate error detected, falling back...")

			// Step 2: Fallback to WORKING certs (should succeed)
			fmt.Println("  [2/2] Falling back to WORKING CA certificates...")
			result, err = c.makeRequest(c.workingClient, url)
			if err != nil {
				return nil, "", fmt.Errorf("fallback also failed: %w", err)
			}
			fmt.Println("  ✓ SUCCESS! Fallback to working certs succeeded!")
			return result, "working-certs-after-fallback", nil
		}

		return nil, "", fmt.Errorf("non-certificate error, cannot fallback: %w", err)
	}

	// Unexpected: broken certs worked (shouldn't happen)
	fmt.Println("  ⚠ Unexpected: broken certs succeeded!")
	return result, "broken-certs-unexpectedly-worked", nil
}

func (c *HTTPClientFallbackTest) makeRequest(client *http.Client, url string) (map[string]interface{}, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}

	return result, nil
}

// isCertError checks if the error is related to certificate verification
// (reusing the function from client_smart_fallback.go)
func isCertErrorFallbackTest(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	certErrors := []string{
		"certificate",
		"x509",
		"tls",
		"unknown authority",
		"certificate signed by unknown authority",
		"certificate has expired",
		"certificate is not valid",
	}

	for _, certErr := range certErrors {
		if containsStr(errStr, certErr) {
			return true
		}
	}
	return false
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
		indexOfStr(s, substr) >= 0)
}

func indexOfStr(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
