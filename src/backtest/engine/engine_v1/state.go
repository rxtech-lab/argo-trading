package engine

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/alifiroozi80/duckdb"
	"github.com/shopspring/decimal"
	"github.com/sirily11/argo-trading-go/src/logger"
	"github.com/sirily11/argo-trading-go/src/strategy"
	"github.com/sirily11/argo-trading-go/src/types"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// OrderModel represents an order in the database
type OrderModel struct {
	OrderID      int64     `gorm:"column:order_id;primaryKey;autoIncrement"`
	Symbol       string    `gorm:"column:symbol"`
	OrderType    string    `gorm:"column:order_type"`
	Quantity     float64   `gorm:"column:quantity"`
	Price        float64   `gorm:"column:price"`
	Timestamp    time.Time `gorm:"column:timestamp"`
	IsCompleted  bool      `gorm:"column:is_completed"`
	Reason       string    `gorm:"column:reason"`
	Message      string    `gorm:"column:message"`
	StrategyName string    `gorm:"column:strategy_name"`
}

// TableName sets the table name for OrderModel
func (OrderModel) TableName() string {
	return "orders"
}

// ToOrder converts OrderModel to types.Order
func (o OrderModel) ToOrder() types.Order {
	return types.Order{
		OrderID:      fmt.Sprintf("%d", o.OrderID),
		Symbol:       o.Symbol,
		Side:         types.PurchaseType(o.OrderType),
		Quantity:     o.Quantity,
		Price:        o.Price,
		Timestamp:    o.Timestamp,
		IsCompleted:  o.IsCompleted,
		Reason:       types.Reason{Reason: o.Reason, Message: o.Message},
		StrategyName: o.StrategyName,
	}
}

// FromOrder creates OrderModel from types.Order
func OrderModelFromOrder(order types.Order) OrderModel {
	return OrderModel{
		Symbol:       order.Symbol,
		OrderType:    string(order.Side),
		Quantity:     order.Quantity,
		Price:        order.Price,
		Timestamp:    order.Timestamp,
		IsCompleted:  order.IsCompleted,
		Reason:       order.Reason.Reason,
		Message:      order.Reason.Message,
		StrategyName: order.StrategyName,
	}
}

// TradeModel represents a trade in the database
type TradeModel struct {
	OrderID       int64     `gorm:"column:order_id;primaryKey"`
	Symbol        string    `gorm:"column:symbol"`
	OrderType     string    `gorm:"column:order_type"`
	Quantity      float64   `gorm:"column:quantity"`
	Price         float64   `gorm:"column:price"`
	Timestamp     time.Time `gorm:"column:timestamp"`
	IsCompleted   bool      `gorm:"column:is_completed"`
	Reason        string    `gorm:"column:reason"`
	Message       string    `gorm:"column:message"`
	StrategyName  string    `gorm:"column:strategy_name"`
	ExecutedAt    time.Time `gorm:"column:executed_at"`
	ExecutedQty   float64   `gorm:"column:executed_qty"`
	ExecutedPrice float64   `gorm:"column:executed_price"`
	Commission    float64   `gorm:"column:commission"`
	PnL           float64   `gorm:"column:pnl"`
}

// TableName sets the table name for TradeModel
func (TradeModel) TableName() string {
	return "trades"
}

// ToTrade converts TradeModel to types.Trade
func (t TradeModel) ToTrade() types.Trade {
	return types.Trade{
		Order: types.Order{
			OrderID:      fmt.Sprintf("%d", t.OrderID),
			Symbol:       t.Symbol,
			Side:         types.PurchaseType(t.OrderType),
			Quantity:     t.Quantity,
			Price:        t.Price,
			Timestamp:    t.Timestamp,
			IsCompleted:  t.IsCompleted,
			Reason:       types.Reason{Reason: t.Reason, Message: t.Message},
			StrategyName: t.StrategyName,
			Fee:          t.Commission,
		},
		ExecutedAt:    t.ExecutedAt,
		ExecutedQty:   t.ExecutedQty,
		ExecutedPrice: t.ExecutedPrice,
		Fee:           t.Commission,
		PnL:           t.PnL,
	}
}

