package httpclient

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type HTTPClientSmartFallback struct {
	embeddedClient *http.Client
	systemClient   *http.Client
	hybridClient   *http.Client
}

// NewHTTPClientSmartFallback creates a client that tries multiple CA cert sources
// with intelligent fallback based on actual verification results
func NewHTTPClientSmartFallback() (*HTTPClientSmartFallback, error) {
	client := &HTTPClientSmartFallback{}

	// 1. Create client with embedded certs only
	embeddedPool := x509.NewCertPool()
	if !embeddedPool.AppendCertsFromPEM(embeddedCACerts) {
		return nil, fmt.Errorf("failed to load embedded CA certificates")
	}
	client.embeddedClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: embeddedPool},
		},
		Timeout: 10 * time.Second,
	}

	// 2. Create client with system certs only (if available)
	systemPool, err := loadSystemCerts()
	if err == nil && systemPool != nil {
		client.systemClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{RootCAs: systemPool},
			},
			Timeout: 10 * time.Second,
		}
	}

	// 3. Create hybrid client (system + embedded)
	hybridPool, _ := x509.SystemCertPool()
	if hybridPool == nil {
		hybridPool = x509.NewCertPool()
	}
	hybridPool.AppendCertsFromPEM(embeddedCACerts)
	client.hybridClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: hybridPool},
		},
		Timeout: 10 * time.Second,
	}

	return client, nil
}

// loadSystemCerts tries to load system CA certificates from common paths
func loadSystemCerts() (*x509.CertPool, error) {
	certPaths := []string{
		"/etc/ssl/certs/ca-certificates.crt",
		"/etc/pki/tls/certs/ca-bundle.crt",
		"/etc/ssl/ca-bundle.pem",
		"/etc/pki/tls/cacert.pem",
		"/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem",
		"/etc/ssl/cert.pem",
	}

	pool := x509.NewCertPool()
	for _, path := range certPaths {
		if certs, err := os.ReadFile(path); err == nil {
			if pool.AppendCertsFromPEM(certs) {
				return pool, nil
			}
		}
	}
	return nil, fmt.Errorf("no system certificates found")
}

// TestHTTPSRequestWithSmartFallback tries multiple CA cert sources with intelligent fallback
func (c *HTTPClientSmartFallback) TestHTTPSRequestWithSmartFallback(url string) (map[string]interface{}, error) {
	var lastErr error

	// Strategy 1: Try hybrid first (best chance of success)
	fmt.Println("  → Attempting with hybrid CA certs (system + embedded)...")
	result, err := c.makeRequest(c.hybridClient, url)
	if err == nil {
		fmt.Println("  ✓ Success with hybrid CA certs!")
		return result, nil
	}
	lastErr = err
	fmt.Printf("  ✗ Hybrid failed: %v\n", err)

	// Strategy 2: Try system certs only (if available)
	if c.systemClient != nil {
		fmt.Println("  → Attempting with system CA certs only...")
		result, err := c.makeRequest(c.systemClient, url)
		if err == nil {
			fmt.Println("  ✓ Success with system CA certs!")
			return result, nil
		}
		lastErr = err
		fmt.Printf("  ✗ System certs failed: %v\n", err)
	}

	// Strategy 3: Try embedded certs only (final fallback)
	fmt.Println("  → Attempting with embedded CA certs only...")
	result, err = c.makeRequest(c.embeddedClient, url)
	if err == nil {
		fmt.Println("  ✓ Success with embedded CA certs!")
		return result, nil
	}
	fmt.Printf("  ✗ Embedded certs failed: %v\n", err)

	// All strategies failed
	return nil, fmt.Errorf("all CA cert strategies failed, last error: %w", lastErr)
}

// makeRequest executes the HTTP request with the given client
func (c *HTTPClientSmartFallback) makeRequest(client *http.Client, url string) (map[string]interface{}, error) {
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

// TestHTTPSRequestWithRetry tries a request and falls back to different cert pools on TLS errors
func (c *HTTPClientSmartFallback) TestHTTPSRequestWithRetry(url string) (result map[string]interface{}, certSource string, err error) {
	// Try 1: Hybrid (system + embedded)
	result, err = c.makeRequest(c.hybridClient, url)
	if err == nil {
		return result, "hybrid (system + embedded)", nil
	}

	// Check if it's a certificate verification error
	if !isCertError(err) {
		return nil, "", fmt.Errorf("non-certificate error: %w", err)
	}

	// Try 2: System only
	if c.systemClient != nil {
		result, err = c.makeRequest(c.systemClient, url)
		if err == nil {
			return result, "system only", nil
		}
	}

	// Try 3: Embedded only
	result, err = c.makeRequest(c.embeddedClient, url)
	if err == nil {
		return result, "embedded only", nil
	}

	return nil, "", fmt.Errorf("all certificate sources failed: %w", err)
}

// isCertError checks if the error is related to certificate verification
func isCertError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Check for common certificate-related error messages
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
		if contains(errStr, certErr) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
		indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
