---
title: Wallet API Design
description: API design for the frontend wallet feature supporting multi-asset balances
---

# Wallet API Design

## Overview

This document describes the wallet API design for the frontend (Swift) wallet feature. The wallet API exposes the user's account balances as a list of assets, supporting multiple currencies and asset types (e.g., crypto). This is necessary because trading providers like Binance hold balances across multiple assets (BTC, ETH, USDT, etc.), unlike traditional brokers that report a single cash balance.

## Requirements

### Functional Requirements

1. **Multi-Asset Balance**
   - The wallet must return a list of asset balances, not a single balance value
   - Each asset entry contains:
     - `title`: Human-readable asset name (e.g., "Bitcoin", "Ethereum", "US Dollar")
     - `symbol`: Asset ticker symbol (e.g., "BTC", "ETH", "USD")
     - `balance`: Current balance as a string (string to preserve decimal precision)
     - `usdBalance`: Optional USD-equivalent value as a string
   - USD balance is optional because not all assets have a direct USD conversion, or the conversion rate may not be available

2. **Wallet Balance Callback**
   - The live trading engine should emit wallet balance updates via a callback
   - The callback should be triggered on order fills, trades, and periodic refreshes
   - The frontend can subscribe to this callback to keep the wallet UI up to date

3. **On-Demand Query**
   - The Swift API should expose a method to query the current wallet balance on demand
   - This is needed for initial load and manual refresh

## Types

### Go Types

**File: `internal/types/wallet.go` (new file)**

```go
package types

// WalletAsset represents a single asset balance in the user's wallet.
type WalletAsset struct {
    // Title is the human-readable name of the asset (e.g., "Bitcoin", "Ethereum")
    Title string `json:"title" yaml:"title"`

    // Symbol is the ticker symbol (e.g., "BTC", "ETH", "USD")
    Symbol string `json:"symbol" yaml:"symbol"`

    // Balance is the current balance as a string to preserve decimal precision
    Balance string `json:"balance" yaml:"balance"`

    // UsdBalance is the optional USD-equivalent value as a string.
    // Empty string means the USD conversion is not available.
    UsdBalance string `json:"usd_balance,omitempty" yaml:"usd_balance,omitempty"`
}
```

### Protocol Buffer Messages

**File: `pkg/strategy/strategy.proto`**

```protobuf
// WalletAsset represents a single asset balance in the user's wallet.
message WalletAsset {
  string title = 1;                  // Human-readable asset name
  string symbol = 2;                 // Ticker symbol (e.g., "BTC")
  string balance = 3;                // Balance as string for precision
  optional string usd_balance = 4;   // Optional USD-equivalent value
}

// GetWalletResponse contains all wallet asset balances.
message GetWalletResponse {
  repeated WalletAsset assets = 1;
}
```

Add the RPC method to the `StrategyApi` service:

```protobuf
service StrategyApi {
  // ... existing methods ...

  // GetWallet returns the current wallet balances for all assets.
  rpc GetWallet(google.protobuf.Empty) returns (GetWalletResponse) {}
}
```

## Swift API

### Gomobile-Compatible Collection

Since gomobile does not support returning slices, the wallet uses a collection interface:

```go
// WalletAssetCollection provides index-based access to wallet assets (gomobile-compatible).
type WalletAssetCollection interface {
    Get(i int) *WalletAssetItem
    Size() int
}

// WalletAssetItem is a gomobile-compatible wrapper for a single wallet asset.
// Gomobile requires concrete struct types with simple field accessors.
type WalletAssetItem struct {
    Title      string
    Symbol     string
    Balance    string
    UsdBalance string // Empty string if not available
}
```

### TradingEngineHelper Callback

Add a new callback to `TradingEngineHelper` for wallet balance updates:

```go
type TradingEngineHelper interface {
    // ... existing callbacks ...

    // OnWalletUpdate is called when wallet balances change (e.g., after order fills).
    // assets is a collection of WalletAssetItem that can be iterated with Get(i)/Size().
    OnWalletUpdate(assets WalletAssetCollection) error
}
```

### Engine Callback Type

Add a new callback type in `internal/trading/engine/engine.go`:

```go
// OnWalletUpdateCallback is called when wallet balances are updated.
type OnWalletUpdateCallback func(assets []types.WalletAsset) error

type LiveTradingCallbacks struct {
    // ... existing callbacks ...

    // OnWalletUpdate is called when wallet balances change.
    OnWalletUpdate *OnWalletUpdateCallback
}
```

### Swift Usage Example

