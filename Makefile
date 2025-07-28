# tcons-signal Makefile

.PHONY: help build test test-coverage clean install lint fmt vet deps

# Default target
help: ## Show this help message
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build the binary
build: ## Build the tcons-signal binary
	go build -o tcons-signal ./cmd

# Run tests
test: ## Run all tests
	go test -v ./...

# Run tests with coverage
test-coverage: ## Run tests with coverage report
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean: ## Remove build artifacts and coverage files
	rm -f tcons-signal
	rm -f coverage.out coverage.html
	rm -f cmd/coverage.out

# Install binary to $GOPATH/bin
install: ## Install binary to $GOPATH/bin
	go install ./cmd

# Lint code
lint: ## Run golangci-lint
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2" && exit 1)
	golangci-lint run

# Format code
fmt: ## Format Go code
	go fmt ./...

# Run go vet
vet: ## Run go vet
	go vet ./...

# Update dependencies
deps: ## Update and tidy dependencies
	go mod tidy
	go mod download

# Run all checks (fmt, vet, lint, test)
check: fmt vet lint test ## Run all code quality checks

# Quick test with fixtures
test-fixtures: build ## Test binary with fixture scripts
	@echo "Testing success fixture..."
	./tcons-signal --queue-url "mock://test" --id "test-success" --exec "./test/fixtures/success.sh" || echo "Expected to fail (no real SQS)"
	@echo "Testing failure fixture..."
	./tcons-signal --queue-url "mock://test" --id "test-failure" --exec "./test/fixtures/fail.sh" || echo "Expected to fail (no real SQS)"

# Show help flags
usage: build ## Show binary usage
	./tcons-signal --help

# Integration testing with ElasticMQ and EC2 metadata mock
integration-up: ## Start integration test environment
	go run ./test/helpers.go up

integration-down: ## Stop integration test environment
	go run ./test/helpers.go down

integration-test: ## Run full integration test suite
	go run ./test/helpers.go test