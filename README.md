# Socket Server POC

Minimal TCP socket server proof-of-concept.

## What It Does

- Accepts TCP connections on port 8080
- Receives text messages (newline-terminated)
- Echoes messages back with timestamp
- Tracks active connections

## Quick Start

```bash
# Terminal 1 - Start server
go run main.go

# Terminal 2 - Interactive client
go run client.go
```

## Testing Methods

### 1. Command Line Tools

**netcat (simplest)**
```bash
nc localhost 8080
# Type messages and press Enter
```

**Echo single message**
```bash
echo "Hello World" | nc localhost 8080
```

**Send from file**
```bash
cat test_messages.txt | nc localhost 8080
```

**telnet**
```bash
telnet localhost 8080
```

### 2. Advanced Test Client

**Interactive mode**
```bash
go run test_client.go
```

**Send single message**
```bash
go run test_client.go -msg "Hello"
```

**Send 100 messages**
```bash
go run test_client.go -msg "Test" -count 100
```

**Load test: 100 concurrent connections**
```bash
go run test_client.go -msg "Load" -count 1000 -concurrent 100
```

**Connect to different host**
```bash
go run test_client.go -host "192.168.1.100:8080" -msg "Hello"
```

### 3. Run All Tests

**Linux/Mac**
```bash
chmod +x test.sh
./test.sh
```

**Windows**
```bash
test.bat
```

### 4. GUI Tools (Postman-like)

**Recommended: Packet Sender**
- Download: https://packetsender.com/
- Free, cross-platform
- TCP/UDP client with GUI
- Save and replay requests

**Alternative: SocketTest**
- Download: https://sourceforge.net/projects/sockettest/
- Simple Windows GUI

**Alternative: Hercules**
- Download: https://www.hw-group.com/software/hercules-setup-utility
- Popular TCP/UDP testing tool

## Files

```
main.go           - TCP server
client.go         - Simple interactive client
test_client.go    - Advanced test client with load testing
test.sh           - Test suite (Linux/Mac)
test.bat          - Test suite (Windows)
test_messages.txt - Sample messages for testing
```

## Example Test Session

```bash
# Start server
$ go run main.go
2024/10/13 19:00:00 Server started on 0.0.0.0:8080

# In another terminal - send single message
$ go run test_client.go -msg "Hello"
Connected to localhost:8080
Sending 'Hello' 1 times with 1 concurrent connections
Connection 0 completed: 1/1 messages

Completed in 5.234ms
Messages/second: 191.06

# Load test
$ go run test_client.go -msg "Test" -count 1000 -concurrent 100
Sending 'Test' 1000 times with 100 concurrent connections
Connection 0 completed: 10/10 messages
Connection 1 completed: 10/10 messages
...
Connection 99 completed: 10/10 messages

Completed in 234.567ms
Messages/second: 4263.58
```

## Current Features

- Connection management
- Multiple concurrent clients
- Simple echo protocol
- Basic logging
- Load testing capability

## Next Steps

Based on bridge_system_specs.md:
1. Add binary protocol support
2. Add database integration
3. Add API client
4. Add data transformation
