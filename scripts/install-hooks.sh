#!/bin/bash
#
# Script to install git pre-commit hook
# This can be run independently or as part of scripts/setup.sh

set -e

# Get the directory of this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Get git directory
GIT_DIR="$(git rev-parse --git-dir 2>/dev/null)"

if [ -z "$GIT_DIR" ]; then
    echo "Error: Not in a git repository"
    exit 1
fi

echo "Installing git pre-commit hook..."
cp "$SCRIPT_DIR/hooks/pre-commit" "$GIT_DIR/hooks/pre-commit"
chmod +x "$GIT_DIR/hooks/pre-commit"
echo "Git pre-commit hook installed successfully!"
echo ""
echo "The pre-commit hook will now run 'go fmt ./...' before each commit."
echo "If formatting changes are needed, the commit will be aborted and you'll"
echo "need to stage the formatted files and commit again."