// FromTrade creates TradeModel from types.Trade
func TradeModelFromTrade(trade types.Trade, orderID int64) TradeModel {
	return TradeModel{
		OrderID:       orderID,
		Symbol:        trade.Order.Symbol,
		OrderType:     string(trade.Order.Side),
		Quantity:      trade.Order.Quantity,
		Price:         trade.Order.Price,
		Timestamp:     trade.Order.Timestamp,
		IsCompleted:   trade.Order.IsCompleted,
		Reason:        trade.Order.Reason.Reason,
		Message:       trade.Order.Reason.Message,
		StrategyName:  trade.Order.StrategyName,
		ExecutedAt:    trade.ExecutedAt,
		ExecutedQty:   trade.ExecutedQty,
		ExecutedPrice: trade.ExecutedPrice,
		Commission:    trade.Fee,
		PnL:           trade.PnL,
	}
}

type BacktestState struct {
	db     *gorm.DB
	logger *logger.Logger
}

func NewBacktestState(logger *logger.Logger) *BacktestState {
	// Initialize GORM with DuckDB in-memory database
	db, err := gorm.Open(duckdb.Open(":memory:"), &gorm.Config{})
	if err != nil {
		logger.Error("Failed to open database", zap.Error(err))
		return nil
	}

	return &BacktestState{
		logger: logger,
		db:     db,
	}
}

// Initialize creates the necessary tables for tracking trades and positions
func (b *BacktestState) Initialize() error {
	// Create tables using raw SQL since GORM AutoMigrate doesn't work well with DuckDB
	sqlDB, err := b.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB: %w", err)
	}

	// Create sequence for order IDs
	_, err = sqlDB.Exec(`CREATE SEQUENCE IF NOT EXISTS order_id_seq`)
	if err != nil {
		return fmt.Errorf("failed to create sequence: %w", err)
	}

	// Create orders table with sequence-based order_id
	_, err = sqlDB.Exec(`
		CREATE TABLE IF NOT EXISTS orders (
			order_id INTEGER PRIMARY KEY DEFAULT nextval('order_id_seq'),
			symbol TEXT,
			order_type TEXT,
			quantity DOUBLE,
			price DOUBLE,
			timestamp TIMESTAMP,
			is_completed BOOLEAN,
			reason TEXT,
			message TEXT,
			strategy_name TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create orders table: %w", err)
	}

	// Create trades table
	_, err = sqlDB.Exec(`
		CREATE TABLE IF NOT EXISTS trades (
			order_id INTEGER PRIMARY KEY,
			symbol TEXT,
			order_type TEXT,
			quantity DOUBLE,
			price DOUBLE,
			timestamp TIMESTAMP,
			is_completed BOOLEAN,
			reason TEXT,
			message TEXT,
			strategy_name TEXT,
			executed_at TIMESTAMP,
			executed_qty DOUBLE,
			executed_price DOUBLE,
			commission DOUBLE,
			pnl DOUBLE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create trades table: %w", err)
	}

	return nil
}

// UpdateResult contains the results of processing an order
type UpdateResult struct {
	Order         types.Order
	Trade         types.Trade
	IsNewPosition bool
}

