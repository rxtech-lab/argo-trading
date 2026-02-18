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

2. **Position Info**
   - The wallet must return a list of currently open positions
   - Each position entry contains:
     - `symbol`: The trading pair (e.g., "BTCUSDT")
     - `side`: Position direction ("LONG" or "SHORT")
     - `quantity`: Position size as a string
     - `entryPrice`: Average entry price as a string
     - `currentPrice`: Latest market price (optional)
     - `unrealizedPnl`: Unrealized profit/loss (optional)

3. **Order Info**
   - The wallet must return a list of pending (open) orders
   - Each order entry contains:
     - `orderID`: Unique order identifier
     - `symbol`: The trading pair
     - `side`: Order direction ("BUY" or "SELL")
     - `orderType`: Order type ("MARKET" or "LIMIT")
     - `quantity`: Order quantity as a string
     - `price`: Order price as a string
     - `status`: Order status (e.g., "PENDING")

4. **Periodic Polling**
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

// WalletPosition represents a current open position.
type WalletPosition struct {
    // Symbol is the ticker symbol (e.g., "BTCUSDT", "AAPL")
    Symbol string `json:"symbol" yaml:"symbol"`

    // Side is the position direction: "LONG" or "SHORT"
    Side string `json:"side" yaml:"side"`

    // Quantity is the current position size as a string for precision
    Quantity string `json:"quantity" yaml:"quantity"`

    // EntryPrice is the average entry price as a string for precision
    EntryPrice string `json:"entry_price" yaml:"entry_price"`

    // CurrentPrice is the latest market price as a string (optional, empty if unavailable)
    CurrentPrice string `json:"current_price,omitempty" yaml:"current_price,omitempty"`

    // UnrealizedPnl is the unrealized profit/loss as a string (optional, empty if unavailable)
    UnrealizedPnl string `json:"unrealized_pnl,omitempty" yaml:"unrealized_pnl,omitempty"`
}

// WalletOrder represents a pending (open) order.
type WalletOrder struct {
    // OrderID is the unique identifier for the order
    OrderID string `json:"order_id" yaml:"order_id"`

    // Symbol is the ticker symbol (e.g., "BTCUSDT", "AAPL")
    Symbol string `json:"symbol" yaml:"symbol"`

    // Side is the order direction: "BUY" or "SELL"
    Side string `json:"side" yaml:"side"`

    // OrderType is the order type: "MARKET" or "LIMIT"
    OrderType string `json:"order_type" yaml:"order_type"`

    // Quantity is the order quantity as a string for precision
    Quantity string `json:"quantity" yaml:"quantity"`

    // Price is the order price as a string for precision (relevant for LIMIT orders)
    Price string `json:"price" yaml:"price"`

    // Status is the order status (e.g., "PENDING")
    Status string `json:"status" yaml:"status"`
}

// WalletInfo is the combined response for the GetWallet API.
// It includes asset balances, open positions, and pending orders.
type WalletInfo struct {
    // Assets is the list of asset balances in the wallet
    Assets []WalletAsset `json:"assets" yaml:"assets"`

    // Positions is the list of currently open positions
    Positions []WalletPosition `json:"positions" yaml:"positions"`

    // Orders is the list of pending (open) orders
    Orders []WalletOrder `json:"orders" yaml:"orders"`
}
```

## Swift API

### Gomobile-Compatible Collection

Since gomobile does not support returning slices, the wallet uses collection interfaces:

```go
// WalletAssetCollection provides index-based access to wallet assets (gomobile-compatible).
type WalletAssetCollection interface {
    Get(i int) *WalletAssetItem
    Size() int
}

// WalletAssetItem is a gomobile-compatible wrapper for a single wallet asset.
type WalletAssetItem struct {
    Title      string
    Symbol     string
    Balance    string
    UsdBalance string // Empty string if not available
}

// WalletPositionCollection provides index-based access to positions (gomobile-compatible).
type WalletPositionCollection interface {
    Get(i int) *WalletPositionItem
    Size() int
}

// WalletPositionItem is a gomobile-compatible wrapper for a single position.
type WalletPositionItem struct {
    Symbol        string
    Side          string // "LONG" or "SHORT"
    Quantity      string
    EntryPrice    string
    CurrentPrice  string // Empty if unavailable
    UnrealizedPnl string // Empty if unavailable
}

// WalletOrderCollection provides index-based access to pending orders (gomobile-compatible).
type WalletOrderCollection interface {
    Get(i int) *WalletOrderItem
    Size() int
}

