.PHONY: help ingest build cli test clean setup

help: 
	@echo 'usage: make [target]'
	@echo ''
	@echo 'available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

setup: ## initial project setup (run once)
	@echo "setting up algorave..."
	@cp .env.example .env
	@echo "✓ created .env file - please edit with your API keys"
	@go mod download
	@echo "✓ downloaded dependencies"
	@mkdir -p bin docs/strudel
	@echo "✓ created directories"
	@echo ""
	@echo "next steps:"
	@echo "  1. edit .env with your API keys"
	@echo "  2. run schema.sql in supabase SQL Editor"
	@echo "  3. add markdown docs to docs/strudel/"
	@echo "  4. run 'make ingest' to index documentation"

ingest: ## run document ingestion
	@echo "ingesting documentation..."
	go run ./cmd/ingester all --clear

ingest-no-clear: ## run ingestion without clearing existing data
	@echo "ingesting documentation (no clear)..."
	go run ./cmd/ingester all

server: ## run API server
	@echo "starting API server..."
	go run ./cmd/server

build: ## build binaries
	@echo "building binaries..."
	@mkdir -p bin
	go build -o bin/ingester ./cmd/ingester
	go build -o bin/server ./cmd/server
	go build -o bin/algorave ./cmd/tui
	@echo "✓ built all binaries in bin/"

cli: ## build local CLI
	@echo "building local CLI..."
	@mkdir -p bin
	go build -o bin/algorave ./cmd/tui
	@echo "✓ built bin/algorave"

test: ## run all tests
	@echo "running tests..."
	@go test -v ./...

test-coverage: ## run tests and generate coverage report
	@echo "running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "coverage report: coverage.html"

clean: ## clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html
	@echo "✓ cleaned build artifacts"

fmt: ## format code
	go fmt ./...
	@echo "✓ formatted code"

lint: ## run linter (requires golangci-lint)
	golangci-lint run

deps: ## Download dependencies
	go mod download
	go mod tidy
	@echo "✓ dependencies updated"

ci: ## run CI checks locally (lint + unit tests)
	@echo "running CI checks..."
	@echo "\n→ checking formatting..."
	@if [ -n "$$(gofmt -s -l .)" ]; then \
		echo "x: code is not formatted. run 'make fmt' to fix."; \
		gofmt -s -l .; \
		exit 1; \
	fi
	@echo "✓ code is formatted"
	@echo "\n→ running linter..."
	@golangci-lint run || (echo "x: linting failed" && exit 1)
	@echo "✓ linting passed"
	@echo "\n→ running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@echo "✓ all CI checks passed!"

.DEFAULT_GOAL := help

db-migrate: ## apply pending migrations
	@echo "applying migrations to supabase..."
	supabase db push
	@echo "✓ migrations applied"

docs: ## generate API documentation
	swag init -g cmd/server/main.go --output ./docs --parseDependency --parseInternal	