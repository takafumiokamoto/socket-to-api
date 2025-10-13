package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/okamoto/socket-to-api/internal/config"
	"github.com/okamoto/socket-to-api/pkg/protocol"
	"go.uber.org/zap"
)

// TCPServer represents the TCP socket server
type TCPServer struct {
	config    *config.ServerConfig
	listener  net.Listener
	connMgr   *ConnectionManager
	logger    *zap.Logger
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewTCPServer creates a new TCP server
func NewTCPServer(cfg *config.ServerConfig, connMgr *ConnectionManager, logger *zap.Logger) *TCPServer {
	ctx, cancel := context.WithCancel(context.Background())

	return &TCPServer{
		config:  cfg,
		connMgr: connMgr,
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start starts the TCP server
func (s *TCPServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start TCP server on %s: %w", addr, err)
	}

	s.listener = listener
	s.logger.Info("TCP server started", zap.String("address", addr))

	// Start accepting connections
	s.wg.Add(1)
	go s.acceptConnections()

	// Start cleanup routine for stale connections
	s.wg.Add(1)
	go s.cleanupRoutine()

	return nil
}

// acceptConnections accepts incoming TCP connections
func (s *TCPServer) acceptConnections() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		// Set a deadline to prevent blocking forever
		s.listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))

		conn, err := s.listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Timeout is expected, continue
				continue
			}
			if s.ctx.Err() != nil {
				// Context cancelled, shutting down
				return
			}
			s.logger.Error("failed to accept connection", zap.Error(err))
			continue
		}

		// Check connection limit
		if s.connMgr.Count() >= s.config.MaxConnections {
			s.logger.Warn("max connections reached, rejecting connection",
				zap.String("remote_addr", conn.RemoteAddr().String()))
			conn.Close()
			continue
		}

		// Handle the connection
		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// handleConnection handles an individual client connection
func (s *TCPServer) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	// Configure TCP keep-alive
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(s.config.KeepAlive)
		if s.config.KeepAlive {
			tcpConn.SetKeepAlivePeriod(s.config.KeepAlivePeriod)
		}
	}

	remoteAddr := conn.RemoteAddr().String()
	s.logger.Info("new connection accepted", zap.String("remote_addr", remoteAddr))

	// Extract port from remote address
	// Note: In production, you might need a handshake protocol where the client
	// sends its port number, or use a port mapping mechanism
	tcpAddr, ok := conn.RemoteAddr().(*net.TCPAddr)
	if !ok {
		s.logger.Error("failed to parse TCP address", zap.String("remote_addr", remoteAddr))
		return
	}
	clientPort := tcpAddr.Port

	// Register connection
	if err := s.connMgr.Register(clientPort, conn); err != nil {
		s.logger.Error("failed to register connection",
			zap.Int("port", clientPort),
			zap.Error(err))
		return
	}
	defer s.connMgr.Unregister(clientPort)

	// Read messages from client
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		// Set read deadline
		if err := conn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout)); err != nil {
			s.logger.Error("failed to set read deadline", zap.Error(err))
			return
		}

		// Read message using protocol
		msg, err := protocol.ReadMessage(conn)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Read timeout, send ping to keep connection alive
				if err := protocol.WriteMessage(conn, protocol.MessageTypePing, nil); err != nil {
					s.logger.Error("failed to send ping", zap.Error(err))
					return
				}
				continue
			}
			s.logger.Error("failed to read message",
				zap.Int("port", clientPort),
				zap.Error(err))
			return
		}

		s.connMgr.IncrementReceived(clientPort)

		// Handle different message types
		switch msg.Type {
		case protocol.MessageTypeRequest:
			s.handleRequest(clientPort, msg.Data)
		case protocol.MessageTypePing:
			// Respond with pong
			if err := protocol.WriteMessage(conn, protocol.MessageTypePong, nil); err != nil {
				s.logger.Error("failed to send pong", zap.Error(err))
				return
			}
		case protocol.MessageTypePong:
			// Just update last active time
			s.connMgr.UpdateLastActive(clientPort)
		default:
			s.logger.Warn("unknown message type",
				zap.Int("port", clientPort),
				zap.Uint8("type", uint8(msg.Type)))
		}
	}
}

// handleRequest handles a request message from a client
func (s *TCPServer) handleRequest(clientPort int, data []byte) {
	// TODO: This will be implemented to insert data into database
	// For now, just log it
	s.logger.Debug("received request",
		zap.Int("port", clientPort),
		zap.Int("data_size", len(data)))

	// The actual implementation will:
	// 1. Insert binary data into database with client port
	// 2. The worker pool will pick it up and process it
}

// cleanupRoutine periodically cleans up stale connections
func (s *TCPServer) cleanupRoutine() {
	defer s.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	maxIdleTime := 5 * time.Minute

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.connMgr.CleanupStaleConnections(maxIdleTime)
		}
	}
}

// Stop gracefully stops the TCP server
func (s *TCPServer) Stop() error {
	s.logger.Info("stopping TCP server")

	// Cancel context to signal shutdown
	s.cancel()

	// Close listener
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			s.logger.Error("error closing listener", zap.Error(err))
		}
	}

	// Close all connections
	s.connMgr.CloseAll()

	// Wait for all goroutines to finish
	s.wg.Wait()

	s.logger.Info("TCP server stopped")
	return nil
}

// GetConnectionManager returns the connection manager
func (s *TCPServer) GetConnectionManager() *ConnectionManager {
	return s.connMgr
}
