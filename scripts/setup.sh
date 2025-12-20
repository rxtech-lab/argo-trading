#!/bin/bash
set -e

# Variables for go-plugin
PLUGIN_URL="https://github.com/knqyf263/go-plugin/releases/download/v0.9.0/go-plugin_0.9.0_linux-amd64.deb"
DOWNLOAD_DIR="/tmp"
PACKAGE_NAME="go-plugin_0.9.0_linux-amd64.deb"


# if on macos, download url should be https://github.com/knqyf263/go-plugin/releases/download/v0.9.0/go-plugin_0.9.0_darwin-arm64.tar.gz
if [ "$(uname)" == "Darwin" ]; then
    PLUGIN_URL="https://github.com/knqyf263/go-plugin/releases/download/v0.9.0/go-plugin_0.9.0_darwin-arm64.tar.gz"
    PACKAGE_NAME="go-plugin_0.9.0_darwin-arm64.tar.gz"
fi

echo "Downloading go-plugin package..."
curl -L -o "$DOWNLOAD_DIR/$PACKAGE_NAME" "$PLUGIN_URL"

echo "Installing go-plugin package..."
if [ "$(uname)" == "Linux" ]; then
    sudo dpkg -i "$DOWNLOAD_DIR/$PACKAGE_NAME"
    echo "Go-plugin installation complete!"

    echo "Installing Protocol Buffers compiler (protoc)..."
    # Install protoc using apt
    echo "Updating package lists..."
    sudo apt-get update

    echo "Installing protoc via apt..."
    sudo apt-get install -y protobuf-compiler
elif [ "$(uname)" == "Darwin" ]; then
    tar -xzf "$DOWNLOAD_DIR/$PACKAGE_NAME" -C /usr/local/bin
    echo "Go-plugin installation complete!"

    echo "Installing Protocol Buffers compiler (protoc)..."
    brew install protobuf
    echo "Protocol Buffers compiler (protoc) installation complete!"
else
    echo "Error: This package is for Linux systems only."
    echo "Package downloaded to $DOWNLOAD_DIR/$PACKAGE_NAME but not installed."
    exit 1
fi

# Verify installation
echo "Verifying protoc installation..."
protoc --version

# Install mockgen
echo "Installing mockgen..."
go install go.uber.org/mock/mockgen@latest

# Install gomobile
go install golang.org/x/mobile/cmd/gomobile@latest
go get golang.org/x/mobile@latest

# Clean up
echo "Cleaning up..."
rm -f "$DOWNLOAD_DIR/$PACKAGE_NAME"

echo "Installations complete and temporary files removed."