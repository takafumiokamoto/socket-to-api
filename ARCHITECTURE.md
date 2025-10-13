# Architecture Overview

## Design Philosophy

This bridge system is designed with the following principles:

1. **High Concurrency**: Handle thousands of concurrent operations efficiently
2. **Reliability**: Graceful error handling and automatic retries
3. **Observability**: Comprehensive logging and metrics
4. **Maintainability**: Clean architecture with clear separation of concerns
5. **Performance**: Optimized for low latency and high throughput

## Component Interaction Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Request Processing Flow                         │
└─────────────────────────────────────────────────────────────────────────┘

Legacy Client                Bridge System                     External API
     │                            │                                  │
     │ 1. TCP Connect            │                                  │
     ├───────────────────────────▶│                                  │
     │                            │                                  │
     │                            │ 2. Register Connection           │
     │                            │    (ConnectionManager)           │
     │                            │                                  │
     │ 3. Send Request (Binary)  │                                  │
     ├───────────────────────────▶│                                  │
     │                            │                                  │
     │                            │ 4. Insert to DB                  │
     │                            │    (Repository)                  │
     │                            │                                  │
     │                            │ 5. Poll DB                       │
     │                            │    (Main Loop)                   │
     │                            │                                  │
     │                            │ 6. Fetch & Lock Record           │
     │                            │    (Repository)                  │
     │                            │                                  │
     │                            │ 7. Submit Job                    │
     │                            │    (WorkerPool)                  │
     │                            │                                  │
     │                            │ 8. Process Job                   │
     │                            │    (Processor)                   │
     │                            │                                  │
     │                            │ 9. Binary → JSON                 │
     │                            │    (Transformer)                 │
     │                            │                                  │
     │                            │ 10. HTTPS POST                   │
     │                            ├─────────────────────────────────▶│
     │                            │                                  │
     │                            │ 11. JSON Response                │
     │                            │◀─────────────────────────────────┤
     │                            │                                  │
     │                            │ 12. Store Response               │
     │                            │     (Repository)                 │
     │                            │                                  │
     │                            │ 13. JSON → Binary                │
     │                            │     (Transformer)                │
     │                            │                                  │
     │                            │ 14. Route by Port                │
     │                            │     (ConnectionManager)          │
     │                            │                                  │
     │ 15. Response (Binary)     │                                  │
     │◀───────────────────────────┤                                  │
     │                            │                                  │
     │                            │ 16. Cleanup DB                   │
     │                            │     (Repository)                 │
     │                            │                                  │
```

## Concurrency Model

### Worker Pool Pattern

The system uses a worker pool pattern to handle concurrent request processing:

```
Database Poll Loop ─────▶ Job Queue ─────▶ Worker 1 ─┐
                              │            Worker 2  │
                              │            Worker 3  ├─▶ Result Handler
                              │              ...     │
                              └──────────▶ Worker N ─┘
```

**Benefits:**
- Bounded concurrency prevents resource exhaustion
- Queue provides backpressure mechanism
- Workers are reusable and efficient
- Easy to scale by adjusting pool size

### Connection Management

The ConnectionManager uses a concurrent-safe map pattern:

```go
type ConnectionManager struct {
    connections map[int]net.Conn
    connInfo    map[int]*ConnectionInfo
    mu          sync.RWMutex
    logger      *zap.Logger
}
```

**Key Features:**
- RWMutex allows concurrent reads, exclusive writes
- O(1) lookup by client port
- Thread-safe registration and unregistration
- Automatic cleanup of stale connections

## Database Optimization

### Row-Level Locking

Uses Oracle's `FOR UPDATE SKIP LOCKED` for optimal concurrency:

```sql
SELECT * FROM unsend_data
WHERE status = 'pending'
ORDER BY created_at ASC
FETCH FIRST :limit ROWS ONLY
FOR UPDATE SKIP LOCKED
```

**Advantages:**
- Multiple workers can fetch different records simultaneously
- No blocking on locked records
- Automatic retry on contention
- High throughput under load

### Connection Pooling

Database connection pool is tuned for high concurrency:

```yaml
database:
  max_open_conns: 50      # Maximum concurrent connections
  max_idle_conns: 25      # Idle connections for fast reuse
  conn_max_lifetime: 5m   # Rotate connections periodically
```

## Error Handling Strategy

### Multi-Layer Error Handling

1. **Network Layer**: TCP timeouts, connection errors
2. **Database Layer**: Transaction rollback, retry logic
3. **API Layer**: HTTP errors, timeout, retry with exponential backoff
4. **Application Layer**: Business logic errors, validation

### Retry Strategy

```
Request Failed
    │
    ├─ Increment Retry Count
    │
    ├─ Mark as Failed in DB
    │
    ├─ Update Last Error Message
    │
    └─ Next Poll: Retry if retry_count < max_retries
```

**API Retry Logic:**
- Maximum 3 retries (configurable)
- Exponential backoff: 1s, 2s, 4s
- Context-aware cancellation
- Detailed error logging

## Performance Characteristics

### Throughput

**Target Performance:**
- 1,000+ requests/second sustained
- 10,000+ concurrent TCP connections
- Sub-100ms p99 latency for simple requests

**Bottleneck Analysis:**
- Database polling interval: 100ms default
- Worker pool size: 100 workers default
- API timeout: 30s default
- Network latency: Variable

### Memory Usage

**Memory Optimization:**
- Streaming for large payloads
- Bounded queues (queue_size: 1000)
- Connection pooling (reuse vs allocation)
- Periodic GC via connection cleanup

### CPU Usage

**CPU Profile:**
- Database polling: Low (ticker-based)
- Worker processing: Medium (goroutine-based)
- Data transformation: Low-Medium (depends on payload size)
- Network I/O: Low (async, non-blocking)

## Scalability Options

### Vertical Scaling

Increase resources on single instance:

```yaml
worker:
  pool_size: 200        # Increase workers
  queue_size: 2000      # Increase queue

