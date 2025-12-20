#!/bin/bash

# Exit on any error
set -e

# Check if required variables are set
if [ -z "${SIGNING_CERTIFICATE_NAME}" ]; then
  echo "Error: SIGNING_CERTIFICATE_NAME is not set"
  exit 1
fi

# Import binary configuration
source "$(dirname "$0")/binaries.sh"

# Sign and verify each binary
for binary in "${BINARIES[@]}"; do
  BINARY_PATH="output/${binary}"
  
  echo "Signing binary: ${BINARY_PATH}"

  # Sign the binary with hardened runtime and entitlements
  codesign --force --options runtime --timestamp --entitlements "$(dirname "$0")/app.entitlements" --sign "${SIGNING_CERTIFICATE_NAME}" "${BINARY_PATH}"
  
  # Verify signature
  codesign --verify --verbose "${BINARY_PATH}"
done

echo "Binaries signed successfully"