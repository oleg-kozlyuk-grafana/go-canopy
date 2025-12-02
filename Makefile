.PHONY: help build test test-coverage lint docker-build local-up local-down clean deps

# Variables
BINARY_NAME=canopy
BUILD_DIR=.
CMD_DIR=./cmd/canopy
DOCKER_IMAGE=canopy:latest
DOCKER_COMPOSE_FILE=deployments/docker-compose.yml
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

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

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -coverprofile=$(COVERAGE_FILE) ./...
	@echo ""
	@echo "Coverage summary:"
	$(GOCMD) tool cover -func=$(COVERAGE_FILE)
	@echo ""
	@echo "Checking coverage threshold (80%)..."
	@$(GOCMD) tool cover -func=$(COVERAGE_FILE) | grep total | \
		awk '{print $$3}' | sed 's/%//' | \
		awk '{if ($$1 < 80) {print "❌ Coverage is below 80%: " $$1 "%"; exit 1} else {print "✅ Coverage is above 80%: " $$1 "%"}}'
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
	rm -f $(BUILD_DIR)/$(BINARY_NAME)
	rm -f $(COVERAGE_FILE)
	rm -f $(COVERAGE_HTML)
	@echo "Clean complete"

## run: Build and run the application locally
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

## install: Install the binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOCMD) install $(CMD_DIR)
	@echo "Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"
