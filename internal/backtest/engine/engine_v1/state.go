package engine

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
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
	db                        *sql.DB
	logger                    *logger.Logger
	sq                        squirrel.StatementBuilderType
	initialBalance            float64
	portfolioStrategy         PortfolioCalculationStrategy
	riskFreeRate              float64
	sharpeAnnualizationFactor int
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
		logger:                    logger,
		db:                        db,
		sq:                        squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
		initialBalance:            0,
		portfolioStrategy:         PortfolioCalculationFIFO,
		riskFreeRate:              0,
		sharpeAnnualizationFactor: DefaultSharpeAnnualizationFactor,
	}, nil
}

// SetInitialBalance sets the initial cash balance for the backtest run.
func (b *BacktestState) SetInitialBalance(balance float64) {
	b.initialBalance = balance
}

// SetPortfolioCalculationStrategy selects how per-trade PnL is computed for
// closing trades. Unknown values fall back to average-cost.
func (b *BacktestState) SetPortfolioCalculationStrategy(strategy PortfolioCalculationStrategy) {
	b.portfolioStrategy = ResolvePortfolioCalculation(strategy)
}

// PortfolioCalculationStrategy returns the active portfolio calculation strategy.
func (b *BacktestState) PortfolioCalculationStrategy() PortfolioCalculationStrategy {
	return b.portfolioStrategy
}

// SetRiskFreeRate sets the annualized risk-free rate (as a decimal fraction;
// e.g. 0.04 = 4%) used when computing the Sharpe ratio. Defaults to 0.
func (b *BacktestState) SetRiskFreeRate(rate float64) {
	b.riskFreeRate = rate
}

