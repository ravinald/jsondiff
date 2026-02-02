.PHONY: all build test clean install lint help

# Binary name
BINARY_NAME=jsondiff
BINARY_PATH=./cmd/jsondiff
BUILD_DIR=./build
COVERAGE_FILE=coverage.out

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build flags
LDFLAGS=-ldflags "-s -w"

# Default target
all: test build

## help: Show this help message
help:
	@echo "Available targets:"
	@echo "  make build              - Build the binary"
	@echo "  make test               - Run all tests"
	@echo "  make test-coverage      - Run tests with coverage report"
	@echo "  make test-race          - Run tests with race detector"
	@echo "  make install            - Install binary to GOPATH/bin"
	@echo "  make clean              - Remove build artifacts"
	@echo "  make fmt                - Format code"
	@echo "  make vet                - Run go vet"
	@echo "  make lint               - Run golangci-lint"
	@echo "  make mod-tidy           - Tidy go modules"
	@echo ""
	@echo "Example targets:"
	@echo "  make example            - Run basic diff example"
	@echo "  make example-sort       - Run example with sorting"
	@echo "  make example-config     - Run example with config file"
	@echo "  make example-side       - Run side-by-side example"
	@echo "  make example-include    - Run example with field inclusion"
	@echo "  make example-exclude    - Run example with field exclusion"
	@echo "  make example-nested     - Run example with nested field filter"

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(BINARY_PATH)
	@echo "Binary built: $(BINARY_NAME)"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Generating coverage report..."
	$(GOTEST) -v -coverprofile=$(COVERAGE_FILE) ./...
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report: coverage.html"
	@$(GOCMD) tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print "Total coverage: " $$3}'

## test-race: Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	$(GOTEST) -race ./...

## install: Install the binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	$(GOCMD) install $(BINARY_PATH)
	@echo "$(BINARY_NAME) installed to GOPATH/bin"

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(COVERAGE_FILE) coverage.html
	rm -rf $(BUILD_DIR)
	@echo "Clean complete"

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

## lint: Run golangci-lint
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. See: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run

## mod-tidy: Tidy go modules
mod-tidy:
	@echo "Tidying modules..."
	$(GOMOD) tidy

## mod-download: Download go modules
mod-download:
	@echo "Downloading modules..."
	$(GOMOD) download

# Examples
## example: Run basic diff example
example: build
	@echo "=== Basic diff example ==="
	./$(BINARY_NAME) examples/file1.json examples/file2.json

## example-sort: Run example with sorting
example-sort: build
	@echo "=== Diff with sorted keys ==="
	./$(BINARY_NAME) -s examples/file1.json examples/file2.json

## example-config: Run example with custom config
example-config: build
	@echo "=== Diff with custom color config ==="
	./$(BINARY_NAME) --config examples/config.json examples/file1.json examples/file2.json

## example-side: Run side-by-side diff example
example-side: build
	@echo "=== Side-by-side diff ==="
	./$(BINARY_NAME) -y examples/user1.json examples/user2.json

## example-include: Run example with field inclusion
example-include: build
	@echo "=== Diff with field inclusion (only name and email) ==="
	./$(BINARY_NAME) --include name,email examples/user1.json examples/user2.json

## example-exclude: Run example with field exclusion
example-exclude: build
	@echo "=== Diff with field exclusion (exclude metadata and preferences) ==="
	./$(BINARY_NAME) --exclude metadata,preferences examples/user1.json examples/user2.json

## example-nested: Run example with nested field filter
example-nested: build
	@echo "=== Diff with nested field filter (only address.city) ==="
	./$(BINARY_NAME) --include address.city examples/user1.json examples/user2.json

## example-combined: Run example with combined filters
example-combined: build
	@echo "=== Diff with combined include/exclude filters ==="
	./$(BINARY_NAME) --include user --exclude user.preferences examples/user1.json examples/user2.json

# Development helpers
## watch: Watch for changes and rebuild (requires entr)
watch:
	@which entr > /dev/null || (echo "entr not installed. Install with: brew install entr (macOS) or apt-get install entr (Linux)" && exit 1)
	find . -name "*.go" | entr -c make build

## bench: Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...