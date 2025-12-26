.PHONY: help ingest build cli test clean setup

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

setup: ## Initial project setup (run once)
	@echo "Setting up algorave..."
	@cp .env.example .env
	@echo "✓ Created .env file - please edit with your API keys"
	@go mod download
	@echo "✓ Downloaded dependencies"
	@mkdir -p bin docs/strudel
	@echo "✓ Created directories"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Edit .env with your API keys"
	@echo "  2. Run schema.sql in Supabase SQL Editor"
	@echo "  3. Add markdown docs to docs/strudel/"
	@echo "  4. Run 'make ingest' to index documentation"

ingest: ## Run document ingestion
	@echo "Ingesting documentation..."
	go run ./cmd/ingester all --clear

ingest-no-clear: ## Run ingestion without clearing existing data
	@echo "Ingesting documentation (no clear)..."
	go run ./cmd/ingester all

server: ## Run API server
	@echo "Starting API server..."
	go run ./cmd/server

build: ## Build binaries
	@echo "Building binaries..."
	@mkdir -p bin
	go build -o bin/ingester ./cmd/ingester
	go build -o bin/server ./cmd/server
	go build -o bin/algorave ./cmd/tui
	@echo "✓ Built all binaries in bin/"

cli: ## Build local CLI
	@echo "Building local CLI..."
	@mkdir -p bin
	go build -o bin/algorave ./cmd/tui
	@echo "✓ Built bin/algorave"

test: ## Run tests
	go test -v ./...

test-coverage: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html
	@echo "✓ Cleaned build artifacts"

fmt: ## Format code
	go fmt ./...
	@echo "✓ Formatted code"

lint: ## Run linter (requires golangci-lint)
	golangci-lint run

deps: ## Download dependencies
	go mod download
	go mod tidy
	@echo "✓ Dependencies updated"

ci: ## Run CI checks locally (lint + test)
	@echo "Running CI checks..."
	@echo "\n→ Checking formatting..."
	@if [ -n "$$(gofmt -s -l .)" ]; then \
		echo "❌ Code is not formatted. Run 'make fmt' to fix."; \
		gofmt -s -l .; \
		exit 1; \
	fi
	@echo "✓ Code is formatted"
	@echo "\n→ Running linter..."
	@golangci-lint run || (echo "❌ Linting failed" && exit 1)
	@echo "✓ Linting passed"
	@echo "\n→ Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@echo "✓ All CI checks passed!"

.DEFAULT_GOAL := help

db-migrate: ## Apply pending migrations
	@echo "Applying migrations to Supabase..."
	supabase db push
	@echo "✓ Migrations applied"