// Update processes orders and updates trades
func (b *BacktestState) Update(orders []types.Order) ([]UpdateResult, error) {
	results := make([]UpdateResult, 0, len(orders))

	for _, order := range orders {
		// Use raw SQL approach since GORM has issues with the sequence
		sqlDB, err := b.db.DB()
		if err != nil {
			return nil, fmt.Errorf("failed to get SQL DB: %w", err)
		}

		// Start transaction
		tx, err := sqlDB.Begin()
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}

		// Insert order and get the auto-generated order_id
		var orderID int64
		err = tx.QueryRow(`
			INSERT INTO orders (
				symbol, order_type, quantity, price, timestamp,
				is_completed, reason, message, strategy_name
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			RETURNING order_id
		`,
			order.Symbol, order.Side, order.Quantity, order.Price,
			order.Timestamp, order.IsCompleted, order.Reason.Reason, order.Reason.Message,
			order.StrategyName,
		).Scan(&orderID)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to insert order: %w", err)
		}

		// Get current position
		currentPosition, err := b.GetPosition(order.Symbol)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to get position: %w", err)
		}

		// Calculate PnL if closing position
		var pnl float64 = 0
		if order.Side == types.PurchaseTypeSell && currentPosition.Quantity > 0 {
			// For sell orders, calculate PnL using decimal arithmetic
			avgEntryPrice := currentPosition.GetAverageEntryPrice()
			entryDec := decimal.NewFromFloat(order.Quantity).Mul(decimal.NewFromFloat(avgEntryPrice))
			exitDec := decimal.NewFromFloat(order.Quantity).Mul(decimal.NewFromFloat(order.Price)).Sub(decimal.NewFromFloat(order.Fee))
			resultDec := exitDec.Sub(entryDec)
			pnl, _ = resultDec.Float64()
		}

		// Create trade record
		trade := types.Trade{
			Order: types.Order{
				OrderID:      fmt.Sprintf("%d", orderID),
				Symbol:       order.Symbol,
				Side:         order.Side,
				Quantity:     order.Quantity,
				Price:        order.Price,
				Timestamp:    order.Timestamp,
				IsCompleted:  order.IsCompleted,
				Reason:       order.Reason,
				StrategyName: order.StrategyName,
				Fee:          order.Fee,
			},
			ExecutedAt:    order.Timestamp,
			ExecutedQty:   order.Quantity,
			ExecutedPrice: order.Price,
			Fee:           order.Fee,
			PnL:           pnl,
		}

		// Insert trade
		_, err = tx.Exec(`
			INSERT INTO trades (
				order_id, symbol, order_type, quantity, price, timestamp,
				is_completed, reason, message, strategy_name,
				executed_at, executed_qty, executed_price, commission, pnl
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			orderID, trade.Order.Symbol, trade.Order.Side, trade.Order.Quantity, trade.Order.Price,
			trade.Order.Timestamp, trade.Order.IsCompleted, trade.Order.Reason.Reason, trade.Order.Reason.Message,
			order.StrategyName, trade.ExecutedAt, trade.ExecutedQty, trade.ExecutedPrice,
			trade.Fee, trade.PnL,
		)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to insert trade: %w", err)
		}

		// Determine if this is a new position
		isNewPosition := order.Side == types.PurchaseTypeBuy && currentPosition.Quantity == 0

		// Commit transaction
		err = tx.Commit()
		if err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}

		// Add result to return list
		results = append(results, UpdateResult{
			Order:         order,
			Trade:         trade,
			IsNewPosition: isNewPosition,
		})
	}

	return results, nil
}

// GetPosition retrieves the current position for a symbol by calculating from trades
func (b *BacktestState) GetPosition(symbol string) (types.Position, error) {
	sqlDB, err := b.db.DB()
	if err != nil {
		return types.Position{}, fmt.Errorf("failed to get SQL DB: %w", err)
	}

	// Calculate position information from trades using raw SQL (complex aggregations)
	var position types.Position
	err = sqlDB.QueryRow(`
		WITH buy_trades AS (
			SELECT 
				SUM(executed_qty) as total_in_qty,
				SUM(commission) as total_in_fee,
				SUM(executed_qty * executed_price) as total_in_amount,
				MIN(executed_at) as first_trade_time
			FROM trades 
			WHERE symbol = ? AND order_type = ?
		),
		sell_trades AS (
			SELECT 
				SUM(executed_qty) as total_out_qty,
				SUM(commission) as total_out_fee,
				SUM(executed_qty * executed_price) as total_out_amount
			FROM trades 
			WHERE symbol = ? AND order_type = ?
		)
		SELECT 
			? as symbol,
			COALESCE(b.total_in_qty, 0) - COALESCE(s.total_out_qty, 0) as quantity,
			COALESCE(b.total_in_qty, 0) as total_in_quantity,
			COALESCE(s.total_out_qty, 0) as total_out_quantity,
			COALESCE(b.total_in_amount, 0) as total_in_amount,
			COALESCE(s.total_out_amount, 0) as total_out_amount,
			COALESCE(b.total_in_fee, 0) as total_in_fee,
			COALESCE(s.total_out_fee, 0) as total_out_fee,
			COALESCE(b.first_trade_time, CURRENT_TIMESTAMP) as open_timestamp,
			MAX(t.strategy_name) as strategy_name
		FROM trades t
		LEFT JOIN buy_trades b ON 1=1
		LEFT JOIN sell_trades s ON 1=1
		WHERE t.symbol = ?
		GROUP BY b.total_in_qty, s.total_out_qty, b.total_in_amount, s.total_out_amount, b.total_in_fee, s.total_out_fee, b.first_trade_time
	`, symbol, types.PurchaseTypeBuy, symbol, types.PurchaseTypeSell, symbol, symbol).Scan(
		&position.Symbol,
		&position.Quantity,
		&position.TotalInQuantity,
		&position.TotalOutQuantity,
		&position.TotalInAmount,
		&position.TotalOutAmount,
		&position.TotalInFee,
		&position.TotalOutFee,
		&position.OpenTimestamp,
		&position.StrategyName,
	)

	if err != nil {
		// If no records found, return empty position
		return types.Position{
			Symbol:           symbol,
			Quantity:         0,
			TotalInQuantity:  0,
			TotalOutQuantity: 0,
			TotalInAmount:    0,
			TotalOutAmount:   0,
			TotalInFee:       0,
			TotalOutFee:      0,
			OpenTimestamp:    time.Time{},
			StrategyName:     "",
		}, nil
	}

	return position, nil
}

// GetAllTrades returns all trades from the database
func (b *BacktestState) GetAllTrades() ([]types.Trade, error) {
	sqlDB, err := b.db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get SQL DB: %w", err)
	}

	rows, err := sqlDB.Query(`
		SELECT 
			order_id, symbol, order_type, quantity, price, timestamp,
			is_completed, reason, message, strategy_name,
			executed_at, executed_qty, executed_price, commission, pnl
		FROM trades
		ORDER BY executed_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query trades: %w", err)
	}
	defer rows.Close()

	var trades []types.Trade
	for rows.Next() {
		var trade types.Trade
		var orderIDInt int64
		err := rows.Scan(
			&orderIDInt,
			&trade.Order.Symbol,
			&trade.Order.Side,
			&trade.Order.Quantity,
			&trade.Order.Price,
			&trade.Order.Timestamp,
			&trade.Order.IsCompleted,
			&trade.Order.Reason.Reason,
			&trade.Order.Reason.Message,
			&trade.Order.StrategyName,
			&trade.ExecutedAt,
			&trade.ExecutedQty,
			&trade.ExecutedPrice,
			&trade.Fee,
			&trade.PnL,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trade: %w", err)
		}
		trade.Order.OrderID = fmt.Sprintf("%d", orderIDInt)
		trades = append(trades, trade)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating trades: %w", err)
	}

	return trades, nil
}

