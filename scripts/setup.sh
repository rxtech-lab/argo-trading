#!/bin/bash
set -e

# Variables for go-plugin
PLUGIN_URL="https://github.com/knqyf263/go-plugin/releases/download/v0.9.0/go-plugin_0.9.0_linux-amd64.deb"
DOWNLOAD_DIR="/tmp"
PACKAGE_NAME="go-plugin_0.9.0_linux-amd64.deb"

echo "Downloading go-plugin package..."
curl -L -o "$DOWNLOAD_DIR/$PACKAGE_NAME" "$PLUGIN_URL"

echo "Installing go-plugin package..."
if [ "$(uname)" == "Linux" ]; then
    sudo dpkg -i "$DOWNLOAD_DIR/$PACKAGE_NAME"
    echo "Go-plugin installation complete!"
else
    echo "Error: This package is for Linux systems only."
    echo "Package downloaded to $DOWNLOAD_DIR/$PACKAGE_NAME but not installed."
    exit 1
fi

echo "Installing Protocol Buffers compiler (protoc)..."
# Install protoc using apt
echo "Updating package lists..."
sudo apt-get update

echo "Installing protoc via apt..."
sudo apt-get install -y protobuf-compiler

# Verify installation
echo "Verifying protoc installation..."
protoc --version

# Install mockgen
echo "Installing mockgen..."
go install go.uber.org/mock/mockgen@latest

# Clean up
echo "Cleaning up..."
rm -f "$DOWNLOAD_DIR/$PACKAGE_NAME"

echo "Installations complete and temporary files removed."