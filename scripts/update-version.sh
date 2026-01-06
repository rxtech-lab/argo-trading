#!/bin/bash
set -e

VERSION=$1

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    exit 1
fi

VERSION_FILE="internal/version/version.go"

sed -i.bak "s/var Version = \".*\"/var Version = \"${VERSION}\"/" "$VERSION_FILE"
rm -f "${VERSION_FILE}.bak"

echo "Updated ${VERSION_FILE} to version: ${VERSION}"
