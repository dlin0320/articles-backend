# Articles Backend Makefile

.PHONY: help build test test-unit test-integration test-integration-fresh test-integration-cleanup clean-db docker-up docker-down docker-logs

# Default target
help:
	@echo "Articles Backend - Available Commands"
	@echo "===================================="
	@echo ""
	@echo "Development:"
	@echo "  build                 Build the Go application"
	@echo "  run                   Run the application locally"
	@echo ""
	@echo "Docker Services:"
	@echo "  docker-up             Start all services with persistence"
	@echo "  docker-down           Stop all services and remove containers"
	@echo "  docker-logs           Show logs from all services"
	@echo ""
	@echo "Testing:"
	@echo "  test                  Run all tests (unit + integration)"
	@echo "  test-unit             Run unit tests only"
	@echo "  test-integration      Run integration tests (existing services)"
	@echo "  test-integration-fresh Run integration tests with fresh database"
	@echo "  test-integration-cleanup Run integration tests after cleaning database"
	@echo ""
	@echo "Database:"
	@echo "  clean-db              Clean database for fresh test runs"
	@echo ""

# Build commands
build:
	@echo "🔨 Building Go application..."
	go build -o articles-api cmd/api/main.go

run: build
	@echo "🚀 Running application..."
	./articles-api

# Docker commands
docker-up:
	@echo "🐳 Starting Docker services..."
	docker-compose up --build -d

docker-down:
	@echo "🛑 Stopping Docker services..."
	docker-compose down -v --remove-orphans

docker-logs:
	@echo "📋 Showing Docker service logs..."
	docker-compose logs -f

# Test commands
test: test-unit test-integration-cleanup

test-unit:
	@echo "🧪 Running unit tests..."
	go test ./... -v

test-integration:
	@echo "🧪 Running integration tests..."
	./scripts/run-integration-tests.sh

test-integration-fresh:
	@echo "🧪 Running integration tests with fresh database..."
	./scripts/run-integration-tests.sh --fresh-db

test-integration-cleanup:
	@echo "🧪 Running integration tests with database cleanup..."
	./scripts/run-integration-tests.sh --cleanup

# Database commands
clean-db:
	@echo "🧹 Cleaning database..."
	./scripts/cleanup-database.sh

# Convenience aliases
fresh: test-integration-fresh
cleanup: test-integration-cleanup