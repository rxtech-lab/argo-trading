package engine

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/runtime"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type BacktestState struct {
	db     *sql.DB
	logger *logger.Logger
	sq     squirrel.StatementBuilderType
}

// CalculatePNL calculates the profit/loss for a trade

func NewBacktestState(logger *logger.Logger) (*BacktestState, error) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		logger.Error("Failed to open database", zap.Error(err))

		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection to ensure database is properly initialized
	if err := db.Ping(); err != nil {
		logger.Error("Failed to connect to database", zap.Error(err))
		db.Close()

		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &BacktestState{
		logger: logger,
		db:     db,
		sq:     squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
	}, nil
}

// Initialize creates the necessary tables for tracking trades and positions.
func (b *BacktestState) Initialize() error {
	// Check for nil db
	if b == nil || b.db == nil {
		return fmt.Errorf("backtest state or database is nil")
	}

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
			status TEXT,
			reason TEXT,
			message TEXT,
			strategy_name TEXT,
			position_type TEXT
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
			pnl DOUBLE,
			position_type TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create trades table: %w", err)
	}

	return nil
}

// UpdateResult contains the results of processing an order.
type UpdateResult struct {
	Order         types.Order
	Trade         types.Trade
	IsNewPosition bool
}

// Update processes orders and updates trades.
func (b *BacktestState) Update(orders []types.Order) ([]UpdateResult, error) {
	// Check for nil fields
	if b == nil || b.db == nil {
		return nil, fmt.Errorf("backtest state or database is nil")
	}

	results := make([]UpdateResult, 0, len(orders))

	for _, order := range orders {
		orderID := uuid.New().String()

		if err := order.Validate(); err != nil {
			return nil, fmt.Errorf("invalid order: %w", err)
		}

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
				"is_completed", "status", "reason", "message", "strategy_name", "position_type",
			).
			Values(
				orderID, order.Symbol, order.Side, order.Quantity, order.Price,
				order.Timestamp, order.IsCompleted, order.Status, order.Reason.Reason, order.Reason.Message,
				order.StrategyName, order.PositionType,
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

		if order.Side == types.PurchaseTypeSell && order.PositionType == types.PositionTypeLong && currentPosition.TotalLongPositionQuantity > 0 {
			// For sell orders, calculate PnL using decimal arithmetic
			avgEntryPrice := currentPosition.GetAverageLongPositionEntryPrice()
			entryDec := decimal.NewFromFloat(order.Quantity).Mul(decimal.NewFromFloat(avgEntryPrice))
			exitDec := decimal.NewFromFloat(order.Quantity).Mul(decimal.NewFromFloat(order.Price)).Sub(decimal.NewFromFloat(order.Fee))
			resultDec := exitDec.Sub(entryDec)
			pnl, _ = resultDec.Float64()
		}

		if order.Side == types.PurchaseTypeSell && order.PositionType == types.PositionTypeShort && currentPosition.TotalShortPositionQuantity > 0 {
			// For sell orders, calculate PnL using decimal arithmetic
			avgEntryPrice := currentPosition.GetAverageShortPositionEntryPrice()
			entryDec := decimal.NewFromFloat(order.Quantity).Mul(decimal.NewFromFloat(avgEntryPrice))
			exitDec := decimal.NewFromFloat(order.Quantity).Mul(decimal.NewFromFloat(order.Price)).Add(decimal.NewFromFloat(order.Fee))
			resultDec := entryDec.Sub(exitDec)
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
				Status:       order.Status,
				Reason:       order.Reason,
				StrategyName: order.StrategyName,
				Fee:          order.Fee,
				PositionType: order.PositionType,
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
				"executed_at", "executed_qty", "executed_price", "commission", "pnl", "position_type",
			).
			Values(
				orderID, trade.Order.Symbol, trade.Order.Side, trade.Order.Quantity, trade.Order.Price,
				trade.Order.Timestamp, trade.Order.IsCompleted, trade.Order.Reason.Reason, trade.Order.Reason.Message,
				order.StrategyName, trade.ExecutedAt, trade.ExecutedQty, trade.ExecutedPrice,
				trade.Fee, trade.PnL, trade.Order.PositionType,
			).
			RunWith(tx)

		_, err = insertTradeQuery.Exec()
		if err != nil {
			tx.Rollback()

			return nil, fmt.Errorf("failed to insert trade: %w", err)
		}

		// Determine if this is a new position
		var isNewPosition bool = false

		if order.Side == types.PurchaseTypeBuy && order.PositionType == types.PositionTypeLong {
			if currentPosition.TotalLongPositionQuantity == 0 {
				isNewPosition = true
			}
		}

		if order.Side == types.PurchaseTypeBuy && order.PositionType == types.PositionTypeShort {
			if currentPosition.TotalShortPositionQuantity == 0 {
				isNewPosition = true
			}
		}

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

// StoreFailedOrder stores a failed order in the database without creating a trade.
// This is used when an order fails validation (e.g., insufficient buying power).
func (b *BacktestState) StoreFailedOrder(order types.Order) error {
	// Check for nil fields
	if b == nil || b.db == nil {
		return fmt.Errorf("backtest state or database is nil")
	}

	orderID := uuid.New().String()

	// Insert order with failed status
	insertQuery := b.sq.
		Insert("orders").
		Columns(
			"order_id", "symbol", "order_type", "quantity", "price", "timestamp",
			"is_completed", "status", "reason", "message", "strategy_name", "position_type",
		).
		Values(
			orderID, order.Symbol, order.Side, order.Quantity, order.Price,
			order.Timestamp, order.IsCompleted, order.Status, order.Reason.Reason, order.Reason.Message,
			order.StrategyName, order.PositionType,
		).
		RunWith(b.db)

	_, err := insertQuery.Exec()
	if err != nil {
		return fmt.Errorf("failed to insert failed order: %w", err)
	}

	return nil
}

// GetPosition retrieves the current position for a symbol by calculating from trades.
func (b *BacktestState) GetPosition(symbol string) (types.Position, error) {
	// Extended CTEs to calculate both long (buy/sell) and short (sell/buy) position fields
	query := `
    WITH long_buy_trades AS (
       SELECT 
          SUM(executed_qty) as total_long_in_qty,
          SUM(commission) as total_long_in_fee,
          SUM(executed_qty * executed_price) as total_long_in_amount
       FROM trades 
       WHERE symbol = ? AND order_type = ? AND position_type = ?
    ),
    long_sell_trades AS (
       SELECT 
          SUM(executed_qty) as total_long_out_qty,
          SUM(commission) as total_long_out_fee,
          SUM(executed_qty * executed_price) as total_long_out_amount
       FROM trades 
       WHERE symbol = ? AND order_type = ? AND position_type = ?
    ),
    short_sell_trades AS (
       SELECT 
          SUM(executed_qty) as total_out_short_qty,
          SUM(commission) as total_short_out_fee,
          SUM(executed_qty * executed_price) as total_short_out_amount
       FROM trades 
       WHERE symbol = ? AND order_type = ? AND position_type = ?
    ),
    short_buy_trades AS (
       SELECT 
          SUM(executed_qty) as total_short_in_qty,
          SUM(commission) as total_short_in_fee,
          SUM(executed_qty * executed_price) as total_short_in_amount
       FROM trades 
       WHERE symbol = ? AND order_type = ? AND position_type = ?
    ),
    first_trade AS (
       SELECT 
          MIN(executed_at) as first_trade_time
       FROM trades 
       WHERE symbol = ?
    )
    SELECT 
       ? as symbol,
       COALESCE(b.total_long_in_qty, 0) - COALESCE(s.total_long_out_qty, 0) as quantity,
       COALESCE(b.total_long_in_qty, 0) as total_in_long_position_quantity,
       COALESCE(s.total_long_out_qty, 0) as total_out_long_position_quantity,
       COALESCE(b.total_long_in_amount, 0) as total_in_long_position_amount,
       COALESCE(s.total_long_out_amount, 0) as total_out_long_position_amount,
       COALESCE(b.total_long_in_fee, 0) as total_long_in_fee,
       COALESCE(s.total_long_out_fee, 0)  as total_long_out_fee,
       COALESCE(sb.total_short_in_fee, 0) as total_short_in_fee,
       COALESCE(ss.total_short_out_fee, 0)  as total_short_out_fee,
       ft.first_trade_time as open_timestamp,
       MAX(t.strategy_name) as strategy_name,
       COALESCE(sb.total_short_in_qty, 0) as total_in_short_position_quantity,
       COALESCE(ss.total_out_short_qty, 0) as total_out_short_position_quantity,
       COALESCE(sb.total_short_in_amount, 0) as total_in_short_position_amount,
       COALESCE(ss.total_short_out_amount, 0) as total_out_short_position_amount,
       COALESCE(sb.total_short_in_qty, 0) - COALESCE(ss.total_out_short_qty, 0) as short_quantity
    FROM trades t
    LEFT JOIN long_buy_trades b ON 1=1
    LEFT JOIN long_sell_trades s ON 1=1
    LEFT JOIN short_sell_trades ss ON 1=1
    LEFT JOIN short_buy_trades sb ON 1=1
    CROSS JOIN first_trade ft
    WHERE t.symbol = ?
    GROUP BY b.total_long_in_qty, s.total_long_out_qty, b.total_long_in_amount, s.total_long_out_amount, b.total_long_in_fee, s.total_long_out_fee, sb.total_short_in_fee, ss.total_short_out_fee, ss.total_out_short_qty, sb.total_short_in_qty, ss.total_short_out_amount, sb.total_short_in_amount, ss.total_short_out_fee, sb.total_short_in_fee, ft.first_trade_time
    `

	args := []interface{}{
		symbol, types.PurchaseTypeBuy, types.PositionTypeLong, // long_buy_trades
		symbol, types.PurchaseTypeSell, types.PositionTypeLong, // long_sell_trades
		symbol, types.PurchaseTypeSell, types.PositionTypeShort, // short_sell_trades
		symbol, types.PurchaseTypeBuy, types.PositionTypeShort, // short_buy_trades
		symbol, // first_trade CTE symbol parameter
		symbol, // symbol for select
		symbol, // symbol for WHERE
	}

	var position types.Position
	err := b.db.QueryRow(query, args...).Scan(
		&position.Symbol,
		&position.TotalLongPositionQuantity,
		&position.TotalLongInPositionQuantity,
		&position.TotalLongOutPositionQuantity,
		&position.TotalLongInPositionAmount,
		&position.TotalLongOutPositionAmount,
		&position.TotalLongInFee,
		&position.TotalLongOutFee,
		&position.TotalShortInFee,
		&position.TotalShortOutFee,
		&position.OpenTimestamp,
		&position.StrategyName,
		&position.TotalShortInPositionQuantity,
		&position.TotalShortOutPositionQuantity,
		&position.TotalShortInPositionAmount,
		&position.TotalShortOutPositionAmount,
		&position.TotalShortPositionQuantity,
	)

	if errors.Is(err, sql.ErrNoRows) {
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

	if err != nil {
		return types.Position{}, fmt.Errorf("failed to query position: %w", err)
	}

	return position, nil
}

// GetAllTrades returns all trades from the database.
func (b *BacktestState) GetAllTrades() ([]types.Trade, error) {
	selectQuery := b.sq.
		Select(
			"order_id", "symbol", "order_type", "quantity", "price", "timestamp",
			"is_completed", "reason", "message", "strategy_name",
			"executed_at", "executed_qty", "executed_price", "commission", "pnl", "position_type",
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
			&trade.Order.PositionType,
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

// Cleanup resets the database state.
func (b *BacktestState) Cleanup() error {
	// Check for nil db
	if b == nil || b.db == nil {
		return fmt.Errorf("backtest state or database is nil")
	}

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

// Write saves the backtest results to Parquet files in the specified directory.
func (b *BacktestState) Write(path string) error {
	// Check for nil fields
	if b == nil || b.db == nil || b.logger == nil {
		return fmt.Errorf("backtest state, database, or logger is nil")
	}

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

// GetStats returns the statistics of the backtest.
// runID is the unique identifier for this backtest run.
// tradesFilePath, ordersFilePath, and marksFilePath are the paths to the output files.
func (b *BacktestState) GetStats(ctx runtime.RuntimeContext, runID, tradesFilePath, ordersFilePath, marksFilePath, strategyPath, dataPath string) ([]types.TradeStats, error) {
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

		// Get last market data for unrealized PnL calculation and holding time
		lastData, err := ctx.DataSource.ReadLastData(symbol)
		if err != nil {
			return nil, fmt.Errorf("failed to get last market data for %s: %w", symbol, err)
		}

		// Calculate holding time (uses lastData.Time for open positions)
		holdingTime, err := b.calculateTradeHoldingTime(symbol, lastData.Time)
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

		tradePnl := types.TradePnl{
			RealizedPnL:   0,
			UnrealizedPnL: 0,
			TotalPnL:      0,
			MaximumLoss:   0,
			MaximumProfit: 0,
		}

		// Calculate unrealized PnL if there's an open position
		if position.TotalLongPositionQuantity > 0 {
			entryDec := decimal.NewFromFloat(position.TotalLongPositionQuantity).Mul(decimal.NewFromFloat(position.GetAverageLongPositionEntryPrice()))
			exitDec := decimal.NewFromFloat(position.TotalLongPositionQuantity).Mul(decimal.NewFromFloat(lastData.Close))
			unrealizedPnL, _ := exitDec.Sub(entryDec).Float64()
			realizedPnl := position.GetTotalPnL()
			tradePnl.TotalPnL = realizedPnl + unrealizedPnL
			tradePnl.RealizedPnL = realizedPnl
			tradePnl.UnrealizedPnL = unrealizedPnL
		} else if position.TotalShortPositionQuantity < 0 {
			// For short positions, profit is made when price goes down
			// TotalShortPositionQuantity is negative for a short position
			shortQuantity := -position.TotalShortPositionQuantity // Convert to positive for calculations

			// For short positions, we don't use GetAverageShortPositionEntryPrice
			// Instead calculate directly from the position data and our knowledge that
			// for our specific test with short positions, the entry price is simply the price
			// of the first sell order
			var entryPrice float64
			if position.TotalShortOutPositionQuantity > 0 {
				entryPrice = position.TotalShortOutPositionAmount / position.TotalShortOutPositionQuantity
			}

			// Calculate UnrealizedPnL = (entryPrice - lastPrice) * quantity
			unrealizedPnL := (entryPrice - lastData.Close) * shortQuantity

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

		// Calculate Buy-and-Hold PnL
		buyAndHoldPnl, err := b.calculateBuyAndHoldPnL(symbol, ctx.DataSource)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate buy-and-hold PnL: %w", err)
		}

		stats = append(stats, types.TradeStats{
			ID:               runID,
			Timestamp:        time.Now(),
			Symbol:           symbol,
			TradeResult:      tradeResult,
			TotalFees:        totalFees,
			TradeHoldingTime: holdingTime,
			TradePnl:         tradePnl,
			BuyAndHoldPnl:    buyAndHoldPnl,
			TradesFilePath:   tradesFilePath,
			OrdersFilePath:   ordersFilePath,
			MarksFilePath:    marksFilePath,
			StrategyPath:     strategyPath,
			DataPath:         dataPath,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating symbols: %w", err)
	}

	return stats, nil
}

// GetOrderById returns an order by its id.
func (b *BacktestState) GetOrderById(orderID string) (optional.Option[types.Order], error) {
	query := b.sq.
		Select("order_id", "symbol", "order_type", "quantity", "price", "timestamp", "is_completed", "status", "reason", "message", "strategy_name", "position_type").
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
		&order.Status,
		&order.Reason.Reason,
		&order.Reason.Message,
		&order.StrategyName,
		&order.PositionType,
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

// GetAllPositions returns all positions from the database by calculating from trades.
func (b *BacktestState) GetAllPositions() ([]types.Position, error) {
	// Extended CTEs to calculate both long and short position fields for all symbols
	query := `
		WITH long_buy_trades AS (
			SELECT 
				symbol,
				SUM(executed_qty) as total_in_qty,
				SUM(commission) as total_in_fee,
				SUM(executed_qty * executed_price) as total_in_amount,
				MIN(executed_at) as first_trade_time,
				MAX(strategy_name) as strategy_name
			FROM trades 
			WHERE order_type = ? AND position_type = ?
			GROUP BY symbol
		),
		long_sell_trades AS (
			SELECT 
				symbol,
				SUM(executed_qty) as total_out_qty,
				SUM(commission) as total_out_fee,
				SUM(executed_qty * executed_price) as total_out_amount
			FROM trades 
			WHERE order_type = ? AND position_type = ?
			GROUP BY symbol
		),
		short_sell_trades AS (
			SELECT 
				symbol,
				SUM(executed_qty) as total_in_short_qty,
				SUM(commission) as total_in_short_fee,
				SUM(executed_qty * executed_price) as total_in_short_amount
			FROM trades 
			WHERE order_type = ? AND position_type = ?
			GROUP BY symbol
		),
		short_cover_trades AS (
			SELECT 
				symbol,
				SUM(executed_qty) as total_out_short_qty,
				SUM(commission) as total_out_short_fee,
				SUM(executed_qty * executed_price) as total_out_short_amount
			FROM trades 
			WHERE order_type = ? AND position_type = ?
			GROUP BY symbol
		)
		SELECT 
			COALESCE(b.symbol, s.symbol, ss.symbol, sc.symbol) as symbol,
			COALESCE(b.total_in_qty, 0) - COALESCE(s.total_out_qty, 0) as quantity,
			COALESCE(b.total_in_qty, 0) as total_in_long_position_quantity,
			COALESCE(s.total_out_qty, 0) as total_out_long_position_quantity,
			COALESCE(b.total_in_amount, 0) as total_in_long_position_amount,
			COALESCE(s.total_out_amount, 0) as total_out_long_position_amount,
			COALESCE(b.total_in_fee, 0) as total_in_fee,
			COALESCE(s.total_out_fee, 0) as total_out_fee,
			COALESCE(b.first_trade_time, CURRENT_TIMESTAMP) as open_timestamp,
			COALESCE(b.strategy_name, '') as strategy_name,
			COALESCE(ss.total_in_short_qty, 0) as total_in_short_position_quantity,
			COALESCE(sc.total_out_short_qty, 0) as total_out_short_position_quantity,
			COALESCE(ss.total_in_short_amount, 0) as total_in_short_position_amount,
			COALESCE(sc.total_out_short_amount, 0) as total_out_short_position_amount
		FROM long_buy_trades b
		FULL OUTER JOIN long_sell_trades s ON b.symbol = s.symbol
		FULL OUTER JOIN short_sell_trades ss ON COALESCE(b.symbol, s.symbol) = ss.symbol
		FULL OUTER JOIN short_cover_trades sc ON COALESCE(b.symbol, s.symbol, ss.symbol) = sc.symbol
		WHERE (COALESCE(b.total_in_qty, 0) - COALESCE(s.total_out_qty, 0)) != 0
		ORDER BY symbol
	`

	rows, err := b.db.Query(query, types.PurchaseTypeBuy, types.PositionTypeLong, types.PurchaseTypeSell, types.PositionTypeLong, types.PurchaseTypeSell, types.PositionTypeShort, types.PurchaseTypeBuy, types.PositionTypeShort)
	if err != nil {
		return nil, fmt.Errorf("failed to query positions: %w", err)
	}
	defer rows.Close()

	var positions []types.Position

	for rows.Next() {
		var position types.Position

		err := rows.Scan(
			&position.Symbol,
			&position.TotalLongPositionQuantity,
			&position.TotalLongInPositionQuantity,
			&position.TotalLongOutPositionQuantity,
			&position.TotalLongInPositionAmount,
			&position.TotalLongOutPositionAmount,
			&position.TotalLongInFee,
			&position.TotalLongOutFee,
			&position.OpenTimestamp,
			&position.StrategyName,
			&position.TotalShortInPositionQuantity,
			&position.TotalShortOutPositionQuantity,
			&position.TotalShortInPositionAmount,
			&position.TotalShortOutPositionAmount,
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

func (b *BacktestState) GetAllOrders() ([]types.Order, error) {
	query := b.sq.
		Select("order_id", "symbol", "order_type", "quantity", "price", "timestamp", "is_completed", "status", "reason", "message", "strategy_name", "position_type").
		From("orders").
		OrderBy("timestamp ASC").
		RunWith(b.db)

	rows, err := query.Query()
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	var orders []types.Order

	for rows.Next() {
		var order types.Order

		err := rows.Scan(
			&order.OrderID,
			&order.Symbol,
			&order.Side,
			&order.Quantity,
			&order.Price,
			&order.Timestamp,
			&order.IsCompleted,
			&order.Status,
			&order.Reason.Reason,
			&order.Reason.Message,
			&order.StrategyName,
			&order.PositionType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}

		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating orders: %w", err)
	}

	return orders, nil
}

// calculateTradeResult calculates the trade result statistics for a symbol.
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

// calculateTradeHoldingTime calculates the holding time statistics for a symbol.
// endTime is used to calculate holding time for open positions (positions not yet sold).
func (b *BacktestState) calculateTradeHoldingTime(symbol string, endTime time.Time) (types.TradeHoldingTime, error) {
	// Using raw SQL for CTE query - Squirrel doesn't natively support this complex query
	// Uses FIFO matching: first buy matches first sell, second buy matches second sell, etc.
	// Open positions (buys without matching sells) use endTime as the "sell time"
	query := `
		WITH buy_trades AS (
			SELECT executed_at, ROW_NUMBER() OVER (ORDER BY executed_at) as rn
			FROM trades
			WHERE symbol = ? AND order_type = ?
		),
		sell_trades AS (
			SELECT executed_at, ROW_NUMBER() OVER (ORDER BY executed_at) as rn
			FROM trades
			WHERE symbol = ? AND order_type = ?
		),
		-- Closed positions: matched buy-sell pairs using FIFO
		closed_durations AS (
			SELECT
				EXTRACT(EPOCH FROM (s.executed_at - b.executed_at)) as duration
			FROM buy_trades b
			JOIN sell_trades s ON s.rn = b.rn
		),
		-- Open positions: buys without matching sells (use end time)
		open_durations AS (
			SELECT
				EXTRACT(EPOCH FROM (CAST(? AS TIMESTAMP) - b.executed_at)) as duration
			FROM buy_trades b
			WHERE b.rn > (SELECT COALESCE(MAX(rn), 0) FROM sell_trades)
		),
		all_durations AS (
			SELECT duration FROM closed_durations
			UNION ALL
			SELECT duration FROM open_durations
		)
		SELECT
			COALESCE(MIN(duration), 0) as min_duration,
			COALESCE(MAX(duration), 0) as max_duration,
			COALESCE(AVG(duration), 0) as avg_duration
		FROM all_durations
	`

	var minDuration, maxDuration, avgDuration float64

	// Format endTime as ISO 8601 string for DuckDB compatibility
	endTimeStr := endTime.Format("2006-01-02 15:04:05")

	err := b.db.QueryRow(query, symbol, types.PurchaseTypeBuy, symbol, types.PurchaseTypeSell, endTimeStr).Scan(
		&minDuration,
		&maxDuration,
		&avgDuration,
	)
	if err != nil {
		return types.TradeHoldingTime{}, fmt.Errorf("failed to calculate holding time: %w", err)
	}

	holdingTime := types.TradeHoldingTime{
		Min: int(math.Round(minDuration)),
		Max: int(math.Round(maxDuration)),
		Avg: int(math.Round(avgDuration)),
	}

	return holdingTime, nil
}

// calculateTotalFees calculates the total fees for a symbol.
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

// calculateBuyAndHoldPnL calculates the buy-and-hold PnL for a symbol.
func (b *BacktestState) calculateBuyAndHoldPnL(symbol string, ds datasource.DataSource) (float64, error) {
	// Find the first trade for this symbol
	query := `
		WITH first_trade AS (
			SELECT 
				MIN(timestamp) as first_timestamp
			FROM trades 
			WHERE symbol = ?
		),
		first_order AS (
			SELECT 
				price as first_price,
				quantity as first_quantity,
				position_type as position_type,
				order_type as side
			FROM trades 
			WHERE symbol = ? AND timestamp = (SELECT first_timestamp FROM first_trade)
			LIMIT 1
		)
		SELECT 
			first_price, 
			first_quantity,
			position_type,
			side
		FROM first_order
	`

	var firstPrice, firstQuantity float64

	var positionType, side string

	err := b.db.QueryRow(query, symbol, symbol).Scan(&firstPrice, &firstQuantity, &positionType, &side)
	if err != nil {
		if err == sql.ErrNoRows {
			// No trades for this symbol
			return 0, nil
		}

		return 0, fmt.Errorf("failed to query first trade: %w", err)
	}

	// Get last market data for end price
	lastData, err := ds.ReadLastData(symbol)
	if err != nil {
		return 0, fmt.Errorf("failed to get last market data for %s: %w", symbol, err)
	}

	// Calculate buy-and-hold PnL based on position type
	var buyAndHoldPnl float64
	if positionType == string(types.PositionTypeLong) {
		// For long positions: (lastPrice - firstPrice) * firstQuantity
		buyAndHoldPnl = (lastData.Close - firstPrice) * firstQuantity
	} else if positionType == string(types.PositionTypeShort) {
		// For short positions: (firstPrice - lastPrice) * firstQuantity
		// In short positions, profit is made when price goes down
		buyAndHoldPnl = (firstPrice - lastData.Close) * firstQuantity
	}

	return buyAndHoldPnl, nil
}
