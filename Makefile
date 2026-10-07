.PHONY: build test lint run clean install

BINARY_NAME=gtd
BUILD_DIR=./bin

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/gtd

test:
	@echo "Running tests..."
	@go test -v ./...

test-integration:
	@echo "Running integration tests..."
	@go test -v -tags=integration ./tests/...

lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, install it from https://golangci-lint.run/usage/install/" && exit 1)
	@golangci-lint run ./...

run:
	@go run ./cmd/gtd dashboard

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean

install:
	@echo "Installing $(BINARY_NAME)..."
	@go install ./cmd/gtd

deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

fmt:
	@echo "Formatting code..."
	@go fmt ./...

vet:
	@echo "Running go vet..."
	@go vet ./...
