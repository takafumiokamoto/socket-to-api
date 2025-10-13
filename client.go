package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	// Connect to server
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer conn.Close()

	log.Println("Connected to server")

	reader := bufio.NewReader(conn)
	stdin := bufio.NewReader(os.Stdin)

	// Send and receive loop
	for {
		fmt.Print("Enter message: ")
		message, _ := stdin.ReadString('\n')

		// Send message
		_, err := conn.Write([]byte(message))
		if err != nil {
			log.Fatal("Failed to send:", err)
		}

		// Set read timeout
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))

		// Receive response
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal("Failed to receive:", err)
		}

		fmt.Printf("Server response: %s", response)
	}
}
