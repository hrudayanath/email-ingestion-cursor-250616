.PHONY: all build test clean lint run-backend run-frontend docker-build docker-up docker-down

# Variables
BINARY_NAME=email-harvester
DOCKER_COMPOSE=docker-compose
GO=go
NPM=npm

# Default target
all: build

# Build the backend
build-backend:
	cd backend && $(GO) build -o $(BINARY_NAME) ./cmd/server

# Build the frontend
build-frontend:
	cd frontend && $(NPM) install && $(NPM) run build

# Build both backend and frontend
build: build-backend build-frontend

# Run tests
test:
	cd backend && $(GO) test ./...
	cd frontend && $(NPM) test

# Clean build artifacts
clean:
	rm -f backend/$(BINARY_NAME)
	rm -rf frontend/build
	rm -rf frontend/node_modules

# Run linters
lint:
	cd backend && golangci-lint run
	cd frontend && $(NPM) run lint

# Run backend in development mode
run-backend:
	cd backend && $(GO) run ./cmd/server

# Run frontend in development mode
run-frontend:
	cd frontend && $(NPM) start

# Build Docker images
docker-build:
	$(DOCKER_COMPOSE) build

# Start all services
docker-up:
	$(DOCKER_COMPOSE) up -d

# Stop all services
docker-down:
	$(DOCKER_COMPOSE) down

# Show logs
docker-logs:
	$(DOCKER_COMPOSE) logs -f

# Run database migrations
migrate:
	cd backend && $(GO) run ./cmd/migrate

# Generate API documentation
docs:
	cd backend && swag init -g cmd/server/main.go -o docs

# Install development dependencies
install-dev:
	# Backend
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/swaggo/swag/cmd/swag@latest

	# Frontend
	cd frontend && $(NPM) install

# Help
help:
	@echo "Available targets:"
	@echo "  all            - Build both backend and frontend"
	@echo "  build          - Build both backend and frontend"
	@echo "  build-backend  - Build the backend"
	@echo "  build-frontend - Build the frontend"
	@echo "  test           - Run tests"
	@echo "  clean          - Clean build artifacts"
	@echo "  lint           - Run linters"
	@echo "  run-backend    - Run backend in development mode"
	@echo "  run-frontend   - Run frontend in development mode"
	@echo "  docker-build   - Build Docker images"
	@echo "  docker-up      - Start all services"
	@echo "  docker-down    - Stop all services"
	@echo "  docker-logs    - Show logs"
	@echo "  migrate        - Run database migrations"
	@echo "  docs           - Generate API documentation"
	@echo "  install-dev    - Install development dependencies" 