# Trading System Architecture

This document outlines the architecture and implementation plan for the trading system in Argo Trading. The design supports multiple trading system implementations (backtesting, paper trading, live trading) through a unified interface.

## Overview

The trading system is designed with a **provider pattern** that allows strategies to interact with different execution environments through a common interface. This abstraction enables:

- **Backtesting**: Simulate trades against historical data
- **Paper Trading**: Simulate trades against live market data without real money
- **Live Trading**: Execute real trades through broker APIs (Interactive Brokers, Alpaca, Binance, etc.)

## Core Interface

The `TradingSystem` interface defines all trading operations. All implementations must satisfy this contract:

```go
package trading

import "github.com/rxtech-lab/argo-trading/internal/types"

type TradingSystem interface {
    // PlaceOrder places a single order
    PlaceOrder(order types.ExecuteOrder) error

    // PlaceMultipleOrders places multiple orders atomically
    PlaceMultipleOrders(orders []types.ExecuteOrder) error

    // GetPositions returns all current positions
    GetPositions() ([]types.Position, error)

    // GetPosition returns the current position for a specific symbol
    GetPosition(symbol string) (types.Position, error)

    // CancelOrder cancels an order by ID
    CancelOrder(orderID string) error

    // CancelAllOrders cancels all pending orders
    CancelAllOrders() error

    // GetOrderStatus returns the status of an order
    GetOrderStatus(orderID string) (types.OrderStatus, error)

    // GetAccountInfo returns the current account state including balance, equity, and P&L
    GetAccountInfo() (types.AccountInfo, error)

    // GetOpenOrders returns all pending/open orders that have not been executed yet
    GetOpenOrders() ([]types.ExecuteOrder, error)

    // GetTrades returns executed trades with optional filtering
    GetTrades(filter types.TradeFilter) ([]types.Trade, error)

    // GetMaxBuyQuantity returns the maximum quantity that can be bought at the given price
    // It takes into account the current balance and commission fees
    GetMaxBuyQuantity(symbol string, price float64) (float64, error)

    // GetMaxSellQuantity returns the maximum quantity that can be sold for a symbol
    // This is the total long position quantity for the symbol
    GetMaxSellQuantity(symbol string) (float64, error)
}
```

## Core Types

### ExecuteOrder

Request structure sent from strategies to place orders:

```go
type ExecuteOrder struct {
    ID           string                                          // UUID for the order
    Symbol       string                                          // Trading symbol (e.g., "AAPL", "BTC-USD")
    Side         PurchaseType                                    // BUY or SELL
    OrderType    OrderType                                       // MARKET or LIMIT
    Reason       Reason                                          // Why this order was placed
    Price        float64                                         // Execution price (limit price for LIMIT orders)
    StrategyName string                                          // Strategy that created this order
    Quantity     float64                                         // Amount to trade
    PositionType PositionType                                    // LONG or SHORT
    TakeProfit   optional.Option[ExecuteOrderTakeProfitOrStopLoss] // Optional take profit
    StopLoss     optional.Option[ExecuteOrderTakeProfitOrStopLoss] // Optional stop loss
}
```

### Order

Internal representation of an order after processing:

```go
type Order struct {
    OrderID      string       // Unique identifier
    Symbol       string       // Trading symbol
    Side         PurchaseType // BUY or SELL
    Quantity     float64      // Order quantity
    Price        float64      // Order price
    Timestamp    time.Time    // When the order was created
    IsCompleted  bool         // Whether the order is complete
    Status       OrderStatus  // PENDING, FILLED, CANCELLED, REJECTED, FAILED
    Reason       Reason       // Reason for the order
    StrategyName string       // Strategy that created this order
    Fee          float64      // Commission fee
    PositionType PositionType // LONG or SHORT
}
```

### Position

Represents current holdings of an asset:

