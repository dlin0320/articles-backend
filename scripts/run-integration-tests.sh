#!/bin/bash

# Integration Test Runner
# Provides options for running integration tests with clean database state
# Usage: 
#   ./scripts/run-integration-tests.sh [OPTIONS]
#
# Options:
#   --fresh-db    Start services with fresh database (no persistence)
#   --cleanup     Clean existing database and reuse running services  
#   --help        Show this help message

set -e

FRESH_DB=false
CLEANUP_ONLY=false
DOCKER_COMPOSE_FILES="-f docker-compose.yml"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --fresh-db)
            FRESH_DB=true
            DOCKER_COMPOSE_FILES="-f docker-compose.yml -f docker-compose.test.yml"
            shift
            ;;
        --cleanup)
            CLEANUP_ONLY=true
            shift
            ;;
        --help)
            echo "Integration Test Runner"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --fresh-db    Start services with fresh database (no persistence)"
            echo "  --cleanup     Clean existing database and reuse running services"
            echo "  --help        Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                    # Run tests with existing services"
            echo "  $0 --cleanup          # Clean database then run tests"
            echo "  $0 --fresh-db         # Start fresh services and run tests"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

echo "🧪 Articles Backend Integration Test Runner"
echo "==========================================="

if [ "$FRESH_DB" = true ]; then
    echo "🔄 Starting services with fresh database (no persistence)..."
    
    # Stop existing services
    docker-compose down -v --remove-orphans 2>/dev/null || true
    
    # Start with test configuration
    docker-compose $DOCKER_COMPOSE_FILES up --build -d
    
    # Wait for services to be ready
    echo "⏳ Waiting for services to be ready..."
    timeout=60
    while [ $timeout -gt 0 ]; do
        if curl -s http://localhost:8080/health > /dev/null 2>&1 && \
           curl -s http://localhost:8001/health > /dev/null 2>&1; then
            echo "✅ Services are ready!"
            break
        fi
        sleep 2
        timeout=$((timeout - 2))
    done
    
    if [ $timeout -le 0 ]; then
        echo "❌ Services failed to start within timeout"
        docker-compose $DOCKER_COMPOSE_FILES logs
        exit 1
    fi
    
elif [ "$CLEANUP_ONLY" = true ]; then
    echo "🧹 Cleaning existing database..."
    
    # Check if services are running
    if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo "❌ Services are not running. Start them first with: docker-compose up -d"
        exit 1
    fi
    
    # Run cleanup script
    ./scripts/cleanup-database.sh
    
else
    echo "ℹ️  Using existing services (no cleanup)"
    echo "   Tip: Use --cleanup to clean database or --fresh-db for fresh services"
fi

echo ""
echo "🏃 Running integration tests..."
echo ""

# Run the integration tests
if go test -tags=integration ./tests/integration/... -v -timeout=120s; then
    echo ""
    echo "🎉 All integration tests passed!"
    exit 0
else
    echo ""
    echo "❌ Some integration tests failed."
    echo ""
    echo "💡 Troubleshooting tips:"
    echo "   - Try running with --fresh-db for a completely clean environment"
    echo "   - Check service logs: docker-compose logs"
    echo "   - Verify services are healthy:"
    echo "     curl http://localhost:8080/health"
    echo "     curl http://localhost:8001/health"
    exit 1
fi