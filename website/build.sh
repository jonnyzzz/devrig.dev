#!/bin/bash

# Script to build production website using same containers as dev

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Building production website..."

cd "$SCRIPT_DIR"

# Clean previous build
rm -rf public static/js/**

# Step 1: Install dependencies
echo "→ Installing Node.js dependencies..."
docker run --rm -v "$SCRIPT_DIR:/src" -w /src node:20.18.1-alpine3.20 npm ci

# Step 2: Build TypeScript/React with Webpack
echo "→ Building TypeScript and React with Webpack..."
docker run --rm -v "$SCRIPT_DIR:/src" -w /src node:20.18.1-alpine3.20 npm run build

# Step 4: Build Hugo site
echo "→ Building Hugo site..."
docker run --rm -v "$SCRIPT_DIR:/src" -w /src hugomods/hugo:0.141.0 hugo --minify

echo "✓ Production website built successfully in ./public/"
