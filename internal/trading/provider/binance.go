package tradingprovider

import (
	"context"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/internal/utils"
	"github.com/rxtech-lab/argo-trading/pkg/errors"
)

const (
	// BinanceDecimalPrecision is a default decimal precision used as a fallback.
	// 8 decimals allows for satoshi-level precision (0.00000001 BTC) for BTC-like assets.
	// Production systems should use symbol-specific precision from Binance exchange info (e.g. LOT_SIZE, PRICE_FILTER).
	BinanceDecimalPrecision = 8
)

// Service interfaces for mocking the Binance API

// CreateOrderService interface for creating orders.
type CreateOrderService interface {
	Symbol(symbol string) CreateOrderService
	Side(side binance.SideType) CreateOrderService
	Type(orderType binance.OrderType) CreateOrderService
	Quantity(quantity string) CreateOrderService
	Price(price string) CreateOrderService
	TimeInForce(tif binance.TimeInForceType) CreateOrderService
	Do(ctx context.Context) (*binance.CreateOrderResponse, error)
}

// GetAccountService interface for getting account info.
type GetAccountService interface {
	Do(ctx context.Context) (*binance.Account, error)
}

// ListOpenOrdersService interface for listing open orders.
type ListOpenOrdersService interface {
	Do(ctx context.Context) ([]*binance.Order, error)
}

// CancelOrderService interface for canceling orders.
type CancelOrderService interface {
	Symbol(symbol string) CancelOrderService
	OrderID(orderID int64) CancelOrderService
	Do(ctx context.Context) (*binance.CancelOrderResponse, error)
}

// CancelOpenOrdersService interface for canceling all open orders for a symbol.
type CancelOpenOrdersService interface {
	Symbol(symbol string) CancelOpenOrdersService
	Do(ctx context.Context) error
}

// ListTradesService interface for listing trades.
type ListTradesService interface {
	Symbol(symbol string) ListTradesService
	Limit(limit int) ListTradesService
	StartTime(startTime int64) ListTradesService
	EndTime(endTime int64) ListTradesService
	Do(ctx context.Context) ([]*binance.TradeV3, error)
}

// TradeFeeService interface for getting trade fees.
type TradeFeeService interface {
	Symbol(symbol string) TradeFeeService
	Do(ctx context.Context) ([]*binance.TradeFeeDetails, error)
}

// BinanceClient interface abstracts the Binance client for testing.
type BinanceClient interface {
	NewCreateOrderService() CreateOrderService
	NewGetAccountService() GetAccountService
	NewListOpenOrdersService() ListOpenOrdersService
	NewCancelOrderService() CancelOrderService
	NewCancelOpenOrdersService() CancelOpenOrdersService
	NewListTradesService() ListTradesService
	NewTradeFeeService() TradeFeeService
}

// realBinanceClient wraps the actual binance.Client.
type realBinanceClient struct {
	client *binance.Client
}

func (r *realBinanceClient) NewCreateOrderService() CreateOrderService {
	return &realCreateOrderService{service: r.client.NewCreateOrderService()}
}

func (r *realBinanceClient) NewGetAccountService() GetAccountService {
	return &realGetAccountService{service: r.client.NewGetAccountService()}
}

func (r *realBinanceClient) NewListOpenOrdersService() ListOpenOrdersService {
	return &realListOpenOrdersService{service: r.client.NewListOpenOrdersService()}
}

func (r *realBinanceClient) NewCancelOrderService() CancelOrderService {
	return &realCancelOrderService{service: r.client.NewCancelOrderService()}
}

func (r *realBinanceClient) NewCancelOpenOrdersService() CancelOpenOrdersService {
	return &realCancelOpenOrdersService{service: r.client.NewCancelOpenOrdersService()}
}

func (r *realBinanceClient) NewListTradesService() ListTradesService {
	return &realListTradesService{service: r.client.NewListTradesService()}
}

