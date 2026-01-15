package testhelper

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/log"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestingT is an interface that matches testing.T
type TestingT interface {
	require.TestingT
	TempDir() string
}

// ReadLiveStats reads live trade stats from the tmp folder.
// Returns all stats.yaml files found.
func ReadLiveStats(t TestingT, tmpFolder string) ([]types.LiveTradeStats, error) {
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

	if err != nil {
		return nil, err
	}

	if len(statsPaths) == 0 {
		return nil, fmt.Errorf("no stats.yaml files found in %s", tmpFolder)
	}

	var statsSlice []types.LiveTradeStats

	for _, statsPath := range statsPaths {
		content, err := os.ReadFile(statsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read stats file %s: %w", statsPath, err)
		}

		var stats types.LiveTradeStats
		err = yaml.Unmarshal(content, &stats)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal stats file %s: %w", statsPath, err)
		}

		statsSlice = append(statsSlice, stats)
	}

	return statsSlice, nil
}

// ReadTrades reads trades from parquet files in the tmp folder.
func ReadTrades(t TestingT, tmpFolder string) (trades []types.Trade, err error) {
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

	if err != nil {
		return nil, err
	}

	if len(tradesPaths) == 0 {
		return nil, fmt.Errorf("no trades.parquet files found in %s", tmpFolder)
	}

	// Read from all trades files
	for _, tradesPath := range tradesPaths {
		fileTrades, err := readTradesFromParquet(tradesPath)
		if err != nil {
			return nil, err
		}
		trades = append(trades, fileTrades...)
	}

	return trades, nil
}

func readTradesFromParquet(tradesPath string) ([]types.Trade, error) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	createViewSQL := fmt.Sprintf(`CREATE VIEW trades_view AS SELECT * FROM read_parquet('%s');`, tradesPath)
	_, err = db.Exec(createViewSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create view from parquet file: %w", err)
	}

	sq := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	query, args, err := sq.
		Select(
			"order_id", "symbol", "order_type", "quantity", "price", "timestamp", "is_completed",
			"reason", "message", "strategy_name", "commission",
			"executed_at", "executed_qty", "executed_price", "pnl",
		).
		From("trades_view").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL query: %w", err)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query trades: %w", err)
	}
	defer rows.Close()

	var trades []types.Trade
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

		order.Reason = types.Reason{
			Reason:  reason,
			Message: reasonMessage,
		}

		trade.Order = order
		trades = append(trades, trade)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating trade rows: %w", err)
	}

	return trades, nil
}

// ReadOrders reads orders from parquet files in the tmp folder.
func ReadOrders(t TestingT, tmpFolder string) (orders []types.Order, err error) {
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

	if err != nil {
		return nil, err
	}

	if len(ordersPaths) == 0 {
		return nil, fmt.Errorf("no orders.parquet files found in %s", tmpFolder)
	}

	for _, ordersPath := range ordersPaths {
		fileOrders, err := readOrdersFromParquet(ordersPath)
		if err != nil {
			return nil, err
		}
		orders = append(orders, fileOrders...)
	}

	return orders, nil
}