database:
  max_open_conns: 100   # More DB connections

server:
  max_connections: 20000 # More TCP connections
```

### Horizontal Scaling

Multiple bridge instances (requires consideration):

**Challenges:**
- TCP connection affinity (client must connect to same instance)
- Database record contention (solved by SKIP LOCKED)
- Port mapping coordination

**Solutions:**
- Load balancer with sticky sessions
- Shared Oracle DB handles contention
- Stateless processing enables horizontal scaling

## Security Considerations

### TLS/SSL

```yaml
api:
  tls_insecure_skip: false  # Always verify certificates in production
```

### Connection Security

- TCP connections should be on private network
- Consider mTLS for client authentication
- Implement connection limits per client
- Rate limiting on database operations

### Data Protection

- Binary data stored as BLOB in Oracle
- API responses stored for audit trail
- Sensitive data should be encrypted at rest
- PII should be masked in logs

## Monitoring & Observability

### Metrics (Logged every 60s)

```json
{
  "active_connections": 1234,
  "pending_requests": 567,
  "worker_pool_size": 100,
  "jobs_in_queue": 45,
  "db_open_conns": 30,
  "db_idle_conns": 15,
  "db_in_use": 15
}
```

### Health Checks

- Database connectivity check (every 30s)
- API endpoint health check (every 30s)
- Connection pool statistics
- Worker pool utilization

### Logging

Structured JSON logging with fields:
- `timestamp`: ISO8601 format
- `level`: debug, info, warn, error
- `message`: Human-readable message
- `request_id`: For request tracing
- `duration`: For performance monitoring
- `error`: Error details when applicable

## Customization Points

### 1. Binary Protocol (`pkg/protocol/protocol.go`)

Customize message format:
```go
const (
    HeaderSize = 8              // Adjust header size
    MaxMessageSize = 10 * 1024 * 1024  // Adjust max size
)
```

### 2. Data Transformation (`internal/transformer/converter.go`)

Implement custom parsing:
```go
func (t *Transformer) parseBinaryData(binaryData []byte) (map[string]interface{}, error) {
    // Implement your protocol-specific parsing
}

func (t *Transformer) encodeToBinary(response *APIResponsePayload) ([]byte, error) {
    // Implement your protocol-specific encoding
}
```

### 3. API Request Format (`internal/models/models.go`)

Add custom fields:
```go
type APIRequest struct {
    RequestID   string                 `json:"request_id"`
    Timestamp   string                 `json:"timestamp"`
    ClientPort  int                    `json:"client_port"`
    Data        map[string]interface{} `json:"data"`
    // Add your custom fields
    CustomField1 string                `json:"custom_field_1"`
    CustomField2 int                   `json:"custom_field_2"`
}
```

### 4. Processing Logic (`internal/worker/processor.go`)

Customize the processing pipeline:
```go
func (p *Processor) Process(ctx context.Context, request *UnsendData) *ProcessingResult {
    // Add custom validation
    // Add custom transformation
    // Add custom business logic
}
```

## Deployment Considerations

### AWS EC2 Deployment

**Instance Type Selection:**
- c5.2xlarge (8 vCPU, 16 GB) for medium load
- c5.4xlarge (16 vCPU, 32 GB) for high load
- Network-optimized instances for TCP throughput

**Co-location Benefits:**
- Low latency between legacy app and bridge
- Shared Oracle DB connection (localhost)
- Simplified network configuration

### High Availability

**Single Instance Limitations:**
- Single point of failure
- Maintenance requires downtime
- Limited by single machine resources

**HA Design Options:**
1. **Active-Standby**: Health check + auto-failover
2. **Active-Active**: Load balancer + sticky sessions
3. **Kubernetes**: Pod replicas + service mesh

### Disaster Recovery

**Backup Strategy:**
- Database backups (Oracle RMAN)
- Configuration backups
- Binary version control

**Recovery Procedure:**
1. Restore database from backup
2. Deploy bridge application
3. Verify connectivity
4. Resume operations

## Future Enhancements

### Potential Improvements

1. **Metrics Export**: Prometheus/Grafana integration
2. **Distributed Tracing**: OpenTelemetry support
3. **Admin API**: REST API for management
4. **Circuit Breaker**: Prevent cascade failures
5. **Rate Limiting**: Per-client rate limits
6. **Connection Pooling**: Redis for distributed connection registry
7. **Message Queue**: Replace DB polling with queue (Kafka/RabbitMQ)

### Performance Optimizations

1. **Batch Processing**: Batch multiple requests per API call
2. **Connection Multiplexing**: HTTP/2 for API calls
3. **Caching**: Cache frequent API responses
4. **Compression**: Compress large payloads
5. **Zero-Copy**: Reduce memory copies in data path

## Testing Strategy

### Unit Tests
- Test each component in isolation
- Mock external dependencies
- Test error conditions

### Integration Tests
- Test component interactions
- Use test database
- Mock external API

### Load Tests
- Simulate thousands of connections
- Measure throughput and latency
- Identify bottlenecks

### Chaos Engineering
- Simulate database failures
- Simulate API failures
- Test recovery mechanisms
