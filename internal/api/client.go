package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/okamoto/socket-to-api/internal/config"
	"github.com/okamoto/socket-to-api/internal/models"
	"go.uber.org/zap"
)

// Client represents an HTTPS API client
type Client struct {
	config     *config.APIConfig
	httpClient *http.Client
	logger     *zap.Logger
}

// NewClient creates a new API client
func NewClient(cfg *config.APIConfig, logger *zap.Logger) *Client {
	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	// Allow insecure TLS if configured (not recommended for production)
	if cfg.TLSInsecureSkip {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
		logger.Warn("TLS certificate verification is disabled")
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
	}

	return &Client{
		config:     cfg,
		httpClient: httpClient,
		logger:     logger,
	}
}

// SendRequest sends a request to the external API
func (c *Client) SendRequest(ctx context.Context, request *models.APIRequest) (*models.APIResponsePayload, error) {
	// Marshal request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	c.logger.Debug("sending API request",
		zap.String("request_id", request.RequestID),
		zap.Int("client_port", request.ClientPort))

	// Send request with retries
	var response *models.APIResponsePayload
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.config.RetryDelay * time.Duration(attempt)):
			}

			c.logger.Info("retrying API request",
				zap.String("request_id", request.RequestID),
				zap.Int("attempt", attempt))
		}

		response, lastErr = c.sendRequestOnce(ctx, requestBody)
		if lastErr == nil {
			return response, nil
		}

		c.logger.Warn("API request failed",
			zap.String("request_id", request.RequestID),
			zap.Int("attempt", attempt),
			zap.Error(lastErr))
	}

	return nil, fmt.Errorf("API request failed after %d attempts: %w", c.config.MaxRetries+1, lastErr)
}

// sendRequestOnce sends a single API request without retries
func (c *Client) sendRequestOnce(ctx context.Context, requestBody []byte) (*models.APIResponsePayload, error) {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "socket-to-api-bridge/1.0")

	// Send request
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(startTime)
	c.logger.Debug("API request completed",
		zap.Int("status_code", resp.StatusCode),
		zap.Duration("duration", duration))

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API returned error status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Parse response
	var apiResponse models.APIResponsePayload
	if err := json.Unmarshal(responseBody, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &apiResponse, nil
}

// HealthCheck performs a health check on the API
func (c *Client) HealthCheck(ctx context.Context) error {
	// Create a simple GET request to the base URL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.config.BaseURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read and discard body
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 500 {
		return fmt.Errorf("API health check failed with status %d", resp.StatusCode)
	}

	c.logger.Debug("API health check passed", zap.Int("status_code", resp.StatusCode))
	return nil
}

// Close closes the API client and releases resources
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	c.logger.Info("API client closed")
	return nil
}
