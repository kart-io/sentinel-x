# Makefile for Kart Logger project

.PHONY: fmt test clean build bench coverage help

# Format code
fmt:
	gofmt -s -w .
	go fmt ./...
	go vet ./...

# Run tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -cover ./...

# Generate detailed coverage report
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Run benchmarks
bench:
	go test -bench=. ./...

# Build the library
build:
	go build ./...

# Clean build artifacts
clean:
	go clean ./...
	rm -f coverage.out

# Tidy dependencies
tidy:
	go mod tidy

# Run all checks (format, vet, test)
check: fmt test

# Run examples
example-comprehensive:
	cd example/comprehensive && go run main.go

example-performance:
	cd example/performance && go run main.go

example-otlp:
	cd example/otlp && go run main.go

# Help
help:
	@echo "Available commands:"
	@echo "  fmt              - Format code with gofmt -s and run vet"
	@echo "  test             - Run tests"
	@echo "  test-verbose     - Run tests with verbose output"
	@echo "  test-coverage    - Run tests with coverage"
	@echo "  coverage         - Generate detailed coverage report"
	@echo "  bench            - Run benchmarks"
	@echo "  build            - Build the library"
	@echo "  clean            - Clean build artifacts"
	@echo "  tidy             - Tidy dependencies"
	@echo "  check            - Run fmt and test"
	@echo "  example-*        - Run specific examples"
	@echo "  help             - Show this help message"