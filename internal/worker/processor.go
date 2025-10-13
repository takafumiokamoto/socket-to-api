package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/okamoto/socket-to-api/internal/api"
	"github.com/okamoto/socket-to-api/internal/database"
	"github.com/okamoto/socket-to-api/internal/models"
	"github.com/okamoto/socket-to-api/internal/server"
	"github.com/okamoto/socket-to-api/internal/transformer"
	"github.com/okamoto/socket-to-api/pkg/protocol"
	"go.uber.org/zap"
)

// Processor handles the processing of requests
type Processor struct {
	repo        *database.Repository
	apiClient   *api.Client
	transformer *transformer.Transformer
	connMgr     *server.ConnectionManager
	logger      *zap.Logger
}

// NewProcessor creates a new processor
func NewProcessor(
	repo *database.Repository,
	apiClient *api.Client,
	transformer *transformer.Transformer,
	connMgr *server.ConnectionManager,
	logger *zap.Logger,
) *Processor {
	return &Processor{
		repo:        repo,
		apiClient:   apiClient,
		transformer: transformer,
		connMgr:     connMgr,
		logger:      logger,
	}
}

// Process processes a single request through the entire pipeline
func (p *Processor) Process(ctx context.Context, request *models.UnsendData) *models.ProcessingResult {
	result := &models.ProcessingResult{
		RequestID:  request.ID,
		ClientPort: request.ClientPort,
		Success:    false,
	}

	// Step 1: Mark as sending
	if err := p.repo.MarkAsSending(ctx, request.ID); err != nil {
		result.Error = fmt.Errorf("failed to mark as sending: %w", err)
		p.handleError(ctx, request.ID, result.Error)
		return result
	}

	// Step 2: Validate binary data
	if err := p.transformer.ValidateBinaryData(request.BinaryData); err != nil {
		result.Error = fmt.Errorf("invalid binary data: %w", err)
		p.handleError(ctx, request.ID, result.Error)
		return result
	}

	// Step 3: Convert binary to API request
	apiRequest, err := p.transformer.BinaryToAPIRequest(request.ID, request.ClientPort, request.BinaryData)
	if err != nil {
		result.Error = fmt.Errorf("failed to convert to API request: %w", err)
		p.handleError(ctx, request.ID, result.Error)
		return result
	}

	// Step 4: Send API request
	apiResponse, err := p.apiClient.SendRequest(ctx, apiRequest)
	if err != nil {
		result.Error = fmt.Errorf("API request failed: %w", err)
		p.handleError(ctx, request.ID, result.Error)
		return result
	}

	// Step 5: Validate API response
	if err := p.transformer.ValidateAPIResponse(apiResponse); err != nil {
		result.Error = fmt.Errorf("invalid API response: %w", err)
		p.handleError(ctx, request.ID, result.Error)
		return result
	}

	// Step 6: Store API response in database
	dbResponse := &models.APIResponse{
		RequestID:    request.ID,
		StatusCode:   apiResponse.StatusCode,
		ResponseBody: []byte(apiResponse.Message), // Store the message
		CreatedAt:    time.Now(),
	}

	if err := p.repo.InsertAPIResponse(ctx, dbResponse); err != nil {
		p.logger.Error("failed to insert API response",
			zap.Int64("request_id", request.ID),
			zap.Error(err))
		// Continue processing even if DB insert fails
	}

	// Step 7: Convert API response to binary
	binaryResponse, err := p.transformer.APIResponseToBinary(apiResponse)
	if err != nil {
		result.Error = fmt.Errorf("failed to convert response to binary: %w", err)
		p.handleError(ctx, request.ID, result.Error)
		return result
	}

	result.Response = binaryResponse

	// Step 8: Send response to client via TCP
	if err := p.sendToClient(request.ClientPort, binaryResponse); err != nil {
		result.Error = fmt.Errorf("failed to send response to client: %w", err)
		p.handleError(ctx, request.ID, result.Error)
		return result
	}

	// Step 9: Mark as complete and cleanup
	if err := p.repo.MarkAsComplete(ctx, request.ID); err != nil {
		p.logger.Error("failed to mark as complete",
			zap.Int64("request_id", request.ID),
			zap.Error(err))
		// Continue to deletion even if status update fails
	}

	// Step 10: Delete processed record
	if err := p.repo.DeleteProcessedRequest(ctx, request.ID); err != nil {
		p.logger.Error("failed to delete processed request",
			zap.Int64("request_id", request.ID),
			zap.Error(err))
		// Don't fail the result if deletion fails
	}

	result.Success = true
	p.logger.Info("request processed successfully",
		zap.Int64("request_id", request.ID),
		zap.Int("client_port", request.ClientPort))

	return result
}

// sendToClient sends binary data to a client via TCP socket
func (p *Processor) sendToClient(clientPort int, data []byte) error {
	// Wrap data in protocol message
	message, err := protocol.EncodeMessage(protocol.MessageTypeResponse, data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Send to connection manager
	if err := p.connMgr.SendData(clientPort, message); err != nil {
		return fmt.Errorf("connection manager send failed: %w", err)
	}

	p.logger.Debug("sent response to client",
		zap.Int("client_port", clientPort),
		zap.Int("data_size", len(data)))

	return nil
}

// handleError handles processing errors
func (p *Processor) handleError(ctx context.Context, requestID int64, err error) {
	p.logger.Error("processing error",
		zap.Int64("request_id", requestID),
		zap.Error(err))

	// Mark request as failed in database
	errorMsg := err.Error()
	if dbErr := p.repo.MarkAsFailed(ctx, requestID, errorMsg); dbErr != nil {
		p.logger.Error("failed to mark request as failed",
			zap.Int64("request_id", requestID),
			zap.Error(dbErr))
	}

	// Increment retry count
	if dbErr := p.repo.IncrementRetryCount(ctx, requestID); dbErr != nil {
		p.logger.Error("failed to increment retry count",
			zap.Int64("request_id", requestID),
			zap.Error(dbErr))
	}
}

// ProcessBatch processes multiple requests in batch
func (p *Processor) ProcessBatch(ctx context.Context, requests []*models.UnsendData) []*models.ProcessingResult {
	results := make([]*models.ProcessingResult, len(requests))

	for i, req := range requests {
		results[i] = p.Process(ctx, req)
	}

	return results
}

// SendErrorToClient sends an error message to a client
func (p *Processor) SendErrorToClient(clientPort int, errorMsg string) error {
	binaryError, err := p.transformer.CreateErrorResponse(errorMsg)
	if err != nil {
		return fmt.Errorf("failed to create error response: %w", err)
	}

	return p.sendToClient(clientPort, binaryError)
}