func (r *realBinanceClient) NewTradeFeeService() TradeFeeService {
	return &realTradeFeeService{service: r.client.NewTradeFeeService()}
}

// Real service wrappers

type realCreateOrderService struct {
	service *binance.CreateOrderService
}

func (s *realCreateOrderService) Symbol(symbol string) CreateOrderService {
	s.service = s.service.Symbol(symbol)

	return s
}

func (s *realCreateOrderService) Side(side binance.SideType) CreateOrderService {
	s.service = s.service.Side(side)

	return s
}

func (s *realCreateOrderService) Type(orderType binance.OrderType) CreateOrderService {
	s.service = s.service.Type(orderType)

	return s
}

func (s *realCreateOrderService) Quantity(quantity string) CreateOrderService {
	s.service = s.service.Quantity(quantity)

	return s
}

func (s *realCreateOrderService) Price(price string) CreateOrderService {
	s.service = s.service.Price(price)

	return s
}

func (s *realCreateOrderService) TimeInForce(tif binance.TimeInForceType) CreateOrderService {
	s.service = s.service.TimeInForce(tif)

	return s
}

func (s *realCreateOrderService) Do(ctx context.Context) (*binance.CreateOrderResponse, error) {
	return s.service.Do(ctx)
}

type realGetAccountService struct {
	service *binance.GetAccountService
}

func (s *realGetAccountService) Do(ctx context.Context) (*binance.Account, error) {
	return s.service.Do(ctx)
}

type realListOpenOrdersService struct {
	service *binance.ListOpenOrdersService
}

func (s *realListOpenOrdersService) Do(ctx context.Context) ([]*binance.Order, error) {
	return s.service.Do(ctx)
}

type realCancelOrderService struct {
	service *binance.CancelOrderService
}

func (s *realCancelOrderService) Symbol(symbol string) CancelOrderService {
	s.service = s.service.Symbol(symbol)

	return s
}

func (s *realCancelOrderService) OrderID(orderID int64) CancelOrderService {
	s.service = s.service.OrderID(orderID)

	return s
}

func (s *realCancelOrderService) Do(ctx context.Context) (*binance.CancelOrderResponse, error) {
	return s.service.Do(ctx)
}

type realCancelOpenOrdersService struct {
	service *binance.CancelOpenOrdersService
}

func (s *realCancelOpenOrdersService) Symbol(symbol string) CancelOpenOrdersService {
	s.service = s.service.Symbol(symbol)

	return s
}

func (s *realCancelOpenOrdersService) Do(ctx context.Context) error {
	_, err := s.service.Do(ctx)

	return err
}

type realListTradesService struct {
	service *binance.ListTradesService
}

func (s *realListTradesService) Symbol(symbol string) ListTradesService {
	s.service = s.service.Symbol(symbol)

	return s
}

func (s *realListTradesService) Limit(limit int) ListTradesService {
	s.service = s.service.Limit(limit)

	return s
}

func (s *realListTradesService) StartTime(startTime int64) ListTradesService {
	s.service = s.service.StartTime(startTime)

	return s
}

func (s *realListTradesService) EndTime(endTime int64) ListTradesService {
	s.service = s.service.EndTime(endTime)

	return s
}

func (s *realListTradesService) Do(ctx context.Context) ([]*binance.TradeV3, error) {
	return s.service.Do(ctx)
}

type realTradeFeeService struct {
	service *binance.TradeFeeService
}

func (s *realTradeFeeService) Symbol(symbol string) TradeFeeService {
	s.service = s.service.Symbol(symbol)

	return s
}

func (s *realTradeFeeService) Do(ctx context.Context) ([]*binance.TradeFeeDetails, error) {
	return s.service.Do(ctx)
}

// BinanceTradingSystemProvider implements TradingSystemProvider using Binance API.
// It is stateless - all data is fetched directly from the Binance API.
type BinanceTradingSystemProvider struct {
	client           BinanceClient
	decimalPrecision int
	onStatusChange   OnStatusChange
}

