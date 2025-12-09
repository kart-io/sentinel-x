.PHONY: run run-http run-grpc build proto client fmt lint

# JWT key for development (min 32 chars for HS256)
JWT_KEY ?= sentinel-x-dev-secret-key-32chars

# Format the code
fmt:
	go fmt ./...

# Lint the code
lint:
	golangci-lint run ./...

# Run the server in both HTTP and gRPC modes
run:
	go run main.go --server.mode=both --http.addr=:8102 --grpc.addr=:9098 --jwt.key=$(JWT_KEY)

# Run the server in HTTP mode only
run-http:
	go run main.go --server.mode=http --http.addr=:8102 --jwt.key=$(JWT_KEY)

# Run the server in gRPC mode only
run-grpc:
	go run main.go --server.mode=grpc --grpc.addr=:9098 --jwt.key=$(JWT_KEY)

# Build the server binary
build:
	mkdir -p bin
	go build -o bin/user-center main.go

# Generate Go code from .proto files
proto:
	protoc --proto_path=../../.. --proto_path=. --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative api/v1/user.proto

# Run the gRPC test client
client:
	go run client/main.go
