#!/bin/bash
set -e

# Usage: ./scripts/update-package-swift.sh <version> <checksum>
# Example: ./scripts/update-package-swift.sh v1.2.3 abc123...

VERSION=$1
CHECKSUM=$2

if [ -z "$VERSION" ] || [ -z "$CHECKSUM" ]; then
    echo "Usage: $0 <version> <checksum>"
    exit 1
fi

PACKAGE_SWIFT="Package.swift"
DOWNLOAD_URL="https://github.com/rxtech-lab/argo-trading/releases/download/${VERSION}/ArgoTrading.xcframework.zip"

# Update url
sed -i.bak "s|https://github.com/rxtech-lab/argo-trading/releases/download/[^\"]*/ArgoTrading.xcframework.zip|${DOWNLOAD_URL}|g" "$PACKAGE_SWIFT"

# Update checksum
sed -i.bak "s|checksum: \"[a-f0-9]*\"|checksum: \"${CHECKSUM}\"|g" "$PACKAGE_SWIFT"

rm -f "${PACKAGE_SWIFT}.bak"

echo "Updated Package.swift:"
echo "  URL: ${DOWNLOAD_URL}"
echo "  Checksum: ${CHECKSUM}"
