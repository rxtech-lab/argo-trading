package tradingprovider

import (
	"context"

	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"go.uber.org/zap"
)

// LoggingTradingSystemProvider wraps a TradingSystemProvider and emits a log
// line each time the strategy invokes a host API. Used by the live trading
// engine to produce a human-readable running.log of strategy→host calls.
type LoggingTradingSystemProvider struct {
	inner TradingSystemProvider
	log   *logger.Logger
}

// NewLoggingTradingSystemProvider wraps the given provider with API-call logging.
func NewLoggingTradingSystemProvider(inner TradingSystemProvider, log *logger.Logger) TradingSystemProvider {
	return &LoggingTradingSystemProvider{inner: inner, log: log}
}

func (p *LoggingTradingSystemProvider) PlaceOrder(order types.ExecuteOrder) error {
	p.log.Info("strategy wants to call api",
		zap.String("api", "PlaceOrder"),
		zap.String("symbol", order.Symbol),
		zap.Any("side", order.Side),
		zap.Float64("price", order.Price),
		zap.Float64("quantity", order.Quantity),
	)
	err := p.inner.PlaceOrder(order)
	if err != nil {
		p.log.Warn("api call failed", zap.String("api", "PlaceOrder"), zap.Error(err))
	}

	return err
}

func (p *LoggingTradingSystemProvider) PlaceMultipleOrders(orders []types.ExecuteOrder) error {
	p.log.Info("strategy wants to call api",
		zap.String("api", "PlaceMultipleOrders"),
		zap.Int("count", len(orders)),
	)
	err := p.inner.PlaceMultipleOrders(orders)
	if err != nil {
		p.log.Warn("api call failed", zap.String("api", "PlaceMultipleOrders"), zap.Error(err))
	}

	return err
}

func (p *LoggingTradingSystemProvider) GetPositions() ([]types.Position, error) {
	p.log.Info("strategy wants to call api", zap.String("api", "GetPositions"))

	return p.inner.GetPositions()
}

func (p *LoggingTradingSystemProvider) GetPosition(symbol string) (types.Position, error) {
	p.log.Info("strategy wants to call api",
		zap.String("api", "GetPosition"),
		zap.String("symbol", symbol),
	)

	return p.inner.GetPosition(symbol)
}

func (p *LoggingTradingSystemProvider) CancelOrder(orderID string) error {
	p.log.Info("strategy wants to call api",
		zap.String("api", "CancelOrder"),
		zap.String("order_id", orderID),
	)
	err := p.inner.CancelOrder(orderID)
	if err != nil {
		p.log.Warn("api call failed", zap.String("api", "CancelOrder"), zap.Error(err))
	}

	return err
}

func (p *LoggingTradingSystemProvider) CancelAllOrders() error {
	p.log.Info("strategy wants to call api", zap.String("api", "CancelAllOrders"))
	err := p.inner.CancelAllOrders()
	if err != nil {
		p.log.Warn("api call failed", zap.String("api", "CancelAllOrders"), zap.Error(err))
	}

	return err
}

func (p *LoggingTradingSystemProvider) GetOrderStatus(orderID string) (types.OrderStatus, error) {
	p.log.Info("strategy wants to call api",
		zap.String("api", "GetOrderStatus"),
		zap.String("order_id", orderID),
	)

	return p.inner.GetOrderStatus(orderID)
}

func (p *LoggingTradingSystemProvider) GetAccountInfo() (types.AccountInfo, error) {
	p.log.Info("strategy wants to call api", zap.String("api", "GetAccountInfo"))

	return p.inner.GetAccountInfo()
}

func (p *LoggingTradingSystemProvider) GetAssets() ([]types.Asset, error) {
	p.log.Info("strategy wants to call api", zap.String("api", "GetAssets"))

	return p.inner.GetAssets()
}

func (p *LoggingTradingSystemProvider) GetPrices(symbols []string) (map[string]float64, error) {
	p.log.Info("strategy wants to call api",
		zap.String("api", "GetPrices"),
		zap.Int("count", len(symbols)),
	)

	return p.inner.GetPrices(symbols)
}

func (p *LoggingTradingSystemProvider) GetOpenOrders() ([]types.ExecuteOrder, error) {
	p.log.Info("strategy wants to call api", zap.String("api", "GetOpenOrders"))

	return p.inner.GetOpenOrders()
}

func (p *LoggingTradingSystemProvider) GetTrades(filter types.TradeFilter) ([]types.Trade, error) {
	p.log.Info("strategy wants to call api",
		zap.String("api", "GetTrades"),
		zap.Any("filter", filter),
	)

	return p.inner.GetTrades(filter)
}

func (p *LoggingTradingSystemProvider) GetMaxBuyQuantity(symbol string, price float64) (float64, error) {
	p.log.Info("strategy wants to call api",
		zap.String("api", "GetMaxBuyQuantity"),
		zap.String("symbol", symbol),
		zap.Float64("price", price),
	)

	return p.inner.GetMaxBuyQuantity(symbol, price)
}

func (p *LoggingTradingSystemProvider) GetMaxSellQuantity(symbol string) (float64, error) {
	p.log.Info("strategy wants to call api",
		zap.String("api", "GetMaxSellQuantity"),
		zap.String("symbol", symbol),
	)

	return p.inner.GetMaxSellQuantity(symbol)
}

func (p *LoggingTradingSystemProvider) CheckConnection(ctx context.Context) error {
	return p.inner.CheckConnection(ctx)
}

func (p *LoggingTradingSystemProvider) SetOnStatusChange(callback OnStatusChange) {
	p.inner.SetOnStatusChange(callback)
}

var _ TradingSystemProvider = (*LoggingTradingSystemProvider)(nil)