// NewBinanceTradingSystemProvider creates a new Binance trading system.
// If useTestnet is true, connects to Binance Testnet (https://testnet.binance.vision/).
// If config.BaseURL is set, it takes precedence over useTestnet.
func NewBinanceTradingSystemProvider(config BinanceProviderConfig, useTestnet bool) (*BinanceTradingSystemProvider, error) {
	if useTestnet {
		binance.UseTestnet = true
	}

	client := binance.NewClient(config.ApiKey, config.SecretKey)

	// Set custom base URL if provided (takes precedence over useTestnet)
	if config.BaseURL != "" {
		client.BaseURL = config.BaseURL
	}

	return &BinanceTradingSystemProvider{
		client:           &realBinanceClient{client: client},
		decimalPrecision: BinanceDecimalPrecision,
		onStatusChange:   nil,
	}, nil
}

// newBinanceTradingSystemProviderWithClient creates a new Binance trading system with a custom client.
// This is used for testing with mock clients.
func newBinanceTradingSystemProviderWithClient(client BinanceClient) *BinanceTradingSystemProvider {
	return &BinanceTradingSystemProvider{
		client:           client,
		decimalPrecision: BinanceDecimalPrecision,
		onStatusChange:   nil,
	}
}

// newBinanceTradingSystemProviderWithPrecision creates a new Binance trading system with custom precision.
// This is used for testing with different decimal precisions.
func newBinanceTradingSystemProviderWithPrecision(client BinanceClient, decimalPrecision int) *BinanceTradingSystemProvider {
	return &BinanceTradingSystemProvider{
		client:           client,
		decimalPrecision: decimalPrecision,
		onStatusChange:   nil,
	}
}

// PlaceOrder places a single order on Binance.
func (b *BinanceTradingSystemProvider) PlaceOrder(order types.ExecuteOrder) error {
	ctx := context.Background()

	// Map order side
	var side binance.SideType

	switch order.Side {
	case types.PurchaseTypeBuy:
		side = binance.SideTypeBuy
	case types.PurchaseTypeSell:
		side = binance.SideTypeSell
	default:
		return errors.Newf(errors.ErrCodeInvalidParameter, "unsupported order side: %s", order.Side)
	}

	// Map order type
	var orderType binance.OrderType

	switch order.OrderType {
	case types.OrderTypeMarket:
		orderType = binance.OrderTypeMarket
	case types.OrderTypeLimit:
		orderType = binance.OrderTypeLimit
	default:
		return errors.Newf(errors.ErrCodeInvalidParameter, "unsupported order type: %s", order.OrderType)
	}

	// Validate and round quantity to decimal precision
	if order.Quantity <= 0 {
		return errors.New(errors.ErrCodeInvalidParameter, "order quantity must be greater than zero")
	}

	roundedQuantity := utils.RoundToDecimalPrecision(order.Quantity, b.decimalPrecision)
	if roundedQuantity <= 0 {
		return errors.Newf(errors.ErrCodeInvalidParameter,
			"order quantity %.8f is too small after rounding to %d decimal places",
			order.Quantity, b.decimalPrecision)
	}

	// Create order service
	orderService := b.client.NewCreateOrderService().
		Symbol(order.Symbol).
		Side(side).
		Type(orderType).
		Quantity(strconv.FormatFloat(roundedQuantity, 'f', b.decimalPrecision, 64))

	// For limit orders, add price and time in force
	if order.OrderType == types.OrderTypeLimit {
		orderService = orderService.
			Price(strconv.FormatFloat(order.Price, 'f', -1, 64)).
			TimeInForce(binance.TimeInForceTypeGTC)
	}

	// Execute order
	_, err := orderService.Do(ctx)
	if err != nil {
		return errors.Wrap(errors.ErrCodeOrderFailed, "failed to place order on Binance", err)
	}

	return nil
}

