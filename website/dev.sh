#!/bin/bash

# Script to run development environment with Hugo and Node.js

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Starting development environment..."
echo "- Node.js: watching and building TypeScript/React"
echo "- Hugo: development server at http://localhost:1313"
echo ""
echo "Press Ctrl+C to stop all services"
echo ""

cd "$SCRIPT_DIR"

# Start services with docker-compose
docker-compose -f docker-compose.dev.yml up
