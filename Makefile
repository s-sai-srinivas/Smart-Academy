.PHONY: help lint test build run clean docker-up docker-down

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ═══════════════════════════════════════
# Backend
# ═══════════════════════════════════════

backend-lint: ## Lint Go code
	cd backend && golangci-lint run ./...

backend-test: ## Run Go tests
	cd backend && go test ./... -v -count=1 -timeout 60s

backend-test-cover: ## Run Go tests with coverage
	cd backend && go test ./... -v -count=1 -coverprofile=coverage.out -timeout 60s

backend-build: ## Build Go binary
	cd backend && go build -o bin/app .

backend-run: ## Run Go server locally
	cd backend && go run .

backend-tidy: ## Tidy Go modules
	cd backend && go mod tidy

backend-seed: ## Run the database seeder script
	cd backend && go run ../cmd/seed/main.go

# ═══════════════════════════════════════
# Frontend
# ═══════════════════════════════════════

frontend-lint: ## Lint TypeScript/React code
	cd frontend && npx eslint src/ --ext .ts,.tsx

frontend-typecheck: ## TypeScript type checking
	cd frontend && npx tsc --noEmit

frontend-format: ## Format frontend code with Prettier
	cd frontend && npx prettier --write "src/**/*.{ts,tsx,css,json}"

frontend-format-check: ## Check frontend formatting (CI)
	cd frontend && npx prettier --check "src/**/*.{ts,tsx,css,json}"

frontend-build: ## Build frontend for production
	cd frontend && npm run build

frontend-dev: ## Start frontend dev server
	cd frontend && npm run dev

# ═══════════════════════════════════════
# Docker
# ═══════════════════════════════════════

docker-build: ## Build Docker images
	docker compose build

docker-up: ## Start all services
	docker compose up -d

docker-down: ## Stop all services
	docker compose down

docker-logs: ## View Docker logs
	docker compose logs -f

# ═══════════════════════════════════════
# All at once
# ═══════════════════════════════════════

lint: backend-lint frontend-lint ## Lint everything

test: backend-test ## Test everything

build: backend-build frontend-build ## Build everything

clean: ## Remove build artifacts
	rm -f backend/bin/app
	rm -rf frontend/dist
