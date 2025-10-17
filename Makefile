.PHONY: build clean test run lint lint-golangci format validate-examples docs-update-tags help release test-e2e

# Binary name
BINARY_NAME=mcpshell
# Build directory
BUILD_DIR=build

# Go related variables
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/$(BUILD_DIR)

# Default target
all: build

# Build the application
build:
	@echo ">>> Building $(BINARY_NAME)..."
	@mkdir -p $(GOBIN)
	@go build -o $(GOBIN)/$(BINARY_NAME) .
	@echo ">>> ... $(BINARY_NAME) built successfully"

# Clean build artifacts
clean:
	@echo ">>> Cleaning..."
	@rm -rf $(BUILD_DIR)

# Run tests
test:
	@echo ">>> Running tests..."
	@go test -v ./...
	@echo ">>> ... tests completed successfully"

# Run the exe command tests
test-e2e:
	@echo ">>> Running exe command tests..."
	@if [ ! -x "$(GOBIN)/$(BINARY_NAME)" ]; then \
		echo ">>> $(GOBIN)/$(BINARY_NAME) not found. Building..."; \
		$(MAKE) build; \
	fi
	@chmod +x tests/*.sh
	@tests/run_tests.sh
	@echo ">>> ... exe command tests completed"

# Run the application
run:
	@go run main.go

# Install the application
install:
	@echo ">>> Installing $(BINARY_NAME)..."
	@go install .
	@echo ">>> ... $(BINARY_NAME) installed successfully"

# Run linting (golangci-lint)
lint: lint-golangci

# Run golangci-lint (comprehensive linting)
lint-golangci:
	@echo ">>> Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run; \
	fi
	@echo ">>> ... golangci-lint completed successfully"

# Legacy linting - deprecated but kept for backward compatibility
lint-legacy:
	@echo ">>> Running legacy linting (golint)..."
	@golint ./...
	@echo ">>> ... legacy linting completed successfully"

# Format code
format:
	@echo ">>> Formatting Go code..."
	@go fmt ./...
	@go mod tidy
	@echo ">>> ... code formatted successfully"

format-md:
	@echo ">>> Formatting Markdown code..."
	@find . -name "*.md" -type f -exec mdformat {} \;
	@echo ">>> ... markdown code formatted successfully"

# Validate all YAML configuration files in examples directory
validate-examples: build
	@echo ">>> Validating example YAML configurations..."
	@find examples -name "*.yaml" -type f | while read file; do \
		echo "--------------------------------------------------------------"; \
		echo ">>> Validating $$file..."; \
		$(GOBIN)/$(BINARY_NAME) validate --tools $$file || exit 1; \
	done
	@echo ">>>"
	@echo ">>> ... all example configurations validated SUCCESSFULLY !!!"

# Automated release process
release:
	@echo ">>> Checking repository status..."
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Error: Repository has uncommitted changes. Please commit or stash them first."; \
		exit 1; \
	fi
	@echo ">>> Repository is clean."
	@echo ">>> Existing tags:"
	@git tag -l | sort -V
	@echo ""
	@read -p "Enter new version tag (e.g., v1.2.3): " TAG; \
	echo ">>> Using tag: $$TAG"; \
	echo ">>> Updating version tags in documentation..."; \
	find . -name "*.md" -type f -exec sed -i.bak -E "s|github.com/inercia/MCPShell@v[0-9]+\.[0-9]+\.[0-9]+|github.com/inercia/MCPShell@$$TAG|g" {} \; -exec rm {}.bak \; ; \
	echo ">>> Documentation version tags updated successfully"; \
	echo ">>> Adding and committing documentation changes..."; \
	git add -u ; \
	git commit -m "Release $$TAG"; \
	echo ">>> Creating git tag..."; \
	git tag -a "$$TAG" -m "Version $$TAG"; \
	echo ">>> Tag '$$TAG' created successfully."; \
	echo ""; \
	echo "To push the tag and documentation changes, run:"; \
	echo "  git push origin main $$TAG"

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the application"
	@echo "  clean         - Remove build artifacts"
	@echo "  test          - Run tests"
	@echo "  test-e2e      - Run end-to-end tests"
	@echo "  run           - Run the application"
	@echo "  install       - Install the application"
	@echo "  lint          - Run linting (alias for lint-golangci)"
	@echo "  lint-golangci - Run golangci-lint (installs if not present)"
	@echo "  lint-legacy   - Run legacy linting with golint"
	@echo "  format        - Format Go code"
	@echo "  validate-examples - Validate all YAML configs in examples directory"
	@echo "  release       - Automated release process"
	@echo "  help          - Show this help" 