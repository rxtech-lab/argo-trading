package writer

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/sirily11/argo-trading-go/src/types"
	"gopkg.in/yaml.v3"
)

// ResultWriter defines the interface for writing backtest results
type ResultWriter interface {
	// WriteTrade writes a trade to the output
	WriteTrade(trade types.Trade) error

	// WritePosition writes a position to the output
	WritePosition(position types.Position) error

	// WriteOrder writes an order to the output
	WriteOrder(order types.Order) error

	// WriteEquityCurve writes the equity curve data
	WriteEquityCurve(equityCurve []float64, timestamps []time.Time) error

	// WriteStats writes the trade statistics
	WriteStats(stats types.TradeStats) error

	// WriteStrategyStats writes statistics for a specific strategy
	WriteStrategyStats(strategyName string, stats types.TradeStats) error

	// Close finalizes the writing process
	Close() error
}

// CSVWriter implements ResultWriter by writing to CSV files
type CSVWriter struct {
	baseDir         string
	runDir          string
	tradesFile      *os.File
	positionsFile   *os.File
	ordersFile      *os.File
	equityCurveFile *os.File

	tradesCsv      *csv.Writer
	positionsCsv   *csv.Writer
	ordersCsv      *csv.Writer
	equityCurveCsv *csv.Writer
}

// NewCSVWriter creates a new CSVWriter with the given base directory
func NewCSVWriter(baseDir string) (ResultWriter, error) {
	// Create a directory for this run using current timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	runDir := filepath.Join(baseDir, timestamp)

	// Create the run directory
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create run directory: %w", err)
	}

	writer := &CSVWriter{
		baseDir: baseDir,
		runDir:  runDir,
	}

	// Initialize files
	if err := writer.initFiles(); err != nil {
		return nil, err
	}

	return writer, nil
}

// initFiles initializes all CSV files
func (w *CSVWriter) initFiles() error {
	// Create trades file
	tradesFile, err := os.Create(filepath.Join(w.runDir, "trades.csv"))
	if err != nil {
		return fmt.Errorf("failed to create trades file: %w", err)
	}
	w.tradesFile = tradesFile
	w.tradesCsv = csv.NewWriter(tradesFile)

	// Write header for trades file using gocsv
	tradeHeader := []types.Trade{}
	csvContent, err := gocsv.MarshalString(&tradeHeader)
	if err != nil {
		return fmt.Errorf("failed to get trade headers: %w", err)
	}
	// Extract just the header line
	headerLine := strings.Split(csvContent, "\n")[0]
	if headerLine != "" {
		if err := w.tradesCsv.Write(strings.Split(headerLine, ",")); err != nil {
			return fmt.Errorf("failed to write trades header: %w", err)
		}
		w.tradesCsv.Flush()
	}

	// Create positions file
	positionsFile, err := os.Create(filepath.Join(w.runDir, "positions.csv"))
	if err != nil {
		return fmt.Errorf("failed to create positions file: %w", err)
	}
	w.positionsFile = positionsFile
	w.positionsCsv = csv.NewWriter(positionsFile)

	// Write header for positions file using gocsv
	positionHeader := []types.Position{}
	csvContent, err = gocsv.MarshalString(&positionHeader)
	if err != nil {
		return fmt.Errorf("failed to get position headers: %w", err)
	}
	// Extract just the header line
	headerLine = strings.Split(csvContent, "\n")[0]
	if headerLine != "" {
		if err := w.positionsCsv.Write(strings.Split(headerLine, ",")); err != nil {
			return fmt.Errorf("failed to write positions header: %w", err)
		}
		w.positionsCsv.Flush()
	}

	// Create orders file
	ordersFile, err := os.Create(filepath.Join(w.runDir, "orders.csv"))
	if err != nil {
		return fmt.Errorf("failed to create orders file: %w", err)
	}
	w.ordersFile = ordersFile
	w.ordersCsv = csv.NewWriter(ordersFile)

	// Write header for orders file using gocsv
	orderHeader := []types.Order{}
	csvContent, err = gocsv.MarshalString(&orderHeader)
	if err != nil {
		return fmt.Errorf("failed to get order headers: %w", err)
	}
	// Extract just the header line
	headerLine = strings.Split(csvContent, "\n")[0]
	if headerLine != "" {
		if err := w.ordersCsv.Write(strings.Split(headerLine, ",")); err != nil {
			return fmt.Errorf("failed to write orders header: %w", err)
		}
		w.ordersCsv.Flush()
	}

	// Create equity curve file
	equityCurveFile, err := os.Create(filepath.Join(w.runDir, "equity_curve.csv"))
	if err != nil {
		return fmt.Errorf("failed to create equity curve file: %w", err)
	}
	w.equityCurveFile = equityCurveFile
	w.equityCurveCsv = csv.NewWriter(equityCurveFile)

	// Define a struct for equity curve with csv tags
	type EquityCurvePoint struct {
		Timestamp string  `csv:"timestamp"`
		Equity    float64 `csv:"equity"`
	}

	// Write header for equity curve file using gocsv
	equityCurveHeader := []EquityCurvePoint{}
	csvContent, err = gocsv.MarshalString(&equityCurveHeader)
	if err != nil {
		return fmt.Errorf("failed to get equity curve headers: %w", err)
	}
	// Extract just the header line
	headerLine = strings.Split(csvContent, "\n")[0]
	if headerLine != "" {
		if err := w.equityCurveCsv.Write(strings.Split(headerLine, ",")); err != nil {
			return fmt.Errorf("failed to write equity curve header: %w", err)
		}
		w.equityCurveCsv.Flush()
	}

	return nil
}