// PlaceMultipleOrders places multiple orders sequentially.
func (b *BinanceTradingSystemProvider) PlaceMultipleOrders(orders []types.ExecuteOrder) error {
	for _, order := range orders {
		if err := b.PlaceOrder(order); err != nil {
			return err
		}
	}

	return nil
}

// GetPositions returns all positions derived from account balances.
func (b *BinanceTradingSystemProvider) GetPositions() ([]types.Position, error) {
	ctx := context.Background()

	account, err := b.client.NewGetAccountService().Do(ctx)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeOrderFailed, "failed to get account info from Binance", err)
	}

	positions := make([]types.Position, 0)

	for _, balance := range account.Balances {
		free, _ := strconv.ParseFloat(balance.Free, 64)
		locked, _ := strconv.ParseFloat(balance.Locked, 64)
		total := free + locked

		if total > 0 {
			positions = append(positions, types.Position{
				Symbol:                        balance.Asset,
				TotalLongPositionQuantity:     total,
				TotalShortPositionQuantity:    0,
				TotalLongInPositionQuantity:   0,
				TotalLongOutPositionQuantity:  0,
				TotalLongInPositionAmount:     0,
				TotalLongOutPositionAmount:    0,
				TotalShortInPositionQuantity:  0,
				TotalShortOutPositionQuantity: 0,
				TotalShortInPositionAmount:    0,
				TotalShortOutPositionAmount:   0,
				TotalLongInFee:                0,
				TotalLongOutFee:               0,
				TotalShortInFee:               0,
				TotalShortOutFee:              0,
				OpenTimestamp:                 time.Time{},
				StrategyName:                  "",
			})
		}
	}

	return positions, nil
}

// GetPosition returns the position for a specific symbol.
func (b *BinanceTradingSystemProvider) GetPosition(symbol string) (types.Position, error) {
	positions, err := b.GetPositions()
	if err != nil {
		return types.Position{}, err
	}

	for _, pos := range positions {
		if pos.Symbol == symbol {
			return pos, nil
		}
	}

	// Return empty position if not found
	return types.Position{
		Symbol:                        symbol,
		TotalLongPositionQuantity:     0,
		TotalShortPositionQuantity:    0,
		TotalLongInPositionQuantity:   0,
		TotalLongOutPositionQuantity:  0,
		TotalLongInPositionAmount:     0,
		TotalLongOutPositionAmount:    0,
		TotalShortInPositionQuantity:  0,
		TotalShortOutPositionQuantity: 0,
		TotalShortInPositionAmount:    0,
		TotalShortOutPositionAmount:   0,
		TotalLongInFee:                0,
		TotalLongOutFee:               0,
		TotalShortInFee:               0,
		TotalShortOutFee:              0,
		OpenTimestamp:                 time.Time{},
		StrategyName:                  "",
	}, nil
}

// CancelOrder cancels an order by order ID.
func (b *BinanceTradingSystemProvider) CancelOrder(orderID string) error {
	ctx := context.Background()

	// Binance requires symbol to cancel order, but we only have orderID
	// We need to find the order first from open orders
	openOrders, err := b.GetOpenOrders()
	if err != nil {
		return err
	}

	for _, order := range openOrders {
		if order.ID == orderID {
			// Parse order ID as int64 for Binance API
			binanceOrderID, parseErr := strconv.ParseInt(orderID, 10, 64)
			if parseErr != nil {
				return errors.Wrap(errors.ErrCodeInvalidParameter, "invalid order ID format", parseErr)
			}

			_, err := b.client.NewCancelOrderService().
				Symbol(order.Symbol).
				OrderID(binanceOrderID).
				Do(ctx)
			if err != nil {
				return errors.Wrap(errors.ErrCodeOrderFailed, "failed to cancel order on Binance", err)
			}

			return nil
		}
	}

	return errors.Newf(errors.ErrCodeDataNotFound, "order not found: %s", orderID)
}