// Cleanup resets the database state
func (b *BacktestState) Cleanup() error {
	sqlDB, err := b.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB: %w", err)
	}

	// Drop and recreate tables using raw SQL
	_, err = sqlDB.Exec(`
		DROP TABLE IF EXISTS trades;
		DROP TABLE IF EXISTS orders;
		DROP SEQUENCE IF EXISTS order_id_seq;
	`)
	if err != nil {
		return fmt.Errorf("failed to cleanup tables: %w", err)
	}

	// Reinitialize
	return b.Initialize()
}

// Write saves the backtest results to Parquet files in the specified directory
func (b *BacktestState) Write(path string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	sqlDB, err := b.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB: %w", err)
	}

	// Export trades to Parquet
	tradesPath := filepath.Join(path, "trades.parquet")
	_, err = sqlDB.Exec(fmt.Sprintf(`COPY trades TO '%s' (FORMAT PARQUET)`, tradesPath))
	if err != nil {
		return fmt.Errorf("failed to export trades to Parquet: %w", err)
	}

	// Export orders to Parquet
	ordersPath := filepath.Join(path, "orders.parquet")
	_, err = sqlDB.Exec(fmt.Sprintf(`COPY orders TO '%s' (FORMAT PARQUET)`, ordersPath))
	if err != nil {
		return fmt.Errorf("failed to export orders to Parquet: %w", err)
	}

	b.logger.Info("Successfully exported backtest results to Parquet files",
		zap.String("trades", tradesPath),
		zap.String("orders", ordersPath),
	)
	return nil
}