func readOrdersFromParquet(ordersPath string) ([]types.Order, error) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	createViewSQL := fmt.Sprintf(`CREATE VIEW orders_view AS SELECT * FROM read_parquet('%s');`, ordersPath)
	_, err = db.Exec(createViewSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create view from parquet file: %w", err)
	}

	sq := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	query, args, err := sq.
		Select(
			"order_id", "symbol", "order_type", "quantity", "price", "timestamp", "is_completed",
			"status", "reason", "message", "strategy_name",
		).
		From("orders_view").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL query: %w", err)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	var orders []types.Order
	for rows.Next() {
		var (
			order         types.Order
			status        string
			reason        string
			reasonMessage string
		)

		err := rows.Scan(
			&order.OrderID, &order.Symbol, &order.Side, &order.Quantity,
			&order.Price, &order.Timestamp, &order.IsCompleted,
			&status, &reason, &reasonMessage, &order.StrategyName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order row: %w", err)
		}

		order.Status = types.OrderStatus(status)
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

// ReadMarks reads marks from parquet files in the tmp folder.
func ReadMarks(t TestingT, tmpFolder string) (marks []types.Mark, err error) {
	var marksPaths []string

	err = filepath.Walk(tmpFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Base(path) == "marks.parquet" {
			marksPaths = append(marksPaths, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(marksPaths) == 0 {
		return nil, fmt.Errorf("no marks.parquet files found in %s", tmpFolder)
	}

	for _, marksPath := range marksPaths {
		fileMarks, err := readMarksFromParquet(marksPath)
		if err != nil {
			return nil, err
		}
		marks = append(marks, fileMarks...)
	}

	return marks, nil
}

func readMarksFromParquet(marksPath string) ([]types.Mark, error) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	createViewSQL := fmt.Sprintf(`CREATE VIEW marks_view AS SELECT * FROM read_parquet('%s');`, marksPath)
	_, err = db.Exec(createViewSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create view from parquet file: %w", err)
	}

	sq := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	query, args, err := sq.
		Select(
			"id", "market_data_id", "signal_type", "signal_name", "signal_time", "signal_symbol",
			"color", "shape", "level", "title", "message", "category",
		).
		From("marks_view").
		OrderBy("id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL query: %w", err)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query marks: %w", err)
	}
	defer rows.Close()

	var marks []types.Mark
	for rows.Next() {
		var (
			id           int
			marketDataID string
			signalType   sql.NullString
			signalName   sql.NullString
			signalTime   sql.NullTime
			signalSymbol sql.NullString
			color        string
			shapeStr     string
			levelStr     string
			title        string
			message      string
			category     string
		)

		err := rows.Scan(
			&id,
			&marketDataID,
			&signalType,
			&signalName,
			&signalTime,
			&signalSymbol,
			&color,
			&shapeStr,
			&levelStr,
			&title,
			&message,
			&category,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan mark row: %w", err)
		}

		mark := types.Mark{
			MarketDataId: marketDataID,
			Color:        types.MarkColor(color),
			Shape:        types.MarkShape(shapeStr),
			Level:        types.MarkLevel(levelStr),
			Title:        title,
			Message:      message,
			Category:     category,
		}

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

// ReadLogs reads logs from parquet files in the tmp folder.
func ReadLogs(t TestingT, tmpFolder string) (logs []log.LogEntry, err error) {
	var logsPaths []string

	err = filepath.Walk(tmpFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Base(path) == "logs.parquet" {
			logsPaths = append(logsPaths, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(logsPaths) == 0 {
		return nil, fmt.Errorf("no logs.parquet files found in %s", tmpFolder)
	}

	for _, logsPath := range logsPaths {
		fileLogs, err := readLogsFromParquet(logsPath)
		if err != nil {
			return nil, err
		}
		logs = append(logs, fileLogs...)
	}

	return logs, nil
}

func readLogsFromParquet(logsPath string) ([]log.LogEntry, error) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	createViewSQL := fmt.Sprintf(`CREATE VIEW logs_view AS SELECT * FROM read_parquet('%s');`, logsPath)
	_, err = db.Exec(createViewSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create view from parquet file: %w", err)
	}

	sq := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	query, args, err := sq.
		Select("id", "timestamp", "symbol", "level", "message", "fields").
		From("logs_view").
		OrderBy("id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL query: %w", err)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var logs []log.LogEntry
	for rows.Next() {
		var (
			id         int
			entry      log.LogEntry
			levelStr   string
			fieldsJSON sql.NullString
		)

		err := rows.Scan(
			&id,
			&entry.Timestamp,
			&entry.Symbol,
			&levelStr,
			&entry.Message,
			&fieldsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log entry row: %w", err)
		}

		entry.Level = types.LogLevel(levelStr)

		if fieldsJSON.Valid && fieldsJSON.String != "" {
			var fields map[string]string
			if err := json.Unmarshal([]byte(fieldsJSON.String), &fields); err != nil {
				return nil, fmt.Errorf("failed to unmarshal fields from JSON: %w", err)
			}
			entry.Fields = fields
		}

		logs = append(logs, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating log rows: %w", err)
	}

	return logs, nil
}

// GetRunFolders returns all run folders (run_1, run_2, etc.) in the tmp folder.
// The folders are sorted numerically.
func GetRunFolders(tmpFolder string) ([]string, error) {
	var runFolders []string

	err := filepath.Walk(tmpFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && strings.HasPrefix(info.Name(), "run_") {
			runFolders = append(runFolders, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by run number
	slices.Sort(runFolders)

	return runFolders, nil
}
