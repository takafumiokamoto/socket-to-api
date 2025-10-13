# Socket-to-API Bridge System

A high-performance bridge application that translates between legacy TCP socket connections and modern HTTPS API calls, designed to handle thousands of concurrent operations.

## Overview

This bridge system enables legacy applications using TCP socket communication to interact with modern API-based systems without modification. It acts as a transparent middleware layer that:

- Accepts TCP socket connections from legacy clients
- Polls an Oracle database for pending requests
- Converts binary data to JSON format
- Sends requests to external APIs via HTTPS
- Converts API responses back to binary format
- Routes responses to the correct client via TCP sockets

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────────────┐
│                         Bridge System                            │
│                                                                   │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐      │
│  │              │    │              │    │              │      │
│  │  TCP Server  │───▶│ Connection   │    │   Worker     │      │
│  │              │    │  Manager     │◀───│    Pool      │      │
│  └──────────────┘    └──────────────┘    └──────────────┘      │
│         │                                        │               │
│         │                                        ▼               │
│         │                                ┌──────────────┐       │
│         │                                │              │       │
│         └───────────────────────────────▶│  Processor   │       │
│                                           │              │       │
│                                           └──────┬───────┘       │
│                                                  │               │
│  ┌──────────────┐    ┌──────────────┐    ┌─────▼───────┐      │
│  │              │    │              │    │              │      │
│  │   Oracle DB  │◀───│ Repository   │◀───│ Transformer  │      │
│  │              │    │              │    │              │      │
│  └──────────────┘    └──────────────┘    └──────────────┘      │
│                                                  │               │
│                                           ┌──────▼───────┐       │
│                                           │              │       │
│                                           │  API Client  │       │
│                                           │              │       │
│                                           └──────────────┘       │
└─────────────────────────────────────────────────────────────────┘
                                                   │
                                                   ▼
                                          External API System
```

### Key Components

1. **TCP Server** (`internal/server/tcp_server.go`)
   - Accepts and manages long-lived TCP connections
   - Handles thousands of concurrent connections
   - Implements keep-alive and timeout mechanisms
   - Routes incoming requests to the processing pipeline

2. **Connection Manager** (`internal/server/connection_manager.go`)
   - Thread-safe connection registry
   - Maps client ports to active connections
   - Tracks connection statistics and health
   - Manages connection lifecycle

3. **Database Layer** (`internal/database/`)
   - Oracle DB client with connection pooling
   - Repository pattern for data operations
   - Row-level locking for concurrent access
   - Transaction management

4. **Worker Pool** (`internal/worker/pool.go`)
   - Configurable worker pool for concurrent processing
   - Job queue with backpressure handling
   - Timeout management per job
   - Graceful shutdown support

5. **Processor** (`internal/worker/processor.go`)
   - Orchestrates the complete request pipeline
   - Handles errors and retries
   - Manages database state transitions
   - Routes responses to clients

6. **Transformer** (`internal/transformer/converter.go`)
   - Converts binary data ↔ JSON
   - Protocol-aware data parsing
   - Validation and error handling
   - Customizable for different binary formats

7. **API Client** (`internal/api/client.go`)
   - HTTPS client with connection pooling
   - Automatic retry logic
   - Timeout and deadline management
   - TLS configuration support

## Data Flow

1. **Database Polling**: Main loop continuously scans for unsent data
2. **Fetch & Lock**: Retrieve pending records with row-level locking
3. **Update Status**: Mark record as "sending"
4. **Transform**: Convert binary data to JSON
5. **API Call**: Send JSON to external API via HTTPS
6. **Store Response**: Insert API response into database
7. **Transform Back**: Convert JSON response to binary
8. **Route**: Use port mapping to find client connection
9. **Send**: Deliver binary response via TCP socket
10. **Cleanup**: Delete processed record

## Installation

### Prerequisites

- Go 1.21 or higher
- Oracle Database with Oracle Instant Client
- Access to external API system

### Build

```bash
# Clone the repository
git clone https://github.com/okamoto/socket-to-api.git
cd socket-to-api

# Download dependencies
go mod download

# Build the application
go build -o bridge ./cmd/bridge
```

### Run

```bash
# Run with default config
./bridge

# Run with custom config
./bridge -config /path/to/config.yaml
```

## Configuration

Configuration is managed via YAML. See `config/config.yaml` for all options.

### Key Configuration Sections

#### Server Configuration
```yaml
server:
  host: "0.0.0.0"
  port: 8080
  max_connections: 10000
  read_timeout: 30s
  write_timeout: 30s
```

#### Database Configuration
```yaml
database:
  connection_string: "user/password@localhost:1521/ORCL"
  max_open_conns: 50
  max_idle_conns: 25
  poll_interval: 100ms
```

#### API Configuration
```yaml
api:
  base_url: "https://api.example.com"
  timeout: 30s
  max_retries: 3
  retry_delay: 1s
```

#### Worker Configuration
```yaml
worker:
  pool_size: 100      # Number of concurrent workers
  queue_size: 1000    # Job queue capacity
  process_timeout: 60s
