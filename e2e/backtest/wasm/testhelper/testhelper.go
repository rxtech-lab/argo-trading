package testhelper

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Masterminds/squirrel"
	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine"
	v1 "github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/runtime/wasm"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"
)

// E2ETestSuite is a base test suite for E2E tests
type E2ETestSuite struct {
	suite.Suite
	Backtest engine.Engine
}

// SetupTest initializes the backtest engine
func (s *E2ETestSuite) SetupTest(engineConfig string) {
	// initialize backtest engine
	backtest := v1.NewBacktestEngineV1()
	err := backtest.Initialize(engineConfig)
	s.Require().NoError(err)

	// initialize strategy api
	l, err := logger.NewLogger()
	s.Require().NoError(err)

	dataSource, err := datasource.NewDataSource(":memory:", l)
	s.Require().NoError(err)

	err = backtest.SetDataSource(dataSource)
	s.Require().NoError(err)

	s.Backtest = backtest
}

// RunWasmStrategyTest runs a test for a WASM strategy
func RunWasmStrategyTest(s *E2ETestSuite, strategyName string, wasmPath string, dataPath string) (tmpFolder string) {
	type config struct {
		FastPeriod int    `yaml:"fastPeriod"`
		SlowPeriod int    `yaml:"slowPeriod"`
		Symbol     string `yaml:"symbol"`
	}

	cfg := config{
		FastPeriod: 10,
		SlowPeriod: 20,
		Symbol:     "BTCUSDT",
	}

	cfgBytes, err := yaml.Marshal(cfg)
	require.NoError(s.T(), err)

	// write config to file
	tmpFolder = s.T().TempDir()
	configPath := filepath.Join(tmpFolder, "config", "config.yaml")
	resultPath := filepath.Join(tmpFolder, "results")

	// create config folder
	err = os.MkdirAll(filepath.Dir(configPath), 0755)
	require.NoError(s.T(), err)

	// write config to file
	err = os.WriteFile(configPath, cfgBytes, 0644)
	require.NoError(s.T(), err)

	err = s.Backtest.Initialize("")
	require.NoError(s.T(), err)

	runtime, err := wasm.NewStrategyWasmRuntime(wasmPath)
	require.NoError(s.T(), err)

	// generate schema
	_, err = runtime.GetConfigSchema()
	require.NoError(s.T(), err)

	if dataPath == "" {
		dataPath = "../../../../internal/indicator/test_data/test_data.parquet"
	}
	err = s.Backtest.SetDataPath(dataPath)
	require.NoError(s.T(), err)

	err = s.Backtest.LoadStrategy(runtime)
	require.NoError(s.T(), err)

	err = s.Backtest.SetResultsFolder(resultPath)
	require.NoError(s.T(), err)

	// set config path
	err = s.Backtest.SetConfigPath(configPath)
	require.NoError(s.T(), err)

	err = s.Backtest.Run(optional.None[engine.OnProcessDataCallback]())
	require.NoError(s.T(), err)

	return tmpFolder
}

// ReadStats reads the stats from the tmp folder
func ReadStats(s *E2ETestSuite, tmpFolder string) ([]types.TradeStats, error) {
	var statsPaths []string

	err := filepath.Walk(tmpFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Base(path) == "stats.yaml" {
			statsPaths = append(statsPaths, path)
		}
		return nil
	})

	require.NoError(s.T(), err)
	// require at least one stats file
	require.Greater(s.T(), len(statsPaths), 0)

	// read the first stats file
	var statsSlice []types.TradeStats

	for _, statsPath := range statsPaths {
		content, err := os.ReadFile(statsPath)
		require.NoError(s.T(), err)

		var stats []types.TradeStats
		err = yaml.Unmarshal(content, &stats)
		require.NoError(s.T(), err)

		if len(stats) > 0 {
			statsSlice = append(statsSlice, stats[0])
		}
	}

	// Return the entire stats slice
	return statsSlice, nil
}

// ReadTrades reads the trades from the tmp folder
func ReadTrades(s *E2ETestSuite, tmpFolder string) (trades []types.Trade, err error) {
	var tradesPaths []string

	err = filepath.Walk(tmpFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Base(path) == "trades.parquet" {
			tradesPaths = append(tradesPaths, path)
		}
		return nil
	})

	require.NoError(s.T(), err)
	// require at least one trades file
	require.Greater(s.T(), len(tradesPaths), 0)

	// read the first trades file
	tradesPath := tradesPaths[0]

	// Create an in-memory DuckDB instance for reading the parquet file
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	// Create a view from the parquet file - using raw SQL as Squirrel doesn't support CREATE VIEW
	createViewSQL := fmt.Sprintf(`CREATE VIEW trades_view AS SELECT * FROM read_parquet('%s');`, tradesPath)
	_, err = db.Exec(createViewSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create view from parquet file: %w", err)
	}

	// Initialize Squirrel with dollar placeholder format
	sq := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	// Construct query using Squirrel with corrected column name (order_type instead of side)
	query, args, err := sq.
		Select(
			"order_id", "symbol", "order_type", "quantity", "price", "timestamp", "is_completed",
			"reason", "message", "strategy_name", "commission",
			"executed_at", "executed_qty", "executed_price", "commission", "pnl",
		).
		From("trades_view").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL query: %w", err)
	}

	// Execute query
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query trades: %w", err)
	}
	defer rows.Close()

	// Scan rows into trade structs
	trades = []types.Trade{}
	for rows.Next() {
		var (
			trade         types.Trade
			order         types.Order
			reason        string
			reasonMessage string
		)

		err := rows.Scan(
			&order.OrderID, &order.Symbol, &order.Side, &order.Quantity,
			&order.Price, &order.Timestamp, &order.IsCompleted,
			&reason, &reasonMessage, &order.StrategyName, &order.Fee,
			&trade.ExecutedAt, &trade.ExecutedQty, &trade.ExecutedPrice,
			&trade.Fee, &trade.PnL,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trade row: %w", err)
		}

		// Set the Reason struct
		order.Reason = types.Reason{
			Reason:  reason,
			Message: reasonMessage,
		}

		// Set the Order field in Trade
		trade.Order = order
		trades = append(trades, trade)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating trade rows: %w", err)
	}

	return trades, nil
}

