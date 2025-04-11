package engine

import (
	"database/sql"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/moznion/go-optional"
	"github.com/shopspring/decimal"
	"github.com/sirily11/argo-trading-go/internal/logger"
	"github.com/sirily11/argo-trading-go/internal/types"
	"github.com/sirily11/argo-trading-go/pkg/strategy"
	"go.uber.org/zap"
)

type BacktestState struct {
	db     *sql.DB
	logger *logger.Logger
	sq     squirrel.StatementBuilderType
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
		sq:     squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
	}
}

// Initialize creates the necessary tables for tracking trades and positions
func (b *BacktestState) Initialize() error {
	// Create sequence for order IDs
	_, err := b.db.Exec(`CREATE SEQUENCE IF NOT EXISTS order_id_seq`)
	if err != nil {
		return fmt.Errorf("failed to create sequence: %w", err)
	}

	// Create orders table with string-based order_id
	_, err = b.db.Exec(`
		CREATE TABLE IF NOT EXISTS orders (
			order_id TEXT PRIMARY KEY,
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
			order_id TEXT,
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
		orderID := uuid.New().String()
		// Start transaction
		tx, err := b.db.Begin()
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}

		// Insert order
		insertQuery := b.sq.
			Insert("orders").
			Columns(
				"order_id", "symbol", "order_type", "quantity", "price", "timestamp",
				"is_completed", "reason", "message", "strategy_name",
			).
			Values(
				orderID, order.Symbol, order.Side, order.Quantity, order.Price,
				order.Timestamp, order.IsCompleted, order.Reason.Reason, order.Reason.Message,
				order.StrategyName,
			).
			RunWith(tx)

		_, err = insertQuery.Exec()
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
				OrderID:      orderID,
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

		// Insert trade using Squirrel
		insertTradeQuery := b.sq.
			Insert("trades").
			Columns(
				"order_id", "symbol", "order_type", "quantity", "price", "timestamp",
				"is_completed", "reason", "message", "strategy_name",
				"executed_at", "executed_qty", "executed_price", "commission", "pnl",
			).
			Values(
				orderID, trade.Order.Symbol, trade.Order.Side, trade.Order.Quantity, trade.Order.Price,
				trade.Order.Timestamp, trade.Order.IsCompleted, trade.Order.Reason.Reason, trade.Order.Reason.Message,
				order.StrategyName, trade.ExecutedAt, trade.ExecutedQty, trade.ExecutedPrice,
				trade.Fee, trade.PnL,
			).
			RunWith(tx)

		_, err = insertTradeQuery.Exec()
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

		// Add result
		order.OrderID = orderID
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
	// Create a complex query with CTEs using raw SQL as Squirrel doesn't directly support this complex case
	query := `
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
	`

	args := []interface{}{
		symbol, types.PurchaseTypeBuy,
		symbol, types.PurchaseTypeSell,
		symbol,
		symbol,
	}

	var position types.Position
	err := b.db.QueryRow(query, args...).Scan(
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
	selectQuery := b.sq.
		Select(
			"order_id", "symbol", "order_type", "quantity", "price", "timestamp",
			"is_completed", "reason", "message", "strategy_name",
			"executed_at", "executed_qty", "executed_price", "commission", "pnl",
		).
		From("trades").
		OrderBy("executed_at ASC").
		RunWith(b.db)

	rows, err := selectQuery.Query()
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
		trades = append(trades, trade)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating trades: %w", err)
	}

	return trades, nil
}

// Cleanup resets the database state
func (b *BacktestState) Cleanup() error {
	// Use raw SQL for dropping tables - Squirrel doesn't have DROP syntax
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

	// Export trades to Parquet - using raw SQL as Squirrel doesn't support COPY
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

// calculateTradeResult calculates the trade result statistics for a symbol
func (b *BacktestState) calculateTradeResult(symbol string) (types.TradeResult, error) {
	// Using raw SQL for CTE query - Squirrel doesn't natively support CTEs well
	query := `
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
	`

	var result types.TradeResult
	err := b.db.QueryRow(query, symbol).Scan(
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
	// Using raw SQL for CTE query - Squirrel doesn't natively support this complex query
	query := `
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
	`

	var holdingTime types.TradeHoldingTime
	var avgDuration float64
	err := b.db.QueryRow(query, symbol, types.PurchaseTypeBuy, symbol, types.PurchaseTypeSell).Scan(
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
	// Using Squirrel for a simpler query
	query := b.sq.
		Select("SUM(commission)").
		From("trades").
		Where(squirrel.Eq{"symbol": symbol}).
		RunWith(b.db)

	var totalFees float64
	err := query.QueryRow().Scan(&totalFees)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate total fees: %w", err)
	}
	return totalFees, nil
}

// GetStats returns the statistics of the backtest
func (b *BacktestState) GetStats(ctx strategy.StrategyContext) ([]types.TradeStats, error) {
	// Get all unique symbols that have trades using Squirrel
	selectQuery := b.sq.
		Select("DISTINCT symbol").
		From("trades").
		OrderBy("symbol").
		RunWith(b.db)

	rows, err := selectQuery.Query()
	if err != nil {
		return nil, fmt.Errorf("failed to get unique symbols: %w", err)
	}
	defer rows.Close()

	var stats []types.TradeStats
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, fmt.Errorf("failed to scan symbol: %w", err)
		}

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

		// Calculate maximum loss and maximum profit using Squirrel
		maxLossProfit := b.sq.
			Select("COALESCE(MIN(pnl), 0) as max_loss", "COALESCE(MAX(pnl), 0) as max_profit").
			From("trades").
			Where(squirrel.Eq{"symbol": symbol}).
			RunWith(b.db)

		var maxLoss, maxProfit float64
		err = maxLossProfit.QueryRow().Scan(&maxLoss, &maxProfit)
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

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating symbols: %w", err)
	}

	return stats, nil
}

// GetOrderById returns an order by its id
func (b *BacktestState) GetOrderById(orderID string) (optional.Option[types.Order], error) {
	query := b.sq.
		Select("*").
		From("orders").
		Where(squirrel.Eq{"order_id": orderID}).
		RunWith(b.db)

	var order types.Order
	err := query.QueryRow().Scan(
		&order.OrderID,
		&order.Symbol,
		&order.Side,
		&order.Quantity,
		&order.Price,
		&order.Timestamp,
		&order.IsCompleted,
		&order.Reason.Reason,
		&order.Reason.Message,
		&order.StrategyName,
	)
	if err != nil {
		// check if error is no rows in result set
		if err == sql.ErrNoRows {
			return optional.None[types.Order](), nil
		}
		return optional.None[types.Order](), fmt.Errorf("failed to get order by id: %w", err)
	}
	return optional.Some(order), nil
}

// GetAllPositions returns all positions from the database by calculating from trades
func (b *BacktestState) GetAllPositions() ([]types.Position, error) {
	// Using raw SQL for CTE query - Squirrel doesn't natively support this complex case
	query := `
		WITH buy_trades AS (
			SELECT 
				symbol,
				SUM(executed_qty) as total_in_qty,
				SUM(commission) as total_in_fee,
				SUM(executed_qty * executed_price) as total_in_amount,
				MIN(executed_at) as first_trade_time,
				MAX(strategy_name) as strategy_name
			FROM trades 
			WHERE order_type = ?
			GROUP BY symbol
		),
		sell_trades AS (
			SELECT 
				symbol,
				SUM(executed_qty) as total_out_qty,
				SUM(commission) as total_out_fee,
				SUM(executed_qty * executed_price) as total_out_amount
			FROM trades 
			WHERE order_type = ?
			GROUP BY symbol
		)
		SELECT 
			COALESCE(b.symbol, s.symbol) as symbol,
			COALESCE(b.total_in_qty, 0) - COALESCE(s.total_out_qty, 0) as quantity,
			COALESCE(b.total_in_qty, 0) as total_in_quantity,
			COALESCE(s.total_out_qty, 0) as total_out_quantity,
			COALESCE(b.total_in_amount, 0) as total_in_amount,
			COALESCE(s.total_out_amount, 0) as total_out_amount,
			COALESCE(b.total_in_fee, 0) as total_in_fee,
			COALESCE(s.total_out_fee, 0) as total_out_fee,
			COALESCE(b.first_trade_time, CURRENT_TIMESTAMP) as open_timestamp,
			COALESCE(b.strategy_name, '') as strategy_name
		FROM buy_trades b
		FULL OUTER JOIN sell_trades s ON b.symbol = s.symbol
		WHERE COALESCE(b.total_in_qty, 0) - COALESCE(s.total_out_qty, 0) != 0
		ORDER BY symbol
	`

	rows, err := b.db.Query(query, types.PurchaseTypeBuy, types.PurchaseTypeSell)
	if err != nil {
		return nil, fmt.Errorf("failed to query positions: %w", err)
	}
	defer rows.Close()

	var positions []types.Position
	for rows.Next() {
		var position types.Position
		err := rows.Scan(
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
			return nil, fmt.Errorf("failed to scan position: %w", err)
		}
		positions = append(positions, position)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating positions: %w", err)
	}

	return positions, nil
}
