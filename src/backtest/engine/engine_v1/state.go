package engine

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/sirily11/argo-trading-go/src/logger"
	"github.com/sirily11/argo-trading-go/src/types"
	"go.uber.org/zap"
)

type BacktestState struct {
	db     *sql.DB
	logger *logger.Logger
}

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

	// Create positions table
	_, err = b.db.Exec(`
		CREATE TABLE IF NOT EXISTS positions (
			strategy_name TEXT,
			symbol TEXT,
			quantity DOUBLE,
			average_price DOUBLE,
			open_timestamp TIMESTAMP,
			PRIMARY KEY (symbol)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create positions table: %w", err)
	}

	return nil
}

// UpdateResult contains the results of processing an order
type UpdateResult struct {
	Order         types.Order
	Trade         types.Trade
	Position      types.Position
	IsNewPosition bool
}

// Update processes orders and updates trades and positions
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
		var currentPosition types.Position
		err = tx.QueryRow(`
			SELECT symbol, quantity, average_price, open_timestamp
			FROM positions 
			WHERE symbol = ?
		`, order.Symbol).Scan(&currentPosition.Symbol, &currentPosition.Quantity, &currentPosition.AveragePrice, &currentPosition.OpenTimestamp)

		if err != nil && err != sql.ErrNoRows {
			tx.Rollback()
			return nil, fmt.Errorf("failed to query position: %w", err)
		}

		// Calculate PnL if closing position
		var pnl float64
		if order.OrderType == types.OrderTypeSell && currentPosition.Quantity > 0 {
			pnl = (order.Price - currentPosition.AveragePrice) * order.Quantity
		} else if order.OrderType == types.OrderTypeBuy && currentPosition.Quantity < 0 {
			pnl = (currentPosition.AveragePrice - order.Price) * order.Quantity
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
			},
			ExecutedAt:    order.Timestamp,
			ExecutedQty:   order.Quantity,
			ExecutedPrice: order.Price,
			Commission:    0.0,
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
			trade.Commission, trade.PnL,
		)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to insert trade: %w", err)
		}

		// Update position and track if it's a new position
		isNewPosition := false
		var finalPosition types.Position

		if order.OrderType == types.OrderTypeBuy {
			if currentPosition.Quantity == 0 {
				// New position
				isNewPosition = true
				finalPosition = types.Position{
					Symbol:        order.Symbol,
					Quantity:      order.Quantity,
					AveragePrice:  order.Price,
					OpenTimestamp: order.Timestamp,
					StrategyName:  order.StrategyName,
				}
				_, err = tx.Exec(`
					INSERT INTO positions (symbol, quantity, average_price, open_timestamp, strategy_name)
					VALUES (?, ?, ?, ?, ?)
				`, finalPosition.Symbol, finalPosition.Quantity, finalPosition.AveragePrice, finalPosition.OpenTimestamp, finalPosition.StrategyName)
			} else {
				// Update existing position
				newQuantity := currentPosition.Quantity + order.Quantity
				newAvgPrice := (currentPosition.AveragePrice*currentPosition.Quantity + order.Price*order.Quantity) / newQuantity
				finalPosition = types.Position{
					Symbol:        order.Symbol,
					Quantity:      newQuantity,
					AveragePrice:  newAvgPrice,
					OpenTimestamp: currentPosition.OpenTimestamp,
					StrategyName:  order.StrategyName,
				}
				_, err = tx.Exec(`
					UPDATE positions 
					SET quantity = ?, average_price = ?, strategy_name = ?
					WHERE symbol = ?
				`, finalPosition.Quantity, finalPosition.AveragePrice, finalPosition.StrategyName, finalPosition.Symbol)
			}
		} else if order.OrderType == types.OrderTypeSell {
			if currentPosition.Quantity == order.Quantity {
				// Close position
				_, err = tx.Exec(`DELETE FROM positions WHERE symbol = ?`, order.Symbol)
				finalPosition = types.Position{} // Empty position indicates closed
			} else {
				// Update position
				newQuantity := currentPosition.Quantity - order.Quantity
				finalPosition = types.Position{
					Symbol:        order.Symbol,
					Quantity:      newQuantity,
					AveragePrice:  currentPosition.AveragePrice,
					OpenTimestamp: currentPosition.OpenTimestamp,
					StrategyName:  order.StrategyName,
				}
				_, err = tx.Exec(`
					UPDATE positions 
					SET quantity = ?, strategy_name = ?
					WHERE symbol = ?
				`, finalPosition.Quantity, finalPosition.StrategyName, finalPosition.Symbol)
			}
		}

		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to update position: %w", err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}

		// Add result
		results = append(results, UpdateResult{
			Order:         order,
			Trade:         trade,
			Position:      finalPosition,
			IsNewPosition: isNewPosition,
		})
	}

	return results, nil
}

// GetPosition returns the current position for a symbol
func (b *BacktestState) GetPosition(symbol string) (types.Position, error) {
	var position types.Position
	err := b.db.QueryRow(`
		SELECT symbol, quantity, average_price, open_timestamp 
		FROM positions 
		WHERE symbol = ?
	`, symbol).Scan(&position.Symbol, &position.Quantity, &position.AveragePrice, &position.OpenTimestamp)

	if err == sql.ErrNoRows {
		return types.Position{}, nil
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
			&trade.Commission,
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
		DROP TABLE IF EXISTS positions;
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

	// Export positions to Parquet
	positionsPath := filepath.Join(path, "positions.parquet")
	_, err = b.db.Exec(fmt.Sprintf(`COPY positions TO '%s' (FORMAT PARQUET)`, positionsPath))
	if err != nil {
		return fmt.Errorf("failed to export positions to Parquet: %w", err)
	}

	// Export orders to Parquet
	ordersPath := filepath.Join(path, "orders.parquet")
	_, err = b.db.Exec(fmt.Sprintf(`COPY orders TO '%s' (FORMAT PARQUET)`, ordersPath))
	if err != nil {
		return fmt.Errorf("failed to export orders to Parquet: %w", err)
	}

	b.logger.Info("Successfully exported backtest results to Parquet files",
		zap.String("trades", tradesPath),
		zap.String("positions", positionsPath),
		zap.String("orders", ordersPath),
	)
	return nil
}
