package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

const (
	ServerHost = "0.0.0.0"
	ServerPort = "8080"
)

// ConnectionManager tracks active connections
type ConnectionManager struct {
	connections map[string]net.Conn
	mu          sync.RWMutex
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]net.Conn),
	}
}

func (cm *ConnectionManager) Add(addr string, conn net.Conn) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.connections[addr] = conn
	log.Printf("Connection added: %s (total: %d)", addr, len(cm.connections))
}

func (cm *ConnectionManager) Remove(addr string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.connections, addr)
	log.Printf("Connection removed: %s (total: %d)", addr, len(cm.connections))
}

func (cm *ConnectionManager) Count() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.connections)
}

func main() {
	connMgr := NewConnectionManager()

	// Start TCP server
	listener, err := net.Listen("tcp", ServerHost+":"+ServerPort)
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}
	defer listener.Close()

	log.Printf("Server started on %s:%s", ServerHost, ServerPort)

	// Accept connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Failed to accept connection:", err)
			continue
		}

		go handleConnection(conn, connMgr)
	}
}

func handleConnection(conn net.Conn, connMgr *ConnectionManager) {
	defer conn.Close()

	addr := conn.RemoteAddr().String()
	log.Printf("New connection from: %s", addr)

	connMgr.Add(addr, conn)
	defer connMgr.Remove(addr)

	reader := bufio.NewReader(conn)

	for {
		// Read message (line-based for simplicity)
		message, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Connection closed: %s (error: %v)", addr, err)
			return
		}

		log.Printf("Received from %s: %s", addr, message[:len(message)-1])

		// Echo response
		response := fmt.Sprintf("Echo: %s [%s]\n", message[:len(message)-1], time.Now().Format(time.RFC3339))

		_, err = conn.Write([]byte(response))
		if err != nil {
			log.Printf("Failed to send response to %s: %v", addr, err)
			return
		}

		log.Printf("Sent to %s: %s", addr, response[:len(response)-1])
	}
}