// SetSharpeAnnualizationFactor sets the number of return periods per year used
// to annualize the Sharpe ratio. A value of 0 falls back to the default (252);
// negative values disable annualization.
func (b *BacktestState) SetSharpeAnnualizationFactor(n int) {
	b.sharpeAnnualizationFactor = ResolveSharpeAnnualizationFactor(n)
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
			cumulative_pnl DOUBLE,
			position_type TEXT,
			open_position_qty DOUBLE,
			balance DOUBLE,
			hold_time BIGINT,
			average_cost DOUBLE
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

		// Calculate FIFO PnL for closing trades; buys have zero individual PnL.
		fifoPnl, err := b.computeClosingPnL(order, currentPosition)
		if err != nil {
			tx.Rollback()

			return nil, fmt.Errorf("failed to calculate FIFO PnL: %w", err)
		}

		// Cumulative PnL is the running sum of per-trade PnL for this symbol.
		var priorPnlSum float64
		if err := b.db.QueryRow(`SELECT COALESCE(SUM(pnl), 0) FROM trades WHERE symbol = ?`, order.Symbol).Scan(&priorPnlSum); err != nil {
			tx.Rollback()

			return nil, fmt.Errorf("failed to query prior pnl sum: %w", err)
		}

		cumulativePnl := priorPnlSum + fifoPnl

		openPositionQty := computeOpenPositionQty(order, currentPosition)

		// Calculate cash balance after this trade.
		var prevBalance float64
		err = b.db.QueryRow(`SELECT COALESCE((SELECT balance FROM trades ORDER BY rowid DESC LIMIT 1), ?)`, b.initialBalance).Scan(&prevBalance)
		if err != nil {
			tx.Rollback()

			return nil, fmt.Errorf("failed to query previous balance: %w", err)
		}

		balance := computeCashBalance(prevBalance, order)

		// Calculate FIFO-weighted hold time (in seconds) for closing trades; 0 for opening trades.
		holdTime, err := b.computeClosingHoldTime(order, currentPosition)
		if err != nil {
			tx.Rollback()

			return nil, fmt.Errorf("failed to calculate hold time: %w", err)
		}

		// Per-unit weighted-average cost basis relevant to this trade.
		averageCost, err := b.computeTradeAverageCost(order)
		if err != nil {
			tx.Rollback()

			return nil, fmt.Errorf("failed to calculate average cost: %w", err)
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
			ExecutedAt:      order.Timestamp,
			ExecutedQty:     order.Quantity,
			ExecutedPrice:   order.Price,
			Fee:             order.Fee,
			PnL:             fifoPnl,
			CumulativePnL:   cumulativePnl,
			OpenPositionQty: openPositionQty,
			Balance:         balance,
			HoldTime:        holdTime,
			AverageCost:     averageCost,
		}

		// Insert trade using Squirrel
		insertTradeQuery := b.sq.
			Insert("trades").
			Columns(
				"order_id", "symbol", "order_type", "quantity", "price", "timestamp",
				"is_completed", "reason", "message", "strategy_name",
				"executed_at", "executed_qty", "executed_price", "commission", "pnl", "cumulative_pnl", "position_type",
				"open_position_qty", "balance", "hold_time", "average_cost",
			).
			Values(
				orderID, trade.Order.Symbol, trade.Order.Side, trade.Order.Quantity, trade.Order.Price,
				trade.Order.Timestamp, trade.Order.IsCompleted, trade.Order.Reason.Reason, trade.Order.Reason.Message,
				order.StrategyName, trade.ExecutedAt, trade.ExecutedQty, trade.ExecutedPrice,
				trade.Fee, trade.PnL, trade.CumulativePnL, trade.Order.PositionType,
				trade.OpenPositionQty, trade.Balance, trade.HoldTime, trade.AverageCost,
			).
			RunWith(tx)

		_, err = insertTradeQuery.Exec()
		if err != nil {
			tx.Rollback()

			return nil, fmt.Errorf("failed to insert trade: %w", err)
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
			IsNewPosition: isNewPositionOpened(order, currentPosition),
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
			"executed_at", "executed_qty", "executed_price", "commission", "pnl", "cumulative_pnl", "position_type",
			"hold_time",
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
			&trade.CumulativePnL,
			&trade.Order.PositionType,
			&trade.HoldTime,
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

// getStrategyInfo retrieves strategy metadata from the runtime.
func getStrategyInfo(strategyRuntime runtime.StrategyRuntime) (types.StrategyInfo, error) {
	identifier, err := strategyRuntime.GetIdentifier()
	if err != nil {
		return types.StrategyInfo{}, fmt.Errorf("failed to get strategy identifier: %w", err)
	}

	version, err := strategyRuntime.GetRuntimeEngineVersion()
	if err != nil {
		return types.StrategyInfo{}, fmt.Errorf("failed to get strategy version: %w", err)
	}

	return types.StrategyInfo{
		ID:      identifier,
		Version: version,
		Name:    strategyRuntime.Name(),
	}, nil
}

// statsParams contains common parameters for creating TradeStats.
type statsParams struct {
	runID          string
	tradesFilePath string
	ordersFilePath string
	marksFilePath  string
	logsFilePath   string
	strategyInfo   types.StrategyInfo
	strategyPath   string
	dataPath       string
}

// createZeroStats creates a TradeStats with zero values for a symbol without trades.
func createZeroStats(symbol string, params statsParams, initialBalance float64) types.TradeStats {
	return types.TradeStats{
		ID:        params.runID,
		Timestamp: time.Now(),
		Symbol:    symbol,
		TradeResult: types.TradeResult{
			NumberOfTrades:        0,
			NumberOfTradingPairs:  0,
			NumberOfWinningTrades: 0,
			NumberOfLosingTrades:  0,
			WinRate:               0,
			MaxDrawdown:           0,
			SharpeRatio:           0,
		},
		TotalFees: 0,
		TradeHoldingTime: types.TradeHoldingTime{
			Min:         0,
			Max:         0,
			Avg:         0,
			Median:      0,
			Percentiles: types.Percentiles{P25: 0, P50: 0, P75: 0, P90: 0, P95: 0, P99: 0},
		},
		TradePnl: types.TradePnl{
			RealizedPnL:     0,
			UnrealizedPnL:   0,
			TotalPnL:        0,
			MaximumLoss:     0,
			MaximumProfit:   0,
			MedianPnL:       0,
			Percentiles:     types.Percentiles{P25: 0, P50: 0, P75: 0, P90: 0, P95: 0, P99: 0},
			TotalInvestment: 0,
			PnLPercentage:   0,
		},
		BuyAndHoldPnl:        0,
		TradesFilePath:       params.tradesFilePath,
		OrdersFilePath:       params.ordersFilePath,
		MarksFilePath:        params.marksFilePath,
		LogsFilePath:         params.logsFilePath,
		Strategy:             params.strategyInfo,
		StrategyPath:         params.strategyPath,
		DataPath:             params.dataPath,
		InitialBalance:       initialBalance,
		FinalBalance:         initialBalance,
		PortfolioCalculation: "",
		MonthlyTrades:        nil,
		MonthlyBalance:       nil,
		MonthlyHoldingTime:   nil,
		BacktestConfig:       nil,
		StrategyConfig:       nil,
	}
}

// calculateUnrealizedPnL calculates the realized and unrealized PnL for a position.
// Realized PnL is always populated from closed round-trips, even when the final position is flat.
func calculateUnrealizedPnL(position types.Position, lastPrice float64) types.TradePnl {
	realizedPnl := position.GetTotalPnL()

	var unrealizedPnL float64

	if position.TotalLongPositionQuantity > 0 {
		entryDec := decimal.NewFromFloat(position.TotalLongPositionQuantity).Mul(decimal.NewFromFloat(position.GetAverageLongPositionEntryPrice()))
		exitDec := decimal.NewFromFloat(position.TotalLongPositionQuantity).Mul(decimal.NewFromFloat(lastPrice))
		unrealizedPnL, _ = exitDec.Sub(entryDec).Float64()
	} else if position.TotalShortPositionQuantity < 0 {
		shortQuantity := -position.TotalShortPositionQuantity

		var entryPrice float64
		if position.TotalShortOutPositionQuantity > 0 {
			entryPrice = position.TotalShortOutPositionAmount / position.TotalShortOutPositionQuantity
		}

		unrealizedPnL = (entryPrice - lastPrice) * shortQuantity
	}

	return types.TradePnl{
		RealizedPnL:     realizedPnl,
		UnrealizedPnL:   unrealizedPnL,
		TotalPnL:        realizedPnl + unrealizedPnL,
		MaximumLoss:     0,
		MaximumProfit:   0,
		MedianPnL:       0,
		Percentiles:     types.Percentiles{P25: 0, P50: 0, P75: 0, P90: 0, P95: 0, P99: 0},
		TotalInvestment: 0,
		PnLPercentage:   0,
	}
}

// GetStats returns the statistics of the backtest for all symbols.
func (b *BacktestState) GetStats(ctx runtime.RuntimeContext, strategyRuntime runtime.StrategyRuntime, runID, tradesFilePath, ordersFilePath, marksFilePath, logsFilePath, strategyPath, dataPath string) ([]types.TradeStats, error) {
	strategyInfo, err := getStrategyInfo(strategyRuntime)
	if err != nil {
		return nil, err
	}

	params := statsParams{
		runID:          runID,
		tradesFilePath: tradesFilePath,
		ordersFilePath: ordersFilePath,
		marksFilePath:  marksFilePath,
		logsFilePath:   logsFilePath,
		strategyInfo:   strategyInfo,
		strategyPath:   strategyPath,
		dataPath:       dataPath,
	}

	tradeSymbols, err := b.getTradeSymbols()
	if err != nil {
		return nil, err
	}

	symbols := tradeSymbols
	if len(symbols) == 0 {
		symbols, err = ctx.DataSource.GetAllSymbols()
		if err != nil {
			return nil, fmt.Errorf("failed to get symbols from market data: %w", err)
		}
	}

	tradeSymbolSet := make(map[string]bool)
	for _, s := range tradeSymbols {
		tradeSymbolSet[s] = true
	}

	stats := make([]types.TradeStats, 0, len(symbols))

	for _, symbol := range symbols {
		stat, err := b.calculateSymbolStats(ctx, symbol, tradeSymbolSet[symbol], params)
		if err != nil {
			return nil, err
		}

		stats = append(stats, stat)
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

// computeClosingPnL calculates the closing PnL for a trade based on position
// state, using the configured portfolio calculation strategy (FIFO or
// average-cost). It returns 0 for opening trades.
func (b *BacktestState) computeClosingPnL(order types.Order, position types.Position) (float64, error) {
	isLongClose := order.Side == types.PurchaseTypeSell && order.PositionType == types.PositionTypeLong && position.TotalLongPositionQuantity > 0
	isShortClose := order.Side == types.PurchaseTypeSell && order.PositionType == types.PositionTypeShort && position.TotalShortPositionQuantity > 0

	if !isLongClose && !isShortClose {
		return 0, nil
	}

	if b.portfolioStrategy == PortfolioCalculationAverageCost {
		return b.calculateAverageCostPnL(order.Symbol, order.PositionType, order.Quantity, order.Price, order.Fee)
	}

	return b.calculateFIFOPnL(order.Symbol, order.PositionType, order.Quantity, order.Price, order.Fee)
}

// computeClosingHoldTime returns the quantity-weighted-average holding time (in
// seconds) for a closing trade. Closing trades are SELL orders that match prior
// BUY entries (mirroring the convention used by computeClosingPnL). Under FIFO
// the match consumes the oldest unmatched entries first; under average-cost the
// match consumes the most recently acquired entries first (LIFO). Opening
// trades return 0.
func (b *BacktestState) computeClosingHoldTime(order types.Order, position types.Position) (int, error) {
	isLongClose := order.Side == types.PurchaseTypeSell && order.PositionType == types.PositionTypeLong && position.TotalLongPositionQuantity > 0
	isShortClose := order.Side == types.PurchaseTypeSell && order.PositionType == types.PositionTypeShort && position.TotalShortPositionQuantity > 0

	if !isLongClose && !isShortClose {
		return 0, nil
	}

	if b.portfolioStrategy == PortfolioCalculationAverageCost {
		return b.calculateLIFOHoldTime(order.Symbol, order.PositionType, order.Quantity, order.Timestamp)
	}

	return b.calculateFIFOHoldTime(order.Symbol, order.PositionType, order.Quantity, order.Timestamp)
}

// calculateFIFOHoldTime computes the quantity-weighted-average holding time (seconds)
// between the closing trade at closeTime and prior unmatched BUY entry trades, using FIFO.
func (b *BacktestState) calculateFIFOHoldTime(symbol string, positionType types.PositionType, closeQty float64, closeTime time.Time) (int, error) {
	// Get prior entry (BUY) trades in FIFO order.
	entryQuery := b.sq.
		Select("executed_qty", "executed_at").
		From("trades").
		Where(squirrel.Eq{
			"symbol":        symbol,
			"order_type":    types.PurchaseTypeBuy,
			"position_type": positionType,
		}).
		OrderBy("executed_at ASC").
		RunWith(b.db)

	entryRows, err := entryQuery.Query()
	if err != nil {
		return 0, fmt.Errorf("failed to query entry trades for hold time: %w", err)
	}
	defer entryRows.Close()

	type entryTrade struct {
		qty        float64
		executedAt time.Time
	}

	var entries []entryTrade
	for entryRows.Next() {
		var e entryTrade
		if err := entryRows.Scan(&e.qty, &e.executedAt); err != nil {
			return 0, fmt.Errorf("failed to scan entry trade: %w", err)
		}
		entries = append(entries, e)
	}
	if err := entryRows.Err(); err != nil {
		return 0, fmt.Errorf("error iterating entry trades: %w", err)
	}

	// Quantity previously closed (sold) for this symbol+positionType.
	prevSoldQuery := b.sq.
		Select("COALESCE(SUM(executed_qty), 0)").
		From("trades").
		Where(squirrel.Eq{
			"symbol":        symbol,
			"order_type":    types.PurchaseTypeSell,
			"position_type": positionType,
		}).
		RunWith(b.db)

	var prevSoldQty float64
	if err := prevSoldQuery.QueryRow().Scan(&prevSoldQty); err != nil {
		return 0, fmt.Errorf("failed to query previously closed qty: %w", err)
	}

	remaining := closeQty
	skipQty := prevSoldQty
	weightedSeconds := 0.0
	matchedQtyTotal := 0.0

	for _, entry := range entries {
		if remaining <= 0 {
			break
		}

		if skipQty >= entry.qty {
			skipQty -= entry.qty

			continue
		}

		availableQty := entry.qty - skipQty
		skipQty = 0

		matchedQty := math.Min(availableQty, remaining)
		duration := closeTime.Sub(entry.executedAt).Seconds()
		if duration < 0 {
			duration = 0
		}

		weightedSeconds += duration * matchedQty
		matchedQtyTotal += matchedQty
		remaining -= matchedQty
	}

	if matchedQtyTotal == 0 {
		return 0, nil
	}

	return int(math.Round(weightedSeconds / matchedQtyTotal)), nil
}

// calculateLIFOHoldTime computes the quantity-weighted-average holding time
// (seconds) between a closing trade at closeTime and the lots that would be
// consumed under last-in-first-out matching. Because prior partial exits may
// have consumed lots from the top of the stack — leaving older lots partially
// available "below" newer ones — we replay the full lot history to reconstruct
// the stack as it stands immediately before the current close.
func (b *BacktestState) calculateLIFOHoldTime(symbol string, positionType types.PositionType, closeQty float64, closeTime time.Time) (int, error) {
	tradesQuery := b.sq.
		Select("order_type", "executed_qty", "executed_at").
		From("trades").
		Where(squirrel.Eq{
			"symbol":        symbol,
			"position_type": positionType,
		}).
		OrderBy("executed_at ASC", "rowid ASC").
		RunWith(b.db)

	rows, err := tradesQuery.Query()
	if err != nil {
		return 0, fmt.Errorf("failed to query trades for LIFO hold time: %w", err)
	}
	defer rows.Close()

	type lot struct {
		qty        float64
		executedAt time.Time
	}

	var stack []lot

	for rows.Next() {
		var (
			orderType  string
			qty        float64
			executedAt time.Time
		)

		if err := rows.Scan(&orderType, &qty, &executedAt); err != nil {
			return 0, fmt.Errorf("failed to scan trade for LIFO hold time: %w", err)
		}

		if types.PurchaseType(orderType) == types.PurchaseTypeBuy {
			stack = append(stack, lot{qty: qty, executedAt: executedAt})

			continue
		}

		// SELL: pop from top of stack.
		remaining := qty
		for remaining > 0 && len(stack) > 0 {
			top := &stack[len(stack)-1]
			if top.qty <= remaining {
				remaining -= top.qty
				stack = stack[:len(stack)-1]

				continue
			}

			top.qty -= remaining
			remaining = 0
		}
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("error iterating trades for LIFO hold time: %w", err)
	}

	if len(stack) == 0 {
		return 0, nil
	}

	// Simulate the current close against the reconstructed stack without mutating it.
	remaining := closeQty
	weightedSeconds := 0.0
	matchedQtyTotal := 0.0

	for i := len(stack) - 1; i >= 0 && remaining > 0; i-- {
		entry := stack[i]
		matchedQty := math.Min(entry.qty, remaining)

		duration := closeTime.Sub(entry.executedAt).Seconds()
		if duration < 0 {
			duration = 0
		}

		weightedSeconds += duration * matchedQty
		matchedQtyTotal += matchedQty
		remaining -= matchedQty
	}

	if matchedQtyTotal == 0 {
		return 0, nil
	}

	return int(math.Round(weightedSeconds / matchedQtyTotal)), nil
}

// computeOpenPositionQty calculates the open position quantity after a trade.
func computeOpenPositionQty(order types.Order, position types.Position) float64 {
	if order.PositionType == types.PositionTypeLong {
		if order.Side == types.PurchaseTypeBuy {
			return position.TotalLongPositionQuantity + order.Quantity
		}

		return position.TotalLongPositionQuantity - order.Quantity
	}

	if order.Side == types.PurchaseTypeSell {
		return position.TotalShortPositionQuantity + order.Quantity
	}

	return position.TotalShortPositionQuantity - order.Quantity
}

// computeCashBalance calculates the cash balance after a trade.
func computeCashBalance(prevBalance float64, order types.Order) float64 {
	tradeCost := order.Quantity * order.Price
	if order.Side == types.PurchaseTypeBuy {
		return prevBalance - tradeCost - order.Fee
	}

	return prevBalance + tradeCost - order.Fee
}

// isNewPositionOpened checks if the order opens a new position.
func isNewPositionOpened(order types.Order, position types.Position) bool {
	if order.Side != types.PurchaseTypeBuy {
		return false
	}

	if order.PositionType == types.PositionTypeLong {
		return position.TotalLongPositionQuantity == 0
	}

	return order.PositionType == types.PositionTypeShort && position.TotalShortPositionQuantity == 0
}

// calculateFIFOPnL calculates the individual PnL for a sell order using FIFO matching.
// It matches the sell quantity against the earliest unmatched buy orders to determine
// the actual entry cost for this specific trade.
func (b *BacktestState) calculateFIFOPnL(symbol string, positionType types.PositionType, sellQty float64, sellPrice float64, sellFee float64) (float64, error) {
	// Get all entry (buy) trades for this symbol+positionType in FIFO order
	entryQuery := b.sq.
		Select("executed_qty", "executed_price", "commission").
		From("trades").
		Where(squirrel.Eq{
			"symbol":        symbol,
			"order_type":    types.PurchaseTypeBuy,
			"position_type": positionType,
		}).
		OrderBy("executed_at ASC").
		RunWith(b.db)

	entryRows, err := entryQuery.Query()
	if err != nil {
		return 0, fmt.Errorf("failed to query entry trades for FIFO: %w", err)
	}
	defer entryRows.Close()

	type entryTrade struct {
		qty   float64
		price float64
		fee   float64
	}

	var entries []entryTrade
	for entryRows.Next() {
		var e entryTrade
		if err := entryRows.Scan(&e.qty, &e.price, &e.fee); err != nil {
			return 0, fmt.Errorf("failed to scan entry trade: %w", err)
		}
		entries = append(entries, e)
	}
	if err := entryRows.Err(); err != nil {
		return 0, fmt.Errorf("error iterating entry trades: %w", err)
	}

	// Get total quantity previously sold (exited) for this symbol+positionType
	prevSoldQuery := b.sq.
		Select("COALESCE(SUM(executed_qty), 0)").
		From("trades").
		Where(squirrel.Eq{
			"symbol":        symbol,
			"order_type":    types.PurchaseTypeSell,
			"position_type": positionType,
		}).
		RunWith(b.db)

	var prevSoldQty float64
	if err := prevSoldQuery.QueryRow().Scan(&prevSoldQty); err != nil {
		return 0, fmt.Errorf("failed to query previous sold qty: %w", err)
	}

	// FIFO matching: walk through entries, skip consumed portions, match remaining
	remaining := sellQty
	skipQty := prevSoldQty
	fifoCost := decimal.Zero

	for _, entry := range entries {
		if remaining <= 0 {
			break
		}

		if skipQty >= entry.qty {
			skipQty -= entry.qty

			continue
		}

		availableQty := entry.qty - skipQty
		skipQty = 0

		matchedQty := math.Min(availableQty, remaining)
		matchedDec := decimal.NewFromFloat(matchedQty)

		// Pro-rate the entry fee
		entryFeeProrated := decimal.NewFromFloat(entry.fee).Mul(matchedDec).Div(decimal.NewFromFloat(entry.qty))

		if positionType == types.PositionTypeLong {
			// Long: entry cost = price * qty + fee
			cost := decimal.NewFromFloat(entry.price).Mul(matchedDec).Add(entryFeeProrated)
			fifoCost = fifoCost.Add(cost)
		} else {
			// Short: entry value = price * qty - fee
			value := decimal.NewFromFloat(entry.price).Mul(matchedDec).Sub(entryFeeProrated)
			fifoCost = fifoCost.Add(value)
		}

		remaining -= matchedQty
	}

	sellValue := decimal.NewFromFloat(sellPrice).Mul(decimal.NewFromFloat(sellQty))
	sellFeeDec := decimal.NewFromFloat(sellFee)

	var result decimal.Decimal
	if positionType == types.PositionTypeLong {
		// Long PnL = exit_value - sell_fee - entry_cost
		result = sellValue.Sub(sellFeeDec).Sub(fifoCost)
	} else {
		// Short PnL = entry_value - (exit_value + sell_fee)
		result = fifoCost.Sub(sellValue.Add(sellFeeDec))
	}

	pnl, _ := result.Float64()

	return pnl, nil
}

// calculateAverageCostPnL calculates the individual PnL for a closing trade
// using the running weighted-average cost basis of the currently-open position.
// Unlike FIFO, average-cost does not split the closing trade across specific
// entry lots; instead, the per-unit cost basis is the average of all unclosed
// entries, with buys updating the average and sells leaving it unchanged until
// the position is fully flat (at which point the basis resets).
func (b *BacktestState) calculateAverageCostPnL(symbol string, positionType types.PositionType, sellQty float64, sellPrice float64, sellFee float64) (float64, error) {
	// Load all prior trades (buys and sells) for this symbol+positionType in
	// chronological order so we can replay the running average-cost state.
	tradesQuery := b.sq.
		Select("order_type", "executed_qty", "executed_price", "commission").
		From("trades").
		Where(squirrel.Eq{
			"symbol":        symbol,
			"position_type": positionType,
		}).
		OrderBy("executed_at ASC", "rowid ASC").
		RunWith(b.db)

	rows, err := tradesQuery.Query()
	if err != nil {
		return 0, fmt.Errorf("failed to query trades for average cost: %w", err)
	}
	defer rows.Close()

	// Running state: total open quantity and total cost basis for open quantity.
	openQty := decimal.Zero
	costBasis := decimal.Zero

	for rows.Next() {
		var (
			orderType string
			qty       float64
			price     float64
			fee       float64
		)

		if err := rows.Scan(&orderType, &qty, &price, &fee); err != nil {
			return 0, fmt.Errorf("failed to scan trade for average cost: %w", err)
		}

		qtyDec := decimal.NewFromFloat(qty)
		priceDec := decimal.NewFromFloat(price)
		feeDec := decimal.NewFromFloat(fee)

		if types.PurchaseType(orderType) == types.PurchaseTypeBuy {
			// Entry trade — add to open quantity and cost basis. Fees are
			// capitalised into the basis following the same sign convention as
			// calculateFIFOPnL (added for long, subtracted for short).
			var entryValue decimal.Decimal
			if positionType == types.PositionTypeLong {
				entryValue = priceDec.Mul(qtyDec).Add(feeDec)
			} else {
				entryValue = priceDec.Mul(qtyDec).Sub(feeDec)
			}

			openQty = openQty.Add(qtyDec)
			costBasis = costBasis.Add(entryValue)

			continue
		}

		// Exit trade — reduce open quantity and cost basis at the current
		// average cost per unit. This keeps the average unchanged across
		// partial exits.
		if openQty.IsZero() {
			// Defensive: should not happen if orders pass validation; skip.
			continue
		}

		avg := costBasis.Div(openQty)
		matchedDec := decimal.NewFromFloat(math.Min(qty, openQty.InexactFloat64()))

		openQty = openQty.Sub(matchedDec)
		costBasis = costBasis.Sub(avg.Mul(matchedDec))

		if openQty.Sign() <= 0 {
			openQty = decimal.Zero
			costBasis = decimal.Zero
		}
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("error iterating trades for average cost: %w", err)
	}

	if openQty.Sign() <= 0 {
		// Nothing open to close — defensive fallback consistent with
		// computeClosingPnL (which is only called for true closes).
		return 0, nil
	}

	avg := costBasis.Div(openQty)
	sellQtyDec := decimal.NewFromFloat(sellQty)
	// Can't close more than is currently open.
	if sellQtyDec.GreaterThan(openQty) {
		sellQtyDec = openQty
	}

	sellValue := decimal.NewFromFloat(sellPrice).Mul(sellQtyDec)
	sellFeeDec := decimal.NewFromFloat(sellFee)
	avgCost := avg.Mul(sellQtyDec)

	var result decimal.Decimal
	if positionType == types.PositionTypeLong {
		// Long PnL = exit_value - sell_fee - avg_cost
		result = sellValue.Sub(sellFeeDec).Sub(avgCost)
	} else {
		// Short PnL = avg_entry_value - exit_value - sell_fee
		result = avgCost.Sub(sellValue).Sub(sellFeeDec)
	}

	pnl, _ := result.Float64()

	return pnl, nil
}

// computeTradeAverageCost returns the per-unit weighted-average cost basis of
// the position relevant to this trade:
//   - For BUY (entry) trades: the updated running average after the buy is
//     applied — i.e. the new blended cost basis of the resulting position.
//   - For SELL (closing) trades: the running average BEFORE the sell is
//     applied — i.e. the cost basis being closed out. This matches the
//     post-sell average on partial closes and stays non-zero on full closes
//     (so analysts can compare sell price against cost basis on every row).
//
// Entry fees are capitalised into the basis using the same sign convention as
// calculateAverageCostPnL (added for long entries, subtracted for short
// entries). Returns 0 only when there is no open position to reference (e.g.
// a defensive sell against an empty position). Independent of the configured
// portfolio calculation strategy.
func (b *BacktestState) computeTradeAverageCost(order types.Order) (float64, error) {
	tradesQuery := b.sq.
		Select("order_type", "executed_qty", "executed_price", "commission").
		From("trades").
		Where(squirrel.Eq{
			"symbol":        order.Symbol,
			"position_type": order.PositionType,
		}).
		OrderBy("executed_at ASC", "rowid ASC").
		RunWith(b.db)

	rows, err := tradesQuery.Query()
	if err != nil {
		return 0, fmt.Errorf("failed to query trades for average cost: %w", err)
	}
	defer rows.Close()

	openQty := decimal.Zero
	costBasis := decimal.Zero

	apply := func(orderType types.PurchaseType, qty, price, fee float64) {
		qtyDec := decimal.NewFromFloat(qty)
		priceDec := decimal.NewFromFloat(price)
		feeDec := decimal.NewFromFloat(fee)

		if orderType == types.PurchaseTypeBuy {
			var entryValue decimal.Decimal
			if order.PositionType == types.PositionTypeLong {
				entryValue = priceDec.Mul(qtyDec).Add(feeDec)
			} else {
				entryValue = priceDec.Mul(qtyDec).Sub(feeDec)
			}

			openQty = openQty.Add(qtyDec)
			costBasis = costBasis.Add(entryValue)

			return
		}

		if openQty.IsZero() {
			return
		}

		avg := costBasis.Div(openQty)
		matchedDec := decimal.NewFromFloat(math.Min(qty, openQty.InexactFloat64()))
		openQty = openQty.Sub(matchedDec)
		costBasis = costBasis.Sub(avg.Mul(matchedDec))

		if openQty.Sign() <= 0 {
			openQty = decimal.Zero
			costBasis = decimal.Zero
		}
	}

	for rows.Next() {
		var (
			orderType string
			qty       float64
			price     float64
			fee       float64
		)

		if err := rows.Scan(&orderType, &qty, &price, &fee); err != nil {
			return 0, fmt.Errorf("failed to scan trade for average cost: %w", err)
		}

		apply(types.PurchaseType(orderType), qty, price, fee)
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("error iterating trades for average cost: %w", err)
	}

	// For closing (SELL) trades, report the cost basis being closed — i.e. the
	// running average BEFORE applying the sell. Partial closes leave the
	// average unchanged, so this matches the post-close value in that case; on
	// a full close, it stays non-zero rather than resetting to 0.
	if order.Side == types.PurchaseTypeSell {
		if openQty.Sign() <= 0 {
			return 0, nil
		}

		avg, _ := costBasis.Div(openQty).Float64()

		return avg, nil
	}

	apply(order.Side, order.Quantity, order.Price, order.Fee)

	if openQty.Sign() <= 0 {
		return 0, nil
	}

	avg, _ := costBasis.Div(openQty).Float64()

	return avg, nil
}

// calculateTradeResult calculates the trade result statistics for a symbol.
// NumberOfTrades counts all fills (entries and exits). NumberOfTradingPairs counts
// round trips — each exit trade closes a pair against one or more prior entry trades.
// Win/lose counts and win rate are computed against trading pairs.
func (b *BacktestState) calculateTradeResult(symbol string) (types.TradeResult, error) {
	query := `
		WITH trade_stats AS (
			SELECT
				COUNT(*) as total_trades,
				SUM(CASE WHEN order_type = ? THEN 1 ELSE 0 END) as trading_pairs,
				SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END) as winning_trades,
				SUM(CASE WHEN pnl < 0 THEN 1 ELSE 0 END) as losing_trades,
				MIN(pnl) as min_pnl,
				MAX(pnl) as max_pnl
			FROM trades
			WHERE symbol = ?
		)
		SELECT
			COALESCE(total_trades, 0) as total_trades,
			COALESCE(trading_pairs, 0) as trading_pairs,
			COALESCE(winning_trades, 0) as winning_trades,
			COALESCE(losing_trades, 0) as losing_trades,
			CASE WHEN COALESCE(trading_pairs, 0) > 0 THEN CAST(winning_trades AS DOUBLE) / trading_pairs ELSE 0 END as win_rate,
			CASE WHEN min_pnl < 0 THEN ABS(min_pnl) ELSE 0 END as max_drawdown
		FROM trade_stats
	`

	var result types.TradeResult

	err := b.db.QueryRow(query, types.PurchaseTypeSell, symbol).Scan(
		&result.NumberOfTrades,
		&result.NumberOfTradingPairs,
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

// calculateSharpeRatio computes the annualized Sharpe ratio for a symbol using
// daily equity returns derived from the trades table. For each trading day it
// takes the last observed equity (initial_balance + cumulative_pnl; matching
// how monthly balance is computed), forms
// return_t = equity_t / equity_{t-1} - 1 across consecutive trading days, and
// returns (mean(return) - rf_period) / stdev(return) * sqrt(N), where
// rf_period = riskFreeRate / N and N is the annualization factor. Returns 0
// when there are fewer than two observations or when stdev is zero. When
// sharpeAnnualizationFactor is 0 the ratio is reported un-annualized
// (multiplier of 1).
func (b *BacktestState) calculateSharpeRatio(symbol string) (float64, error) {
	query := `
		SELECT arg_max(cumulative_pnl, executed_at) AS ending_pnl
		FROM trades
		WHERE symbol = ?
		GROUP BY date_trunc('day', executed_at)
		ORDER BY date_trunc('day', executed_at)
	`

	rows, err := b.db.Query(query, symbol)
	if err != nil {
		return 0, fmt.Errorf("failed to query daily equity for sharpe ratio: %w", err)
	}
	defer rows.Close()

	var equities []float64

	for rows.Next() {
		var endingPnl float64
		if err := rows.Scan(&endingPnl); err != nil {
			return 0, fmt.Errorf("failed to scan daily equity: %w", err)
		}

		equities = append(equities, b.initialBalance+endingPnl)
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("error iterating daily equity: %w", err)
	}

	if len(equities) < 2 {
		return 0, nil
	}

	returns := make([]float64, 0, len(equities)-1)

	for i := 1; i < len(equities); i++ {
		prev := equities[i-1]
		if prev == 0 {
			// Skip undefined returns when prior equity is zero; this would
			// otherwise produce NaN/Inf and poison the statistics.
			continue
		}

		returns = append(returns, equities[i]/prev-1)
	}

	if len(returns) < 2 {
		return 0, nil
	}

	var sum float64
	for _, r := range returns {
		sum += r
	}

	mean := sum / float64(len(returns))

	var sqSum float64
	for _, r := range returns {
		diff := r - mean
		sqSum += diff * diff
	}

	// Sample standard deviation (n-1) so variance of a 2-point series is well
	// defined; matches common Sharpe ratio conventions.
	variance := sqSum / float64(len(returns)-1)
	if variance <= 0 {
		return 0, nil
	}

	stdev := math.Sqrt(variance)

	annualization := b.sharpeAnnualizationFactor

	var periodRiskFree float64
	if annualization > 0 {
		periodRiskFree = b.riskFreeRate / float64(annualization)
	}

	sharpe := (mean - periodRiskFree) / stdev
	if annualization > 0 {
		sharpe *= math.Sqrt(float64(annualization))
	}

	return sharpe, nil
}

// calculateTradeHoldingTime calculates the holding time statistics for a symbol.
// endTime is used to calculate holding time for open positions (positions not yet sold).
// Under FIFO, matches are row-number-aligned (first buy <-> first sell) via SQL.
// Under average-cost (LIFO for hold time), matches are produced by replaying a
// per-position-type lot stack in Go, emitting one duration per matched slice.
func (b *BacktestState) calculateTradeHoldingTime(symbol string, endTime time.Time, strategy PortfolioCalculationStrategy) (types.TradeHoldingTime, error) {
	if strategy == PortfolioCalculationAverageCost {
		return b.calculateTradeHoldingTimeLIFO(symbol, endTime)
	}

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
			COALESCE(AVG(duration), 0) as avg_duration,
			COALESCE(quantile_cont(duration, 0.5), 0) as median_duration,
			COALESCE(quantile_cont(duration, 0.25), 0) as p25,
			COALESCE(quantile_cont(duration, 0.75), 0) as p75,
			COALESCE(quantile_cont(duration, 0.9), 0) as p90,
			COALESCE(quantile_cont(duration, 0.95), 0) as p95,
			COALESCE(quantile_cont(duration, 0.99), 0) as p99
		FROM all_durations
	`

	var minDuration, maxDuration, avgDuration, medianDuration float64

	var p25, p75, p90, p95, p99 float64

	// Format endTime as ISO 8601 string for DuckDB compatibility
	endTimeStr := endTime.Format("2006-01-02 15:04:05")

	err := b.db.QueryRow(query, symbol, types.PurchaseTypeBuy, symbol, types.PurchaseTypeSell, endTimeStr).Scan(
		&minDuration,
		&maxDuration,
		&avgDuration,
		&medianDuration,
		&p25,
		&p75,
		&p90,
		&p95,
		&p99,
	)
	if err != nil {
		return types.TradeHoldingTime{}, fmt.Errorf("failed to calculate holding time: %w", err)
	}

	holdingTime := types.TradeHoldingTime{
		Min:    int(math.Round(minDuration)),
		Max:    int(math.Round(maxDuration)),
		Avg:    int(math.Round(avgDuration)),
		Median: int(math.Round(medianDuration)),
		Percentiles: types.Percentiles{
			P25: p25,
			P50: medianDuration,
			P75: p75,
			P90: p90,
			P95: p95,
			P99: p99,
		},
	}

	return holdingTime, nil
}

// calculateTradeHoldingTimeLIFO computes min/max/avg holding time by replaying
// the full trade history in Go and matching exits against the most recently
// acquired lots (LIFO). Long and short positions are tracked independently.
// Each matched slice emits one duration record; unmatched open lots at the end
// emit (endTime - lot.executedAt) to mirror the FIFO-SQL behaviour for open
// positions.
func (b *BacktestState) calculateTradeHoldingTimeLIFO(symbol string, endTime time.Time) (types.TradeHoldingTime, error) {
	tradesQuery := b.sq.
		Select("order_type", "position_type", "executed_qty", "executed_at").
		From("trades").
		Where(squirrel.Eq{"symbol": symbol}).
		OrderBy("executed_at ASC", "rowid ASC").
		RunWith(b.db)

	rows, err := tradesQuery.Query()
	if err != nil {
		return types.TradeHoldingTime{}, fmt.Errorf("failed to query trades for LIFO holding time: %w", err)
	}
	defer rows.Close()

	type lot struct {
		qty        float64
		executedAt time.Time
	}

	stacks := map[types.PositionType][]lot{}

	var durations []float64

	for rows.Next() {
		var (
			orderType    string
			positionType string
			qty          float64
			executedAt   time.Time
		)

		if err := rows.Scan(&orderType, &positionType, &qty, &executedAt); err != nil {
			return types.TradeHoldingTime{}, fmt.Errorf("failed to scan trade for LIFO holding time: %w", err)
		}

		pt := types.PositionType(positionType)

		if types.PurchaseType(orderType) == types.PurchaseTypeBuy {
			stacks[pt] = append(stacks[pt], lot{qty: qty, executedAt: executedAt})

			continue
		}

		// SELL: pop LIFO, emitting one duration per consumed slice.
		remaining := qty
		stack := stacks[pt]

		for remaining > 0 && len(stack) > 0 {
			top := &stack[len(stack)-1]
			matched := math.Min(top.qty, remaining)
			duration := executedAt.Sub(top.executedAt).Seconds()
			if duration < 0 {
				duration = 0
			}

			durations = append(durations, duration)

			top.qty -= matched
			remaining -= matched

			if top.qty <= 0 {
				stack = stack[:len(stack)-1]
			}
		}

		stacks[pt] = stack
	}

	if err := rows.Err(); err != nil {
		return types.TradeHoldingTime{}, fmt.Errorf("error iterating trades for LIFO holding time: %w", err)
	}

	// Remaining lots are still-open positions; price hold time against endTime.
	for _, stack := range stacks {
		for _, l := range stack {
			duration := endTime.Sub(l.executedAt).Seconds()
			if duration < 0 {
				duration = 0
			}

			durations = append(durations, duration)
		}
	}

	if len(durations) == 0 {
		return types.TradeHoldingTime{Min: 0, Max: 0, Avg: 0, Median: 0, Percentiles: types.Percentiles{P25: 0, P50: 0, P75: 0, P90: 0, P95: 0, P99: 0}}, nil
	}

	minDur := durations[0]
	maxDur := durations[0]
	sum := 0.0

	for _, d := range durations {
		if d < minDur {
			minDur = d
		}

		if d > maxDur {
			maxDur = d
		}

		sum += d
	}

	avg := sum / float64(len(durations))

	sorted := make([]float64, len(durations))
	copy(sorted, durations)
	sort.Float64s(sorted)

	median := quantileCont(sorted, 0.5)

	return types.TradeHoldingTime{
		Min:    int(math.Round(minDur)),
		Max:    int(math.Round(maxDur)),
		Avg:    int(math.Round(avg)),
		Median: int(math.Round(median)),
		Percentiles: types.Percentiles{
			P25: quantileCont(sorted, 0.25),
			P50: median,
			P75: quantileCont(sorted, 0.75),
			P90: quantileCont(sorted, 0.90),
			P95: quantileCont(sorted, 0.95),
			P99: quantileCont(sorted, 0.99),
		},
	}, nil
}

// quantileCont mirrors DuckDB's quantile_cont: linear interpolation between
// the two nearest ranks on a slice that is already sorted ascending.
func quantileCont(sorted []float64, q float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}

	if n == 1 {
		return sorted[0]
	}

	pos := q * float64(n-1)
	lo := int(pos)
	hi := lo + 1

	if hi >= n {
		return sorted[n-1]
	}

	return sorted[lo] + (sorted[hi]-sorted[lo])*(pos-float64(lo))
}

// calculateTradePnlStats computes median and percentile statistics over the
// per-trade realized PnL of all closing trades for a symbol. Closing trades are
// identified as fills with non-zero pnl (entry trades record pnl as zero).
func (b *BacktestState) calculateTradePnlStats(symbol string) (median float64, percentiles types.Percentiles, err error) {
	query := `
		SELECT
			COALESCE(quantile_cont(pnl, 0.5), 0) as median_pnl,
			COALESCE(quantile_cont(pnl, 0.25), 0) as p25,
			COALESCE(quantile_cont(pnl, 0.75), 0) as p75,
			COALESCE(quantile_cont(pnl, 0.9), 0) as p90,
			COALESCE(quantile_cont(pnl, 0.95), 0) as p95,
			COALESCE(quantile_cont(pnl, 0.99), 0) as p99
		FROM trades
		WHERE symbol = ? AND pnl != 0
	`

	var p25, p75, p90, p95, p99 float64

	if err = b.db.QueryRow(query, symbol).Scan(&median, &p25, &p75, &p90, &p95, &p99); err != nil {
		return 0, types.Percentiles{}, fmt.Errorf("failed to calculate trade pnl percentiles: %w", err)
	}

	percentiles = types.Percentiles{
		P25: p25,
		P50: median,
		P75: p75,
		P90: p90,
		P95: p95,
		P99: p99,
	}

	return median, percentiles, nil
}

// calculateTotalInvestment returns the gross capital deployed across all entry
// trades for a symbol. Both long and short positions are recorded with
// order_type = BUY for their entries (per engine_v1 semantics), so we sum the
// notional (executed_qty * executed_price) of every BUY fill. This is used as
// the denominator for PnL percentage and represents the actual capital put to
// work — distinct from the run-wide initial cash balance.
func (b *BacktestState) calculateTotalInvestment(symbol string) (float64, error) {
	query := `
		SELECT COALESCE(SUM(executed_qty * executed_price), 0)
		FROM trades
		WHERE symbol = ? AND order_type = ?
	`

	var totalInvestment float64
	if err := b.db.QueryRow(query, symbol, types.PurchaseTypeBuy).Scan(&totalInvestment); err != nil {
		return 0, fmt.Errorf("failed to calculate total investment: %w", err)
	}

	return totalInvestment, nil
}

// calculateMonthlyTradeStats returns per-month trade activity for a symbol.
// Months are formatted as YYYY-MM and ordered chronologically. NumberOfTrades
// counts every fill executed in the month (entries and exits). NumberOfTradingPairs
// counts closing trades (sell trades for long positions) and is also the
// denominator for win/lose counts which use the per-trade pnl sign.
func (b *BacktestState) calculateMonthlyTradeStats(symbol string) ([]types.MonthlyTradeStats, error) {
	query := `
		SELECT
			strftime(date_trunc('month', executed_at), '%Y-%m') as month,
			COUNT(*) as total_trades,
			SUM(CASE WHEN order_type = ? THEN 1 ELSE 0 END) as trading_pairs,
			SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END) as winning_trades,
			SUM(CASE WHEN pnl < 0 THEN 1 ELSE 0 END) as losing_trades
		FROM trades
		WHERE symbol = ?
		GROUP BY month
		ORDER BY month
	`

	rows, err := b.db.Query(query, types.PurchaseTypeSell, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to query monthly trade stats: %w", err)
	}
	defer rows.Close()

	var result []types.MonthlyTradeStats

	for rows.Next() {
		var stat types.MonthlyTradeStats
		if err := rows.Scan(&stat.Month, &stat.NumberOfTrades, &stat.NumberOfTradingPairs, &stat.NumberOfWinningTrades, &stat.NumberOfLosingTrades); err != nil {
			return nil, fmt.Errorf("failed to scan monthly trade stat: %w", err)
		}

		result = append(result, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating monthly trade stats: %w", err)
	}

	return result, nil
}

// calculateMonthlyBalance returns per-month equity balance evolution for a
// symbol. Equity for a trade is computed as initial_balance + cumulative_pnl
// at the time of the trade. StartingBalance for the first month with trades is
// the initial balance; subsequent months use the previous month's ending
// balance as starting balance.
func (b *BacktestState) calculateMonthlyBalance(symbol string) ([]types.MonthlyBalanceChange, error) {
	query := `
		SELECT
			strftime(date_trunc('month', executed_at), '%Y-%m') as month,
			arg_max(cumulative_pnl, executed_at) as ending_pnl,
			COALESCE(SUM(pnl), 0) as realized_pnl
		FROM trades
		WHERE symbol = ?
		GROUP BY month
		ORDER BY month
	`

	rows, err := b.db.Query(query, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to query monthly balance: %w", err)
	}
	defer rows.Close()

	var (
		result      []types.MonthlyBalanceChange
		prevEnding  = b.initialBalance
		hasPrevious bool
	)

	for rows.Next() {
		var (
			month       string
			endingPnl   float64
			realizedPnl float64
		)

		if err := rows.Scan(&month, &endingPnl, &realizedPnl); err != nil {
			return nil, fmt.Errorf("failed to scan monthly balance: %w", err)
		}

		ending := b.initialBalance + endingPnl

		starting := b.initialBalance
		if hasPrevious {
			starting = prevEnding
		}

		result = append(result, types.MonthlyBalanceChange{
			Month:           month,
			StartingBalance: starting,
			EndingBalance:   ending,
			Change:          ending - starting,
			RealizedPnL:     realizedPnl,
		})

		prevEnding = ending
		hasPrevious = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating monthly balance: %w", err)
	}

	return result, nil
}

// calculateMonthlyHoldingTime returns per-month holding-time statistics for
// closing trades exited within each month. Uses the per-trade hold_time stored
// on closing trades (entry trades have hold_time = 0 and are excluded).
func (b *BacktestState) calculateMonthlyHoldingTime(symbol string) ([]types.MonthlyHoldingTime, error) {
	query := `
		SELECT
			strftime(date_trunc('month', executed_at), '%Y-%m') as month,
			COALESCE(MIN(hold_time), 0) as min_hold,
			COALESCE(MAX(hold_time), 0) as max_hold,
			COALESCE(AVG(hold_time), 0) as avg_hold,
			COALESCE(quantile_cont(hold_time, 0.5), 0) as median_hold
		FROM trades
		WHERE symbol = ? AND hold_time > 0
		GROUP BY month
		ORDER BY month
	`

	rows, err := b.db.Query(query, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to query monthly holding time: %w", err)
	}
	defer rows.Close()

	var result []types.MonthlyHoldingTime

	for rows.Next() {
		var (
			month                                 string
			minHold, maxHold, avgHold, medianHold float64
		)

		if err := rows.Scan(&month, &minHold, &maxHold, &avgHold, &medianHold); err != nil {
			return nil, fmt.Errorf("failed to scan monthly holding time: %w", err)
		}

		result = append(result, types.MonthlyHoldingTime{
			Month:  month,
			Min:    int(math.Round(minHold)),
			Max:    int(math.Round(maxHold)),
			Avg:    int(math.Round(avgHold)),
			Median: int(math.Round(medianHold)),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating monthly holding time: %w", err)
	}

	return result, nil
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

// getTradeSymbols returns all unique symbols that have trades in the database.
func (b *BacktestState) getTradeSymbols() ([]string, error) {
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

	return symbols, nil
}

// calculateMaxLossProfit retrieves the maximum loss and profit for a symbol.
func (b *BacktestState) calculateMaxLossProfit(symbol string) (maxLoss, maxProfit float64, err error) {
	query := b.sq.
		Select("COALESCE(MIN(pnl), 0) as max_loss", "COALESCE(MAX(pnl), 0) as max_profit").
		From("trades").
		Where(squirrel.Eq{"symbol": symbol}).
		RunWith(b.db)

	err = query.QueryRow().Scan(&maxLoss, &maxProfit)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to calculate max loss/profit: %w", err)
	}

	return maxLoss, maxProfit, nil
}

// calculateSymbolStats calculates trade statistics for a single symbol.
func (b *BacktestState) calculateSymbolStats(ctx runtime.RuntimeContext, symbol string, hasTrades bool, params statsParams) (types.TradeStats, error) {
	lastData, err := ctx.DataSource.ReadLastData(symbol)
	if err != nil {
		return types.TradeStats{}, fmt.Errorf("failed to get last market data for %s: %w", symbol, err)
	}

	if !hasTrades {
		zero := createZeroStats(symbol, params, b.initialBalance)
		zero.PortfolioCalculation = string(b.portfolioStrategy)

		return zero, nil
	}

	tradeResult, err := b.calculateTradeResult(symbol)
	if err != nil {
		return types.TradeStats{}, err
	}

	sharpeRatio, err := b.calculateSharpeRatio(symbol)
	if err != nil {
		return types.TradeStats{}, err
	}

	tradeResult.SharpeRatio = sharpeRatio

	holdingTime, err := b.calculateTradeHoldingTime(symbol, lastData.Time, b.portfolioStrategy)
	if err != nil {
		return types.TradeStats{}, err
	}

	totalFees, err := b.calculateTotalFees(symbol)
	if err != nil {
		return types.TradeStats{}, err
	}

	position, err := b.GetPosition(symbol)
	if err != nil {
		return types.TradeStats{}, err
	}

	tradePnl := calculateUnrealizedPnL(position, lastData.Close)

	maxLoss, maxProfit, err := b.calculateMaxLossProfit(symbol)
	if err != nil {
		return types.TradeStats{}, err
	}

	tradePnl.MaximumLoss = maxLoss
	tradePnl.MaximumProfit = maxProfit

	medianPnl, pnlPercentiles, err := b.calculateTradePnlStats(symbol)
	if err != nil {
		return types.TradeStats{}, err
	}

	tradePnl.MedianPnL = medianPnl
	tradePnl.Percentiles = pnlPercentiles

	totalInvestment, err := b.calculateTotalInvestment(symbol)
	if err != nil {
		return types.TradeStats{}, err
	}

	tradePnl.TotalInvestment = totalInvestment
	if totalInvestment > 0 {
		tradePnl.PnLPercentage = tradePnl.TotalPnL / totalInvestment
	}

	buyAndHoldPnl, err := b.calculateBuyAndHoldPnL(symbol, ctx.DataSource)
	if err != nil {
		return types.TradeStats{}, fmt.Errorf("failed to calculate buy-and-hold PnL: %w", err)
	}

	monthlyTrades, err := b.calculateMonthlyTradeStats(symbol)
	if err != nil {
		return types.TradeStats{}, err
	}

	monthlyBalance, err := b.calculateMonthlyBalance(symbol)
	if err != nil {
		return types.TradeStats{}, err
	}

	monthlyHoldingTime, err := b.calculateMonthlyHoldingTime(symbol)
	if err != nil {
		return types.TradeStats{}, err
	}

	// Calculate final balance as equity: initial balance + total PnL.
	// TotalPnL already includes fee impacts (fees are embedded in average entry/exit prices),
	// so we don't subtract fees separately.
	finalBalance := b.initialBalance + tradePnl.TotalPnL

	return types.TradeStats{
		ID:                   params.runID,
		Timestamp:            time.Now(),
		Symbol:               symbol,
		TradeResult:          tradeResult,
		TotalFees:            totalFees,
		TradeHoldingTime:     holdingTime,
		TradePnl:             tradePnl,
		BuyAndHoldPnl:        buyAndHoldPnl,
		TradesFilePath:       params.tradesFilePath,
		OrdersFilePath:       params.ordersFilePath,
		MarksFilePath:        params.marksFilePath,
		LogsFilePath:         params.logsFilePath,
		Strategy:             params.strategyInfo,
		StrategyPath:         params.strategyPath,
		DataPath:             params.dataPath,
		InitialBalance:       b.initialBalance,
		FinalBalance:         finalBalance,
		PortfolioCalculation: string(b.portfolioStrategy),
		BacktestConfig:       nil,
		StrategyConfig:       nil,
		MonthlyTrades:        monthlyTrades,
		MonthlyBalance:       monthlyBalance,
		MonthlyHoldingTime:   monthlyHoldingTime,
	}, nil
}
