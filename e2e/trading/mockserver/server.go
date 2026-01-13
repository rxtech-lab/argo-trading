// Package mockserver provides a mock Binance server for testing.
// It implements both WebSocket and REST API endpoints that mimic Binance's behavior.
package mockserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	backtestTesthelper "github.com/rxtech-lab/argo-trading/e2e/backtest/wasm/testhelper"
	"github.com/rxtech-lab/argo-trading/internal/types"
)

// MockBinanceServer provides a mock Binance server for testing.
// It supports both REST API endpoints and WebSocket streaming.
type MockBinanceServer struct {
	mu sync.RWMutex

	// HTTP server
	httpServer *http.Server
	listener   net.Listener

	// WebSocket upgrader
	upgrader websocket.Upgrader

	// State management
	balances   map[string]*Balance
	orders     map[int64]*Order
	trades     []*Trade
	orderIDSeq int64
	tradeIDSeq int64
	tradeFees  map[string]*TradeFee
	symbolInfo map[string]*SymbolInfo

	// Market data
	currentPrices map[string]float64
	dataGenerator *MarketDataGeneratorConfig

	// WebSocket connections
	wsConnections map[*websocket.Conn]bool
	wsMu          sync.RWMutex

	// Streaming configuration
	streamInterval time.Duration
	stopStreaming  chan struct{}
}

// Balance represents an account balance.
type Balance struct {
	Asset  string
	Free   float64
	Locked float64
}

// OrderStatus represents the status of an order.
type OrderStatus string

const (
	OrderStatusNew             OrderStatus = "NEW"
	OrderStatusPartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	OrderStatusFilled          OrderStatus = "FILLED"
	OrderStatusCanceled        OrderStatus = "CANCELED"
	OrderStatusRejected        OrderStatus = "REJECTED"
	OrderStatusExpired         OrderStatus = "EXPIRED"
)

// OrderType represents the type of an order.
type OrderType string

const (
	OrderTypeMarket OrderType = "MARKET"
	OrderTypeLimit  OrderType = "LIMIT"
)

// OrderSide represents the side of an order.
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

// Order represents a trading order.
type Order struct {
	OrderID      int64
	Symbol       string
	Side         OrderSide
	Type         OrderType
	Quantity     float64
	Price        float64
	Status       OrderStatus
	TimeInForce  string
	CreatedAt    time.Time
	ExecutedQty  float64
	CummulateQty float64
}

// Trade represents an executed trade.
type Trade struct {
	ID         int64
	OrderID    int64
	Symbol     string
	Price      float64
	Quantity   float64
	Commission float64
	Time       time.Time
	IsBuyer    bool
}

// TradeFee represents trading fee configuration.
type TradeFee struct {
	Symbol          string
	MakerCommission float64
	TakerCommission float64
}

// SymbolInfo represents symbol trading information.
type SymbolInfo struct {
	Symbol     string
	BaseAsset  string
	QuoteAsset string
}

// MarketDataGeneratorConfig holds configuration for market data generation.
type MarketDataGeneratorConfig struct {
	Symbols           []string
	Pattern           backtestTesthelper.SimulationPattern
	InitialPrice      float64
	VolatilityPercent float64
	TrendStrength     float64
	Seed              int64
}

// ServerConfig holds configuration for the mock server.
type ServerConfig struct {
	// InitialBalances maps asset to initial balance amount
	InitialBalances map[string]float64
	// TradeFees maps symbol to fee configuration
	TradeFees map[string]*TradeFee
	// MarketData configuration for generating prices
	MarketData *MarketDataGeneratorConfig
	// StreamInterval is the interval between price updates
	StreamInterval time.Duration
}

