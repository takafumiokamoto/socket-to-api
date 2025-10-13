package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"time"
)

func main() {
	host := flag.String("host", "localhost:8080", "Server host:port")
	message := flag.String("msg", "", "Message to send (if empty, enters interactive mode)")
	count := flag.Int("count", 1, "Number of times to send the message")
	concurrent := flag.Int("concurrent", 1, "Number of concurrent connections")
	flag.Parse()

	if *message == "" {
		// Interactive mode
		runInteractiveMode(*host)
	} else {
		// Batch mode
		runBatchMode(*host, *message, *count, *concurrent)
	}
}

func runInteractiveMode(host string) {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer conn.Close()

	fmt.Println("Connected to", host)
	fmt.Println("Type messages and press Enter (Ctrl+C to quit)")
	fmt.Println("---")

	reader := bufio.NewReader(conn)

	for {
		var message string
		fmt.Print("> ")
		fmt.Scanln(&message)

		// Send
		_, err := conn.Write([]byte(message + "\n"))
		if err != nil {
			log.Fatal("Send failed:", err)
		}

		// Receive
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal("Receive failed:", err)
		}

		fmt.Printf("< %s", response)
	}
}

func runBatchMode(host, message string, count, concurrent int) {
	fmt.Printf("Sending '%s' %d times with %d concurrent connections\n", message, count, concurrent)

	start := time.Now()
	done := make(chan bool, concurrent)

	messagesPerConn := count / concurrent
	if messagesPerConn == 0 {
		messagesPerConn = 1
		concurrent = count
	}

	for i := 0; i < concurrent; i++ {
		go func(id int) {
			conn, err := net.Dial("tcp", host)
			if err != nil {
				log.Printf("Connection %d failed: %v", id, err)
				done <- false
				return
			}
			defer conn.Close()

			reader := bufio.NewReader(conn)

			successCount := 0
			for j := 0; j < messagesPerConn; j++ {
				// Send
				_, err := conn.Write([]byte(message + "\n"))
				if err != nil {
					log.Printf("Connection %d send failed: %v", id, err)
					break
				}

				// Receive
				conn.SetReadDeadline(time.Now().Add(5 * time.Second))
				_, err = reader.ReadString('\n')
				if err != nil {
					log.Printf("Connection %d receive failed: %v", id, err)
					break
				}

				successCount++
			}

			fmt.Printf("Connection %d completed: %d/%d messages\n", id, successCount, messagesPerConn)
			done <- true
		}(i)
	}

	// Wait for all connections
	for i := 0; i < concurrent; i++ {
		<-done
	}

	duration := time.Since(start)
	fmt.Printf("\nCompleted in %v\n", duration)
	fmt.Printf("Messages/second: %.2f\n", float64(count)/duration.Seconds())
}
