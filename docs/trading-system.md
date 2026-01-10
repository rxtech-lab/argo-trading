# Trading System Provider Architecture

This document outlines the architecture and implementation plan for the trading system in Argo Trading. The design follows the same **provider pattern** as the existing market data system, where each trading system is a registered provider with its own configuration schema.

## Overview

The trading system uses a **provider-based architecture** identical to the market data provider pattern. Key design principles:

- Each broker/exchange is a separate **provider** (e.g., `binance-paper`, `binance-live`, `ibkr-paper`, `ibkr-live`)
- Providers are registered in a **registry** with metadata and configuration schemas
- **Data provider** and **trading provider** are independent - you can trade via IBKR while getting data from Polygon
- Each provider has an `IsPaperTrading` attribute to indicate simulation mode
- Backtesting uses its own engine and is not part of the trading provider system

## Provider Registry

### Provider Type Definition

```go
package trading

// ProviderType defines the type of trading provider.
type ProviderType string

const (
    // Binance providers
    ProviderBinancePaper ProviderType = "binance-paper"
    ProviderBinanceLive  ProviderType = "binance-live"

    // Interactive Brokers providers
    ProviderIBKRPaper ProviderType = "ibkr-paper"
    ProviderIBKRLive  ProviderType = "ibkr-live"
)
```

### Provider Info Structure

```go
// ProviderInfo holds metadata about a trading provider.
type ProviderInfo struct {
    Name           string `json:"name"`
    DisplayName    string `json:"displayName"`
    Description    string `json:"description"`
    IsPaperTrading bool   `json:"isPaperTrading"`
}
```

### Provider Registry

```go
// providerRegistry holds metadata about all supported trading providers.
var providerRegistry = map[ProviderType]ProviderInfo{
    ProviderBinancePaper: {
        Name:           string(ProviderBinancePaper),
        DisplayName:    "Binance Testnet",
        Description:    "Binance testnet for paper trading cryptocurrency without real funds",
        IsPaperTrading: true,
    },
    ProviderBinanceLive: {
        Name:           string(ProviderBinanceLive),
        DisplayName:    "Binance",
        Description:    "Binance exchange for live cryptocurrency trading",
        IsPaperTrading: false,
    },
    ProviderIBKRPaper: {
        Name:           string(ProviderIBKRPaper),
        DisplayName:    "Interactive Brokers Paper",
        Description:    "Interactive Brokers paper trading account for simulation",
        IsPaperTrading: true,
    },
    ProviderIBKRLive: {
        Name:           string(ProviderIBKRLive),
        DisplayName:    "Interactive Brokers",
        Description:    "Interactive Brokers live trading account",
        IsPaperTrading: false,
    },
}
```

### Registry Functions

```go
// GetSupportedProviders returns a list of all supported trading provider names.
func GetSupportedProviders() []string {
    providers := make([]string, 0, len(providerRegistry))
    for providerType := range providerRegistry {
        providers = append(providers, string(providerType))
    }
    return providers
}

// GetProviderInfo returns metadata for a specific trading provider.
func GetProviderInfo(providerName string) (ProviderInfo, error) {
    info, exists := providerRegistry[ProviderType(providerName)]
    if !exists {
        return ProviderInfo{}, fmt.Errorf("unsupported trading provider: %s", providerName)
    }
    return info, nil
}

// GetProviderConfigSchema returns the JSON schema for a provider's configuration.
func GetProviderConfigSchema(providerName string) (string, error) {
    switch ProviderType(providerName) {
    case ProviderBinancePaper, ProviderBinanceLive:
        return strategy.ToJSONSchema(BinanceProviderConfig{})
    case ProviderIBKRPaper, ProviderIBKRLive:
        return strategy.ToJSONSchema(IBKRProviderConfig{})
    default:
        return "", fmt.Errorf("unsupported trading provider: %s", providerName)
    }
}

// ParseProviderConfig parses a JSON configuration string for the given provider.
func ParseProviderConfig(providerName string, jsonConfig string) (interface{}, error) {
    switch ProviderType(providerName) {
    case ProviderBinancePaper, ProviderBinanceLive:
        return parseBinanceConfig(jsonConfig)
    case ProviderIBKRPaper, ProviderIBKRLive:
        return parseIBKRConfig(jsonConfig)
    default:
        return nil, fmt.Errorf("unsupported trading provider: %s", providerName)
    }
}
```

