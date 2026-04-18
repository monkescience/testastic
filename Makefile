VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: test test-unit lint fmt clean mod-tidy coverage help

help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

test: ## Run all tests
	go test -race ./...

test-unit: ## Run unit tests only (skip integration via -short)
	go test -race -short ./...

coverage: ## Run tests with coverage
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

lint: ## Run linter
	golangci-lint run --timeout=5m

fmt: ## Format code
	golangci-lint fmt

clean: ## Clean build artifacts
	rm -f coverage.out coverage.html

mod-tidy: ## Tidy Go modules
	go mod tidy
