# GitHub Copilot Instructions for Argo Trading

This document provides guidelines for GitHub Copilot when working with the Argo Trading repository.

## Commit Style

Always use conventional commit style for all commits. Follow this format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code (white-space, formatting, etc.)
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `perf`: A code change that improves performance
- `test`: Adding missing tests or correcting existing tests
- `chore`: Changes to the build process or auxiliary tools and libraries
- `ci`: Changes to CI configuration files and scripts

**Examples:**
```
feat(indicator): add RSI indicator implementation
fix(backtest): correct position calculation in engine
docs(readme): update installation instructions
test(trading): add unit tests for order validation
```

## Code Organization

**Split large files into smaller files:**
- Keep files focused on a single responsibility
- When a file exceeds 500 lines, consider splitting it into smaller, logical components
- Group related functionality into separate files
- Use meaningful file names that reflect their contents
- For large structs or interfaces, consider splitting implementations across multiple files

## Testing Requirements

**All changes must include unit tests:**
- Every new feature must have corresponding unit tests
- Bug fixes should include tests that would have caught the bug
- Use table-driven tests with `testify/suite` framework (see `internal/indicator/rsi_test.go` for reference)
- For indicator and DB tests, use real datasources (not mocks)
- Aim for meaningful test coverage, not just high percentages
- Test edge cases and error conditions

## Pre-Commit Checklist

**Before committing any code, run these commands in order:**

1. **Format code:**
   ```bash
   go fmt ./...
   ```

2. **Run linters:**
   ```bash
   make lint
   ```
   This runs `golangci-lint run ./...`. Also consider running nilaway:
   ```bash
   ./scripts/lint.sh
   ```

3. **Run tests:**
   ```bash
   make test
   ```
   This runs `go test ./...`. For more thorough testing:
   ```bash
   go test -v -race -cover ./...
   ```

4. **If you modified protobuf files or generated code:**
   ```bash
   make generate
   ```

## Additional Guidelines

- Follow existing code patterns and conventions in the repository
- Keep functions small and focused
- Use meaningful variable and function names
- Add comments for complex logic, but prefer self-documenting code
- Handle errors explicitly - don't ignore them
- For strategies, remember they are stateless - store state in cache via `SetCache`/`GetCache`
- When working with WASM strategies, ensure they compile with `GOOS=wasip1 GOARCH=wasm`

## Project-Specific Notes

- This is a Go-based algorithmic trading framework
- Strategies run as WASM plugins with gRPC communication (go-plugin)
- DuckDB handles all data storage with Parquet files for market data
- See `CLAUDE.md` for detailed architecture and development patterns