## Provider Configuration Schemas

### Base Configuration

```go
// BaseTradingConfig contains common fields for all trading configurations.
type BaseTradingConfig struct {
    InitialCapital   float64 `json:"initialCapital" jsonschema:"description=Starting capital for trading,minimum=0" validate:"required,gt=0"`
    DecimalPrecision int     `json:"decimalPrecision" jsonschema:"description=Decimal precision for quantity calculations,minimum=0,maximum=8,default=2"`
}
```

### Binance Provider Configuration

```go
// BinanceProviderConfig contains configuration for Binance trading.
type BinanceProviderConfig struct {
    BaseTradingConfig
    ApiKey    string `json:"apiKey" jsonschema:"description=Binance API key" validate:"required"`
    SecretKey string `json:"secretKey" jsonschema:"description=Binance API secret" validate:"required"`
}
```

### IBKR Provider Configuration

```go
// IBKRProviderConfig contains configuration for Interactive Brokers trading.
type IBKRProviderConfig struct {
    BaseTradingConfig
    Host      string `json:"host" jsonschema:"description=TWS/Gateway host address,default=127.0.0.1" validate:"required"`
    Port      int    `json:"port" jsonschema:"description=TWS/Gateway port (7496=live 7497=paper for TWS; 4001=live 4002=paper for Gateway),default=7497" validate:"required"`
    ClientID  int    `json:"clientId" jsonschema:"description=Unique client ID for this connection,minimum=0" validate:"required"`
    AccountID string `json:"accountId" jsonschema:"description=IB account ID" validate:"required"`
}
```

## Trading Provider Interface

### Core Interface

```go
package trading

import (
    "context"
    "github.com/rxtech-lab/argo-trading/internal/types"
)

// Provider defines the interface for all trading providers.
type Provider interface {
    // Lifecycle methods
    Initialize(ctx context.Context) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    IsConnected() bool

    // Order management
    PlaceOrder(order types.ExecuteOrder) error
    PlaceMultipleOrders(orders []types.ExecuteOrder) error
    CancelOrder(orderID string) error
    CancelAllOrders() error
    GetOrderStatus(orderID string) (types.OrderStatus, error)
    GetOpenOrders() ([]types.ExecuteOrder, error)

    // Position and account
    GetPositions() ([]types.Position, error)
    GetPosition(symbol string) (types.Position, error)
    GetAccountInfo() (types.AccountInfo, error)
    GetTrades(filter types.TradeFilter) ([]types.Trade, error)

    // Buying/selling power
    GetMaxBuyQuantity(symbol string, price float64) (float64, error)
    GetMaxSellQuantity(symbol string) (float64, error)

    // Provider info
    GetProviderType() ProviderType
    GetProviderInfo() ProviderInfo
}
```

### Provider Factory

```go
// NewProvider creates a new trading provider based on the provider type.
func NewProvider(providerType ProviderType, config interface{}) (Provider, error) {
    switch providerType {
    case ProviderBinancePaper:
        cfg, ok := config.(BinanceProviderConfig)
        if !ok {
            return nil, fmt.Errorf("invalid config type for binance provider")
        }
        return NewBinanceProvider(cfg, true) // testnet=true

    case ProviderBinanceLive:
        cfg, ok := config.(BinanceProviderConfig)
        if !ok {
            return nil, fmt.Errorf("invalid config type for binance provider")
        }
        return NewBinanceProvider(cfg, false) // testnet=false

    case ProviderIBKRPaper, ProviderIBKRLive:
        cfg, ok := config.(IBKRProviderConfig)
        if !ok {
            return nil, fmt.Errorf("invalid config type for ibkr provider")
        }
        isPaper := providerType == ProviderIBKRPaper
        return NewIBKRProvider(cfg, isPaper)

    default:
        return nil, fmt.Errorf("unsupported trading provider: %s", providerType)
    }
}
```