// calculateTradeResult calculates the trade result statistics for a symbol
func (b *BacktestState) calculateTradeResult(symbol string) (types.TradeResult, error) {
	sqlDB, err := b.db.DB()
	if err != nil {
		return types.TradeResult{}, fmt.Errorf("failed to get SQL DB: %w", err)
	}

	var result types.TradeResult
	err = sqlDB.QueryRow(`
		WITH trade_stats AS (
			SELECT 
				COUNT(*) as total_trades,
				SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END) as winning_trades,
				SUM(CASE WHEN pnl < 0 THEN 1 ELSE 0 END) as losing_trades,
				MIN(pnl) as min_pnl,
				MAX(pnl) as max_pnl
			FROM trades
			WHERE symbol = ?
		)
		SELECT 
			total_trades,
			winning_trades,
			losing_trades,
			CASE WHEN total_trades > 0 THEN CAST(winning_trades AS DOUBLE) / total_trades ELSE 0 END as win_rate,
			CASE WHEN min_pnl < 0 THEN ABS(min_pnl) ELSE 0 END as max_drawdown
		FROM trade_stats
	`, symbol).Scan(
		&result.NumberOfTrades,
		&result.NumberOfWinningTrades,
		&result.NumberOfLosingTrades,
		&result.WinRate,
		&result.MaxDrawdown,
	)
	if err != nil {
		return types.TradeResult{}, fmt.Errorf("failed to calculate trade result: %w", err)
	}

	return result, nil
}

// calculateTradeHoldingTime calculates the holding time statistics for a symbol
func (b *BacktestState) calculateTradeHoldingTime(symbol string) (types.TradeHoldingTime, error) {
	sqlDB, err := b.db.DB()
	if err != nil {
		return types.TradeHoldingTime{}, fmt.Errorf("failed to get SQL DB: %w", err)
	}

	var holdingTime types.TradeHoldingTime
	var avgDuration float64
	err = sqlDB.QueryRow(`
		WITH buy_trades AS (
			SELECT executed_at
			FROM trades
			WHERE symbol = ? AND order_type = ?
		),
		sell_trades AS (
			SELECT executed_at
			FROM trades
			WHERE symbol = ? AND order_type = ?
		),
		trade_durations AS (
			SELECT 
				EXTRACT(EPOCH FROM (s.executed_at - b.executed_at)) / 3600 as duration
			FROM buy_trades b
			JOIN sell_trades s ON s.executed_at > b.executed_at
		)
		SELECT 
			COALESCE(MIN(duration), 0) as min_duration,
			COALESCE(MAX(duration), 0) as max_duration,
			COALESCE(AVG(duration), 0) as avg_duration
		FROM trade_durations
	`, symbol, types.PurchaseTypeBuy, symbol, types.PurchaseTypeSell).Scan(
		&holdingTime.Min,
		&holdingTime.Max,
		&avgDuration,
	)
	if err != nil {
		return types.TradeHoldingTime{}, fmt.Errorf("failed to calculate holding time: %w", err)
	}
	holdingTime.Avg = int(math.Round(avgDuration))
	return holdingTime, nil
}