```swift
import SwiftUI
import Swiftargo

struct WalletView: View {
    @State private var assets: [WalletDisplayItem] = []

    var body: some View {
        List(assets, id: \.symbol) { asset in
            HStack {
                VStack(alignment: .leading) {
                    Text(asset.title)
                        .font(.headline)
                    Text(asset.symbol)
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
                Spacer()
                VStack(alignment: .trailing) {
                    Text(asset.balance)
                        .font(.body)
                    if !asset.usdBalance.isEmpty {
                        Text("≈ $\(asset.usdBalance)")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }
            }
        }
        .navigationTitle("Wallet")
    }
}

struct WalletDisplayItem {
    let title: String
    let symbol: String
    let balance: String
    let usdBalance: String
}

// In your TradingEngineHelper implementation:
class MyTradingHelper: NSObject, SwiftargoTradingEngineHelperProtocol {
    weak var walletDelegate: WalletDelegate?

    func onWalletUpdate(_ assets: SwiftargoWalletAssetCollection?) throws {
        guard let assets = assets else { return }
        var items: [WalletDisplayItem] = []
        for i in 0..<assets.size() {
            if let asset = assets.get(i) {
                items.append(WalletDisplayItem(
                    title: asset.title,
                    symbol: asset.symbol,
                    balance: asset.balance,
                    usdBalance: asset.usdBalance
                ))
            }
        }
        DispatchQueue.main.async {
            self.walletDelegate?.didUpdateWallet(items)
        }
    }

    // ... other existing callbacks ...
}
```

## Trading Provider Integration

### TradingSystemProvider Interface

Add a `GetWallet` method to the `TradingSystemProvider` interface:

```go
type TradingSystemProvider interface {
    // ... existing methods ...

    // GetWallet returns the current wallet balances for all assets.
    GetWallet() ([]types.WalletAsset, error)
}
```

### Binance Implementation

For Binance, the wallet maps to the account balances returned by the Binance API:

```go
func (b *BinanceTradingProvider) GetWallet() ([]types.WalletAsset, error) {
    account, err := b.client.NewGetAccountService().Do(context.Background())
    if err != nil {
        return nil, fmt.Errorf("failed to get account info: %w", err)
    }

    var assets []types.WalletAsset
    for _, balance := range account.Balances {
        free, _ := strconv.ParseFloat(balance.Free, 64)
        locked, _ := strconv.ParseFloat(balance.Locked, 64)
        total := free + locked
        if total == 0 {
            continue // Skip zero-balance assets
        }

        assets = append(assets, types.WalletAsset{
            Title:      balance.Asset, // e.g., "BTC"
            Symbol:     balance.Asset,
            Balance:    strconv.FormatFloat(total, 'f', -1, 64),
            UsdBalance: "", // Can be enriched with price data if available
        })
    }

    return assets, nil
}
```

### Backtest Implementation

For backtesting, the wallet returns a single cash asset since backtests operate with a single currency:

```go
func (b *BacktestTrading) GetWallet() ([]types.WalletAsset, error) {
    return []types.WalletAsset{
        {
            Title:      "US Dollar",
            Symbol:     "USD",
            Balance:    strconv.FormatFloat(b.balance, 'f', 2, 64),
            UsdBalance: strconv.FormatFloat(b.balance, 'f', 2, 64),
        },
    }, nil
}
```

## Files to be Modified

### New Files

| File | Description |
|------|-------------|
| `internal/types/wallet.go` | `WalletAsset` struct definition |

### Modified Files

| File | Change |
|------|--------|
| `pkg/strategy/strategy.proto` | Add `WalletAsset` message, `GetWalletResponse` message, and `GetWallet` RPC |
| `internal/trading/provider/trading_provider.go` | Add `GetWallet()` to `TradingSystemProvider` interface |
| `internal/trading/provider/binance.go` | Implement `GetWallet()` for Binance |
| `internal/backtest/engine/engine_v1/backtest_trading.go` | Implement `GetWallet()` for backtest |
| `internal/trading/engine/engine.go` | Add `OnWalletUpdateCallback` and update `LiveTradingCallbacks` |
| `internal/trading/engine/engine_v1/live_trading_v1.go` | Call wallet callback after order fills |
| `pkg/swift-argo/trading.go` | Add `OnWalletUpdate` to `TradingEngineHelper`, bridge callback |
| `pkg/swift-argo/collections.go` | Add `WalletAssetCollection` and `WalletAssetItem` types |

## Implementation TODOs

### Phase 1: Core Types

- [ ] **TODO-1**: Create `internal/types/wallet.go` with `WalletAsset` struct
- [ ] **TODO-2**: Add `WalletAsset` and `GetWalletResponse` messages to `strategy.proto`
- [ ] **TODO-3**: Add `GetWallet` RPC to `StrategyApi` service in `strategy.proto`
- [ ] **TODO-4**: Run `make generate` to regenerate protobuf code

### Phase 2: Provider Integration

- [ ] **TODO-5**: Add `GetWallet()` method to `TradingSystemProvider` interface
- [ ] **TODO-6**: Implement `GetWallet()` in Binance trading provider
- [ ] **TODO-7**: Implement `GetWallet()` in backtest trading system

### Phase 3: Engine Callbacks

- [ ] **TODO-8**: Add `OnWalletUpdateCallback` to `LiveTradingCallbacks` in `engine.go`
- [ ] **TODO-9**: Trigger wallet callback after order fills in `live_trading_v1.go`
- [ ] **TODO-10**: Add periodic wallet refresh (optional, for providers that support streaming balance updates)

