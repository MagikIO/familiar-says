.PHONY: build test clean install run help

# Binary name
BINARY=familiar-says

# Build the application
build:
	@echo "Building $(BINARY)..."
	@go build -o $(BINARY) .
	@echo "Build complete!"

# Run tests
test:
	@echo "Running tests..."
	@go test ./... -v

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY)
	@rm -f coverage.out coverage.html
	@echo "Clean complete!"

# Install the application
install: build
	@echo "Installing $(BINARY)..."
	@cp $(BINARY) $(GOPATH)/bin/
	@echo "Installed to $(GOPATH)/bin/$(BINARY)"

# Run the application with example
run: build
	@echo "Running example..."
	@./$(BINARY) "Hello from familiar-says!"

# Run with animation example
demo: build
	@echo "Running animated demo..."
	@./$(BINARY) --animate --theme rainbow --mood happy --effect confetti "Welcome to familiar-says!"

# Display help
help:
	@echo "Available targets:"
	@echo "  build          - Build the application"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  clean          - Remove build artifacts"
	@echo "  install        - Install to GOPATH/bin"
	@echo "  run            - Build and run with example"
	@echo "  demo           - Run animated demo"
	@echo "  help           - Show this help message"