```go
type Position struct {
    Symbol                     string    // Trading symbol
    TotalLongPositionQuantity  float64   // Current long holdings
    TotalShortPositionQuantity float64   // Current short holdings

    // Long position tracking
    TotalLongInPositionQuantity  float64 // Total quantity bought (long)
    TotalLongOutPositionQuantity float64 // Total quantity sold (long)
    TotalLongInPositionAmount    float64 // Total amount spent buying
    TotalLongOutPositionAmount   float64 // Total amount received selling
    TotalLongInFee               float64 // Fees paid on long buys
    TotalLongOutFee              float64 // Fees paid on long sells

    // Short position tracking
    TotalShortInPositionQuantity  float64 // Total quantity sold (short)
    TotalShortOutPositionQuantity float64 // Total quantity covered
    TotalShortInPositionAmount    float64 // Total amount received shorting
    TotalShortOutPositionAmount   float64 // Total amount spent covering
    TotalShortInFee               float64 // Fees paid on short sells
    TotalShortOutFee              float64 // Fees paid on covering

    OpenTimestamp time.Time // When position was opened
    StrategyName  string    // Strategy managing this position
}
```

### AccountInfo

Current account state:

```go
type AccountInfo struct {
    Balance       float64 // Cash balance (excluding unrealized P&L)
    Equity        float64 // Total account value (balance + unrealized P&L)
    BuyingPower   float64 // Available amount for new purchases
    RealizedPnL   float64 // Total realized profit/loss from closed positions
    UnrealizedPnL float64 // Total unrealized profit/loss from open positions
    TotalFees     float64 // Total fees paid
    MarginUsed    float64 // Margin currently in use (for margin trading)
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

### Phase 1: Core Infrastructure

1. **Define Extended Interface** - Extend the base `TradingSystem` interface with lifecycle methods:

```go
type TradingSystemLifecycle interface {
    TradingSystem

    // Initialize sets up the trading system with configuration
    Initialize(config TradingSystemConfig) error

    // Start begins the trading system (connect to broker, start data feeds, etc.)
    Start(ctx context.Context) error

    // Stop gracefully shuts down the trading system
    Stop(ctx context.Context) error

    // IsConnected returns whether the system is connected and ready
    IsConnected() bool

    // GetSystemType returns the type of trading system
    GetSystemType() TradingSystemType
}

type TradingSystemType string

const (
    TradingSystemTypeBacktest TradingSystemType = "backtest"
    TradingSystemTypePaper    TradingSystemType = "paper"
    TradingSystemTypeLive     TradingSystemType = "live"
)
```

2. **Create Configuration Structure**:

```go
type TradingSystemConfig struct {
    Type             TradingSystemType
    InitialCapital   float64
    Commission       CommissionConfig
    DecimalPrecision int

    // Broker-specific configuration
    BrokerConfig     map[string]interface{}
}

type CommissionConfig struct {
    Type       string  // "fixed", "percentage", "tiered"
    FixedFee   float64 // Fixed fee per trade
    Percentage float64 // Percentage of trade value
    MinFee     float64 // Minimum fee
    MaxFee     float64 // Maximum fee (0 = no max)
}
```

3. **Implement Factory Pattern**:

```go
type TradingSystemFactory interface {
    Create(config TradingSystemConfig) (TradingSystemLifecycle, error)
}

// Registry for trading system implementations
var tradingSystemRegistry = make(map[TradingSystemType]TradingSystemFactory)

func RegisterTradingSystem(systemType TradingSystemType, factory TradingSystemFactory) {
    tradingSystemRegistry[systemType] = factory
}

func CreateTradingSystem(config TradingSystemConfig) (TradingSystemLifecycle, error) {
    factory, ok := tradingSystemRegistry[config.Type]
    if !ok {
        return nil, fmt.Errorf("unknown trading system type: %s", config.Type)
    }
    return factory.Create(config)
}
```

### Phase 2: Backtest Implementation (Existing)

The backtest trading system is already implemented in `internal/backtest/engine/engine_v1/backtest_trading.go`. Key features:

- Simulates order execution against historical data
- Supports market and limit orders
- Tracks pending orders and executes when price conditions are met
- Calculates P&L and position tracking via DuckDB

### Phase 3: Paper Trading Implementation

Paper trading simulates trades against live market data without real money:

```go
type PaperTradingSystem struct {
    config        TradingSystemConfig
    dataFeed      MarketDataFeed       // Live market data source
    state         *TradingState        // Order/position state
    commission    CommissionCalculator
    ctx           context.Context
    cancel        context.CancelFunc
    mu            sync.RWMutex
}

