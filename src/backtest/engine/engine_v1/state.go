package engine

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/shopspring/decimal"
	"github.com/sirily11/argo-trading-go/src/logger"
	"github.com/sirily11/argo-trading-go/src/types"
	"go.uber.org/zap"
)

type BacktestState struct {
	db     *sql.DB
	logger *logger.Logger
}

// CalculatePNL calculates the profit/loss for a trade

func NewBacktestState(logger *logger.Logger) *BacktestState {
	db, err := sql.Open("duckdb", ":memory:")
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
	// Create sequence for order IDs
	_, err := b.db.Exec(`CREATE SEQUENCE IF NOT EXISTS order_id_seq`)
	if err != nil {
		return fmt.Errorf("failed to create sequence: %w", err)
	}

	// Create orders table with sequence-based order_id
	_, err = b.db.Exec(`
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
	_, err = b.db.Exec(`
		CREATE TABLE IF NOT EXISTS trades (
			order_id INTEGER,
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
		// Start transaction
		tx, err := b.db.Begin()
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
			order.Symbol, order.OrderType, order.Quantity, order.Price,
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
		if order.OrderType == types.OrderTypeSell && currentPosition.Quantity > 0 {
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
				OrderType:    order.OrderType,
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
			orderID, trade.Order.Symbol, trade.Order.OrderType, trade.Order.Quantity, trade.Order.Price,
			trade.Order.Timestamp, trade.Order.IsCompleted, trade.Order.Reason.Reason, trade.Order.Reason.Message,
			order.StrategyName, trade.ExecutedAt, trade.ExecutedQty, trade.ExecutedPrice,
			trade.Fee, trade.PnL,
		)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to insert trade: %w", err)
		}

		// Determine if this is a new position
		isNewPosition := order.OrderType == types.OrderTypeBuy && currentPosition.Quantity == 0

		// Commit transaction
		err = tx.Commit()
		if err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}

		// Add result
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
	// Calculate position information from trades
	var position types.Position
	err := b.db.QueryRow(`
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
	`, symbol, types.OrderTypeBuy, symbol, types.OrderTypeSell, symbol, symbol).Scan(
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

	if err == sql.ErrNoRows {
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

	if err != nil {
		return types.Position{}, fmt.Errorf("failed to query position: %w", err)
	}

	return position, nil
}

// GetAllTrades returns all trades from the database
func (b *BacktestState) GetAllTrades() ([]types.Trade, error) {
	rows, err := b.db.Query(`
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
		err := rows.Scan(
			&trade.Order.OrderID,
			&trade.Order.Symbol,
			&trade.Order.OrderType,
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
		trades = append(trades, trade)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating trades: %w", err)
	}

	return trades, nil
}

// Cleanup resets the database state
func (b *BacktestState) Cleanup() error {
	// Drop and recreate tables
	_, err := b.db.Exec(`
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

	// Export trades to Parquet
	tradesPath := filepath.Join(path, "trades.parquet")
	_, err := b.db.Exec(fmt.Sprintf(`COPY trades TO '%s' (FORMAT PARQUET)`, tradesPath))
	if err != nil {
		return fmt.Errorf("failed to export trades to Parquet: %w", err)
	}

	// Export orders to Parquet
	ordersPath := filepath.Join(path, "orders.parquet")
	_, err = b.db.Exec(fmt.Sprintf(`COPY orders TO '%s' (FORMAT PARQUET)`, ordersPath))
	if err != nil {
		return fmt.Errorf("failed to export orders to Parquet: %w", err)
	}

	b.logger.Info("Successfully exported backtest results to Parquet files",
		zap.String("trades", tradesPath),
		zap.String("orders", ordersPath),
	)
	return nil
}
