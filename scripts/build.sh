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
# Note: We inject version into both main package and internal/version package
LDFLAGS="-X main.Version=${VERSION} -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X github.com/rxtech-lab/argo-trading/internal/version.Version=${VERSION}"

go build -o output/trading-backtest \
  -ldflags "${LDFLAGS}" \
  ./cmd/backtest

go build -o output/trading-market \
  -ldflags "${LDFLAGS}" \
  ./cmd/market

echo "Build completed"

# Make the binary executable
chmod +x output/trading-backtest 
chmod +x output/trading-market