// CancelAllOrders cancels all open orders.
func (b *BinanceTradingSystemProvider) CancelAllOrders() error {
	ctx := context.Background()

	// Get all open orders first
	openOrders, err := b.GetOpenOrders()
	if err != nil {
		return err
	}

	// Group orders by symbol
	symbolOrders := make(map[string]bool)
	for _, order := range openOrders {
		symbolOrders[order.Symbol] = true
	}

	// Cancel all orders for each symbol
	for symbol := range symbolOrders {
		err := b.client.NewCancelOpenOrdersService().
			Symbol(symbol).
			Do(ctx)
		if err != nil {
			return errors.Wrap(errors.ErrCodeOrderFailed, "failed to cancel orders on Binance", err)
		}
	}

	return nil
}

// GetOrderStatus returns the status of an order.
func (b *BinanceTradingSystemProvider) GetOrderStatus(orderID string) (types.OrderStatus, error) {
	ctx := context.Background()

	// First check open orders
	openOrders, err := b.client.NewListOpenOrdersService().Do(ctx)
	if err != nil {
		return types.OrderStatusFailed, errors.Wrap(errors.ErrCodeOrderFailed, "failed to get open orders from Binance", err)
	}

	for _, order := range openOrders {
		if strconv.FormatInt(order.OrderID, 10) == orderID {
			return mapBinanceOrderStatus(order.Status), nil
		}
	}

	// Order not in open orders - it might be filled or cancelled
	// We would need the symbol to query specific order, returning failed for now
	return types.OrderStatusFailed, nil
}

// GetAccountInfo returns the current account state.
func (b *BinanceTradingSystemProvider) GetAccountInfo() (types.AccountInfo, error) {
	ctx := context.Background()

	account, err := b.client.NewGetAccountService().Do(ctx)
	if err != nil {
		return types.AccountInfo{}, errors.Wrap(errors.ErrCodeOrderFailed, "failed to get account info from Binance", err)
	}

	// Calculate total balance in quote currency (assuming USDT as base)
	var totalBalance float64

	for _, balance := range account.Balances {
		if balance.Asset == "USDT" || balance.Asset == "BUSD" || balance.Asset == "USD" {
			free, _ := strconv.ParseFloat(balance.Free, 64)
			locked, _ := strconv.ParseFloat(balance.Locked, 64)
			totalBalance += free + locked
		}
	}

	// Calculate buying power (free balance in quote currency)
	var buyingPower float64

	for _, balance := range account.Balances {
		if balance.Asset == "USDT" || balance.Asset == "BUSD" || balance.Asset == "USD" {
			free, _ := strconv.ParseFloat(balance.Free, 64)
			buyingPower += free
		}
	}

	return types.AccountInfo{
		Balance:       totalBalance,
		Equity:        totalBalance, // For spot, equity equals balance
		BuyingPower:   buyingPower,
		RealizedPnL:   0, // Not tracked in spot
		UnrealizedPnL: 0, // Would need current prices to calculate
		TotalFees:     0, // Not directly available from account info
		MarginUsed:    0, // Not applicable for spot
	}, nil
}

// GetOpenOrders returns all pending/open orders.
func (b *BinanceTradingSystemProvider) GetOpenOrders() ([]types.ExecuteOrder, error) {
	ctx := context.Background()

	binanceOrders, err := b.client.NewListOpenOrdersService().Do(ctx)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeOrderFailed, "failed to get open orders from Binance", err)
	}

	orders := make([]types.ExecuteOrder, 0, len(binanceOrders))

	for _, bo := range binanceOrders {
		order, convertErr := convertBinanceOrderToExecuteOrder(bo)
		if convertErr != nil {
			continue // Skip orders that can't be converted
		}

		orders = append(orders, order)
	}

	return orders, nil
}

