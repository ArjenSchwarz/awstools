# Declare all phony targets
.PHONY: all build deps test local clean compile shapes go-functions help

# Default target
all: build

# Help target
help:
	@echo "Available targets:"
	@echo "  all         - Build the project (same as build)"
	@echo "  build       - Run tests, clean, and compile"
	@echo "  test        - Run linting and tests"
	@echo "  clean       - Clean build artifacts"
	@echo "  compile     - Compile the binary"
	@echo "  shapes      - Convert draw.io shapes"
	@echo "  go-functions - List all Go functions in the project"

# Build targets
build: test clean compile

compile:
	go build

# Testing and linting
test:
	golangci-lint run
	go test ./...

# Utility targets
clean:
	go clean

shapes:
	cd drawio/shapes && ./convert.sh

go-functions:
	@echo "Finding all functions in the project..."
	@grep -r "^func " . --include="*.go" | grep -v vendor/