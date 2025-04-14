#!/bin/bash

# Exit on any error
set -e

# Check if VERSION is set
if [ -z "${VERSION}" ]; then
  echo "Error: VERSION is not set"
  exit 1
fi

# Create output directory
mkdir -p output

echo "Building trading-backtest binary version: ${VERSION}"

# Build the Go binary with version information
go build -o output/trading-backtest \
  -ldflags "-X main.Version=${VERSION} -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  ./cmd/backtest

echo "Build completed: output/trading-backtest"

# Make the binary executable
chmod +x output/trading-backtest 