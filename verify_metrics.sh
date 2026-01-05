#!/bin/bash

# Function to cleanup background processes on exit
cleanup() {
    echo "Stopping services..."
    pkill -P $$
    wait
}
trap cleanup EXIT

echo "Starting API service..."
make run BIN=api ENV=dev > api.log 2>&1 &
PID_API=$!

echo "Starting User Center service..."
make run BIN=user-center ENV=dev > user-center.log 2>&1 &
PID_USER=$!

echo "Starting RAG service..."
make run BIN=rag ENV=dev > rag.log 2>&1 &
PID_RAG=$!

echo "Waiting for services to start (15s)..."
sleep 15

echo "=== API Logs (tail) ==="
tail -n 10 api.log
echo ""

echo "=== User Center Logs (tail) ==="
tail -n 10 user-center.log
echo ""

echo "=== RAG Logs (tail) ==="
tail -n 10 rag.log
echo ""

echo "=== API Metrics (Port 8100) ==="
curl -s http://localhost:8100/metrics | head -n 5
echo ""

echo "=== User Center Metrics (Port 8081) ==="
curl -s http://localhost:8081/metrics | head -n 5
echo ""

echo "=== RAG Metrics (Port 8082) ==="
curl -s http://localhost:8082/metrics | head -n 5
echo ""

