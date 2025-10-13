package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/okamoto/socket-to-api/internal/models"
	"go.uber.org/zap"
)

// Repository handles database operations for the bridge system
type Repository struct {
	db     *OracleDB
	logger *zap.Logger
}

// NewRepository creates a new repository
func NewRepository(db *OracleDB, logger *zap.Logger) *Repository {
	return &Repository{
		db:     db,
		logger: logger,
	}
}

// FetchPendingRequests fetches and locks pending requests from the database
func (r *Repository) FetchPendingRequests(ctx context.Context, limit int) ([]*models.UnsendData, error) {
	query := fmt.Sprintf(`
		SELECT id, client_port, binary_data, status, created_at, updated_at, retry_count, last_error
		FROM %s
		WHERE status = :1
		ORDER BY created_at ASC
		FETCH FIRST :2 ROWS ONLY
		FOR UPDATE SKIP LOCKED
	`, r.db.config.TableName)

	rows, err := r.db.db.QueryContext(ctx, query, models.StatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pending requests: %w", err)
	}
	defer rows.Close()

	var requests []*models.UnsendData
	for rows.Next() {
		req := &models.UnsendData{}
		var lastError sql.NullString

		err := rows.Scan(
			&req.ID,
			&req.ClientPort,
			&req.BinaryData,
			&req.Status,
			&req.CreatedAt,
			&req.UpdatedAt,
			&req.RetryCount,
			&lastError,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan request: %w", err)
		}

		if lastError.Valid {
			req.LastError = &lastError.String
		}

		requests = append(requests, req)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	r.logger.Debug("fetched pending requests", zap.Int("count", len(requests)))
	return requests, nil
}

// UpdateRequestStatus updates the status of a request
func (r *Repository) UpdateRequestStatus(ctx context.Context, requestID int64, status models.RequestStatus, errorMsg *string) error {
	query := fmt.Sprintf(`
		UPDATE %s
		SET status = :1, updated_at = :2, last_error = :3
		WHERE id = :4
	`, r.db.config.TableName)

	now := time.Now()
	_, err := r.db.db.ExecContext(ctx, query, status, now, errorMsg, requestID)
	if err != nil {
		return fmt.Errorf("failed to update request status: %w", err)
	}

	r.logger.Debug("updated request status",
		zap.Int64("request_id", requestID),
		zap.String("status", string(status)))

	return nil
}

// IncrementRetryCount increments the retry count for a request
func (r *Repository) IncrementRetryCount(ctx context.Context, requestID int64) error {
	query := fmt.Sprintf(`
		UPDATE %s
		SET retry_count = retry_count + 1, updated_at = :1
		WHERE id = :2
	`, r.db.config.TableName)

	_, err := r.db.db.ExecContext(ctx, query, time.Now(), requestID)
	if err != nil {
		return fmt.Errorf("failed to increment retry count: %w", err)
	}

	return nil
}

// InsertAPIResponse inserts an API response into the database
func (r *Repository) InsertAPIResponse(ctx context.Context, response *models.APIResponse) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (request_id, status_code, response_body, created_at)
		VALUES (:1, :2, :3, :4)
	`, r.db.config.ResponseTable)

	_, err := r.db.db.ExecContext(ctx,
		query,
		response.RequestID,
		response.StatusCode,
		response.ResponseBody,
		response.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert API response: %w", err)
	}

	r.logger.Debug("inserted API response",
		zap.Int64("request_id", response.RequestID),
		zap.Int("status_code", response.StatusCode))

	return nil
}

// DeleteProcessedRequest deletes a processed request from the database
func (r *Repository) DeleteProcessedRequest(ctx context.Context, requestID int64) error {
	query := fmt.Sprintf(`
		DELETE FROM %s WHERE id = :1
	`, r.db.config.TableName)

	result, err := r.db.db.ExecContext(ctx, query, requestID)
	if err != nil {
		return fmt.Errorf("failed to delete processed request: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("no request found with id %d", requestID)
	}

	r.logger.Debug("deleted processed request", zap.Int64("request_id", requestID))
	return nil
}

// InsertRequest inserts a new request into the database
func (r *Repository) InsertRequest(ctx context.Context, clientPort int, binaryData []byte) (int64, error) {
	query := fmt.Sprintf(`
		INSERT INTO %s (client_port, binary_data, status, created_at, updated_at, retry_count)
		VALUES (:1, :2, :3, :4, :5, :6)
		RETURNING id INTO :7
	`, r.db.config.TableName)

	now := time.Now()
	var requestID int64

	_, err := r.db.db.ExecContext(ctx,
		query,
		clientPort,
		binaryData,
		models.StatusPending,
		now,
		now,
		0,
		sql.Out{Dest: &requestID},
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert request: %w", err)
	}

	r.logger.Debug("inserted new request",
		zap.Int64("request_id", requestID),
		zap.Int("client_port", clientPort))

	return requestID, nil
}

// MarkAsSending marks a request as being sent
func (r *Repository) MarkAsSending(ctx context.Context, requestID int64) error {
	return r.UpdateRequestStatus(ctx, requestID, models.StatusSending, nil)
}

// MarkAsComplete marks a request as complete
func (r *Repository) MarkAsComplete(ctx context.Context, requestID int64) error {
	return r.UpdateRequestStatus(ctx, requestID, models.StatusComplete, nil)
}

// MarkAsFailed marks a request as failed with an error message
func (r *Repository) MarkAsFailed(ctx context.Context, requestID int64, errorMsg string) error {
	return r.UpdateRequestStatus(ctx, requestID, models.StatusFailed, &errorMsg)
}

// GetPendingCount returns the count of pending requests
func (r *Repository) GetPendingCount(ctx context.Context) (int64, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s WHERE status = :1
	`, r.db.config.TableName)

	var count int64
	err := r.db.db.QueryRowContext(ctx, query, models.StatusPending).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get pending count: %w", err)
	}

	return count, nil
}