func (p *PaperTradingSystem) Initialize(config TradingSystemConfig) error {
    // Set up data feed connection
    // Initialize state storage
    // Configure commission calculator
}

func (p *PaperTradingSystem) Start(ctx context.Context) error {
    // Connect to live data feed
    // Start order processing goroutine
    // Begin monitoring pending orders
}

func (p *PaperTradingSystem) PlaceOrder(order types.ExecuteOrder) error {
    // Validate order
    // Check buying/selling power against current market price
    // For market orders: execute immediately at current price
    // For limit orders: add to pending queue
}
```

### Phase 4: Live Trading Implementation

Live trading executes real orders through broker APIs:

```go
type LiveTradingSystem struct {
    config       TradingSystemConfig
    broker       BrokerClient          // Broker API client
    dataFeed     MarketDataFeed        // Market data source
    orderStore   OrderStore            // Persistent order storage
    positionSync PositionSynchronizer  // Sync positions with broker
    ctx          context.Context
    cancel       context.CancelFunc
}

// BrokerClient interface for broker implementations
type BrokerClient interface {
    Connect(ctx context.Context) error
    Disconnect(ctx context.Context) error

    SubmitOrder(order types.ExecuteOrder) (string, error)
    CancelOrder(orderID string) error
    GetOrderStatus(orderID string) (types.OrderStatus, error)

    GetPositions() ([]types.Position, error)
    GetAccountInfo() (types.AccountInfo, error)

    SubscribeOrderUpdates(handler OrderUpdateHandler) error
    SubscribePositionUpdates(handler PositionUpdateHandler) error
}
```

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Strategy (WASM)                          │
└─────────────────────────────────────────────────────────────────┘
                                │
                                │ gRPC (go-plugin)
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Strategy Host Service                         │
│  - Data Access (GetRange, ReadLastData, ExecuteSQL)             │
│  - Indicators (ConfigureIndicator, GetSignal)                   │
│  - Cache (GetCache, SetCache)                                   │
│  - Trading (PlaceOrder, GetPositions, etc.)                     │
└─────────────────────────────────────────────────────────────────┘
                                │
                                │ TradingSystem Interface
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Trading System Factory                        │
│                                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  Backtest   │  │   Paper     │  │         Live            │  │
│  │   Trading   │  │   Trading   │  │        Trading          │  │
│  │   System    │  │   System    │  │        System           │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
│         │                │                      │                │
│         ▼                ▼                      ▼                │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  Historical │  │    Live     │  │     Broker API          │  │
│  │    Data     │  │ Data Feed   │  │ (IB, Alpaca, Binance)   │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                        State Storage                             │
│           (DuckDB for backtest, PostgreSQL for live)            │
└─────────────────────────────────────────────────────────────────┘
```

## Order Flow

### Market Order Flow

```
1. Strategy creates ExecuteOrder
   └─> Validates order fields (UUID, symbol, quantity, price)

2. TradingSystem.PlaceOrder() called
   └─> Validates buying/selling power
   └─> Checks current market price

3. Order Execution
   ├─> Backtest: Execute at (High + Low) / 2
   ├─> Paper: Execute at current market price
   └─> Live: Submit to broker, wait for fill

4. Post-Execution
   └─> Update balance
   └─> Update position
   └─> Calculate and store P&L
   └─> Deduct commission fees
```

### Limit Order Flow

```
1. Strategy creates ExecuteOrder with OrderType = LIMIT
   └─> Specifies desired execution price

2. TradingSystem.PlaceOrder() called
   └─> Validates buying/selling power at limit price

3. Order Queuing
   ├─> If price condition met: execute immediately
   └─> Otherwise: add to pending orders queue

4. Pending Order Monitoring
   └─> On each market data update:
       ├─> Buy: If market Low <= limit price → execute
       └─> Sell: If market High >= limit price → execute

5. Order Expiration (optional)
   └─> Cancel pending orders based on expiration policy
```

