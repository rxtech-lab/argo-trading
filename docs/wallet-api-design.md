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

2. **Periodic Polling**
   - The frontend periodically fetches wallet balances from the trading provider
   - No push-based callback is needed; the frontend controls the refresh interval
   - This simplifies the engine and avoids coupling wallet updates to trading events

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

### GetWallet Method

The `TradingEngine` exposes a `GetWallet` method that the frontend calls periodically to fetch the current wallet balances:

```go
// GetWallet returns the current wallet balances from the trading provider.
// The frontend should call this periodically to refresh the wallet UI.
func (t *TradingEngine) GetWallet() (WalletAssetCollection, error) {
    assets, err := t.engine.GetWallet()
    if err != nil {
        return nil, err
    }

    items := make([]*WalletAssetItem, len(assets))
    for i, a := range assets {
        items[i] = &WalletAssetItem{
            Title:      a.Title,
            Symbol:     a.Symbol,
            Balance:    a.Balance,
            UsdBalance: a.UsdBalance,
        }
    }

    return &WalletAssetArray{items: items}, nil
}
```

### Swift Usage Example

```swift
import SwiftUI
import Swiftargo

struct WalletView: View {
    @State private var assets: [WalletDisplayItem] = []
    @State private var engine: SwiftargoTradingEngine?

    let timer = Timer.publish(every: 5, on: .main, in: .common).autoconnect()

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
        .onAppear { fetchWallet() }
        .onReceive(timer) { _ in fetchWallet() }
    }

    private func fetchWallet() {
        guard let engine = engine else { return }
        do {
            let collection = try engine.getWallet()
            var items: [WalletDisplayItem] = []
            for i in 0..<collection.size() {
                if let asset = collection.get(i) {
                    items.append(WalletDisplayItem(
                        title: asset.title,
                        symbol: asset.symbol,
                        balance: asset.balance,
                        usdBalance: asset.usdBalance
                    ))
                }
            }
            self.assets = items
        } catch {
            print("Failed to fetch wallet: \(error)")
        }
    }
}

struct WalletDisplayItem {
    let title: String
    let symbol: String
    let balance: String
    let usdBalance: String
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
| `internal/trading/provider/trading_provider.go` | Add `GetWallet()` to `TradingSystemProvider` interface |
| `internal/trading/provider/binance.go` | Implement `GetWallet()` for Binance |
| `internal/backtest/engine/engine_v1/backtest_trading.go` | Implement `GetWallet()` for backtest |
| `pkg/swift-argo/trading.go` | Add `GetWallet()` method to `TradingEngine` |
| `pkg/swift-argo/collections.go` | Add `WalletAssetCollection` and `WalletAssetItem` types |

## Implementation TODOs

### Phase 1: Core Types

- [ ] **TODO-1**: Create `internal/types/wallet.go` with `WalletAsset` struct

### Phase 2: Provider Integration

- [ ] **TODO-2**: Add `GetWallet()` method to `TradingSystemProvider` interface
- [ ] **TODO-3**: Implement `GetWallet()` in Binance trading provider
- [ ] **TODO-4**: Implement `GetWallet()` in backtest trading system

### Phase 3: Swift API

- [ ] **TODO-5**: Add `WalletAssetCollection` and `WalletAssetItem` to `pkg/swift-argo/collections.go`
- [ ] **TODO-6**: Add `GetWallet()` method to `TradingEngine` in `pkg/swift-argo/trading.go`

### Phase 4: Testing

- [ ] **TODO-7**: Add unit tests for `WalletAsset` JSON serialization
- [ ] **TODO-8**: Add unit tests for Binance `GetWallet()` implementation
- [ ] **TODO-9**: Add unit tests for backtest `GetWallet()` implementation
- [ ] **TODO-10**: Add unit tests for `GetWallet()` in Swift bridge

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
func TestTradingEngine_GetWallet(t *testing.T) {
    // 1. Set up trading engine with mock provider
    // 2. Call GetWallet()
    // 3. Verify returned assets match provider balances
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

4. **Polling-Based Updates**: The frontend periodically fetches wallet balances rather than relying on push-based callbacks. This simplifies the engine, avoids coupling wallet updates to trading events, and gives the frontend full control over refresh frequency.

## Backward Compatibility

1. **Existing `AccountInfo`**: The current `AccountInfo` struct and `GetAccountInfo` RPC are preserved. The wallet API is additive and does not replace existing account functionality.
2. **Provider Interface**: Adding `GetWallet()` to `TradingSystemProvider` requires all implementations to add the method. A default implementation returning the balance as a single USD asset can ease migration.

## Related Files

- [account.go](../internal/types/account.go) - Existing `AccountInfo` struct
- [trading_provider.go](../internal/trading/provider/trading_provider.go) - `TradingSystemProvider` interface
- [trading.go](../pkg/swift-argo/trading.go) - Swift bindings
- [collections.go](../pkg/swift-argo/collections.go) - Gomobile collection types
