package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// MessageType represents the type of message in the protocol
type MessageType byte

const (
	// Message types
	MessageTypeRequest  MessageType = 0x01
	MessageTypeResponse MessageType = 0x02
	MessageTypeError    MessageType = 0x03
	MessageTypePing     MessageType = 0x04
	MessageTypePong     MessageType = 0x05
)

// Protocol constants
const (
	// Header format: [Version:1][Type:1][Length:4][Reserved:2]
	HeaderSize      = 8
	MaxMessageSize  = 10 * 1024 * 1024 // 10MB max message size
	ProtocolVersion = 0x01
)

// Message represents a protocol message
type Message struct {
	Version byte
	Type    MessageType
	Length  uint32
	Data    []byte
}

// EncodeMessage encodes a message to binary format
// Format: [Version:1][Type:1][Length:4][Reserved:2][Data:N]
func EncodeMessage(msgType MessageType, data []byte) ([]byte, error) {
	dataLen := len(data)
	if dataLen > MaxMessageSize {
		return nil, fmt.Errorf("message size %d exceeds maximum %d", dataLen, MaxMessageSize)
	}

	msg := make([]byte, HeaderSize+dataLen)

	// Write header
	msg[0] = ProtocolVersion
	msg[1] = byte(msgType)
	binary.BigEndian.PutUint32(msg[2:6], uint32(dataLen))
	// msg[6:8] are reserved bytes (zero)

	// Write data
	if dataLen > 0 {
		copy(msg[HeaderSize:], data)
	}

	return msg, nil
}

// DecodeMessage decodes a binary message
func DecodeMessage(reader io.Reader) (*Message, error) {
	// Read header
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	msg := &Message{
		Version: header[0],
		Type:    MessageType(header[1]),
		Length:  binary.BigEndian.Uint32(header[2:6]),
	}

	// Validate protocol version
	if msg.Version != ProtocolVersion {
		return nil, fmt.Errorf("unsupported protocol version: %d", msg.Version)
	}

	// Validate message size
	if msg.Length > MaxMessageSize {
		return nil, fmt.Errorf("message size %d exceeds maximum %d", msg.Length, MaxMessageSize)
	}

	// Read data if present
	if msg.Length > 0 {
		msg.Data = make([]byte, msg.Length)
		if _, err := io.ReadFull(reader, msg.Data); err != nil {
			return nil, fmt.Errorf("failed to read message data: %w", err)
		}
	}

	return msg, nil
}

// WriteMessage writes a message to a writer
func WriteMessage(writer io.Writer, msgType MessageType, data []byte) error {
	encoded, err := EncodeMessage(msgType, data)
	if err != nil {
		return err
	}

	_, err = writer.Write(encoded)
	return err
}

// ReadMessage reads a message from a reader
func ReadMessage(reader io.Reader) (*Message, error) {
	return DecodeMessage(reader)
}

// CreatePingMessage creates a ping message
func CreatePingMessage() ([]byte, error) {
	return EncodeMessage(MessageTypePing, nil)
}

// CreatePongMessage creates a pong message
func CreatePongMessage() ([]byte, error) {
	return EncodeMessage(MessageTypePong, nil)
}

// CreateErrorMessage creates an error message
func CreateErrorMessage(errorMsg string) ([]byte, error) {
	return EncodeMessage(MessageTypeError, []byte(errorMsg))
}
