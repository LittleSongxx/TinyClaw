# Golang Makefile Example

# Project name and binary file name
APP_NAME := TinyClaw
VERSION := 0.0.14
BUILD_DIR := ./bin
SRC := $(shell find . -name '*.go' -not -path "./vendor/*")

# Go build flags
BUILD := $(BUILD_DIR)/$(APP_NAME)

# Default task
all: build

# Build the project
build:
	@echo "🔨 Building Golang project..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD) main.go
	@echo "✅ Build complete -> $(BUILD)"

# Run the project
run: build
	@echo "🚀 Running the project..."
	@$(BUILD)

# Run tests
test:
	@echo "🧪 Running tests..."
	@go test ./... -v

# Format the code
fmt:
	@echo "🛠️ Formatting the code..."
	@go fmt ./...

# Static analysis using go vet
vet:
	@echo "🔍 Running go vet..."
	@go vet ./...

# Clean generated files
clean:
	@echo "🧹 Cleaning up..."
	@rm -rf $(BUILD_DIR)

# Static analysis using golangci-lint
lint:
	@echo "🔍 Running static analysis with golangci-lint..."
	@golangci-lint run ./...

# Help information
help:
	@echo "🛠️ Golang Makefile Usage"
	@echo "make           -> Build the project"
	@echo "make run       -> Build and run the project"
	@echo "make test      -> Run tests"
	@echo "make fmt       -> Format the code"
	@echo "make vet       -> Run go vet (static analysis)"
	@echo "make lint      -> Run static analysis with golangci-lint"
	@echo "make clean     -> Clean generated files"
	@echo "make help      -> Display help information"

# Declare phony targets to avoid conflicts with file names
.PHONY: all build run test fmt vet clean lint help
