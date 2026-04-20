# Declare all phony targets
.PHONY: all build test test-verbose test-coverage lint fmt vet modernize check \
	clean compile install deps-tidy deps-update security-scan \
	shapes go-functions help

# Default target
all: build

# Help target
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  build                 - Run tests, clean, and compile"
	@echo "  compile               - Compile the binary"
	@echo "  install               - Install the application"
	@echo "  clean                 - Clean build artifacts and coverage files"
	@echo ""
	@echo "Testing targets:"
	@echo "  test                  - Run Go unit tests"
	@echo "  test-verbose          - Run tests with verbose output and coverage"
	@echo "  test-coverage         - Generate test coverage report (HTML)"
	@echo ""
	@echo "Code quality targets:"
	@echo "  fmt                   - Format Go code"
	@echo "  vet                   - Run go vet for static analysis"
	@echo "  lint                  - Run linter (requires golangci-lint)"
	@echo "  modernize             - Update code to modern Go patterns (requires modernize)"
	@echo "  check                 - Run full validation suite (fmt, vet, lint, test)"
	@echo "  security-scan         - Run security analysis (requires gosec)"
	@echo ""
	@echo "Dependency management:"
	@echo "  deps-tidy             - Clean up go.mod and go.sum"
	@echo "  deps-update           - Update dependencies to latest versions"
	@echo ""
	@echo "Development utilities:"
	@echo "  go-functions          - List all Go functions in the project"
	@echo "  shapes                - Convert draw.io shapes"

# Build targets
build: test clean compile

compile:
	CGO_ENABLED=0 go build -ldflags "-w -s" -buildvcs=false

install:
	go install .

# Testing
test:
	@echo "Running tests..."
	CGO_ENABLED=0 go test ./...

test-verbose:
	CGO_ENABLED=0 go test -v -cover ./...

test-coverage:
	CGO_ENABLED=0 go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

# Code quality
fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	@echo "Running linters..."
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "Error: golangci-lint is not installed."; \
		echo "This repository's .golangci.yml uses the v2 config schema, so v2 is required."; \
		echo "Install it from https://golangci-lint.run/welcome/install/"; \
		exit 1; \
	}
	@major=$$(golangci-lint --version 2>/dev/null | sed -n 's/.*has version \([0-9][0-9]*\)\..*/\1/p'); \
	if [ "$$major" != "2" ]; then \
		version_line=$$(golangci-lint --version 2>/dev/null | head -n 1); \
		echo "Error: golangci-lint v2 is required (detected: $${version_line:-unknown version})."; \
		echo "This repository's .golangci.yml uses the v2 config schema."; \
		echo "Install v2 from https://golangci-lint.run/welcome/install/"; \
		exit 1; \
	fi
	golangci-lint run --timeout 5m

modernize:
	@which modernize > /dev/null || (echo "modernize not installed. Run: go install github.com/gaissmai/modernize@latest" && exit 1)
	modernize -fix -test ./...

check: fmt vet lint test

security-scan:
	@which gosec > /dev/null || (echo "gosec not installed. Run: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest" && exit 1)
	gosec ./...

# Dependency management
deps-tidy:
	go mod tidy

deps-update:
	go get -u ./...
	go mod tidy

# Utility targets
clean:
	go clean
	rm -f coverage.out coverage.html

go-functions:
	@echo "Finding all functions in the project..."
	@grep -r "^func " . --include="*.go" | grep -v vendor/