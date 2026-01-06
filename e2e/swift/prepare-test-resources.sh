#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
RESOURCES_DIR="$SCRIPT_DIR/Tests/ArgoTradingE2ETests/Resources"

echo "Preparing test resources for Swift e2e tests..."

# Create resources directory if it doesn't exist
mkdir -p "$RESOURCES_DIR"

# Build the SMA strategy WASM plugin
echo "Building SMA strategy WASM plugin..."
cd "$PROJECT_ROOT/e2e/backtest/wasm/sma"

# Check if go is available
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH"
    exit 1
fi

# Build the WASM plugin
GOOS=wasip1 GOARCH=wasm go build -o sma_plugin.wasm .
echo "WASM plugin built: sma_plugin.wasm"

# Copy the WASM plugin to resources
cp sma_plugin.wasm "$RESOURCES_DIR/"
echo "Copied sma_plugin.wasm to resources"

# Copy test data
echo "Copying test data..."
cp "$PROJECT_ROOT/internal/indicator/test_data/test_data.parquet" "$RESOURCES_DIR/"
echo "Copied test_data.parquet to resources"

# Create test configuration file
echo "Creating test configuration file..."
cat > "$RESOURCES_DIR/config.json" << 'EOF'
{
    "fastPeriod": 10,
    "slowPeriod": 20,
    "symbol": "BTCUSDT"
}
EOF
echo "Created config.json"

echo "Test resources prepared successfully!"
ls -la "$RESOURCES_DIR"