// GetTrades returns executed trades with optional filtering.
func (b *BinanceTradingSystemProvider) GetTrades(filter types.TradeFilter) ([]types.Trade, error) {
	ctx := context.Background()

	if filter.Symbol == "" {
		return nil, errors.New(errors.ErrCodeInvalidParameter, "symbol is required for GetTrades on Binance")
	}

	tradeService := b.client.NewListTradesService().Symbol(filter.Symbol)

	if filter.Limit > 0 {
		tradeService = tradeService.Limit(filter.Limit)
	}

	if !filter.StartTime.IsZero() {
		tradeService = tradeService.StartTime(filter.StartTime.UnixMilli())
	}

	if !filter.EndTime.IsZero() {
		tradeService = tradeService.EndTime(filter.EndTime.UnixMilli())
	}

	binanceTrades, err := tradeService.Do(ctx)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeOrderFailed, "failed to get trades from Binance", err)
	}

	trades := make([]types.Trade, 0, len(binanceTrades))

	for _, bt := range binanceTrades {
		trade := convertBinanceTradeToTrade(bt, filter.Symbol)
		trades = append(trades, trade)
	}

	return trades, nil
}

// GetMaxBuyQuantity returns the maximum quantity that can be bought at the given price.
// It takes into account the current balance and commission fees.
func (b *BinanceTradingSystemProvider) GetMaxBuyQuantity(symbol string, price float64) (float64, error) {
	if price <= 0 {
		return 0, errors.New(errors.ErrCodeInvalidParameter, "price must be greater than zero")
	}

	accountInfo, err := b.GetAccountInfo()
	if err != nil {
		return 0, err
	}

	buyingPower := accountInfo.BuyingPower

	// Try to fetch symbol-specific trade fee to reserve enough balance for commission.
	// If this call fails or returns invalid data, we fall back to a default fee rate.
	var feeRate float64 = 0.001 // Default 0.1% fee rate as fallback

	if symbol != "" {
		ctx := context.Background()
		tradeFees, feeErr := b.client.NewTradeFeeService().Symbol(symbol).Do(ctx)

		if feeErr == nil && len(tradeFees) > 0 {
			feeInfo := tradeFees[0]

			// Use taker fee as it's typically the higher rate and safer for market orders
			if feeInfo.TakerCommission != "" {
				if takerFee, parseErr := strconv.ParseFloat(feeInfo.TakerCommission, 64); parseErr == nil {
					feeRate = takerFee
				}
			}
		}
	}

	// Adjust buying power to account for fees
	// effectiveBuyingPower = buyingPower / (1 + feeRate)
	effectiveBuyingPower := buyingPower / (1 + feeRate)

	// Max quantity = effective buying power / price
	maxQty := effectiveBuyingPower / price

	return maxQty, nil
}

// GetMaxSellQuantity returns the maximum quantity that can be sold for a symbol.
func (b *BinanceTradingSystemProvider) GetMaxSellQuantity(symbol string) (float64, error) {
	position, err := b.GetPosition(symbol)
	if err != nil {
		return 0, err
	}

	return position.TotalLongPositionQuantity, nil
}

// CheckConnection verifies if the trading provider is connected by performing a health check.
// For Binance, it uses the GetAccountService to verify connectivity and authentication.
func (b *BinanceTradingSystemProvider) CheckConnection(ctx context.Context) error {
	_, err := b.client.NewGetAccountService().Do(ctx)
	if err != nil {
		return errors.Wrap(errors.ErrCodeOrderFailed, "failed to connect to Binance API", err)
	}

	return nil
}

// SetOnStatusChange sets a callback that will be called when the connection status changes.
func (b *BinanceTradingSystemProvider) SetOnStatusChange(callback OnStatusChange) {
	b.onStatusChange = callback
}

// emitStatus emits a status change if a callback is registered.
func (b *BinanceTradingSystemProvider) emitStatus(status types.ProviderConnectionStatus) {
	if b.onStatusChange != nil {
		b.onStatusChange(status)
	}
}

// Helper functions