// WriteTrade writes a trade to the output
func (w *CSVWriter) WriteTrade(trade types.Trade) error {
	record := []string{
		trade.Order.Symbol,
		string(trade.Order.OrderType),
		trade.ExecutedAt.Format(time.RFC3339),
		fmt.Sprintf("%f", trade.ExecutedQty),
		fmt.Sprintf("%f", trade.ExecutedPrice),
		fmt.Sprintf("%f", trade.Commission),
		fmt.Sprintf("%f", trade.PnL),
		trade.Order.StrategyName,
	}

	if err := w.tradesCsv.Write(record); err != nil {
		return fmt.Errorf("failed to write trade: %w", err)
	}

	w.tradesCsv.Flush()
	return nil
}

// WritePosition writes a position to the output
func (w *CSVWriter) WritePosition(position types.Position) error {
	record := []string{
		position.Symbol,
		fmt.Sprintf("%f", position.Quantity),
		fmt.Sprintf("%f", position.AveragePrice),
		position.OpenTimestamp.Format(time.RFC3339),
	}

	if err := w.positionsCsv.Write(record); err != nil {
		return fmt.Errorf("failed to write position: %w", err)
	}

	w.positionsCsv.Flush()
	return nil
}

// WriteOrder writes an order to the output
func (w *CSVWriter) WriteOrder(order types.Order) error {
	// Create a buffer to hold the CSV data
	var buf bytes.Buffer

	// Marshal a single order to the buffer without headers
	if err := gocsv.MarshalWithoutHeaders([]*types.Order{&order}, &buf); err != nil {
		return fmt.Errorf("failed to marshal order: %w", err)
	}

	// Read the CSV line from the buffer
	csvLine := strings.TrimSpace(buf.String())

	// Parse the CSV line into fields
	r := csv.NewReader(strings.NewReader(csvLine))
	record, err := r.Read()
	if err != nil {
		return fmt.Errorf("failed to parse CSV line: %w", err)
	}

	// Write the record to the CSV writer
	if err := w.ordersCsv.Write(record); err != nil {
		return fmt.Errorf("failed to write order: %w", err)
	}

	w.ordersCsv.Flush()
	return nil
}

// WriteEquityCurve writes the equity curve data
func (w *CSVWriter) WriteEquityCurve(equityCurve []float64, timestamps []time.Time) error {
	// If timestamps are not provided, use current time
	if len(timestamps) != len(equityCurve) {
		timestamp := time.Now().Format(time.RFC3339)
		for _, equity := range equityCurve {
			record := []string{
				timestamp,
				fmt.Sprintf("%f", equity),
			}

			if err := w.equityCurveCsv.Write(record); err != nil {
				return fmt.Errorf("failed to write equity curve point: %w", err)
			}
		}
	} else {
		for i, equity := range equityCurve {
			record := []string{
				timestamps[i].Format(time.RFC3339),
				fmt.Sprintf("%f", equity),
			}

			if err := w.equityCurveCsv.Write(record); err != nil {
				return fmt.Errorf("failed to write equity curve point: %w", err)
			}
		}
	}

	w.equityCurveCsv.Flush()
	return nil
}

// WriteStats writes the trade statistics
func (w *CSVWriter) WriteStats(stats types.TradeStats) error {
	statsFile, err := os.Create(filepath.Join(w.runDir, "stats.yaml"))
	if err != nil {
		return fmt.Errorf("failed to create stats file: %w", err)
	}
	defer statsFile.Close()

	data, err := yaml.Marshal(stats)
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	if _, err := statsFile.Write(data); err != nil {
		return fmt.Errorf("failed to write stats: %w", err)
	}

	return nil
}

// WriteStrategyStats writes statistics for a specific strategy
func (w *CSVWriter) WriteStrategyStats(strategyName string, stats types.TradeStats) error {
	statsDir := filepath.Join(w.runDir, "strategy_stats")
	if err := os.MkdirAll(statsDir, 0755); err != nil {
		return fmt.Errorf("failed to create strategy stats directory: %w", err)
	}

	statsFile, err := os.Create(filepath.Join(statsDir, strategyName+".yaml"))
	if err != nil {
		return fmt.Errorf("failed to create strategy stats file: %w", err)
	}
	defer statsFile.Close()

	data, err := yaml.Marshal(stats)
	if err != nil {
		return fmt.Errorf("failed to marshal strategy stats: %w", err)
	}

	if _, err := statsFile.Write(data); err != nil {
		return fmt.Errorf("failed to write strategy stats: %w", err)
	}

	return nil
}

// Close finalizes the writing process
func (w *CSVWriter) Close() error {
	// Flush and close all files
	if w.tradesCsv != nil {
		w.tradesCsv.Flush()
	}
	if w.positionsCsv != nil {
		w.positionsCsv.Flush()
	}
	if w.ordersCsv != nil {
		w.ordersCsv.Flush()
	}
	if w.equityCurveCsv != nil {
		w.equityCurveCsv.Flush()
	}

	// Close all files
	if w.tradesFile != nil {
		w.tradesFile.Close()
	}
	if w.positionsFile != nil {
		w.positionsFile.Close()
	}
	if w.ordersFile != nil {
		w.ordersFile.Close()
	}
	if w.equityCurveFile != nil {
		w.equityCurveFile.Close()
	}

	return nil
}
