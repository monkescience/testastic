VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: test lint fmt clean mod-tidy coverage help

help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

test: ## Run all tests with race detection and coverage profile
	@mkdir -p coverage
	go test -race -covermode=atomic -coverprofile=coverage/coverage.out ./...

coverage: ## Generate HTML coverage report from coverage/coverage.out
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html

lint: ## Run linter
	golangci-lint run --timeout=5m

fmt: ## Format code
	golangci-lint fmt

clean: ## Clean coverage artifacts
	rm -rf coverage/

mod-tidy: ## Tidy Go modules
	go mod tidy
