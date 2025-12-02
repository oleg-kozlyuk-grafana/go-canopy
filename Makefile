.PHONY: help build build-all build-canopy build-initiator build-worker build-all-in-one test test-coverage lint docker-build local-up local-down clean deps

# Variables
BUILD_DIR=.
DOCKER_IMAGE=canopy:latest
DOCKER_COMPOSE_FILE=deployments/docker-compose.yml
COVERAGE_FILE=.coverage/coverage.out
COVERAGE_HTML=.coverage/report.html

# Executables
CANOPY_BINARY=canopy
INITIATOR_BINARY=canopy-initiator
WORKER_BINARY=canopy-worker
ALL_IN_ONE_BINARY=canopy-all-in-one

# Command directories
CANOPY_CMD=./cmd/canopy
INITIATOR_CMD=./cmd/initiator
WORKER_CMD=./cmd/worker
ALL_IN_ONE_CMD=./cmd/all-in-one

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build flags
LDFLAGS=-ldflags "-s -w"

## help: Display this help message
help:
	@echo "Canopy - Coverage Annotation Service"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /' | column -t -s ':'

## build: Build all binaries
build: build-all

## build-all: Build all binaries (canopy, initiator, worker, all-in-one)
build-all: build-canopy build-initiator build-worker build-all-in-one
	@echo "All binaries built successfully"

## build-canopy: Build the canopy binary (local mode)
build-canopy:
	@echo "Building $(CANOPY_BINARY)..."
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CANOPY_BINARY) $(CANOPY_CMD)
	@echo "Build complete: $(BUILD_DIR)/$(CANOPY_BINARY)"

## build-initiator: Build the initiator binary
build-initiator:
	@echo "Building $(INITIATOR_BINARY)..."
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(INITIATOR_BINARY) $(INITIATOR_CMD)
	@echo "Build complete: $(BUILD_DIR)/$(INITIATOR_BINARY)"

## build-worker: Build the worker binary
build-worker:
	@echo "Building $(WORKER_BINARY)..."
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(WORKER_BINARY) $(WORKER_CMD)
	@echo "Build complete: $(BUILD_DIR)/$(WORKER_BINARY)"

## build-all-in-one: Build the all-in-one binary
build-all-in-one:
	@echo "Building $(ALL_IN_ONE_BINARY)..."
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(ALL_IN_ONE_BINARY) $(ALL_IN_ONE_CMD)
	@echo "Build complete: $(BUILD_DIR)/$(ALL_IN_ONE_BINARY)"

## test: Run all tests (excluding integration tests)
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -short ./...

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -race ./internal/storage/integration_test.go ./internal/storage/interface.go ./internal/storage/minio.go ./internal/storage/gcs.go

## test-all: Run all tests including integration tests
test-all:
	@echo "Running all tests including integration tests..."
	$(GOTEST) -v -race ./...

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

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image: $(DOCKER_IMAGE)"
	docker build -t $(DOCKER_IMAGE) -f deployments/Dockerfile .
	@echo "Docker image built: $(DOCKER_IMAGE)"

## local-up: Start local development environment (docker-compose)
local-up:
	@echo "Starting local development environment..."
	@if [ ! -f $(DOCKER_COMPOSE_FILE) ]; then \
		echo "Error: $(DOCKER_COMPOSE_FILE) not found"; \
		echo "Please create the docker-compose.yml file first"; \
		exit 1; \
	fi
	docker-compose -f $(DOCKER_COMPOSE_FILE) up -d
	@echo ""
	@echo "Local environment started"
	@echo "Services:"
	@echo "  - Canopy:  http://localhost:8080"
	@echo "  - Redis:   localhost:6379"
	@echo "  - MinIO:   http://localhost:9000 (console: http://localhost:9001)"
	@echo ""
	@echo "To view logs: docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f"
	@echo "To stop:      make local-down"

## local-down: Stop local development environment
local-down:
	@echo "Stopping local development environment..."
	@if [ ! -f $(DOCKER_COMPOSE_FILE) ]; then \
		echo "Error: $(DOCKER_COMPOSE_FILE) not found"; \
		exit 1; \
	fi
	docker-compose -f $(DOCKER_COMPOSE_FILE) down
	@echo "Local environment stopped"

## local-logs: Show logs from local development environment
local-logs:
	@if [ ! -f $(DOCKER_COMPOSE_FILE) ]; then \
		echo "Error: $(DOCKER_COMPOSE_FILE) not found"; \
		exit 1; \
	fi
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f

## clean: Remove build artifacts and test outputs
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BUILD_DIR)/$(CANOPY_BINARY)
	rm -f $(BUILD_DIR)/$(INITIATOR_BINARY)
	rm -f $(BUILD_DIR)/$(WORKER_BINARY)
	rm -f $(BUILD_DIR)/$(ALL_IN_ONE_BINARY)
	rm -rf $(COVERAGE_FILE)
	rm -rf $(COVERAGE_HTML)
	@echo "Clean complete"

## run-canopy: Build and run canopy (local mode)
run-canopy: build-canopy
	@echo "Running $(CANOPY_BINARY)..."
	./$(CANOPY_BINARY)

## run-initiator: Build and run initiator
run-initiator: build-initiator
	@echo "Running $(INITIATOR_BINARY)..."
	./$(INITIATOR_BINARY)

## run-worker: Build and run worker
run-worker: build-worker
	@echo "Running $(WORKER_BINARY)..."
	./$(WORKER_BINARY)

## run-all-in-one: Build and run all-in-one
run-all-in-one: build-all-in-one
	@echo "Running $(ALL_IN_ONE_BINARY)..."
	./$(ALL_IN_ONE_BINARY)

## install: Install all binaries to GOPATH/bin
install:
	@echo "Installing binaries..."
	$(GOCMD) install $(CANOPY_CMD)
	$(GOCMD) install $(INITIATOR_CMD)
	$(GOCMD) install $(WORKER_CMD)
	$(GOCMD) install $(ALL_IN_ONE_CMD)
	@echo "Installed to $(shell go env GOPATH)/bin/"
