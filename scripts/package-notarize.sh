#!/bin/bash

# Exit on any error
set -e

# Check if required variables are set
if [ -z "${INSTALLER_SIGNING_CERTIFICATE_NAME}" ]; then
  echo "Error: INSTALLER_SIGNING_CERTIFICATE_NAME is not set"
  exit 1
fi

if [ -z "${APPLE_ID}" ] || [ -z "${APPLE_ID_PWD}" ] || [ -z "${APPLE_TEAM_ID}" ]; then
  echo "Error: Apple ID credentials not set (APPLE_ID, APPLE_ID_PWD, APPLE_TEAM_ID)"
  exit 1
fi

# Set binary path
TRADING_BACKTEST_BINARY_PATH="output/trading-backtest"
TRADING_DOWNLOAD_BINARY_PATH="output/trading-download"
PKG_FILE="ArgoTrading_macOS_arm64.pkg"
TMP_DIR="tmp_pkg_build"

# Verify the binary is signed
echo "Verifying binary signature: ${TRADING_BACKTEST_BINARY_PATH}"
codesign --verify --verbose "${TRADING_BACKTEST_BINARY_PATH}" || {
  echo "Error: Binary is not properly signed. Run sign.sh first."
  exit 1
}

echo "Verifying binary signature: ${TRADING_DOWNLOAD_BINARY_PATH}"
codesign --verify --verbose "${TRADING_DOWNLOAD_BINARY_PATH}" || {
  echo "Error: Binary is not properly signed. Run sign.sh first."
  exit 1
}

# Create a temporary directory structure for pkgbuild
echo "Creating package structure"
mkdir -p "${TMP_DIR}/usr/local/bin"
cp "${TRADING_BACKTEST_BINARY_PATH}" "${TMP_DIR}/usr/local/bin/"
cp "${TRADING_DOWNLOAD_BINARY_PATH}" "${TMP_DIR}/usr/local/bin/"

# Create the pkg file
echo "Building pkg installer"
pkgbuild --root "${TMP_DIR}" \
  --identifier "com.argotrading.backtest" \
  --version "1.0" \
  --sign "${INSTALLER_SIGNING_CERTIFICATE_NAME}" \
  --install-location "/" \
  "${PKG_FILE}"

# Clean up temporary directory
rm -rf "${TMP_DIR}"

# Notarize the pkg file
echo "Submitting for notarization"
xcrun notarytool submit "${PKG_FILE}" --verbose --apple-id "${APPLE_ID}" --team-id "${APPLE_TEAM_ID}" --password "${APPLE_ID_PWD}" --wait

# Staple the notarization ticket to the pkg
echo "Stapling notarization ticket"
xcrun stapler staple -v "${PKG_FILE}"

echo "Package created, signed, notarized and stapled successfully: ${PKG_FILE}" 