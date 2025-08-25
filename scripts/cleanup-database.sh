#!/bin/bash

# Database cleanup script for integration tests
# Usage: ./scripts/cleanup-database.sh

set -e

echo "🧹 Cleaning up database for integration tests..."

# Check if PostgreSQL is accessible
if ! docker exec articles-postgres pg_isready -U postgres > /dev/null 2>&1; then
    echo "❌ PostgreSQL is not accessible. Make sure docker-compose services are running."
    echo "   Try: docker-compose up -d"
    exit 1
fi

# Run the cleanup SQL script
docker exec -i articles-postgres psql -U postgres -d articles < scripts/cleanup-db.sql

echo "✅ Database cleanup completed successfully!"
echo ""
echo "Ready to run integration tests with clean database state."