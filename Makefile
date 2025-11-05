.PHONY: help all build build-web build-go test test-coverage lint fmt fmt-check ci clean dev install-web

# Default target
all: build

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install-web: ## Install web dependencies
	cd web && yarn install

build-web: ## Build CSS and web assets
	cd web && yarn build

build-go: ## Build Go binary
	go build -o bin/chatbotgate ./cmd/chatbotgate

build: build-web build-go ## Build everything (web + go)

test: ## Run all tests
	go test ./...

test-coverage: ## Run tests with coverage
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint: ## Run linters (golangci-lint)
	golangci-lint run ./...

fmt: ## Format Go code
	gofmt -w .

fmt-check: ## Check code formatting (CI)
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "The following files are not formatted:"; \
		gofmt -l .; \
		exit 1; \
	fi

ci: fmt-check lint test ## Run all CI checks (format, lint, test)

dev-web: ## Run web dev server (design system catalog)
	cd web && yarn dev

clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf web/dist/
	rm -rf web/node_modules/
	rm -f coverage.out coverage.html

run: build ## Build and run the server
	./bin/chatbotgate -c config.example.yaml
