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

go build -o output/trading-market \
  -ldflags "-X main.Version=${VERSION} -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  ./cmd/market

echo "Build completed"

# Make the binary executable
chmod +x output/trading-backtest 
chmod +x output/trading-market