## Data Provider and Trading Provider Separation

A key design feature is the separation of **data provider** and **trading provider**. This allows flexible configurations like:

- Trade via IBKR, get data from Polygon
- Trade via Binance, get data from a different source
- Use the same strategy with different broker/data combinations

### Trading Session Configuration

```go
// TradingSessionConfig defines the configuration for a trading session.
type TradingSessionConfig struct {
    // Trading provider configuration
    TradingProvider     ProviderType `json:"tradingProvider" jsonschema:"description=Trading provider for order execution" validate:"required"`
    TradingProviderConfig interface{} `json:"tradingProviderConfig" jsonschema:"description=Provider-specific configuration" validate:"required"`

    // Data provider configuration (optional - can use trading provider's data)
    DataProvider       *marketdata.ProviderType `json:"dataProvider,omitempty" jsonschema:"description=Market data provider (optional - defaults to trading provider if supported)"`
    DataProviderConfig interface{}              `json:"dataProviderConfig,omitempty" jsonschema:"description=Data provider configuration"`

    // Session settings
    Symbols []string `json:"symbols" jsonschema:"description=Symbols to trade" validate:"required,min=1"`
}
```

### Example Configurations

#### Trade via IBKR with Polygon Data

```yaml
session:
  trading_provider: ibkr-paper
  trading_provider_config:
    initial_capital: 100000.0
    decimal_precision: 2
    host: "127.0.0.1"
    port: 7497
    client_id: 1
    account_id: "DU123456"

  data_provider: polygon
  data_provider_config:
    api_key: ${POLYGON_API_KEY}

  symbols:
    - AAPL
    - GOOGL
    - MSFT
```

#### Trade via Binance (Paper) with Built-in Data

```yaml
session:
  trading_provider: binance-paper
  trading_provider_config:
    initial_capital: 10000.0
    decimal_precision: 8
    api_key: ${BINANCE_TESTNET_API_KEY}
    secret_key: ${BINANCE_TESTNET_SECRET_KEY}

  # No data_provider specified - uses Binance's built-in market data
  symbols:
    - BTCUSDT
    - ETHUSDT
```

#### Trade via IBKR Live with Binance Data

```yaml
session:
  trading_provider: ibkr-live
  trading_provider_config:
    initial_capital: 50000.0
    decimal_precision: 2
    host: "127.0.0.1"
    port: 7496
    client_id: 1
    account_id: "U123456"

  data_provider: binance
  data_provider_config: {}

  symbols:
    - AAPL
```

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                            Strategy (WASM)                               │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ gRPC (go-plugin)
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        Strategy Host Service                             │
│  ┌─────────────────────────────┐  ┌─────────────────────────────────┐   │
│  │      Data Operations        │  │      Trading Operations         │   │
│  │  - GetRange                 │  │  - PlaceOrder                   │   │
│  │  - ReadLastData             │  │  - GetPositions                 │   │
│  │  - ExecuteSQL               │  │  - CancelOrder                  │   │
│  │  - ConfigureIndicator       │  │  - GetAccountInfo               │   │
│  │  - GetSignal                │  │  - GetTrades                    │   │
│  └──────────────┬──────────────┘  └──────────────┬──────────────────┘   │
└─────────────────┼────────────────────────────────┼──────────────────────┘
                  │                                │
                  ▼                                ▼
