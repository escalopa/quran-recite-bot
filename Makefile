.PHONY: help build run test clean docker-build docker-up docker-down docker-logs

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the bot binary
	go build -o bin/quran-bot ./cmd/bot

run: ## Run the bot locally
	go run ./cmd/bot/main.go

test: ## Run tests
	go test -v ./...

clean: ## Clean build artifacts
	rm -rf bin/

deps: ## Download dependencies
	go mod download
	go mod tidy

docker-build: ## Build Docker image
	docker-compose build

docker-up: ## Start services with docker-compose
	docker-compose up -d

docker-down: ## Stop services
	docker-compose down

docker-logs: ## View logs
	docker-compose logs -f bot

docker-restart: ## Restart bot service
	docker-compose restart bot

lint: ## Run linter
	golangci-lint run ./...

fmt: ## Format code
	go fmt ./...
