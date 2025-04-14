#!/bin/bash

# Exit on any error
set -e

# Check if required variables are set
if [ -z "${SIGNING_CERTIFICATE_NAME}" ]; then
  echo "Error: SIGNING_CERTIFICATE_NAME is not set"
  exit 1
fi

# Set binary path
BINARY_PATH="output/trading-backtest"

echo "Signing binary: ${BINARY_PATH}"
# Sign the binary with hardened runtime
codesign --force --options runtime --timestamp --sign "${SIGNING_CERTIFICATE_NAME}" "${BINARY_PATH}"

# Verify signature
codesign --verify --verbose "${BINARY_PATH}"

echo "Binary signed successfully"