┌─────────────────────────────────┐  ┌─────────────────────────────────────┐
│     Data Provider Registry      │  │     Trading Provider Registry       │
│                                 │  │                                     │
│  ┌───────────┐ ┌───────────┐   │  │  ┌─────────────┐ ┌─────────────┐    │
│  │  Polygon  │ │  Binance  │   │  │  │binance-paper│ │binance-live │    │
│  └───────────┘ └───────────┘   │  │  └─────────────┘ └─────────────┘    │
│                                 │  │  ┌─────────────┐ ┌─────────────┐    │
│                                 │  │  │ ibkr-paper  │ │  ibkr-live  │    │
│                                 │  │  └─────────────┘ └─────────────┘    │
└─────────────────────────────────┘  └─────────────────────────────────────┘
                  │                                │
                  ▼                                ▼
┌─────────────────────────────────┐  ┌─────────────────────────────────────┐
│      External Data Sources      │  │         Broker/Exchange APIs        │
│  - Polygon.io API               │  │  - Binance API / Testnet            │
│  - Binance Public API           │  │  - IBKR TWS / Gateway               │
│  - Historical Parquet files     │  │                                     │
└─────────────────────────────────┘  └─────────────────────────────────────┘
```

## Core Types

### ExecuteOrder

```go
type ExecuteOrder struct {
    ID           string                                            `json:"id"`
    Symbol       string                                            `json:"symbol"`
    Side         PurchaseType                                      `json:"side"`          // BUY or SELL
    OrderType    OrderType                                         `json:"orderType"`     // MARKET or LIMIT
    Reason       Reason                                            `json:"reason"`
    Price        float64                                           `json:"price"`
    StrategyName string                                            `json:"strategyName"`
    Quantity     float64                                           `json:"quantity"`
    PositionType PositionType                                      `json:"positionType"`  // LONG or SHORT
    TakeProfit   optional.Option[ExecuteOrderTakeProfitOrStopLoss] `json:"takeProfit"`
    StopLoss     optional.Option[ExecuteOrderTakeProfitOrStopLoss] `json:"stopLoss"`
}
```

### Position

```go
type Position struct {
    Symbol                       string    `json:"symbol"`
    TotalLongPositionQuantity    float64   `json:"totalLongPositionQuantity"`
    TotalShortPositionQuantity   float64   `json:"totalShortPositionQuantity"`

    // Long position tracking
    TotalLongInPositionQuantity  float64   `json:"totalLongInPositionQuantity"`
    TotalLongOutPositionQuantity float64   `json:"totalLongOutPositionQuantity"`
    TotalLongInPositionAmount    float64   `json:"totalLongInPositionAmount"`
    TotalLongOutPositionAmount   float64   `json:"totalLongOutPositionAmount"`
    TotalLongInFee               float64   `json:"totalLongInFee"`
    TotalLongOutFee              float64   `json:"totalLongOutFee"`

    // Short position tracking
    TotalShortInPositionQuantity  float64  `json:"totalShortInPositionQuantity"`
    TotalShortOutPositionQuantity float64  `json:"totalShortOutPositionQuantity"`
    TotalShortInPositionAmount    float64  `json:"totalShortInPositionAmount"`
    TotalShortOutPositionAmount   float64  `json:"totalShortOutPositionAmount"`
    TotalShortInFee               float64  `json:"totalShortInFee"`
    TotalShortOutFee              float64  `json:"totalShortOutFee"`

    OpenTimestamp time.Time `json:"openTimestamp"`
    StrategyName  string    `json:"strategyName"`
}
```

### AccountInfo

```go
type AccountInfo struct {
    Balance       float64 `json:"balance"`
    Equity        float64 `json:"equity"`
    BuyingPower   float64 `json:"buyingPower"`
    RealizedPnL   float64 `json:"realizedPnL"`
    UnrealizedPnL float64 `json:"unrealizedPnL"`
    TotalFees     float64 `json:"totalFees"`
    MarginUsed    float64 `json:"marginUsed"`
}
```

### Enums

```go
// Order Status
const (
    OrderStatusPending   OrderStatus = "PENDING"
    OrderStatusFilled    OrderStatus = "FILLED"
    OrderStatusCancelled OrderStatus = "CANCELLED"
    OrderStatusRejected  OrderStatus = "REJECTED"
    OrderStatusFailed    OrderStatus = "FAILED"
)

