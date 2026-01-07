set -e

# Check any potential nil pointer dereferences in the codebase
nilaway -include-pkgs="github.com/rxtech-lab/argo-trading" --exclude-pkgs="github.com/rxtech-lab/argo-trading/pkg" ./...

# Check if we're running in CI environment
if [ -n "$CI" ]; then
    echo "Running in CI environment"
    # Add any CI-specific linting commands here
else
    echo "Running in local environment"
    # Add any local-specific linting commands here
    golangci-lint run ./... --fix
fi