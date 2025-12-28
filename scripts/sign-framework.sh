#!/bin/bash
set -e

# Check if required variables are set
if [ -z "${SIGNING_CERTIFICATE_NAME}" ]; then
  echo "Error: SIGNING_CERTIFICATE_NAME is not set"
  exit 1
fi

XCFRAMEWORK_PATH="pkg/swift-argo/ArgoTrading.xcframework"

# Sign all inner frameworks first (inside-out signing)
echo "Signing inner frameworks..."
find "$XCFRAMEWORK_PATH" -name "*.framework" -type d | while read -r framework; do
  echo "Signing: $framework"
  codesign --force --sign "$SIGNING_CERTIFICATE_NAME" --options runtime --timestamp "$framework"
done

# Sign the xcframework itself
echo "Signing xcframework..."
codesign --force --sign "$SIGNING_CERTIFICATE_NAME" --options runtime --timestamp "$XCFRAMEWORK_PATH"

# Verify the signature
echo "Verifying signature..."
codesign --verify --deep --strict "$XCFRAMEWORK_PATH"
echo "Signature verified successfully"