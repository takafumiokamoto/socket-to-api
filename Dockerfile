# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 ensures a static binary that works in scratch
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage - scratch container
FROM scratch

# Copy CA certificates from alpine (required for HTTPS)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary from builder
COPY --from=builder /app/main /main

# Copy SQL files (needed for embed)
COPY --from=builder /app/sql /sql

# Set environment variables (can be overridden at runtime)
ENV DB_HOST=oracle-xe
ENV DB_SERVICE=XEPDB1
ENV DB_USER=app
ENV DB_PASSWORD=app

# Run the application
ENTRYPOINT ["/main"]
