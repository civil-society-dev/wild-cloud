.PHONY: help build dev test clean install lint fmt vet

# Binary name
BINARY_NAME=wildd
BUILD_DIR=build

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build flags
LDFLAGS=-ldflags "-s -w"

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the daemon binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

dev: ## Run the daemon in development mode with live reloading
	@echo "Starting $(BINARY_NAME) in development mode..."
	$(GOCMD) run .

test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v ./...

test-cover: ## Run tests with coverage
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

install: build ## Install the binary to $(go env GOPATH)/bin
	@echo "Installing $(BINARY_NAME)..."
	@mkdir -p $$($(GOCMD) env GOPATH)/bin
	@cp $(BUILD_DIR)/$(BINARY_NAME) $$($(GOCMD) env GOPATH)/bin/

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

fmt: ## Format Go code
	@echo "Formatting code..."
	$(GOFMT) ./...

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...

lint: fmt vet ## Run formatters and linters

check: lint test ## Run all checks (lint + test)

all: clean lint test build ## Clean, lint, test, and build
