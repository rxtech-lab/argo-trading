# Argo Trading Strategy Framework

A powerful framework for developing, testing, and implementing algorithmic trading strategies.

## Documents

Documents are located in [`Docs folder`](./docs)

## Prerequisites

- **Go 1.24+** required
- [Go-wasm plugin](https://github.com/knqyf263/go-plugin) installed
- Basic understanding of Go programming and trading concepts

## Project Setup

1. Clone the repository
2. Install dependencies

   ```bash
   go mod download
   ```

3. Generate strategy interfaces
   ```bash
   make generate
   ```

## Project Structure

```
argo-trading/
├── cmd/                # Command-line tools
├── examples/           # Example implementations
│   └── strategy/       # Strategy examples
├── pkg/                # Public API packages
│   └── strategy/       # Strategy interface definitions
```

## Implementing Your Strategy

To create a new trading strategy, you can run

```bash
pnpm create trading-strategy
```

Or

```bash
npx create-trading-strategy
```

This will automatically create a sample strategy on your local machine.

## Use in Swift

### Swift Package Manager

You can add this repository directly as a Swift package in Xcode:

1. In Xcode, go to **File → Add Package Dependencies…**
2. Enter the repository URL: `https://github.com/rxtech-lab/argo-trading`
3. Select the desired version and add the `ArgoTrading` package to your target.

### Local Development

If you need to develop or test locally against a custom build of the framework:

1. Build the Swift framework:
   ```bash
   make build-swift-argo
   ```

2. Sign the framework:
   ```bash
   .scripts/sign-framework.sh
   ```

3. In Xcode, add the local `ArgoTradingLocal` package (pointing to your local repository checkout) instead of the remote package.

### Dependencies

- `libduckdb.dylib` is required.
- Download `libduckdb-osx-universal.zip` from the [DuckDB releases page](https://github.com/duckdb/duckdb/releases).
- Drag `libduckdb.dylib` into your Xcode project and ensure it's included in your target's **Frameworks, Libraries, and Embedded Content** section.