## Broker Integration Plan

### Interactive Brokers

```go
type IBBrokerClient struct {
    config     IBConfig
    gateway    *IBGateway    // TWS or IB Gateway connection
    orderStore map[string]types.Order
}

type IBConfig struct {
    Host       string // TWS/Gateway host
    Port       int    // TWS/Gateway port
    ClientID   int    // Unique client ID
    AccountID  string // IB account ID
    PaperTrade bool   // Use paper trading account
}
```

### Alpaca

```go
type AlpacaBrokerClient struct {
    config   AlpacaConfig
    client   *alpaca.Client
    stream   *alpaca.Stream
}

type AlpacaConfig struct {
    APIKey    string
    SecretKey string
    BaseURL   string // Paper or live URL
    DataFeed  string // "iex" or "sip"
}
```

### Binance

```go
type BinanceBrokerClient struct {
    config BinanceConfig
    client *binance.Client
    stream *binance.UserDataStream
}

type BinanceConfig struct {
    APIKey    string
    SecretKey string
    Testnet   bool // Use testnet for paper trading
}
```

## Error Handling

All trading operations should return structured errors:

```go
// Error codes for trading operations
const (
    ErrCodeInsufficientFunds    = "INSUFFICIENT_FUNDS"
    ErrCodeInsufficientPosition = "INSUFFICIENT_POSITION"
    ErrCodeInvalidOrder         = "INVALID_ORDER"
    ErrCodeOrderNotFound        = "ORDER_NOT_FOUND"
    ErrCodeOrderAlreadyFilled   = "ORDER_ALREADY_FILLED"
    ErrCodeBrokerRejected       = "BROKER_REJECTED"
    ErrCodeConnectionLost       = "CONNECTION_LOST"
    ErrCodeRateLimited          = "RATE_LIMITED"
)

type TradingError struct {
    Code    string
    Message string
    Details map[string]interface{}
    Cause   error
}
```

## Testing Strategy

### Unit Tests

- Test each trading system implementation in isolation
- Mock broker clients for live trading tests
- Test order validation, position calculations, P&L

### Integration Tests

- Test backtest trading with real historical data
- Test paper trading with simulated market data feeds
- Test broker client connections (using paper/testnet accounts)

### End-to-End Tests

- Run complete backtest simulations
- Run paper trading sessions with sample strategies
- Validate P&L calculations match expected results

## Configuration Examples

### Backtest Configuration

```yaml
trading_system:
  type: backtest
  initial_capital: 100000.0
  decimal_precision: 2
  commission:
    type: fixed
    fixed_fee: 1.0
```

### Paper Trading Configuration

```yaml
trading_system:
  type: paper
  initial_capital: 100000.0
  decimal_precision: 2
  commission:
    type: percentage
    percentage: 0.001
    min_fee: 1.0
  data_feed:
    provider: polygon
    symbols:
      - AAPL
      - GOOGL
      - MSFT
```

### Live Trading Configuration

```yaml
trading_system:
  type: live
  decimal_precision: 2
  broker:
    name: alpaca
    api_key: ${ALPACA_API_KEY}
    secret_key: ${ALPACA_SECRET_KEY}
    base_url: https://api.alpaca.markets
    data_feed: sip
```

## Future Enhancements

1. **Risk Management Integration**
   - Position sizing limits
   - Maximum drawdown protection
   - Exposure limits per symbol/sector

2. **Order Types**
   - Stop orders
   - Stop-limit orders
   - Trailing stop orders
   - OCO (One-Cancels-Other) orders

3. **Multi-Account Support**
   - Trade across multiple broker accounts
   - Aggregate position tracking

4. **Event System**
   - Order fill notifications
   - Position change events
   - Account balance updates

5. **Audit Trail**
   - Complete order history
   - Execution reports
   - Compliance logging