// mapBinanceOrderStatus maps Binance order status to our OrderStatus type.
func mapBinanceOrderStatus(status binance.OrderStatusType) types.OrderStatus {
	switch status {
	case binance.OrderStatusTypeNew, binance.OrderStatusTypePartiallyFilled:
		return types.OrderStatusPending
	case binance.OrderStatusTypeFilled:
		return types.OrderStatusFilled
	case binance.OrderStatusTypeCanceled:
		return types.OrderStatusCancelled
	case binance.OrderStatusTypeRejected:
		return types.OrderStatusRejected
	case binance.OrderStatusTypeExpired, binance.OrderStatusTypePendingCancel:
		return types.OrderStatusFailed
	default:
		return types.OrderStatusFailed
	}
}

// convertBinanceOrderToExecuteOrder converts a Binance order to our ExecuteOrder type.
func convertBinanceOrderToExecuteOrder(bo *binance.Order) (types.ExecuteOrder, error) {
	quantity, _ := strconv.ParseFloat(bo.OrigQuantity, 64)
	price, _ := strconv.ParseFloat(bo.Price, 64)

	var side types.PurchaseType

	switch bo.Side {
	case binance.SideTypeBuy:
		side = types.PurchaseTypeBuy
	case binance.SideTypeSell:
		side = types.PurchaseTypeSell
	default:
		return types.ExecuteOrder{}, errors.Newf(errors.ErrCodeInvalidParameter, "unknown side: %s", bo.Side)
	}

	var orderType types.OrderType

	switch bo.Type {
	case binance.OrderTypeMarket:
		orderType = types.OrderTypeMarket
	case binance.OrderTypeLimit:
		orderType = types.OrderTypeLimit
	default:
		orderType = types.OrderTypeLimit // Default to limit for other types
	}

	return types.ExecuteOrder{
		ID:        strconv.FormatInt(bo.OrderID, 10),
		Symbol:    bo.Symbol,
		Side:      side,
		OrderType: orderType,
		Reason: types.Reason{
			Reason:  types.OrderReasonStrategy,
			Message: "Order from Binance",
		},
		Price:        price,
		StrategyName: "",
		Quantity:     quantity,
		PositionType: types.PositionTypeLong, // Spot only supports long
		TakeProfit:   optional.None[types.ExecuteOrderTakeProfitOrStopLoss](),
		StopLoss:     optional.None[types.ExecuteOrderTakeProfitOrStopLoss](),
	}, nil
}

// convertBinanceTradeToTrade converts a Binance trade to our Trade type.
func convertBinanceTradeToTrade(bt *binance.TradeV3, symbol string) types.Trade {
	quantity, _ := strconv.ParseFloat(bt.Quantity, 64)
	price, _ := strconv.ParseFloat(bt.Price, 64)
	commission, _ := strconv.ParseFloat(bt.Commission, 64)

	var side types.PurchaseType
	if bt.IsBuyer {
		side = types.PurchaseTypeBuy
	} else {
		side = types.PurchaseTypeSell
	}

	return types.Trade{
		Order: types.Order{
			OrderID:      strconv.FormatInt(bt.OrderID, 10),
			Symbol:       symbol,
			Side:         side,
			Quantity:     quantity,
			Price:        price,
			Timestamp:    time.UnixMilli(bt.Time),
			IsCompleted:  true,
			Status:       types.OrderStatusFilled,
			Reason:       types.Reason{Reason: types.OrderReasonStrategy, Message: "Trade from Binance"},
			StrategyName: "",
			Fee:          commission,
			PositionType: types.PositionTypeLong,
		},
		ExecutedAt:    time.UnixMilli(bt.Time),
		ExecutedQty:   quantity,
		ExecutedPrice: price,
		Fee:           commission,
		PnL:           0, // Not directly available from trade
	}
}

// Ensure BinanceTradingSystem implements TradingSystemProvider.
var _ TradingSystemProvider = (*BinanceTradingSystemProvider)(nil)