// calculateTotalFees calculates the total fees for a symbol
func (b *BacktestState) calculateTotalFees(symbol string) (float64, error) {
	sqlDB, err := b.db.DB()
	if err != nil {
		return 0, fmt.Errorf("failed to get SQL DB: %w", err)
	}

	var totalFees float64
	err = sqlDB.QueryRow(`
		SELECT COALESCE(SUM(commission), 0)
		FROM trades
		WHERE symbol = ?
	`, symbol).Scan(&totalFees)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate total fees: %w", err)
	}
	return totalFees, nil
}

// GetStats returns the statistics of the backtest
func (b *BacktestState) GetStats(ctx strategy.StrategyContext) ([]types.TradeStats, error) {
	sqlDB, err := b.db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get SQL DB: %w", err)
	}

	// Get all unique symbols that have trades
	rows, err := sqlDB.Query(`
		SELECT DISTINCT symbol
		FROM trades
		ORDER BY symbol
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get unique symbols: %w", err)
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, fmt.Errorf("failed to scan symbol: %w", err)
		}
		symbols = append(symbols, symbol)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating symbols: %w", err)
	}

	var stats []types.TradeStats
	for _, symbol := range symbols {
		// Calculate trade result
		tradeResult, err := b.calculateTradeResult(symbol)
		if err != nil {
			return nil, err
		}

		// Calculate holding time
		holdingTime, err := b.calculateTradeHoldingTime(symbol)
		if err != nil {
			return nil, err
		}

		// Calculate total fees
		totalFees, err := b.calculateTotalFees(symbol)
		if err != nil {
			return nil, err
		}

		// Get current position
		position, err := b.GetPosition(symbol)
		if err != nil {
			return nil, err
		}

		// Get last market data for unrealized PnL calculation
		lastData, err := ctx.DataSource.ReadLastData(symbol)
		if err != nil {
			return nil, fmt.Errorf("failed to get last market data for %s: %w", symbol, err)
		}

		tradePnl := types.TradePnl{}

		// Calculate unrealized PnL if there's an open position
		if position.Quantity > 0 {
			entryDec := decimal.NewFromFloat(position.Quantity).Mul(decimal.NewFromFloat(position.GetAverageEntryPrice()))
			exitDec := decimal.NewFromFloat(position.Quantity).Mul(decimal.NewFromFloat(lastData.Close))
			unrealizedPnL, _ := exitDec.Sub(entryDec).Float64()
			realizedPnl := position.GetTotalPnL()
			tradePnl.TotalPnL = realizedPnl + unrealizedPnL
			tradePnl.RealizedPnL = realizedPnl
			tradePnl.UnrealizedPnL = unrealizedPnL
		}

		// Calculate maximum loss and maximum profit
		var maxLoss, maxProfit float64
		err = sqlDB.QueryRow(`
			SELECT 
				COALESCE(MIN(pnl), 0) as max_loss,
				COALESCE(MAX(pnl), 0) as max_profit
			FROM trades
			WHERE symbol = ?
		`, symbol).Scan(&maxLoss, &maxProfit)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate max loss/profit: %w", err)
		}
		tradePnl.MaximumLoss = maxLoss
		tradePnl.MaximumProfit = maxProfit

		stats = append(stats, types.TradeStats{
			Symbol:           symbol,
			TradeResult:      tradeResult,
			TotalFees:        totalFees,
			TradeHoldingTime: holdingTime,
			TradePnl:         tradePnl,
		})
	}

	return stats, nil
}