// WalletOrderItem is a gomobile-compatible wrapper for a single pending order.
type WalletOrderItem struct {
    OrderID   string
    Symbol    string
    Side      string // "BUY" or "SELL"
    OrderType string // "MARKET" or "LIMIT"
    Quantity  string
    Price     string
    Status    string
}

// WalletInfoResult is the combined gomobile-compatible result for GetWallet.
type WalletInfoResult struct {
    Assets    WalletAssetCollection
    Positions WalletPositionCollection
    Orders    WalletOrderCollection
}
```

### GetWallet Method

The `TradingEngine` exposes a `GetWallet` method that the frontend calls periodically to fetch the current wallet state (balances, positions, and orders):

```go
// GetWallet returns the current wallet info including asset balances, positions, and open orders.
// The frontend should call this periodically to refresh the wallet UI.
func (t *TradingEngine) GetWallet() (*WalletInfoResult, error) {
    info, err := t.engine.GetWallet()
    if err != nil {
        return nil, err
    }

    // Convert assets
    assetItems := make([]*WalletAssetItem, len(info.Assets))
    for i, a := range info.Assets {
        assetItems[i] = &WalletAssetItem{
            Title:      a.Title,
            Symbol:     a.Symbol,
            Balance:    a.Balance,
            UsdBalance: a.UsdBalance,
        }
    }

    // Convert positions
    posItems := make([]*WalletPositionItem, len(info.Positions))
    for i, p := range info.Positions {
        posItems[i] = &WalletPositionItem{
            Symbol:        p.Symbol,
            Side:          p.Side,
            Quantity:      p.Quantity,
            EntryPrice:    p.EntryPrice,
            CurrentPrice:  p.CurrentPrice,
            UnrealizedPnl: p.UnrealizedPnl,
        }
    }

    // Convert orders
    orderItems := make([]*WalletOrderItem, len(info.Orders))
    for i, o := range info.Orders {
        orderItems[i] = &WalletOrderItem{
            OrderID:   o.OrderID,
            Symbol:    o.Symbol,
            Side:      o.Side,
            OrderType: o.OrderType,
            Quantity:  o.Quantity,
            Price:     o.Price,
            Status:    o.Status,
        }
    }

    return &WalletInfoResult{
        Assets:    &WalletAssetArray{items: assetItems},
        Positions: &WalletPositionArray{items: posItems},
        Orders:    &WalletOrderArray{items: orderItems},
    }, nil
}
```

### Swift Usage Example

```swift
import SwiftUI
import Swiftargo

struct WalletView: View {
    @State private var assets: [WalletDisplayItem] = []
    @State private var positions: [PositionDisplayItem] = []
    @State private var orders: [OrderDisplayItem] = []
    @State private var engine: SwiftargoTradingEngine?

    let timer = Timer.publish(every: 5, on: .main, in: .common).autoconnect()