// NewMockBinanceServer creates a new mock Binance server.
func NewMockBinanceServer(config ServerConfig) *MockBinanceServer {
	server := &MockBinanceServer{
		mu: sync.RWMutex{},
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
		balances:       make(map[string]*Balance),
		orders:         make(map[int64]*Order),
		trades:         make([]*Trade, 0),
		orderIDSeq:     1000,
		tradeIDSeq:     1,
		tradeFees:      make(map[string]*TradeFee),
		symbolInfo:     make(map[string]*SymbolInfo),
		currentPrices:  make(map[string]float64),
		wsConnections:  make(map[*websocket.Conn]bool),
		wsMu:           sync.RWMutex{},
		streamInterval: config.StreamInterval,
		stopStreaming:  make(chan struct{}),
		dataGenerator:  config.MarketData,
		httpServer:     nil,
		listener:       nil,
	}

	// Initialize balances
	for asset, amount := range config.InitialBalances {
		server.balances[asset] = &Balance{
			Asset:  asset,
			Free:   amount,
			Locked: 0,
		}
	}

	// Initialize trade fees
	for symbol, fee := range config.TradeFees {
		server.tradeFees[symbol] = fee
	}

	// Initialize default trade fees if not provided
	if len(server.tradeFees) == 0 {
		server.tradeFees["BTCUSDT"] = &TradeFee{Symbol: "BTCUSDT", MakerCommission: 0.001, TakerCommission: 0.001}
		server.tradeFees["ETHUSDT"] = &TradeFee{Symbol: "ETHUSDT", MakerCommission: 0.001, TakerCommission: 0.001}
	}

	// Initialize symbol info
	for symbol := range server.tradeFees {
		server.initSymbolInfo(symbol)
	}

	// Set default stream interval
	if server.streamInterval == 0 {
		server.streamInterval = 100 * time.Millisecond
	}

	// Initialize prices if market data config provided
	if config.MarketData != nil {
		for _, symbol := range config.MarketData.Symbols {
			server.currentPrices[symbol] = config.MarketData.InitialPrice
			server.initSymbolInfo(symbol)
		}
	}

	return server
}

// initSymbolInfo initializes symbol information based on symbol name.
func (s *MockBinanceServer) initSymbolInfo(symbol string) {
	// Parse symbol to get base and quote assets
	// Common patterns: BTCUSDT, ETHBTC, etc.
	quoteAssets := []string{"USDT", "BUSD", "BTC", "ETH", "BNB"}
	for _, quote := range quoteAssets {
		if strings.HasSuffix(symbol, quote) {
			base := strings.TrimSuffix(symbol, quote)
			s.symbolInfo[symbol] = &SymbolInfo{
				Symbol:     symbol,
				BaseAsset:  base,
				QuoteAsset: quote,
			}
			return
		}
	}
	// Default fallback
	s.symbolInfo[symbol] = &SymbolInfo{
		Symbol:     symbol,
		BaseAsset:  symbol[:len(symbol)/2],
		QuoteAsset: symbol[len(symbol)/2:],
	}
}

