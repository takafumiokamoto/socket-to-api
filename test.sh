#!/bin/bash

# Test script for socket server

echo "=== Socket Server Tests ==="
echo ""

# Test 1: Single message
echo "Test 1: Single message"
go run test_client.go -msg "Hello World"
echo ""

# Test 2: Multiple messages
echo "Test 2: Send 10 messages"
go run test_client.go -msg "Test message" -count 10
echo ""

# Test 3: Concurrent connections
echo "Test 3: 5 concurrent connections, 10 messages each"
go run test_client.go -msg "Concurrent test" -count 50 -concurrent 5
echo ""

# Test 4: Load test
echo "Test 4: Load test - 100 connections, 10 messages each"
go run test_client.go -msg "Load test" -count 1000 -concurrent 100
echo ""

echo "All tests completed!"