### Phase 4: Swift API

- [ ] **TODO-11**: Add `WalletAssetCollection` and `WalletAssetItem` to `pkg/swift-argo/collections.go`
- [ ] **TODO-12**: Add `OnWalletUpdate` callback to `TradingEngineHelper` interface
- [ ] **TODO-13**: Bridge the wallet callback in `TradingEngine.createCallbacks()`

### Phase 5: Testing

- [ ] **TODO-14**: Add unit tests for `WalletAsset` JSON serialization
- [ ] **TODO-15**: Add unit tests for Binance `GetWallet()` implementation
- [ ] **TODO-16**: Add unit tests for backtest `GetWallet()` implementation
- [ ] **TODO-17**: Add integration tests for wallet callback in live trading

## Test Plan

### Unit Tests

#### 1. Wallet Types
**File: `internal/types/wallet_test.go`**

```go
func TestWalletAsset_JSONSerialization(t *testing.T) {
    tests := []struct {
        name     string
        asset    WalletAsset
        expected string
    }{
        {
            name: "with usd balance",
            asset: WalletAsset{
                Title:      "Bitcoin",
                Symbol:     "BTC",
                Balance:    "1.5",
                UsdBalance: "45000.00",
            },
            expected: `{"title":"Bitcoin","symbol":"BTC","balance":"1.5","usd_balance":"45000.00"}`,
        },
        {
            name: "without usd balance",
            asset: WalletAsset{
                Title:   "Ethereum",
                Symbol:  "ETH",
                Balance: "10.0",
            },
            expected: `{"title":"Ethereum","symbol":"ETH","balance":"10.0"}`,
        },
    }
    // ... test implementation
}
```

#### 2. Backtest GetWallet
**File: `internal/backtest/engine/engine_v1/backtest_trading_test.go`**

```go
func TestBacktestTrading_GetWallet(t *testing.T) {
    // Verify single USD asset is returned
    // Verify balance matches account balance
}
```

#### 3. Binance GetWallet
**File: `internal/trading/provider/binance_test.go`**

```go
func TestBinanceTradingProvider_GetWallet(t *testing.T) {
    // Verify multi-asset response
    // Verify zero-balance assets are filtered
    // Verify balance precision is preserved
}
```

### Integration Tests

```go
func TestLiveTrading_WalletCallback(t *testing.T) {
    // 1. Set up live trading engine with mock provider
    // 2. Register OnWalletUpdate callback
    // 3. Simulate an order fill
    // 4. Verify callback is invoked with updated balances
}
```

## JSON Response Example

```json
[
  {
    "title": "Bitcoin",
    "symbol": "BTC",
    "balance": "1.50000000",
    "usd_balance": "45000.00"
  },
  {
    "title": "Ethereum",
    "symbol": "ETH",
    "balance": "10.00000000",
    "usd_balance": "3000.00"
  },
  {
    "title": "Tether",
    "symbol": "USDT",
    "balance": "5000.00",
    "usd_balance": "5000.00"
  },
  {
    "title": "Solana",
    "symbol": "SOL",
    "balance": "25.00000000"
  }
]
```

## Design Decisions

1. **String for Balance**: Balances are strings (not floats) to preserve decimal precision. Floating-point representation can introduce rounding errors that are unacceptable for financial data.

2. **Optional USD Balance**: The `usd_balance` field is optional because:
   - Not all assets have USD pairs (e.g., obscure tokens)
   - Price conversion may not be available in all contexts
   - Backtest environments may not have price feeds for conversion

3. **Flat Asset List**: The wallet returns a flat list of assets rather than a nested structure. This keeps the API simple and maps directly to the UI list view.

4. **Callback-Based Updates**: Wallet updates are pushed via callbacks (not polling) to keep the UI responsive and avoid unnecessary API calls to the trading provider.

## Backward Compatibility

1. **Existing `AccountInfo`**: The current `AccountInfo` struct and `GetAccountInfo` RPC are preserved. The wallet API is additive and does not replace existing account functionality.
2. **Optional Callback**: The `OnWalletUpdate` callback is a pointer (nil by default), so existing code that does not use it continues to work unchanged.
3. **Provider Interface**: Adding `GetWallet()` to `TradingSystemProvider` requires all implementations to add the method. A default implementation returning the balance as a single USD asset can ease migration.

## Related Files

- [account.go](../internal/types/account.go) - Existing `AccountInfo` struct
- [trading_provider.go](../internal/trading/provider/trading_provider.go) - `TradingSystemProvider` interface
- [engine.go](../internal/trading/engine/engine.go) - Live trading engine callbacks
- [trading.go](../pkg/swift-argo/trading.go) - Swift bindings
- [collections.go](../pkg/swift-argo/collections.go) - Gomobile collection types
- [strategy.proto](../pkg/strategy/strategy.proto) - Protobuf definitions
