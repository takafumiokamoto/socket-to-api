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

// Use the embedded certs from client_with_embedded_certs.go
// (embeddedCACerts is already declared in that file)

type HTTPClientHybrid struct {
	client *http.Client
}

// NewHTTPClientHybrid creates an HTTP client that:
// 1. First tries to load system CA certificates
// 2. Falls back to embedded CA certificates if system certs aren't available
// 3. If both are available, it uses both (system certs take precedence)
func NewHTTPClientHybrid() (*HTTPClientHybrid, error) {
	// Start with system cert pool (may be nil on systems without certs)
	certPool, err := x509.SystemCertPool()
	if err != nil || certPool == nil {
		// System certs not available, create empty pool
		fmt.Println("System CA certificates not available, using embedded certs only")
		certPool = x509.NewCertPool()
	} else {
		fmt.Println("Using system CA certificates as base")
	}

	// Always append embedded certs as fallback/supplement
	if ok := certPool.AppendCertsFromPEM(embeddedCACerts); !ok {
		return nil, fmt.Errorf("failed to append embedded CA certificates")
	}
	fmt.Println("Added embedded CA certificates to pool")

	// Create custom TLS config with hybrid cert pool
	tlsConfig := &tls.Config{
		RootCAs: certPool,
	}

	// Create HTTP client with custom transport
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &HTTPClientHybrid{
		client: &http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
		},
	}, nil
}

// NewHTTPClientSystemWithFallback creates a client that explicitly checks system paths
// and falls back to embedded certs
func NewHTTPClientSystemWithFallback() (*HTTPClientHybrid, error) {
	certPool := x509.NewCertPool()
	systemCertsLoaded := false

	// Common system CA certificate paths (Linux/Unix)
	certPaths := []string{
		"/etc/ssl/certs/ca-certificates.crt",                // Debian/Ubuntu/Alpine
		"/etc/pki/tls/certs/ca-bundle.crt",                  // Fedora/RHEL/CentOS
		"/etc/ssl/ca-bundle.pem",                            // OpenSUSE
		"/etc/pki/tls/cacert.pem",                           // OpenELEC
		"/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem", // CentOS/RHEL 7+
		"/etc/ssl/cert.pem",                                 // Alpine (alternative)
	}

	// Try to load from system paths
	for _, path := range certPaths {
		if certs, err := os.ReadFile(path); err == nil {
			if ok := certPool.AppendCertsFromPEM(certs); ok {
				fmt.Printf("✓ Loaded system CA certificates from: %s\n", path)
				systemCertsLoaded = true
				break
			}
		}
	}

	if !systemCertsLoaded {
		fmt.Println("⚠ System CA certificates not found in standard paths")
	}

	// Always add embedded certs as fallback
	if ok := certPool.AppendCertsFromPEM(embeddedCACerts); !ok {
		return nil, fmt.Errorf("failed to append embedded CA certificates")
	}
	if systemCertsLoaded {
		fmt.Println("✓ Added embedded CA certificates as supplement")
	} else {
		fmt.Println("✓ Using embedded CA certificates as primary source")
	}

	// Create custom TLS config
	tlsConfig := &tls.Config{
		RootCAs: certPool,
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &HTTPClientHybrid{
		client: &http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
		},
	}, nil
}

// TestHTTPSRequest tests making an HTTPS request using hybrid certificates
func (c *HTTPClientHybrid) TestHTTPSRequest(url string) (map[string]interface{}, error) {
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTPS request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}