```

## Database Schema

### Unsend Data Table

```sql
CREATE TABLE unsend_data (
    id NUMBER PRIMARY KEY,
    client_port NUMBER NOT NULL,
    binary_data BLOB NOT NULL,
    status VARCHAR2(20) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    retry_count NUMBER DEFAULT 0,
    last_error VARCHAR2(4000)
);

CREATE INDEX idx_unsend_data_status ON unsend_data(status);
CREATE INDEX idx_unsend_data_created ON unsend_data(created_at);
```

### API Response Table

```sql
CREATE TABLE api_responses (
    id NUMBER PRIMARY KEY,
    request_id NUMBER NOT NULL,
    status_code NUMBER NOT NULL,
    response_body CLOB,
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY (request_id) REFERENCES unsend_data(id)
);

CREATE INDEX idx_api_responses_request ON api_responses(request_id);
```

## Protocol

The TCP protocol uses a simple binary format:

### Message Format
```
[Version:1][Type:1][Length:4][Reserved:2][Data:N]
```

- **Version** (1 byte): Protocol version (currently 0x01)
- **Type** (1 byte): Message type (Request=0x01, Response=0x02, Error=0x03, Ping=0x04, Pong=0x05)
- **Length** (4 bytes): Length of data payload (big-endian)
- **Reserved** (2 bytes): Reserved for future use
- **Data** (N bytes): Message payload

### Message Types

- `0x01` - Request: Client sends data to be processed
- `0x02` - Response: Server returns processed result
- `0x03` - Error: Server sends error message
- `0x04` - Ping: Keep-alive ping
- `0x05` - Pong: Keep-alive response

## Performance Considerations

### Concurrency
- Designed to handle thousands of concurrent connections
- Worker pool prevents resource exhaustion
- Connection pooling for database and HTTP clients

### Database Optimization
- Row-level locking with `FOR UPDATE SKIP LOCKED`
- Batch fetching to reduce round trips
- Connection pool tuning for high throughput

### Memory Management
- Streaming for large payloads
- Bounded queues prevent memory overflow
- Connection cleanup for idle clients

### Monitoring
- Built-in metrics logging
- Health check endpoints
- Structured logging with zap

## Customization

### Binary Protocol

The transformer in `internal/transformer/converter.go` can be customized for your specific binary format:

```go
// Implement custom binary parsing
func (t *Transformer) parseBinaryData(binaryData []byte) (map[string]interface{}, error) {
    // Your custom parsing logic here
}

// Implement custom binary encoding
func (t *Transformer) encodeToBinary(response *models.APIResponsePayload) ([]byte, error) {
    // Your custom encoding logic here
}
```

### API Request Format

Customize the API request structure in `internal/models/models.go`:

```go
type APIRequest struct {
    RequestID   string                 `json:"request_id"`
    Timestamp   string                 `json:"timestamp"`
    ClientPort  int                    `json:"client_port"`
    Data        map[string]interface{} `json:"data"`
    // Add your custom fields here
}
```

## Monitoring

### Logs

The application provides structured JSON logs with the following information:

- Connection events (open, close, errors)
- Request processing (start, complete, failed)
- Database operations (queries, updates)
- API calls (requests, responses, retries)
- Performance metrics (durations, counts)

### Metrics

Every 60 seconds, the application logs key metrics:

- Active TCP connections
- Pending database requests
- Worker pool utilization
- Database connection pool stats
- API call statistics

## Error Handling

The system implements comprehensive error handling:

1. **Automatic Retries**: API calls retry with exponential backoff
2. **Error Tracking**: Failed requests logged with error messages
3. **Retry Counting**: Track retry attempts per request
4. **Graceful Degradation**: System continues operating when non-critical components fail
5. **Client Notification**: Error messages sent to clients when processing fails

## Deployment

### Docker

Create a `Dockerfile`:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o bridge ./cmd/bridge

FROM alpine:latest
RUN apk --no-cache add ca-certificates libaio libnsl libc6-compat
COPY --from=builder /app/bridge /usr/local/bin/
COPY --from=builder /app/config /etc/bridge/
ENTRYPOINT ["bridge", "-config", "/etc/bridge/config.yaml"]
```

### Systemd Service

Create `/etc/systemd/system/bridge.service`:

```ini
[Unit]
Description=Socket-to-API Bridge
After=network.target

[Service]
Type=simple
User=bridge
WorkingDirectory=/opt/bridge
ExecStart=/opt/bridge/bridge -config /opt/bridge/config.yaml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

## Troubleshooting

### High CPU Usage
- Check worker pool size (reduce if too high)
- Verify database polling interval
- Monitor number of active connections

### Memory Issues
- Check for connection leaks
- Verify message size limits
- Monitor worker queue depth

### Database Lock Contention
- Increase number of workers
- Reduce batch size
- Check database performance

### Connection Timeouts
- Adjust read/write timeouts
- Check network latency
- Verify keep-alive settings

## License

MIT License

## Contributing

Contributions are welcome. Please submit pull requests or open issues for bugs and feature requests.

## Support

For questions or issues, please open a GitHub issue or contact the maintainers.
