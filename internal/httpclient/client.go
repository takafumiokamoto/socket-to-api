package httpclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HTTPClient struct {
	client *http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// TestHTTPSRequest tests making an HTTPS request to a public API
func (c *HTTPClient) TestHTTPSRequest(url string) (map[string]interface{}, error) {
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
