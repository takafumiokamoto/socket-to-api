package server

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/okamoto/socket-to-api/internal/models"
	"go.uber.org/zap"
)

// ConnectionManager manages active TCP connections
type ConnectionManager struct {
	connections map[int]net.Conn // port -> connection
	connInfo    map[int]*models.ConnectionInfo
	mu          sync.RWMutex
	logger      *zap.Logger
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(logger *zap.Logger) *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[int]net.Conn),
		connInfo:    make(map[int]*models.ConnectionInfo),
		logger:      logger,
	}
}

// Register adds a new connection to the manager
func (cm *ConnectionManager) Register(port int, conn net.Conn) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.connections[port]; exists {
		return fmt.Errorf("port %d already registered", port)
	}

	cm.connections[port] = conn
	cm.connInfo[port] = &models.ConnectionInfo{
		Port:        port,
		RemoteAddr:  conn.RemoteAddr().String(),
		ConnectedAt: time.Now(),
		LastActive:  time.Now(),
	}

	cm.logger.Info("connection registered",
		zap.Int("port", port),
		zap.String("remote_addr", conn.RemoteAddr().String()))

	return nil
}

// Unregister removes a connection from the manager
func (cm *ConnectionManager) Unregister(port int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if conn, exists := cm.connections[port]; exists {
		conn.Close()
		delete(cm.connections, port)
		delete(cm.connInfo, port)

		cm.logger.Info("connection unregistered", zap.Int("port", port))
	}
}

// GetConnection retrieves a connection by port
func (cm *ConnectionManager) GetConnection(port int) (net.Conn, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conn, exists := cm.connections[port]
	return conn, exists
}

// SendData sends data to a specific client by port
func (cm *ConnectionManager) SendData(port int, data []byte) error {
	conn, exists := cm.GetConnection(port)
	if !exists {
		return fmt.Errorf("no connection found for port %d", port)
	}

	// Update stats
	cm.mu.Lock()
	if info, ok := cm.connInfo[port]; ok {
		info.MessagesSent++
		info.LastActive = time.Now()
	}
	cm.mu.Unlock()

	// Write data
	if _, err := conn.Write(data); err != nil {
		cm.logger.Error("failed to send data",
			zap.Int("port", port),
			zap.Error(err))
		cm.Unregister(port)
		return fmt.Errorf("failed to send data to port %d: %w", port, err)
	}

	cm.logger.Debug("data sent to client",
		zap.Int("port", port),
		zap.Int("bytes", len(data)))

	return nil
}

// UpdateLastActive updates the last active time for a connection
func (cm *ConnectionManager) UpdateLastActive(port int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if info, exists := cm.connInfo[port]; exists {
		info.LastActive = time.Now()
	}
}

// IncrementReceived increments the received message counter
func (cm *ConnectionManager) IncrementReceived(port int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if info, exists := cm.connInfo[port]; exists {
		info.MessagesReceived++
		info.LastActive = time.Now()
	}
}

// GetConnectionInfo returns information about a connection
func (cm *ConnectionManager) GetConnectionInfo(port int) (*models.ConnectionInfo, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	info, exists := cm.connInfo[port]
	return info, exists
}

// GetAllConnections returns a list of all active connection ports
func (cm *ConnectionManager) GetAllConnections() []int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	ports := make([]int, 0, len(cm.connections))
	for port := range cm.connections {
		ports = append(ports, port)
	}

	return ports
}

// Count returns the number of active connections
func (cm *ConnectionManager) Count() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return len(cm.connections)
}

// CloseAll closes all active connections
func (cm *ConnectionManager) CloseAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for port, conn := range cm.connections {
		conn.Close()
		cm.logger.Info("closing connection", zap.Int("port", port))
	}

	cm.connections = make(map[int]net.Conn)
	cm.connInfo = make(map[int]*models.ConnectionInfo)
}

// CleanupStaleConnections removes connections that haven't been active
func (cm *ConnectionManager) CleanupStaleConnections(maxIdleTime time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	for port, info := range cm.connInfo {
		if now.Sub(info.LastActive) > maxIdleTime {
			if conn, exists := cm.connections[port]; exists {
				conn.Close()
				delete(cm.connections, port)
				delete(cm.connInfo, port)

				cm.logger.Info("stale connection removed",
					zap.Int("port", port),
					zap.Duration("idle_time", now.Sub(info.LastActive)))
			}
		}
	}
}