// ReadOrders reads the orders from the tmp folder
func ReadOrders(s *E2ETestSuite, tmpFolder string) (orders []types.Order, err error) {
	var ordersPaths []string

	err = filepath.Walk(tmpFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Base(path) == "orders.parquet" {
			ordersPaths = append(ordersPaths, path)
		}
		return nil
	})

	require.NoError(s.T(), err)
	// require at least one orders file
	require.Greater(s.T(), len(ordersPaths), 0)

	// read the first orders file
	ordersPath := ordersPaths[0]

	// Create an in-memory DuckDB instance for reading the parquet file
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	// Create a view from the parquet file - using raw SQL as Squirrel doesn't support CREATE VIEW
	createViewSQL := fmt.Sprintf(`CREATE VIEW orders_view AS SELECT * FROM read_parquet('%s');`, ordersPath)
	_, err = db.Exec(createViewSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create view from parquet file: %w", err)
	}

	// Initialize Squirrel with dollar placeholder format
	sq := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	// Construct query using Squirrel
	query, args, err := sq.
		Select(
			"order_id", "symbol", "side", "quantity", "price", "timestamp", "is_completed",
			"reason", "reason_message", "strategy_name", "fee",
		).
		From("orders_view").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL query: %w", err)
	}

	// Execute query
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	// Scan rows into order structs
	orders = []types.Order{}
	for rows.Next() {
		var (
			order         types.Order
			reason        string
			reasonMessage string
		)

		err := rows.Scan(
			&order.OrderID, &order.Symbol, &order.Side, &order.Quantity,
			&order.Price, &order.Timestamp, &order.IsCompleted,
			&reason, &reasonMessage, &order.StrategyName, &order.Fee,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order row: %w", err)
		}

		// Set the Reason struct
		order.Reason = types.Reason{
			Reason:  reason,
			Message: reasonMessage,
		}

		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating order rows: %w", err)
	}

	return orders, nil
}

func ReadMarker(s *E2ETestSuite, tmpFolder string) (marker []types.Mark, err error) {
	var markerPaths []string

	err = filepath.Walk(tmpFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Base(path) == "marks.parquet" {
			markerPaths = append(markerPaths, path)
		}
		return nil
	})

	require.NoError(s.T(), err)
	// require at least one marks file
	require.Greater(s.T(), len(markerPaths), 0)

	// read the first marks file
	marksPath := markerPaths[0]

	// Create an in-memory DuckDB instance for reading the parquet file
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	// Create a view from the parquet file - using raw SQL as Squirrel doesn't support CREATE VIEW
	createViewSQL := fmt.Sprintf(`CREATE VIEW marks_view AS SELECT * FROM read_parquet('%s');`, marksPath)
	_, err = db.Exec(createViewSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create view from parquet file: %w", err)
	}

	// Initialize Squirrel with dollar placeholder format
	sq := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	// Construct query using Squirrel with the new schema
	query, args, err := sq.
		Select(
			"id", "market_data_id", "signal_type", "signal_name", "signal_time", "signal_symbol",
			"color", "shape", "title", "message", "category",
		).
		From("marks_view").
		OrderBy("id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL query: %w", err)
	}

	// Execute query
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query marks: %w", err)
	}
	defer rows.Close()

	// Scan rows into mark structs
	marks := []types.Mark{}
	for rows.Next() {
		var (
			id           int
			marketDataId string
			signalType   sql.NullString
			signalName   sql.NullString
			signalTime   sql.NullTime
			signalSymbol sql.NullString
			color        string
			shapeStr     string
			title        string
			message      string
			category     string
		)

		err := rows.Scan(
			&id,
			&marketDataId,
			&signalType,
			&signalName,
			&signalTime,
			&signalSymbol,
			&color,
			&shapeStr,
			&title,
			&message,
			&category,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan mark row: %w", err)
		}

		// Create mark with the new fields
		mark := types.Mark{
			MarketDataId: marketDataId,
			Color:        color,
			Shape:        types.MarkShape(shapeStr),
			Title:        title,
			Message:      message,
			Category:     category,
		}

		// Add signal if present
		if signalTime.Valid && signalSymbol.Valid && signalType.Valid && signalName.Valid {
			signal := types.Signal{
				Type:   types.SignalType(signalType.String),
				Name:   signalName.String,
				Time:   signalTime.Time,
				Symbol: signalSymbol.String,
			}
			mark.Signal = optional.Some(signal)
		} else {
			mark.Signal = optional.None[types.Signal]()
		}

		marks = append(marks, mark)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating mark rows: %w", err)
	}

	return marks, nil
}
