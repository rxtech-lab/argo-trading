#!/bin/bash

# Exit on any error
set -e

# Check if required variables are set
if [ -z "${SIGNING_CERTIFICATE_NAME}" ]; then
  echo "Error: SIGNING_CERTIFICATE_NAME is not set"
  exit 1
fi

# Set binary path
TRADING_BACKTEST_BINARY_PATH="output/trading-backtest"
TRADING_DOWNLOAD_BINARY_PATH="output/trading-download"

echo "Signing binary: ${TRADING_BACKTEST_BINARY_PATH}"
echo "Signing binary: ${TRADING_DOWNLOAD_BINARY_PATH}"
# Sign the binary with hardened runtime
codesign --force --options runtime --timestamp --sign "${SIGNING_CERTIFICATE_NAME}" "${TRADING_BACKTEST_BINARY_PATH}"
codesign --force --options runtime --timestamp --sign "${SIGNING_CERTIFICATE_NAME}" "${TRADING_DOWNLOAD_BINARY_PATH}"

# Verify signature
codesign --verify --verbose "${TRADING_BACKTEST_BINARY_PATH}"
codesign --verify --verbose "${TRADING_DOWNLOAD_BINARY_PATH}"

echo "Binary signed successfully"