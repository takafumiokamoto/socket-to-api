.PHONY: build run test clean deps install

# Binary name
BINARY_NAME=bridge
BINARY_PATH=./cmd/bridge

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build the application
build:
	$(GOBUILD) -o $(BINARY_NAME) $(BINARY_PATH)

# Run the application
run: build
	./$(BINARY_NAME)

# Run with custom config
run-config: build
	./$(BINARY_NAME) -config $(CONFIG)

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

# Install dependencies
deps:
	$(GOMOD) download
	$(GOMOD) verify

# Update dependencies
deps-update:
	$(GOGET) -u ./...
	$(GOMOD) tidy

# Format code
fmt:
	$(GOFMT) ./...

# Run linter
lint:
	$(GOVET) ./...

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out

# Install the binary
install: build
	mv $(BINARY_NAME) $(GOPATH)/bin/

# Run with race detector
run-race:
	$(GOCMD) run -race $(BINARY_PATH)

# Build for different platforms
build-linux:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-linux-amd64 $(BINARY_PATH)

build-windows:
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-windows-amd64.exe $(BINARY_PATH)

build-darwin:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-darwin-amd64 $(BINARY_PATH)

build-all: build-linux build-windows build-darwin

# Docker commands
docker-build:
	docker build -t socket-to-api:latest .

docker-run:
	docker run -v $(PWD)/config:/etc/bridge socket-to-api:latest
