package models

import (
	"time"
)

// RequestStatus represents the processing status of a request
type RequestStatus string

const (
	StatusPending  RequestStatus = "pending"
	StatusSending  RequestStatus = "sending"
	StatusSent     RequestStatus = "sent"
	StatusFailed   RequestStatus = "failed"
	StatusComplete RequestStatus = "complete"
)

// UnsendData represents a database record waiting to be processed
type UnsendData struct {
	ID          int64         `db:"id"`
	ClientPort  int           `db:"client_port"`
	BinaryData  []byte        `db:"binary_data"`
	Status      RequestStatus `db:"status"`
	CreatedAt   time.Time     `db:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at"`
	RetryCount  int           `db:"retry_count"`
	LastError   *string       `db:"last_error"`
}

// APIResponse represents the response from the external API
type APIResponse struct {
	ID          int64     `db:"id"`
	RequestID   int64     `db:"request_id"`
	StatusCode  int       `db:"status_code"`
	ResponseBody []byte   `db:"response_body"`
	CreatedAt   time.Time `db:"created_at"`
}

// ProcessingJob represents a job to be processed by workers
type ProcessingJob struct {
	Request     *UnsendData
	RetryCount  int
	SubmittedAt time.Time
}

// ProcessingResult represents the result of processing a job
type ProcessingResult struct {
	RequestID   int64
	ClientPort  int
	Success     bool
	Response    []byte
	Error       error
	ProcessedAt time.Time
}

// APIRequest represents the JSON structure sent to the external API
type APIRequest struct {
	RequestID   string                 `json:"request_id"`
	Timestamp   string                 `json:"timestamp"`
	ClientPort  int                    `json:"client_port"`
	Data        map[string]interface{} `json:"data"`
}

// APIResponsePayload represents the JSON structure received from the external API
type APIResponsePayload struct {
	RequestID  string                 `json:"request_id"`
	Status     string                 `json:"status"`
	StatusCode int                    `json:"status_code"`
	Message    string                 `json:"message"`
	Data       map[string]interface{} `json:"data"`
	Timestamp  string                 `json:"timestamp"`
}

// ConnectionInfo represents an active TCP connection
type ConnectionInfo struct {
	Port        int
	RemoteAddr  string
	ConnectedAt time.Time
	LastActive  time.Time
	MessagesSent int64
	MessagesReceived int64
}
