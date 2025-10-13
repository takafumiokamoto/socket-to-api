package transformer

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/okamoto/socket-to-api/internal/models"
	"go.uber.org/zap"
)

// Transformer handles data conversion between binary and JSON formats
type Transformer struct {
	logger *zap.Logger
}

// NewTransformer creates a new transformer
func NewTransformer(logger *zap.Logger) *Transformer {
	return &Transformer{
		logger: logger,
	}
}

// BinaryToAPIRequest converts binary data to an API request
func (t *Transformer) BinaryToAPIRequest(requestID int64, clientPort int, binaryData []byte) (*models.APIRequest, error) {
	if len(binaryData) == 0 {
		return nil, fmt.Errorf("binary data is empty")
	}

	// Parse binary data structure
	// This is a placeholder implementation - adjust based on actual binary format
	data, err := t.parseBinaryData(binaryData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse binary data: %w", err)
	}

	apiRequest := &models.APIRequest{
		RequestID:  fmt.Sprintf("%d", requestID),
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		ClientPort: clientPort,
		Data:       data,
	}

	t.logger.Debug("converted binary to API request",
		zap.Int64("request_id", requestID),
		zap.Int("client_port", clientPort),
		zap.Int("binary_size", len(binaryData)))

	return apiRequest, nil
}

// APIResponseToBinary converts an API response to binary format
func (t *Transformer) APIResponseToBinary(response *models.APIResponsePayload) ([]byte, error) {
	if response == nil {
		return nil, fmt.Errorf("API response is nil")
	}

	// Convert the response data to binary format
	// This is a placeholder implementation - adjust based on actual binary format
	binaryData, err := t.encodeToBinary(response)
	if err != nil {
		return nil, fmt.Errorf("failed to encode to binary: %w", err)
	}

	t.logger.Debug("converted API response to binary",
		zap.String("request_id", response.RequestID),
		zap.Int("binary_size", len(binaryData)))

	return binaryData, nil
}

// parseBinaryData parses binary data into a map structure
// This is a placeholder implementation that should be customized based on
// the actual binary protocol used by the legacy system
func (t *Transformer) parseBinaryData(binaryData []byte) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Example parsing logic - customize based on actual protocol
	// This implementation assumes a simple structure for demonstration

	if len(binaryData) < 4 {
		return nil, fmt.Errorf("binary data too short: %d bytes", len(binaryData))
	}

	// Option 1: If binary data contains structured fields
	// Extract fields based on known offsets and types
	// Example:
	// messageType := binaryData[0]
	// length := binary.BigEndian.Uint16(binaryData[1:3])
	// payload := binaryData[4:]

	// Option 2: If binary data should be sent as-is (base64 encoded)
	// This is useful when the API expects the raw binary data
	result["binary_data"] = base64.StdEncoding.EncodeToString(binaryData)
	result["data_length"] = len(binaryData)

	// Option 3: If binary data contains JSON
	// Try to parse as JSON first
	var jsonData map[string]interface{}
	if err := json.Unmarshal(binaryData, &jsonData); err == nil {
		// Binary data was actually JSON
		return jsonData, nil
	}

	// Add metadata
	result["encoding"] = "base64"
	result["format"] = "binary"

	return result, nil
}

// encodeToBinary encodes an API response to binary format
// This is a placeholder implementation that should be customized based on
// the actual binary protocol expected by the legacy system
func (t *Transformer) encodeToBinary(response *models.APIResponsePayload) ([]byte, error) {
	// Option 1: If legacy system expects structured binary
	// Build binary message with proper structure
	// Example:
	// buf := new(bytes.Buffer)
	// binary.Write(buf, binary.BigEndian, uint8(responseType))
	// binary.Write(buf, binary.BigEndian, uint32(len(payload)))
	// buf.Write(payload)
	// return buf.Bytes(), nil

	// Option 2: If response contains base64-encoded binary data
	if response.Data != nil {
		if binaryStr, ok := response.Data["binary_data"].(string); ok {
			decoded, err := base64.StdEncoding.DecodeString(binaryStr)
			if err != nil {
				return nil, fmt.Errorf("failed to decode base64 data: %w", err)
			}
			return decoded, nil
		}
	}

	// Option 3: Encode the entire response as JSON and convert to binary
	// This is a simple approach that works for many cases
	jsonData, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response to JSON: %w", err)
	}

	// Create a simple binary wrapper
	// Format: [Length:4 bytes][JSON data]
	binaryData := make([]byte, 4+len(jsonData))
	binary.BigEndian.PutUint32(binaryData[0:4], uint32(len(jsonData)))
	copy(binaryData[4:], jsonData)

	return binaryData, nil
}

// ValidateBinaryData validates binary data before processing
func (t *Transformer) ValidateBinaryData(binaryData []byte) error {
	if len(binaryData) == 0 {
		return fmt.Errorf("binary data is empty")
	}

	// Add custom validation logic based on protocol requirements
	// Example:
	// - Check minimum length
	// - Validate magic bytes
	// - Check checksum
	// - Validate message type

	const maxSize = 10 * 1024 * 1024 // 10MB
	if len(binaryData) > maxSize {
		return fmt.Errorf("binary data exceeds maximum size: %d bytes", len(binaryData))
	}

	return nil
}

// ValidateAPIResponse validates an API response before conversion
func (t *Transformer) ValidateAPIResponse(response *models.APIResponsePayload) error {
	if response == nil {
		return fmt.Errorf("API response is nil")
	}

	if response.RequestID == "" {
		return fmt.Errorf("API response missing request_id")
	}

	if response.Status == "" {
		return fmt.Errorf("API response missing status")
	}

	return nil
}

// ExtractRequestID extracts request ID from binary data if embedded
// This is useful if the binary protocol includes request tracking
func (t *Transformer) ExtractRequestID(binaryData []byte) (string, error) {
	// Placeholder implementation
	// Customize based on actual protocol

	if len(binaryData) < 8 {
		return "", fmt.Errorf("binary data too short to contain request ID")
	}

	// Example: if first 8 bytes are request ID
	requestID := binary.BigEndian.Uint64(binaryData[0:8])
	return fmt.Sprintf("%d", requestID), nil
}

// CreateErrorResponse creates a binary error response for the client
func (t *Transformer) CreateErrorResponse(errorMsg string) ([]byte, error) {
	errorResponse := map[string]interface{}{
		"status":    "error",
		"message":   errorMsg,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(errorResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal error response: %w", err)
	}

	// Wrap in binary format
	binaryData := make([]byte, 4+len(jsonData))
	binary.BigEndian.PutUint32(binaryData[0:4], uint32(len(jsonData)))
	copy(binaryData[4:], jsonData)

	return binaryData, nil
}