// Start starts the mock server on the given address.
// If address is empty or ":0", a random available port is used.
func (s *MockBinanceServer) Start(address string) error {
	if address == "" {
		address = ":0"
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	s.listener = listener

	router := mux.NewRouter()

	// REST API endpoints
	router.HandleFunc("/api/v3/ticker/price", s.handleTickerPrice).Methods("GET")
	router.HandleFunc("/api/v3/klines", s.handleKlines).Methods("GET")
	router.HandleFunc("/api/v3/account", s.handleAccount).Methods("GET")
	router.HandleFunc("/api/v3/order", s.handleCreateOrder).Methods("POST")
	router.HandleFunc("/api/v3/order", s.handleCancelOrder).Methods("DELETE")
	router.HandleFunc("/api/v3/openOrders", s.handleOpenOrders).Methods("GET")
	router.HandleFunc("/api/v3/openOrders", s.handleCancelAllOrders).Methods("DELETE")
	router.HandleFunc("/api/v3/myTrades", s.handleMyTrades).Methods("GET")
	router.HandleFunc("/sapi/v1/asset/tradeFee", s.handleTradeFee).Methods("GET")

	// WebSocket endpoint
	router.HandleFunc("/ws/{symbol}@kline_{interval}", s.handleWebSocket)

	s.httpServer = &http.Server{
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := s.httpServer.Serve(listener); err != http.ErrServerClosed {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	return nil
}

// Stop stops the mock server.
func (s *MockBinanceServer) Stop() error {
	// Signal streaming to stop
	close(s.stopStreaming)

	// Close all WebSocket connections
	s.wsMu.Lock()
	for conn := range s.wsConnections {
		conn.Close()
	}
	s.wsConnections = make(map[*websocket.Conn]bool)
	s.wsMu.Unlock()

	// Shutdown HTTP server
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}

	return nil
}

// Address returns the address the server is listening on.
func (s *MockBinanceServer) Address() string {
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// BaseURL returns the base URL for the server.
func (s *MockBinanceServer) BaseURL() string {
	return "http://" + s.Address()
}

// WebSocketURL returns the WebSocket URL for the server.
func (s *MockBinanceServer) WebSocketURL() string {
	return "ws://" + s.Address()
}

// SetPrice sets the current price for a symbol.
func (s *MockBinanceServer) SetPrice(symbol string, price float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentPrices[symbol] = price
}

// GetPrice returns the current price for a symbol.
func (s *MockBinanceServer) GetPrice(symbol string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentPrices[symbol]
}

// GetBalance returns the balance for an asset.
func (s *MockBinanceServer) GetBalance(asset string) *Balance {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if bal, ok := s.balances[asset]; ok {
		return &Balance{Asset: bal.Asset, Free: bal.Free, Locked: bal.Locked}
	}
	return nil
}

// SetBalance sets the balance for an asset.
func (s *MockBinanceServer) SetBalance(asset string, free, locked float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.balances[asset] = &Balance{Asset: asset, Free: free, Locked: locked}
}

// GetOrder returns an order by ID.
func (s *MockBinanceServer) GetOrder(orderID int64) *Order {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if order, ok := s.orders[orderID]; ok {
		return order
	}
	return nil
}

// GetTrades returns all trades.
func (s *MockBinanceServer) GetTrades() []*Trade {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Trade, len(s.trades))
	copy(result, s.trades)
	return result
}

// Reset resets the server state.
func (s *MockBinanceServer) Reset(config ServerConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.orders = make(map[int64]*Order)
	s.trades = make([]*Trade, 0)
	s.orderIDSeq = 1000
	s.tradeIDSeq = 1

	// Reset balances
	s.balances = make(map[string]*Balance)
	for asset, amount := range config.InitialBalances {
		s.balances[asset] = &Balance{Asset: asset, Free: amount, Locked: 0}
	}
}

// REST API Handlers

// handleTickerPrice handles GET /api/v3/ticker/price
func (s *MockBinanceServer) handleTickerPrice(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	symbolsParam := r.URL.Query().Get("symbols")

	var symbols []string
	if symbolsParam != "" {
		// Parse JSON array of symbols
		if err := json.Unmarshal([]byte(symbolsParam), &symbols); err != nil {
			http.Error(w, "Invalid symbols parameter", http.StatusBadRequest)
			return
		}
	}

	type priceResponse struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	}

	var response []priceResponse
	if len(symbols) > 0 {
		for _, sym := range symbols {
			price, ok := s.currentPrices[sym]
			if ok {
				response = append(response, priceResponse{
					Symbol: sym,
					Price:  strconv.FormatFloat(price, 'f', 8, 64),
				})
			}
		}
	} else {
		for sym, price := range s.currentPrices {
			response = append(response, priceResponse{
				Symbol: sym,
				Price:  strconv.FormatFloat(price, 'f', 8, 64),
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleKlines handles GET /api/v3/klines
func (s *MockBinanceServer) handleKlines(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	interval := r.URL.Query().Get("interval")
	startTimeStr := r.URL.Query().Get("startTime")
	endTimeStr := r.URL.Query().Get("endTime")

	if symbol == "" || interval == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// Parse times
	var startTime, endTime time.Time
	if startTimeStr != "" {
		ms, _ := strconv.ParseInt(startTimeStr, 10, 64)
		startTime = time.UnixMilli(ms)
	} else {
		startTime = time.Now().Add(-24 * time.Hour)
	}
	if endTimeStr != "" {
		ms, _ := strconv.ParseInt(endTimeStr, 10, 64)
		endTime = time.UnixMilli(ms)
	} else {
		endTime = time.Now()
	}

	// Parse interval
	intervalDuration := parseInterval(interval)
	if intervalDuration == 0 {
		http.Error(w, "Invalid interval", http.StatusBadRequest)
		return
	}

	// Generate klines using mock data generator
	s.mu.RLock()
	initialPrice := s.currentPrices[symbol]
	if initialPrice == 0 {
		initialPrice = 100.0
	}
	s.mu.RUnlock()

	numPoints := int(endTime.Sub(startTime) / intervalDuration)
	if numPoints <= 0 {
		numPoints = 1
	}
	if numPoints > 500 {
		numPoints = 500 // Binance limit
	}

	pattern := backtestTesthelper.PatternVolatile
	if s.dataGenerator != nil {
		pattern = s.dataGenerator.Pattern
	}

	config := backtestTesthelper.MockDataConfig{
		Symbol:             symbol,
		StartTime:          startTime,
		EndTime:            time.Time{},
		Interval:           intervalDuration,
		NumDataPoints:      numPoints,
		Pattern:            pattern,
		InitialPrice:       initialPrice,
		MaxDrawdownPercent: 10.0,
		VolatilityPercent:  2.0,
		TrendStrength:      0.01,
		Seed:               time.Now().UnixNano(),
	}

	generator := backtestTesthelper.NewMockDataGenerator(config)
	data, err := generator.Generate()
	if err != nil {
		http.Error(w, "Failed to generate klines", http.StatusInternalServerError)
		return
	}

	// Convert to Binance kline format: [openTime, open, high, low, close, volume, closeTime, ...]
	var klines [][]interface{}
	for _, d := range data {
		closeTime := d.Time.Add(intervalDuration).UnixMilli() - 1
		klines = append(klines, []interface{}{
			d.Time.UnixMilli(),                        // Open time
			strconv.FormatFloat(d.Open, 'f', 8, 64),   // Open
			strconv.FormatFloat(d.High, 'f', 8, 64),   // High
			strconv.FormatFloat(d.Low, 'f', 8, 64),    // Low
			strconv.FormatFloat(d.Close, 'f', 8, 64),  // Close
			strconv.FormatFloat(d.Volume, 'f', 8, 64), // Volume
			closeTime, // Close time
			"0",       // Quote asset volume
			0,         // Number of trades
			"0",       // Taker buy base asset volume
			"0",       // Taker buy quote asset volume
			"0",       // Ignore
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(klines)
}

// handleAccount handles GET /api/v3/account
func (s *MockBinanceServer) handleAccount(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type balanceResponse struct {
		Asset  string `json:"asset"`
		Free   string `json:"free"`
		Locked string `json:"locked"`
	}

	var balances []balanceResponse
	for _, bal := range s.balances {
		balances = append(balances, balanceResponse{
			Asset:  bal.Asset,
			Free:   strconv.FormatFloat(bal.Free, 'f', 8, 64),
			Locked: strconv.FormatFloat(bal.Locked, 'f', 8, 64),
		})
	}

	response := map[string]interface{}{
		"makerCommission":  10,
		"takerCommission":  10,
		"buyerCommission":  0,
		"sellerCommission": 0,
		"canTrade":         true,
		"canWithdraw":      true,
		"canDeposit":       true,
		"updateTime":       time.Now().UnixMilli(),
		"accountType":      "SPOT",
		"balances":         balances,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCreateOrder handles POST /api/v3/order
func (s *MockBinanceServer) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	symbol := r.FormValue("symbol")
	side := OrderSide(r.FormValue("side"))
	orderType := OrderType(r.FormValue("type"))
	quantityStr := r.FormValue("quantity")
	priceStr := r.FormValue("price")
	timeInForce := r.FormValue("timeInForce")

	if symbol == "" || side == "" || orderType == "" || quantityStr == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	quantity, err := strconv.ParseFloat(quantityStr, 64)
	if err != nil {
		http.Error(w, "Invalid quantity", http.StatusBadRequest)
		return
	}

	var price float64
	if priceStr != "" {
		price, err = strconv.ParseFloat(priceStr, 64)
		if err != nil {
			http.Error(w, "Invalid price", http.StatusBadRequest)
			return
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Get execution price
	execPrice := price
	if orderType == OrderTypeMarket {
		if p, ok := s.currentPrices[symbol]; ok {
			execPrice = p
		} else {
			http.Error(w, "No price available for symbol", http.StatusBadRequest)
			return
		}
	}

	// Get symbol info
	symbolInfo, ok := s.symbolInfo[symbol]
	if !ok {
		http.Error(w, "Unknown symbol", http.StatusBadRequest)
		return
	}

	// Validate and execute order
	cost := execPrice * quantity
	if side == OrderSideBuy {
		// Check quote asset balance
		quoteBal := s.balances[symbolInfo.QuoteAsset]
		if quoteBal == nil || quoteBal.Free < cost {
			http.Error(w, "Insufficient balance", http.StatusBadRequest)
			return
		}

		// Deduct from quote asset
		quoteBal.Free -= cost

		// Add to base asset
		baseBal, ok := s.balances[symbolInfo.BaseAsset]
		if !ok {
			baseBal = &Balance{Asset: symbolInfo.BaseAsset, Free: 0, Locked: 0}
			s.balances[symbolInfo.BaseAsset] = baseBal
		}
		baseBal.Free += quantity
	} else {
		// Sell - check base asset balance
		baseBal := s.balances[symbolInfo.BaseAsset]
		if baseBal == nil || baseBal.Free < quantity {
			http.Error(w, "Insufficient balance", http.StatusBadRequest)
			return
		}

		// Deduct from base asset
		baseBal.Free -= quantity

		// Add to quote asset
		quoteBal, ok := s.balances[symbolInfo.QuoteAsset]
		if !ok {
			quoteBal = &Balance{Asset: symbolInfo.QuoteAsset, Free: 0, Locked: 0}
			s.balances[symbolInfo.QuoteAsset] = quoteBal
		}
		quoteBal.Free += cost
	}

	// Create order (immediately filled for market orders, or stored for limit orders)
	s.orderIDSeq++
	order := &Order{
		OrderID:      s.orderIDSeq,
		Symbol:       symbol,
		Side:         side,
		Type:         orderType,
		Quantity:     quantity,
		Price:        price,
		Status:       OrderStatusFilled,
		TimeInForce:  timeInForce,
		CreatedAt:    time.Now(),
		ExecutedQty:  quantity,
		CummulateQty: cost,
	}

	if orderType == OrderTypeLimit {
		order.Status = OrderStatusNew
		order.ExecutedQty = 0
		order.CummulateQty = 0
	}

	s.orders[order.OrderID] = order

	// Create trade for market orders
	if orderType == OrderTypeMarket {
		s.tradeIDSeq++
		trade := &Trade{
			ID:         s.tradeIDSeq,
			OrderID:    order.OrderID,
			Symbol:     symbol,
			Price:      execPrice,
			Quantity:   quantity,
			Commission: cost * 0.001, // 0.1% commission
			Time:       time.Now(),
			IsBuyer:    side == OrderSideBuy,
		}
		s.trades = append(s.trades, trade)
	}

	// Return response
	response := map[string]interface{}{
		"symbol":              symbol,
		"orderId":             order.OrderID,
		"orderListId":         -1,
		"clientOrderId":       uuid.New().String(),
		"transactTime":        time.Now().UnixMilli(),
		"price":               strconv.FormatFloat(price, 'f', 8, 64),
		"origQty":             strconv.FormatFloat(quantity, 'f', 8, 64),
		"executedQty":         strconv.FormatFloat(order.ExecutedQty, 'f', 8, 64),
		"cummulativeQuoteQty": strconv.FormatFloat(order.CummulateQty, 'f', 8, 64),
		"status":              string(order.Status),
		"timeInForce":         timeInForce,
		"type":                string(orderType),
		"side":                string(side),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCancelOrder handles DELETE /api/v3/order
func (s *MockBinanceServer) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	orderIDStr := r.URL.Query().Get("orderId")

	if symbol == "" || orderIDStr == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid orderId", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	order, ok := s.orders[orderID]
	if !ok || order.Symbol != symbol {
		http.Error(w, "Order not found", http.StatusBadRequest)
		return
	}

	if order.Status != OrderStatusNew && order.Status != OrderStatusPartiallyFilled {
		http.Error(w, "Order cannot be canceled", http.StatusBadRequest)
		return
	}

	order.Status = OrderStatusCanceled

	response := map[string]interface{}{
		"symbol":              symbol,
		"orderId":             order.OrderID,
		"origClientOrderId":   "",
		"clientOrderId":       uuid.New().String(),
		"price":               strconv.FormatFloat(order.Price, 'f', 8, 64),
		"origQty":             strconv.FormatFloat(order.Quantity, 'f', 8, 64),
		"executedQty":         strconv.FormatFloat(order.ExecutedQty, 'f', 8, 64),
		"cummulativeQuoteQty": strconv.FormatFloat(order.CummulateQty, 'f', 8, 64),
		"status":              string(order.Status),
		"timeInForce":         order.TimeInForce,
		"type":                string(order.Type),
		"side":                string(order.Side),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleOpenOrders handles GET /api/v3/openOrders
func (s *MockBinanceServer) handleOpenOrders(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var openOrders []map[string]interface{}
	for _, order := range s.orders {
		if order.Status == OrderStatusNew || order.Status == OrderStatusPartiallyFilled {
			openOrders = append(openOrders, map[string]interface{}{
				"symbol":              order.Symbol,
				"orderId":             order.OrderID,
				"orderListId":         -1,
				"clientOrderId":       "",
				"price":               strconv.FormatFloat(order.Price, 'f', 8, 64),
				"origQty":             strconv.FormatFloat(order.Quantity, 'f', 8, 64),
				"executedQty":         strconv.FormatFloat(order.ExecutedQty, 'f', 8, 64),
				"cummulativeQuoteQty": strconv.FormatFloat(order.CummulateQty, 'f', 8, 64),
				"status":              string(order.Status),
				"timeInForce":         order.TimeInForce,
				"type":                string(order.Type),
				"side":                string(order.Side),
				"time":                order.CreatedAt.UnixMilli(),
				"updateTime":          order.CreatedAt.UnixMilli(),
				"isWorking":           true,
				"origQuoteOrderQty":   "0.00000000",
			})
		}
	}

	if openOrders == nil {
		openOrders = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(openOrders)
}

// handleCancelAllOrders handles DELETE /api/v3/openOrders
func (s *MockBinanceServer) handleCancelAllOrders(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		http.Error(w, "Missing symbol parameter", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var canceledOrders []map[string]interface{}
	for _, order := range s.orders {
		if order.Symbol == symbol && (order.Status == OrderStatusNew || order.Status == OrderStatusPartiallyFilled) {
			order.Status = OrderStatusCanceled
			canceledOrders = append(canceledOrders, map[string]interface{}{
				"symbol":  order.Symbol,
				"orderId": order.OrderID,
				"status":  string(order.Status),
			})
		}
	}

	if canceledOrders == nil {
		canceledOrders = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(canceledOrders)
}

// handleMyTrades handles GET /api/v3/myTrades
func (s *MockBinanceServer) handleMyTrades(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	limitStr := r.URL.Query().Get("limit")
	startTimeStr := r.URL.Query().Get("startTime")
	endTimeStr := r.URL.Query().Get("endTime")

	if symbol == "" {
		http.Error(w, "Missing symbol parameter", http.StatusBadRequest)
		return
	}

	limit := 500
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	var startTime, endTime time.Time
	if startTimeStr != "" {
		if ms, err := strconv.ParseInt(startTimeStr, 10, 64); err == nil {
			startTime = time.UnixMilli(ms)
		}
	}
	if endTimeStr != "" {
		if ms, err := strconv.ParseInt(endTimeStr, 10, 64); err == nil {
			endTime = time.UnixMilli(ms)
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var trades []map[string]interface{}
	for _, trade := range s.trades {
		if trade.Symbol != symbol {
			continue
		}
		if !startTime.IsZero() && trade.Time.Before(startTime) {
			continue
		}
		if !endTime.IsZero() && trade.Time.After(endTime) {
			continue
		}

		trades = append(trades, map[string]interface{}{
			"symbol":          trade.Symbol,
			"id":              trade.ID,
			"orderId":         trade.OrderID,
			"price":           strconv.FormatFloat(trade.Price, 'f', 8, 64),
			"qty":             strconv.FormatFloat(trade.Quantity, 'f', 8, 64),
			"commission":      strconv.FormatFloat(trade.Commission, 'f', 8, 64),
			"commissionAsset": "USDT",
			"time":            trade.Time.UnixMilli(),
			"isBuyer":         trade.IsBuyer,
			"isMaker":         false,
			"isBestMatch":     true,
		})

		if len(trades) >= limit {
			break
		}
	}

	if trades == nil {
		trades = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trades)
}

// handleTradeFee handles GET /sapi/v1/asset/tradeFee
func (s *MockBinanceServer) handleTradeFee(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")

	s.mu.RLock()
	defer s.mu.RUnlock()

	var fees []map[string]interface{}
	if symbol != "" {
		if fee, ok := s.tradeFees[symbol]; ok {
			fees = append(fees, map[string]interface{}{
				"symbol":          fee.Symbol,
				"makerCommission": strconv.FormatFloat(fee.MakerCommission, 'f', 4, 64),
				"takerCommission": strconv.FormatFloat(fee.TakerCommission, 'f', 4, 64),
			})
		}
	} else {
		for _, fee := range s.tradeFees {
			fees = append(fees, map[string]interface{}{
				"symbol":          fee.Symbol,
				"makerCommission": strconv.FormatFloat(fee.MakerCommission, 'f', 4, 64),
				"takerCommission": strconv.FormatFloat(fee.TakerCommission, 'f', 4, 64),
			})
		}
	}

	if fees == nil {
		fees = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fees)
}

// WebSocket Handler

// handleWebSocket handles WebSocket connections for kline streaming.
func (s *MockBinanceServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	symbolInterval := vars["symbol"] + "@kline_" + vars["interval"]
	parts := strings.Split(symbolInterval, "@kline_")
	if len(parts) != 2 {
		http.Error(w, "Invalid WebSocket path", http.StatusBadRequest)
		return
	}
	symbol := strings.ToUpper(parts[0])
	interval := parts[1]

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	s.wsMu.Lock()
	s.wsConnections[conn] = true
	s.wsMu.Unlock()

	defer func() {
		s.wsMu.Lock()
		delete(s.wsConnections, conn)
		s.wsMu.Unlock()
		conn.Close()
	}()

	// Start streaming klines
	s.streamKlines(conn, symbol, interval)
}

// streamKlines streams kline data to a WebSocket connection.
func (s *MockBinanceServer) streamKlines(conn *websocket.Conn, symbol, interval string) {
	intervalDuration := parseInterval(interval)
	if intervalDuration == 0 {
		intervalDuration = time.Minute
	}

	ticker := time.NewTicker(s.streamInterval)
	defer ticker.Stop()

	s.mu.RLock()
	initialPrice := s.currentPrices[symbol]
	if initialPrice == 0 {
		initialPrice = 100.0
	}
	s.mu.RUnlock()

	currentPrice := initialPrice
	klineStartTime := time.Now().Truncate(intervalDuration)
	klineOpen := currentPrice
	klineHigh := currentPrice
	klineLow := currentPrice
	klineVolume := 0.0

	pattern := backtestTesthelper.PatternVolatile
	if s.dataGenerator != nil {
		pattern = s.dataGenerator.Pattern
	}

	// Create a generator for price updates
	config := backtestTesthelper.MockDataConfig{
		Symbol:             symbol,
		StartTime:          time.Now(),
		EndTime:            time.Time{},
		Interval:           s.streamInterval,
		NumDataPoints:      1000,
		Pattern:            pattern,
		InitialPrice:       initialPrice,
		MaxDrawdownPercent: 10.0,
		VolatilityPercent:  2.0,
		TrendStrength:      0.01,
		Seed:               time.Now().UnixNano(),
	}

	generator := backtestTesthelper.NewMockDataGenerator(config)
	data, err := generator.Generate()
	if err != nil {
		return
	}

	dataIndex := 0
	for {
		select {
		case <-s.stopStreaming:
			return
		case <-ticker.C:
			// Get next price from generated data
			if dataIndex < len(data) {
				currentPrice = data[dataIndex].Close
				dataIndex++
			}

			// Update kline stats
			if currentPrice > klineHigh {
				klineHigh = currentPrice
			}
			if currentPrice < klineLow {
				klineLow = currentPrice
			}
			klineVolume += data[min(dataIndex, len(data)-1)].Volume / float64(intervalDuration/s.streamInterval)

			// Check if kline is complete
			now := time.Now()
			isFinal := now.Sub(klineStartTime) >= intervalDuration

			// Send kline event
			event := map[string]interface{}{
				"e": "kline",
				"E": now.UnixMilli(),
				"s": symbol,
				"k": map[string]interface{}{
					"t": klineStartTime.UnixMilli(),
					"T": klineStartTime.Add(intervalDuration).UnixMilli() - 1,
					"s": symbol,
					"i": interval,
					"o": strconv.FormatFloat(klineOpen, 'f', 8, 64),
					"c": strconv.FormatFloat(currentPrice, 'f', 8, 64),
					"h": strconv.FormatFloat(klineHigh, 'f', 8, 64),
					"l": strconv.FormatFloat(klineLow, 'f', 8, 64),
					"v": strconv.FormatFloat(klineVolume, 'f', 8, 64),
					"x": isFinal,
				},
			}

			if err := conn.WriteJSON(event); err != nil {
				return
			}

			// Update current price in server state
			s.SetPrice(symbol, currentPrice)

			// Reset for new kline if complete
			if isFinal {
				klineStartTime = now.Truncate(intervalDuration)
				klineOpen = currentPrice
				klineHigh = currentPrice
				klineLow = currentPrice
				klineVolume = 0
			}
		}
	}
}

// parseInterval parses a Binance interval string to a duration.
func parseInterval(interval string) time.Duration {
	if len(interval) < 2 {
		return 0
	}

	numStr := interval[:len(interval)-1]
	unit := interval[len(interval)-1:]

	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0
	}

	switch unit {
	case "s":
		return time.Duration(num) * time.Second
	case "m":
		return time.Duration(num) * time.Minute
	case "h":
		return time.Duration(num) * time.Hour
	case "d":
		return time.Duration(num) * 24 * time.Hour
	case "w":
		return time.Duration(num) * 7 * 24 * time.Hour
	case "M":
		return time.Duration(num) * 30 * 24 * time.Hour
	default:
		return 0
	}
}

// GenerateMarketData generates market data using the configured generator.
func (s *MockBinanceServer) GenerateMarketData(symbol string, numPoints int) ([]types.MarketData, error) {
	s.mu.RLock()
	initialPrice := s.currentPrices[symbol]
	if initialPrice == 0 {
		initialPrice = 100.0
	}
	pattern := backtestTesthelper.PatternVolatile
	if s.dataGenerator != nil {
		pattern = s.dataGenerator.Pattern
	}
	s.mu.RUnlock()

	config := backtestTesthelper.MockDataConfig{
		Symbol:             symbol,
		StartTime:          time.Now(),
		EndTime:            time.Time{},
		Interval:           time.Minute,
		NumDataPoints:      numPoints,
		Pattern:            pattern,
		InitialPrice:       initialPrice,
		MaxDrawdownPercent: 10.0,
		VolatilityPercent:  2.0,
		TrendStrength:      0.01,
		Seed:               time.Now().UnixNano(),
	}

	generator := backtestTesthelper.NewMockDataGenerator(config)
	return generator.Generate()
}
