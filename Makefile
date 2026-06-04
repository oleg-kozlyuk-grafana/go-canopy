.PHONY: help build test test-coverage coverage-html lint deps clean install run

# Variables
BUILD_DIR=.
COVERAGE_FILE=.coverage/coverage.out
COVERAGE_HTML=.coverage/report.html

# Executables
CANOPY_BINARY=canopy

# Command directories
CANOPY_CMD=./cmd/canopy

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build flags
LDFLAGS=-ldflags "-s -w"

## help: Display this help message
help:
	@echo "Canopy - Local code coverage diff analysis"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /' | column -t -s ':'

## build: Build the canopy binary
build:
	@echo "Building $(CANOPY_BINARY)..."
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CANOPY_BINARY) $(CANOPY_CMD)
	@echo "Build complete: $(BUILD_DIR)/$(CANOPY_BINARY)"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -short ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	mkdir -p .coverage
	$(GOTEST) -coverprofile=$(COVERAGE_FILE) ./...
	@echo ""
	@echo "Coverage summary:"
	$(GOCMD) tool cover -func=$(COVERAGE_FILE)
	@echo ""
	@echo "To view detailed coverage report in browser:"
	@echo "  make coverage-html"

## coverage-html: Generate and open HTML coverage report
coverage-html: test-coverage
	@echo "Generating HTML coverage report..."
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Opening coverage report in browser..."
	@if command -v open > /dev/null; then \
		open $(COVERAGE_HTML); \
	elif command -v xdg-open > /dev/null; then \
		xdg-open $(COVERAGE_HTML); \
	else \
		echo "Coverage report generated: $(COVERAGE_HTML)"; \
	fi

## lint: Run linters (go fmt, go vet)
lint:
	@echo "Running go fmt..."
	$(GOFMT) ./...
	@echo "Running go vet..."
	$(GOVET) ./...
	@echo "Lint complete"

## deps: Download and tidy dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	@echo "Tidying dependencies..."
	$(GOMOD) tidy
	@echo "Dependencies updated"

## clean: Remove build artifacts and test outputs
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BUILD_DIR)/$(CANOPY_BINARY)
	rm -rf $(COVERAGE_FILE)
	rm -rf $(COVERAGE_HTML)
	@echo "Clean complete"

## run: Build and run canopy
run: build
	@echo "Running $(CANOPY_BINARY)..."
	./$(CANOPY_BINARY)

## install: Install canopy binary to GOPATH/bin
install:
	@echo "Installing $(CANOPY_BINARY)..."
	$(GOCMD) install $(CANOPY_CMD)
	@echo "Installed to $(shell go env GOPATH)/bin/"