    var body: some View {
        List {
            // Assets section
            Section("Assets") {
                ForEach(assets, id: \.symbol) { asset in
                    HStack {
                        VStack(alignment: .leading) {
                            Text(asset.title).font(.headline)
                            Text(asset.symbol).font(.caption).foregroundColor(.secondary)
                        }
                        Spacer()
                        VStack(alignment: .trailing) {
                            Text(asset.balance).font(.body)
                            if !asset.usdBalance.isEmpty {
                                Text("≈ $\(asset.usdBalance)").font(.caption).foregroundColor(.secondary)
                            }
                        }
                    }
                }
            }

            // Positions section
            Section("Positions") {
                ForEach(positions, id: \.symbol) { pos in
                    HStack {
                        VStack(alignment: .leading) {
                            Text(pos.symbol).font(.headline)
                            Text(pos.side).font(.caption).foregroundColor(pos.side == "LONG" ? .green : .red)
                        }
                        Spacer()
                        VStack(alignment: .trailing) {
                            Text("Qty: \(pos.quantity)").font(.body)
                            Text("Entry: \(pos.entryPrice)").font(.caption)
                            if !pos.unrealizedPnl.isEmpty {
                                Text("PnL: \(pos.unrealizedPnl)").font(.caption).foregroundColor(.secondary)
                            }
                        }
                    }
                }
            }

            // Open orders section
            Section("Open Orders") {
                ForEach(orders, id: \.orderID) { order in
                    HStack {
                        VStack(alignment: .leading) {
                            Text(order.symbol).font(.headline)
                            Text("\(order.side) \(order.orderType)").font(.caption)
                        }
                        Spacer()
                        VStack(alignment: .trailing) {
                            Text("Qty: \(order.quantity)").font(.body)
                            Text("Price: \(order.price)").font(.caption)
                        }
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
            let wallet = try engine.getWallet()

            // Parse assets
            var assetItems: [WalletDisplayItem] = []
            let assetCollection = wallet.assets
            for i in 0..<assetCollection.size() {
                if let a = assetCollection.get(i) {
                    assetItems.append(WalletDisplayItem(title: a.title, symbol: a.symbol, balance: a.balance, usdBalance: a.usdBalance))
                }
            }
            self.assets = assetItems

            // Parse positions
            var posItems: [PositionDisplayItem] = []
            let posCollection = wallet.positions
            for i in 0..<posCollection.size() {
                if let p = posCollection.get(i) {
                    posItems.append(PositionDisplayItem(symbol: p.symbol, side: p.side, quantity: p.quantity, entryPrice: p.entryPrice, unrealizedPnl: p.unrealizedPnl))
                }
            }
            self.positions = posItems

            // Parse orders
            var orderItems: [OrderDisplayItem] = []
            let orderCollection = wallet.orders
            for i in 0..<orderCollection.size() {
                if let o = orderCollection.get(i) {
                    orderItems.append(OrderDisplayItem(orderID: o.orderID, symbol: o.symbol, side: o.side, orderType: o.orderType, quantity: o.quantity, price: o.price))
                }
            }
            self.orders = orderItems
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

struct PositionDisplayItem {
    let symbol: String
    let side: String
    let quantity: String
    let entryPrice: String
    let unrealizedPnl: String
}

struct OrderDisplayItem {
    let orderID: String
    let symbol: String
    let side: String
    let orderType: String
    let quantity: String
    let price: String
}
```

## Trading Provider Integration

### TradingSystemProvider Interface

Add a `GetWallet` method to the `TradingSystemProvider` interface:

```go
type TradingSystemProvider interface {
    // ... existing methods (GetPositions, GetOpenOrders, GetAccountInfo, etc.) ...

    // GetWallet returns the combined wallet info: asset balances, open positions, and pending orders.
    GetWallet() (types.WalletInfo, error)
}
```

The `GetWallet` implementation aggregates data from the existing provider methods (`GetPositions`, `GetOpenOrders`, and account balance data) into a single `WalletInfo` response.

### Binance Implementation

For Binance, the wallet aggregates account balances, open positions, and pending orders:

```go
func (b *BinanceTradingProvider) GetWallet() (types.WalletInfo, error) {
    // Fetch account balances
    account, err := b.client.NewGetAccountService().Do(context.Background())
    if err != nil {
        return types.WalletInfo{}, fmt.Errorf("failed to get account info: %w", err)
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
            Title:      balance.Asset,
            Symbol:     balance.Asset,
            Balance:    strconv.FormatFloat(total, 'f', -1, 64),
            UsdBalance: "", // Can be enriched with price data if available
        })
    }

    // Fetch open positions
    positions, err := b.GetPositions()
    if err != nil {
        return types.WalletInfo{}, fmt.Errorf("failed to get positions: %w", err)
    }
    var walletPositions []types.WalletPosition
    for _, pos := range positions {
        if pos.TotalLongPositionQuantity > 0 {
            walletPositions = append(walletPositions, types.WalletPosition{
                Symbol:     pos.Symbol,
                Side:       "LONG",
                Quantity:   strconv.FormatFloat(pos.TotalLongPositionQuantity, 'f', -1, 64),
                EntryPrice: strconv.FormatFloat(pos.GetAverageLongPositionEntryPrice(), 'f', -1, 64),
            })
        }
        if pos.TotalShortPositionQuantity > 0 {
            walletPositions = append(walletPositions, types.WalletPosition{
                Symbol:     pos.Symbol,
                Side:       "SHORT",
                Quantity:   strconv.FormatFloat(pos.TotalShortPositionQuantity, 'f', -1, 64),
                EntryPrice: strconv.FormatFloat(pos.GetAverageShortPositionEntryPrice(), 'f', -1, 64),
            })
        }
    }

    // Fetch open orders
    openOrders, err := b.GetOpenOrders()
    if err != nil {
        return types.WalletInfo{}, fmt.Errorf("failed to get open orders: %w", err)
    }
    var walletOrders []types.WalletOrder
    for _, o := range openOrders {
        walletOrders = append(walletOrders, types.WalletOrder{
            OrderID:   o.ID,
            Symbol:    o.Symbol,
            Side:      string(o.Side),
            OrderType: string(o.OrderType),
            Quantity:  strconv.FormatFloat(o.Quantity, 'f', -1, 64),
            Price:     strconv.FormatFloat(o.Price, 'f', -1, 64),
            Status:    "PENDING",
        })
    }

    return types.WalletInfo{
        Assets:    assets,
        Positions: walletPositions,
        Orders:    walletOrders,
    }, nil
}
```

### Backtest Implementation

For backtesting, the wallet aggregates the single cash balance, current positions, and open orders:

```go
func (b *BacktestTrading) GetWallet() (types.WalletInfo, error) {
    // Single cash asset
    assets := []types.WalletAsset{
        {
            Title:      "US Dollar",
            Symbol:     "USD",
            Balance:    strconv.FormatFloat(b.balance, 'f', 2, 64),
            UsdBalance: strconv.FormatFloat(b.balance, 'f', 2, 64),
        },
    }

    // Open positions
    positions, _ := b.GetPositions()
    var walletPositions []types.WalletPosition
    for _, pos := range positions {
        if pos.TotalLongPositionQuantity > 0 {
            walletPositions = append(walletPositions, types.WalletPosition{
                Symbol:     pos.Symbol,
                Side:       "LONG",
                Quantity:   strconv.FormatFloat(pos.TotalLongPositionQuantity, 'f', -1, 64),
                EntryPrice: strconv.FormatFloat(pos.GetAverageLongPositionEntryPrice(), 'f', -1, 64),
            })
        }
        if pos.TotalShortPositionQuantity > 0 {
            walletPositions = append(walletPositions, types.WalletPosition{
                Symbol:     pos.Symbol,
                Side:       "SHORT",
                Quantity:   strconv.FormatFloat(pos.TotalShortPositionQuantity, 'f', -1, 64),
                EntryPrice: strconv.FormatFloat(pos.GetAverageShortPositionEntryPrice(), 'f', -1, 64),
            })
        }
    }

    // Open orders
    openOrders, _ := b.GetOpenOrders()
    var walletOrders []types.WalletOrder
    for _, o := range openOrders {
        walletOrders = append(walletOrders, types.WalletOrder{
            OrderID:   o.ID,
            Symbol:    o.Symbol,
            Side:      string(o.Side),
            OrderType: string(o.OrderType),
            Quantity:  strconv.FormatFloat(o.Quantity, 'f', -1, 64),
            Price:     strconv.FormatFloat(o.Price, 'f', -1, 64),
            Status:    "PENDING",
        })
    }

    return types.WalletInfo{
        Assets:    assets,
        Positions: walletPositions,
        Orders:    walletOrders,
    }, nil
}
```

## Files to be Modified

### New Files

| File | Description |
|------|-------------|
| `internal/types/wallet.go` | `WalletAsset`, `WalletPosition`, `WalletOrder`, and `WalletInfo` struct definitions |

### Modified Files

| File | Change |
|------|--------|
| `internal/trading/provider/trading_provider.go` | Add `GetWallet()` to `TradingSystemProvider` interface |
| `internal/trading/provider/binance.go` | Implement `GetWallet()` for Binance |
| `internal/backtest/engine/engine_v1/backtest_trading.go` | Implement `GetWallet()` for backtest |
| `pkg/swift-argo/trading.go` | Add `GetWallet()` method to `TradingEngine` |
| `pkg/swift-argo/collections.go` | Add `WalletAssetCollection`, `WalletPositionCollection`, `WalletOrderCollection`, and `WalletInfoResult` types |

## Implementation TODOs

### Phase 1: Core Types

- [ ] **TODO-1**: Create `internal/types/wallet.go` with `WalletAsset`, `WalletPosition`, `WalletOrder`, and `WalletInfo` structs

### Phase 2: Provider Integration

- [ ] **TODO-2**: Add `GetWallet()` method to `TradingSystemProvider` interface
- [ ] **TODO-3**: Implement `GetWallet()` in Binance trading provider
- [ ] **TODO-4**: Implement `GetWallet()` in backtest trading system

### Phase 3: Swift API

- [ ] **TODO-5**: Add `WalletAssetCollection`, `WalletPositionCollection`, `WalletOrderCollection`, and `WalletInfoResult` to `pkg/swift-argo/collections.go`
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
{
  "assets": [
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
    }
  ],
  "positions": [
    {
      "symbol": "BTCUSDT",
      "side": "LONG",
      "quantity": "0.50000000",
      "entry_price": "30000.00",
      "current_price": "31500.00",
      "unrealized_pnl": "750.00"
    },
    {
      "symbol": "ETHUSDT",
      "side": "SHORT",
      "quantity": "5.00000000",
      "entry_price": "2000.00",
      "current_price": "1950.00",
      "unrealized_pnl": "250.00"
    }
  ],
  "orders": [
    {
      "order_id": "abc-123",
      "symbol": "BTCUSDT",
      "side": "BUY",
      "order_type": "LIMIT",
      "quantity": "0.10000000",
      "price": "29000.00",
      "status": "PENDING"
    }
  ]
}
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
