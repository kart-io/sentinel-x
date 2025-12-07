#!/bin/bash

# Web Framework Integration Test Script
# This script demonstrates the unified logger integration with Gin and Echo

echo "=== Unified Logger Web Framework Integration Demo ==="
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to check if a port is in use
check_port() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        return 0  # Port is in use
    else
        return 1  # Port is free
    fi
}

# Function to wait for server to be ready
wait_for_server() {
    local url=$1
    local timeout=30
    local count=0
    
    echo -n "Waiting for server to be ready"
    while [ $count -lt $timeout ]; do
        if curl -s "$url" > /dev/null 2>&1; then
            echo -e " ${GREEN}✓${NC}"
            return 0
        fi
        echo -n "."
        sleep 1
        count=$((count + 1))
    done
    echo -e " ${RED}✗${NC}"
    return 1
}

# Function to make test requests
test_endpoints() {
    local base_url=$1
    local framework=$2
    
    echo -e "${BLUE}Testing $framework endpoints:${NC}"
    
    # Test root endpoint
    echo -n "  GET / ... "
    response=$(curl -s "$base_url/")
    if [[ $? -eq 0 && $response == *"$framework"* ]]; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
    fi
    
    # Test specific endpoints based on framework
    if [ "$framework" == "Gin" ]; then
        # Test user endpoint
        echo -n "  GET /users/123 ... "
        response=$(curl -s "$base_url/users/123")
        if [[ $? -eq 0 && $response == *"123"* ]]; then
            echo -e "${GREEN}✓${NC}"
        else
            echo -e "${RED}✗${NC}"
        fi
        
        # Test health endpoint
        echo -n "  GET /health ... "
        response=$(curl -s "$base_url/health")
        if [[ $? -eq 0 && $response == *"healthy"* ]]; then
            echo -e "${GREEN}✓${NC}"
        else
            echo -e "${RED}✗${NC}"
        fi
    else
        # Test product endpoint
        echo -n "  GET /api/products/123 ... "
        response=$(curl -s "$base_url/api/products/123")
        if [[ $? -eq 0 && $response == *"123"* ]]; then
            echo -e "${GREEN}✓${NC}"
        else
            echo -e "${RED}✗${NC}"
        fi
        
        # Test health endpoint
        echo -n "  GET /health ... "
        response=$(curl -s "$base_url/health")
        if [[ $? -eq 0 && $response == *"healthy"* ]]; then
            echo -e "${GREEN}✓${NC}"
        else
            echo -e "${RED}✗${NC}"
        fi
    fi
}

# Check if required tools are available
echo "Checking prerequisites..."
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${NC}"
    exit 1
fi

if ! command -v curl &> /dev/null; then
    echo -e "${RED}Error: curl is not installed${NC}"
    exit 1
fi

echo -e "${GREEN}Prerequisites OK${NC}"
echo

# Check if ports are already in use
if check_port 8080; then
    echo -e "${YELLOW}Warning: Port 8080 is already in use. Gin server may fail to start.${NC}"
fi

if check_port 8081; then
    echo -e "${YELLOW}Warning: Port 8081 is already in use. Echo server may fail to start.${NC}"
fi

# Start Gin server
echo -e "${BLUE}Starting Gin server on port 8080...${NC}"
cd ../gin && go run main.go > gin.log 2>&1 &
GIN_PID=$!
echo "Gin server PID: $GIN_PID"

# Start Echo server
echo -e "${BLUE}Starting Echo server on port 8081...${NC}"
cd ../echo && go run main.go > echo.log 2>&1 &
ECHO_PID=$!
echo "Echo server PID: $ECHO_PID"

# Function to cleanup servers
cleanup() {
    echo
    echo -e "${YELLOW}Cleaning up servers...${NC}"
    if [ ! -z "$GIN_PID" ]; then
        kill $GIN_PID 2>/dev/null
        echo "Stopped Gin server"
    fi
    if [ ! -z "$ECHO_PID" ]; then
        kill $ECHO_PID 2>/dev/null
        echo "Stopped Echo server"
    fi
    exit 0
}

# Set up cleanup on script exit
trap cleanup EXIT INT TERM

# Wait for servers to start
echo
if wait_for_server "http://localhost:8080"; then
    echo -e "${GREEN}Gin server is ready${NC}"
else
    echo -e "${RED}Gin server failed to start${NC}"
    exit 1
fi

if wait_for_server "http://localhost:8081"; then
    echo -e "${GREEN}Echo server is ready${NC}"
else
    echo -e "${RED}Echo server failed to start${NC}"
    exit 1
fi

echo
echo -e "${GREEN}Both servers are running!${NC}"
echo

# Test endpoints
test_endpoints "http://localhost:8080" "Gin"
echo
test_endpoints "http://localhost:8081" "Echo"

echo
echo -e "${BLUE}Log Output Samples:${NC}"
echo
echo -e "${YELLOW}Gin Logs (last 5 lines):${NC}"
cd ../gin && tail -n 5 gin.log

echo
echo -e "${YELLOW}Echo Logs (last 5 lines):${NC}"
cd ../echo && tail -n 5 echo.log

echo
echo -e "${BLUE}Demo URLs:${NC}"
echo "  Gin Framework:  http://localhost:8080"
echo "  Echo Framework: http://localhost:8081"
echo
echo -e "${BLUE}Try these commands:${NC}"
echo "  curl http://localhost:8080/docs"
echo "  curl http://localhost:8081/docs"
echo "  curl -X POST http://localhost:8080/users -H 'Content-Type: application/json' -d '{\"name\":\"Test User\",\"email\":\"test@example.com\"}'"
echo "  curl -X POST http://localhost:8081/api/products -H 'Content-Type: application/json' -d '{\"name\":\"Test Product\",\"price\":29.99,\"category\":\"test\"}'"
echo

echo -e "${GREEN}Demo is running. Press Ctrl+C to stop both servers.${NC}"

# Keep script running until interrupted
while true; do
    sleep 1
done