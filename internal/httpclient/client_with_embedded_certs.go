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

//go:embed certs/ca-certificates.crt
var embeddedCACerts []byte

type HTTPClientWithEmbeddedCerts struct {
	client *http.Client
}

func NewHTTPClientWithEmbeddedCerts() (*HTTPClientWithEmbeddedCerts, error) {
	// Create a custom CA cert pool
	certPool := x509.NewCertPool()

	// Append embedded CA certs to the pool
	if !certPool.AppendCertsFromPEM(embeddedCACerts) {
		return nil, fmt.Errorf("failed to append embedded CA certificates")
	}

	// Create custom TLS config with embedded certs
	tlsConfig := &tls.Config{
		RootCAs: certPool,
	}

	// Create HTTP client with custom transport
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &HTTPClientWithEmbeddedCerts{
		client: &http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
		},
	}, nil
}

// TestHTTPSRequest tests making an HTTPS request using embedded certificates
func (c *HTTPClientWithEmbeddedCerts) TestHTTPSRequest(url string) (map[string]interface{}, error) {
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