// Position Type
const (
    PositionTypeLong  PositionType = "LONG"
    PositionTypeShort PositionType = "SHORT"
)

// Purchase Type
const (
    PurchaseTypeBuy  PurchaseType = "BUY"
    PurchaseTypeSell PurchaseType = "SELL"
)

// Order Type
const (
    OrderTypeMarket OrderType = "MARKET"
    OrderTypeLimit  OrderType = "LIMIT"
)
```

## Implementation Plan

### Phase 1: Provider Registry Infrastructure

1. Create `pkg/trading/provider_registry.go` with:
   - Provider type constants
   - Provider info registry
   - `GetSupportedProviders()`, `GetProviderInfo()`, `GetProviderConfigSchema()`

2. Create `pkg/trading/provider_config.go` with:
   - Base and provider-specific config structs
   - JSON schema tags for schema generation
   - Config parsing functions

3. Create `pkg/trading/provider.go` with:
   - `Provider` interface definition
   - `NewProvider()` factory function

### Phase 2: Binance Providers

1. Create `pkg/trading/provider/binance.go`:
   - `BinanceProvider` struct implementing `Provider`
   - Support for both testnet (paper) and mainnet (live)
   - WebSocket connection for real-time updates

2. Implement order management via Binance API
3. Implement position tracking and account info

### Phase 3: IBKR Providers

1. Create `pkg/trading/provider/ibkr.go`:
   - `IBKRProvider` struct implementing `Provider`
   - TWS/Gateway connection handling
   - Paper vs live account detection

2. Implement order management via IBKR API
3. Implement position sync and account info

### Phase 4: Trading Session Manager

1. Create `TradingSessionManager` that:
   - Initializes both data and trading providers
   - Routes data requests to data provider
   - Routes trading requests to trading provider
   - Handles provider lifecycle

## File Structure

```
pkg/
└── trading/
    ├── provider_registry.go    # Registry with GetSupportedProviders(), GetProviderInfo()
    ├── provider_config.go      # Config structs with JSON schema tags
    ├── provider.go             # Provider interface and NewProvider factory
    ├── session.go              # TradingSessionConfig and TradingSessionManager
    └── provider/
        ├── binance.go          # BinanceProvider implementation
        └── ibkr.go             # IBKRProvider implementation
```

## Error Handling

```go
const (
    ErrCodeInsufficientFunds    = "INSUFFICIENT_FUNDS"
    ErrCodeInsufficientPosition = "INSUFFICIENT_POSITION"
    ErrCodeInvalidOrder         = "INVALID_ORDER"
    ErrCodeOrderNotFound        = "ORDER_NOT_FOUND"
    ErrCodeOrderAlreadyFilled   = "ORDER_ALREADY_FILLED"
    ErrCodeBrokerRejected       = "BROKER_REJECTED"
    ErrCodeConnectionLost       = "CONNECTION_LOST"
    ErrCodeRateLimited          = "RATE_LIMITED"
    ErrCodeProviderNotConnected = "PROVIDER_NOT_CONNECTED"
)

type TradingError struct {
    Code     string                 `json:"code"`
    Message  string                 `json:"message"`
    Provider ProviderType           `json:"provider"`
    Details  map[string]interface{} `json:"details,omitempty"`
    Cause    error                  `json:"-"`
}
```

## Testing Strategy

### Unit Tests

- Test provider registry functions
- Test config parsing and validation
- Mock external APIs for provider implementations

### Integration Tests

- Test each provider with real/testnet APIs
- Test data provider + trading provider combinations
- Test order lifecycle (place → fill → position update)

### End-to-End Tests

- Run paper trading sessions with sample strategies
- Validate P&L calculations across providers
