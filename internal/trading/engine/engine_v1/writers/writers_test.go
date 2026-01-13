package writers

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/log"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

type WritersTestSuite struct {
	suite.Suite
	tempDir string
}

func (s *WritersTestSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "writers_test_*")
	s.Require().NoError(err)
	s.tempDir = tempDir
}

func (s *WritersTestSuite) TearDownTest() {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}

func TestWritersTestSuite(t *testing.T) {
	suite.Run(t, new(WritersTestSuite))
}

// ============================================================================
// OrdersWriter Edge Case Tests
// ============================================================================

func (s *WritersTestSuite) TestOrdersWriter_Write_NotInitialized() {
	outputPath := filepath.Join(s.tempDir, "orders.parquet")
	w := NewOrdersWriter(outputPath)
	// Don't call Initialize

	order := types.Order{
		OrderID:      "order-123",
		Symbol:       "BTCUSDT",
		Side:         types.PurchaseTypeBuy,
		Quantity:     1.0,
		Price:        50000.0,
		Timestamp:    time.Now(),
		IsCompleted:  false,
		Status:       types.OrderStatusPending,
		Reason:       types.Reason{Reason: "strategy", Message: "Test"},
		StrategyName: "test",
		PositionType: types.PositionTypeLong,
	}

	err := w.Write(order)
	s.Error(err)
	s.Contains(err.Error(), "writer not initialized")
}

func (s *WritersTestSuite) TestOrdersWriter_Flush_NotInitialized() {
	outputPath := filepath.Join(s.tempDir, "orders.parquet")
	w := NewOrdersWriter(outputPath)

	err := w.Flush()
	s.Error(err)
	s.Contains(err.Error(), "writer not initialized")
}

func (s *WritersTestSuite) TestOrdersWriter_Flush_Success() {
	outputPath := filepath.Join(s.tempDir, "orders.parquet")
	w := NewOrdersWriter(outputPath)

	err := w.Initialize()
	s.Require().NoError(err)
	defer w.Close()

	// Write an order first
	order := types.Order{
		OrderID:      "order-456",
		Symbol:       "ETHUSDT",
		Side:         types.PurchaseTypeSell,
		Quantity:     2.0,
		Price:        3000.0,
		Timestamp:    time.Now(),
		IsCompleted:  true,
		Status:       types.OrderStatusFilled,
		Reason:       types.Reason{Reason: "strategy", Message: "Test"},
		StrategyName: "test",
		PositionType: types.PositionTypeLong,
	}

	err = w.Write(order)
	s.Require().NoError(err)

	// Flush should succeed
	err = w.Flush()
	s.NoError(err)

	// Verify file exists
	s.FileExists(outputPath)
}

func (s *WritersTestSuite) TestOrdersWriter_Close_NotInitialized() {
	outputPath := filepath.Join(s.tempDir, "orders.parquet")
	w := NewOrdersWriter(outputPath)

	// Close without Initialize should be no-op
	err := w.Close()
	s.NoError(err)
}

func (s *WritersTestSuite) TestOrdersWriter_GetOrderCount_NotInitialized() {
	outputPath := filepath.Join(s.tempDir, "orders.parquet")
	w := NewOrdersWriter(outputPath)

	_, err := w.GetOrderCount()
	s.Error(err)
	s.Contains(err.Error(), "writer not initialized")
}

func (s *WritersTestSuite) TestOrdersWriter_GetOutputPath() {
	outputPath := filepath.Join(s.tempDir, "orders.parquet")
	w := NewOrdersWriter(outputPath)

	s.Equal(outputPath, w.GetOutputPath())
}

// Orders Writer Tests

func (s *WritersTestSuite) TestOrdersWriter_Initialize() {
	outputPath := filepath.Join(s.tempDir, "orders.parquet")
	w := NewOrdersWriter(outputPath)

	err := w.Initialize()
	s.Require().NoError(err)

	defer w.Close()

	count, err := w.GetOrderCount()
	s.Require().NoError(err)
	s.Equal(0, count)
}

func (s *WritersTestSuite) TestOrdersWriter_WriteAndCount() {
	outputPath := filepath.Join(s.tempDir, "orders.parquet")
	w := NewOrdersWriter(outputPath)

	err := w.Initialize()
	s.Require().NoError(err)

	defer w.Close()

	// Write an order
	order := types.Order{
		OrderID:     "order-123",
		Symbol:      "BTCUSDT",
		Side:        types.PurchaseTypeBuy,
		Quantity:    1.0,
		Price:       50000.0,
		Timestamp:   time.Now(),
		IsCompleted: false,
		Status:      types.OrderStatusPending,
		Reason: types.Reason{
			Reason:  "strategy",
			Message: "Test order",
		},
		StrategyName: "test-strategy",
		Fee:          0,
		PositionType: types.PositionTypeLong,
	}

	err = w.Write(order)
	s.Require().NoError(err)

	count, err := w.GetOrderCount()
	s.Require().NoError(err)
	s.Equal(1, count)

	// Verify parquet file was created
	s.FileExists(outputPath)
}

func (s *WritersTestSuite) TestOrdersWriter_Upsert() {
	outputPath := filepath.Join(s.tempDir, "orders.parquet")
	w := NewOrdersWriter(outputPath)

	err := w.Initialize()
	s.Require().NoError(err)

	defer w.Close()

	// Write initial order
	order := types.Order{
		OrderID:     "order-123",
		Symbol:      "BTCUSDT",
		Side:        types.PurchaseTypeBuy,
		Quantity:    1.0,
		Price:       50000.0,
		Timestamp:   time.Now(),
		IsCompleted: false,
		Status:      types.OrderStatusPending,
		Reason: types.Reason{
			Reason:  "strategy",
			Message: "Test order",
		},
		StrategyName: "test-strategy",
		Fee:          0,
		PositionType: types.PositionTypeLong,
	}

	err = w.Write(order)
	s.Require().NoError(err)

	// Update order status
	order.IsCompleted = true
	order.Status = types.OrderStatusFilled
	order.Fee = 10.0

	err = w.Write(order)
	s.Require().NoError(err)

	// Should still be 1 order (upsert, not insert)
	count, err := w.GetOrderCount()
	s.Require().NoError(err)
	s.Equal(1, count)
}

// Trades Writer Tests

func (s *WritersTestSuite) TestTradesWriter_Initialize() {
	outputPath := filepath.Join(s.tempDir, "trades.parquet")
	w := NewTradesWriter(outputPath)

	err := w.Initialize()
	s.Require().NoError(err)

	defer w.Close()

	count, err := w.GetTradeCount()
	s.Require().NoError(err)
	s.Equal(0, count)
}

func (s *WritersTestSuite) TestTradesWriter_WriteAndCount() {
	outputPath := filepath.Join(s.tempDir, "trades.parquet")
	w := NewTradesWriter(outputPath)

	err := w.Initialize()
	s.Require().NoError(err)

	defer w.Close()

	// Write a trade
	trade := types.Trade{
		Order: types.Order{
			OrderID:     "order-123",
			Symbol:      "BTCUSDT",
			Side:        types.PurchaseTypeSell,
			Quantity:    1.0,
			Price:       51000.0,
			Timestamp:   time.Now(),
			IsCompleted: true,
			Status:      types.OrderStatusFilled,
			Reason: types.Reason{
				Reason:  "strategy",
				Message: "Test trade",
			},
			StrategyName: "test-strategy",
			Fee:          10.0,
			PositionType: types.PositionTypeLong,
		},
		ExecutedAt:    time.Now(),
		ExecutedQty:   1.0,
		ExecutedPrice: 51000.0,
		Fee:           10.0,
		PnL:           1000.0,
	}

	err = w.Write(trade)
	s.Require().NoError(err)

	count, err := w.GetTradeCount()
	s.Require().NoError(err)
	s.Equal(1, count)

	// Verify parquet file was created
	s.FileExists(outputPath)
}

func (s *WritersTestSuite) TestTradesWriter_GetTotalPnL() {
	outputPath := filepath.Join(s.tempDir, "trades.parquet")
	w := NewTradesWriter(outputPath)

	err := w.Initialize()
	s.Require().NoError(err)

	defer w.Close()

	// Write multiple trades
	trades := []types.Trade{
		{
			Order: types.Order{
				OrderID:      "order-1",
				Symbol:       "BTCUSDT",
				Side:         types.PurchaseTypeSell,
				Quantity:     1.0,
				Price:        51000.0,
				Timestamp:    time.Now(),
				IsCompleted:  true,
				Status:       types.OrderStatusFilled,
				Reason:       types.Reason{Reason: "strategy", Message: "Test"},
				StrategyName: "test",
				Fee:          10.0,
				PositionType: types.PositionTypeLong,
			},
			ExecutedAt:    time.Now(),
			ExecutedQty:   1.0,
			ExecutedPrice: 51000.0,
			Fee:           10.0,
			PnL:           500.0,
		},
		{
			Order: types.Order{
				OrderID:      "order-2",
				Symbol:       "BTCUSDT",
				Side:         types.PurchaseTypeSell,
				Quantity:     1.0,
				Price:        52000.0,
				Timestamp:    time.Now(),
				IsCompleted:  true,
				Status:       types.OrderStatusFilled,
				Reason:       types.Reason{Reason: "strategy", Message: "Test"},
				StrategyName: "test",
				Fee:          10.0,
				PositionType: types.PositionTypeLong,
			},
			ExecutedAt:    time.Now(),
			ExecutedQty:   1.0,
			ExecutedPrice: 52000.0,
			Fee:           10.0,
			PnL:           -200.0,
		},
	}

	for _, trade := range trades {
		err = w.Write(trade)
		s.Require().NoError(err)
	}

	totalPnL, err := w.GetTotalPnL()
	s.Require().NoError(err)
	s.Equal(300.0, totalPnL) // 500 + (-200) = 300
}

// Marks Writer Tests

func (s *WritersTestSuite) TestMarksWriter_Initialize() {
	outputPath := filepath.Join(s.tempDir, "marks.parquet")
	w := NewMarksWriter(outputPath)

	err := w.Initialize()
	s.Require().NoError(err)

	defer w.Close()

	count, err := w.GetMarkCount()
	s.Require().NoError(err)
	s.Equal(0, count)
}

func (s *WritersTestSuite) TestMarksWriter_WriteAndCount() {
	outputPath := filepath.Join(s.tempDir, "marks.parquet")
	w := NewMarksWriter(outputPath)

	err := w.Initialize()
	s.Require().NoError(err)

	defer w.Close()

	// Write a mark without signal
	mark := types.Mark{
		MarketDataId: "md-123",
		Color:        types.MarkColorGreen,
		Shape:        types.MarkShapeCircle,
		Level:        types.MarkLevelInfo,
		Title:        "Buy Signal",
		Message:      "Strategy detected buy signal",
		Category:     "entry",
		Signal:       optional.None[types.Signal](),
	}

	err = w.Write(mark)
	s.Require().NoError(err)

	count, err := w.GetMarkCount()
	s.Require().NoError(err)
	s.Equal(1, count)

	// Verify parquet file was created
	s.FileExists(outputPath)
}

func (s *WritersTestSuite) TestMarksWriter_WriteWithSignal() {
	outputPath := filepath.Join(s.tempDir, "marks.parquet")
	w := NewMarksWriter(outputPath)

	err := w.Initialize()
	s.Require().NoError(err)

	defer w.Close()

	// Write a mark with signal
	signal := types.Signal{
		Time:      time.Now(),
		Type:      types.SignalTypeBuyLong,
		Name:      "RSI Oversold",
		Reason:    "RSI below 30",
		RawValue:  28.5,
		Symbol:    "BTCUSDT",
		Indicator: "RSI",
	}

	mark := types.Mark{
		MarketDataId: "md-123",
		Color:        types.MarkColorGreen,
		Shape:        types.MarkShapeCircle,
		Level:        types.MarkLevelInfo,
		Title:        "Buy Signal",
		Message:      "Strategy detected buy signal",
		Category:     "entry",
		Signal:       optional.Some(signal),
	}

	err = w.Write(mark)
	s.Require().NoError(err)

	count, err := w.GetMarkCount()
	s.Require().NoError(err)
	s.Equal(1, count)
}

// Logs Writer Tests

func (s *WritersTestSuite) TestLogsWriter_Initialize() {
	outputPath := filepath.Join(s.tempDir, "logs.parquet")
	w := NewLogsWriter(outputPath)

	err := w.Initialize()
	s.Require().NoError(err)

	defer w.Close()

	count, err := w.GetLogCount()
	s.Require().NoError(err)
	s.Equal(0, count)
}

func (s *WritersTestSuite) TestLogsWriter_WriteAndCount() {
	outputPath := filepath.Join(s.tempDir, "logs.parquet")
	w := NewLogsWriter(outputPath)

	err := w.Initialize()
	s.Require().NoError(err)

	defer w.Close()

	// Write a log entry
	entry := log.LogEntry{
		Timestamp: time.Now(),
		Symbol:    "BTCUSDT",
		Level:     types.LogLevelInfo,
		Message:   "Strategy started processing",
		Fields: map[string]string{
			"price":  "50000",
			"volume": "1.5",
		},
	}

	err = w.Write(entry)
	s.Require().NoError(err)

	count, err := w.GetLogCount()
	s.Require().NoError(err)
	s.Equal(1, count)

	// Verify parquet file was created
	s.FileExists(outputPath)
}

func (s *WritersTestSuite) TestLogsWriter_WriteWithoutFields() {
	outputPath := filepath.Join(s.tempDir, "logs.parquet")
	w := NewLogsWriter(outputPath)

	err := w.Initialize()
	s.Require().NoError(err)

	defer w.Close()

	// Write a log entry without fields
	entry := log.LogEntry{
		Timestamp: time.Now(),
		Symbol:    "BTCUSDT",
		Level:     types.LogLevelWarning,
		Message:   "Warning: Low volume detected",
		Fields:    nil,
	}

	err = w.Write(entry)
	s.Require().NoError(err)

	count, err := w.GetLogCount()
	s.Require().NoError(err)
	s.Equal(1, count)
}

// Persistence Tests

func (s *WritersTestSuite) TestOrdersWriter_Persistence() {
	outputPath := filepath.Join(s.tempDir, "orders.parquet")

	// Write with first instance
	w1 := NewOrdersWriter(outputPath)
	err := w1.Initialize()
	s.Require().NoError(err)

	order := types.Order{
		OrderID:      "order-123",
		Symbol:       "BTCUSDT",
		Side:         types.PurchaseTypeBuy,
		Quantity:     1.0,
		Price:        50000.0,
		Timestamp:    time.Now(),
		IsCompleted:  false,
		Status:       types.OrderStatusPending,
		Reason:       types.Reason{Reason: "strategy", Message: "Test"},
		StrategyName: "test",
		Fee:          0,
		PositionType: types.PositionTypeLong,
	}

	err = w1.Write(order)
	s.Require().NoError(err)
	w1.Close()

	// Read with second instance
	w2 := NewOrdersWriter(outputPath)
	err = w2.Initialize()
	s.Require().NoError(err)

	defer w2.Close()

	count, err := w2.GetOrderCount()
	s.Require().NoError(err)
	s.Equal(1, count) // Data persisted from first instance
}

func (s *WritersTestSuite) TestTradesWriter_Persistence() {
	outputPath := filepath.Join(s.tempDir, "trades.parquet")

	// Write with first instance
	w1 := NewTradesWriter(outputPath)
	err := w1.Initialize()
	s.Require().NoError(err)

	trade := types.Trade{
		Order: types.Order{
			OrderID:      "order-123",
			Symbol:       "BTCUSDT",
			Side:         types.PurchaseTypeSell,
			Quantity:     1.0,
			Price:        51000.0,
			Timestamp:    time.Now(),
			IsCompleted:  true,
			Status:       types.OrderStatusFilled,
			Reason:       types.Reason{Reason: "strategy", Message: "Test"},
			StrategyName: "test",
			Fee:          10.0,
			PositionType: types.PositionTypeLong,
		},
		ExecutedAt:    time.Now(),
		ExecutedQty:   1.0,
		ExecutedPrice: 51000.0,
		Fee:           10.0,
		PnL:           1000.0,
	}

	err = w1.Write(trade)
	s.Require().NoError(err)
	w1.Close()

	// Read with second instance
	w2 := NewTradesWriter(outputPath)
	err = w2.Initialize()
	s.Require().NoError(err)

	defer w2.Close()

	count, err := w2.GetTradeCount()
	s.Require().NoError(err)
	s.Equal(1, count)

	pnl, err := w2.GetTotalPnL()
	s.Require().NoError(err)
	s.Equal(1000.0, pnl)
}

// ============================================================================
// TradesWriter Edge Case Tests
// ============================================================================

func (s *WritersTestSuite) TestTradesWriter_Write_NotInitialized() {
	outputPath := filepath.Join(s.tempDir, "trades.parquet")
	w := NewTradesWriter(outputPath)

	trade := types.Trade{
		Order: types.Order{
			OrderID:      "order-123",
			Symbol:       "BTCUSDT",
			Side:         types.PurchaseTypeSell,
			Quantity:     1.0,
			Price:        51000.0,
			Timestamp:    time.Now(),
			IsCompleted:  true,
			Status:       types.OrderStatusFilled,
			Reason:       types.Reason{Reason: "strategy", Message: "Test"},
			StrategyName: "test",
			PositionType: types.PositionTypeLong,
		},
		ExecutedAt:    time.Now(),
		ExecutedQty:   1.0,
		ExecutedPrice: 51000.0,
		Fee:           10.0,
		PnL:           1000.0,
	}

	err := w.Write(trade)
	s.Error(err)
	s.Contains(err.Error(), "writer not initialized")
}

func (s *WritersTestSuite) TestTradesWriter_Flush_NotInitialized() {
	outputPath := filepath.Join(s.tempDir, "trades.parquet")
	w := NewTradesWriter(outputPath)

	err := w.Flush()
	s.Error(err)
	s.Contains(err.Error(), "writer not initialized")
}

func (s *WritersTestSuite) TestTradesWriter_Flush_Success() {
	outputPath := filepath.Join(s.tempDir, "trades.parquet")
	w := NewTradesWriter(outputPath)

	err := w.Initialize()
	s.Require().NoError(err)
	defer w.Close()

	trade := types.Trade{
		Order: types.Order{
			OrderID:      "order-flush",
			Symbol:       "BTCUSDT",
			Side:         types.PurchaseTypeSell,
			Quantity:     1.0,
			Price:        51000.0,
			Timestamp:    time.Now(),
			IsCompleted:  true,
			Status:       types.OrderStatusFilled,
			Reason:       types.Reason{Reason: "strategy", Message: "Test"},
			StrategyName: "test",
			PositionType: types.PositionTypeLong,
		},
		ExecutedAt:    time.Now(),
		ExecutedQty:   1.0,
		ExecutedPrice: 51000.0,
		Fee:           5.0,
		PnL:           500.0,
	}

	err = w.Write(trade)
	s.Require().NoError(err)

	err = w.Flush()
	s.NoError(err)
	s.FileExists(outputPath)
}

func (s *WritersTestSuite) TestTradesWriter_Close_NotInitialized() {
	outputPath := filepath.Join(s.tempDir, "trades.parquet")
	w := NewTradesWriter(outputPath)

	err := w.Close()
	s.NoError(err)
}

func (s *WritersTestSuite) TestTradesWriter_GetTradeCount_NotInitialized() {
	outputPath := filepath.Join(s.tempDir, "trades.parquet")
	w := NewTradesWriter(outputPath)

	_, err := w.GetTradeCount()
	s.Error(err)
	s.Contains(err.Error(), "writer not initialized")
}

func (s *WritersTestSuite) TestTradesWriter_GetTotalPnL_NotInitialized() {
	outputPath := filepath.Join(s.tempDir, "trades.parquet")
	w := NewTradesWriter(outputPath)

	_, err := w.GetTotalPnL()
	s.Error(err)
	s.Contains(err.Error(), "writer not initialized")
}

func (s *WritersTestSuite) TestTradesWriter_GetTotalPnL_NoTrades() {
	outputPath := filepath.Join(s.tempDir, "trades.parquet")
	w := NewTradesWriter(outputPath)

	err := w.Initialize()
	s.Require().NoError(err)
	defer w.Close()

	pnl, err := w.GetTotalPnL()
	s.NoError(err)
	s.Equal(0.0, pnl)
}

func (s *WritersTestSuite) TestTradesWriter_GetTotalFees() {
	outputPath := filepath.Join(s.tempDir, "trades.parquet")
	w := NewTradesWriter(outputPath)

	err := w.Initialize()
	s.Require().NoError(err)
	defer w.Close()

	trades := []types.Trade{
		{
			Order: types.Order{
				OrderID:      "order-1",
				Symbol:       "BTCUSDT",
				Side:         types.PurchaseTypeSell,
				Quantity:     1.0,
				Price:        51000.0,
				Timestamp:    time.Now(),
				IsCompleted:  true,
				Status:       types.OrderStatusFilled,
				Reason:       types.Reason{Reason: "strategy", Message: "Test"},
				StrategyName: "test",
				PositionType: types.PositionTypeLong,
			},
			ExecutedAt:    time.Now(),
			ExecutedQty:   1.0,
			ExecutedPrice: 51000.0,
			Fee:           10.0,
			PnL:           500.0,
		},
		{
			Order: types.Order{
				OrderID:      "order-2",
				Symbol:       "ETHUSDT",
				Side:         types.PurchaseTypeSell,
				Quantity:     2.0,
				Price:        3000.0,
				Timestamp:    time.Now(),
				IsCompleted:  true,
				Status:       types.OrderStatusFilled,
				Reason:       types.Reason{Reason: "strategy", Message: "Test"},
				StrategyName: "test",
				PositionType: types.PositionTypeLong,
			},
			ExecutedAt:    time.Now(),
			ExecutedQty:   2.0,
			ExecutedPrice: 3000.0,
			Fee:           15.0,
			PnL:           200.0,
		},
	}

	for _, trade := range trades {
		err = w.Write(trade)
		s.Require().NoError(err)
	}

	totalFees, err := w.GetTotalFees()
	s.Require().NoError(err)
	s.Equal(25.0, totalFees)
}

func (s *WritersTestSuite) TestTradesWriter_GetTotalFees_NoTrades() {
	outputPath := filepath.Join(s.tempDir, "trades.parquet")
	w := NewTradesWriter(outputPath)

	err := w.Initialize()
	s.Require().NoError(err)
	defer w.Close()

	fees, err := w.GetTotalFees()
	s.NoError(err)
	s.Equal(0.0, fees)
}

func (s *WritersTestSuite) TestTradesWriter_GetTotalFees_NotInitialized() {
	outputPath := filepath.Join(s.tempDir, "trades.parquet")
	w := NewTradesWriter(outputPath)

	_, err := w.GetTotalFees()
	s.Error(err)
	s.Contains(err.Error(), "writer not initialized")
}

func (s *WritersTestSuite) TestTradesWriter_GetOutputPath() {
	outputPath := filepath.Join(s.tempDir, "trades.parquet")
	w := NewTradesWriter(outputPath)

	s.Equal(outputPath, w.GetOutputPath())
}

// ============================================================================
// MarksWriter Edge Case Tests
// ============================================================================

func (s *WritersTestSuite) TestMarksWriter_Write_NotInitialized() {
	outputPath := filepath.Join(s.tempDir, "marks.parquet")
	w := NewMarksWriter(outputPath)

	mark := types.Mark{
		MarketDataId: "md-123",
		Color:        types.MarkColorGreen,
		Shape:        types.MarkShapeCircle,
		Level:        types.MarkLevelInfo,
		Title:        "Buy Signal",
		Message:      "Strategy detected buy signal",
		Category:     "entry",
		Signal:       optional.None[types.Signal](),
	}

	err := w.Write(mark)
	s.Error(err)
	s.Contains(err.Error(), "writer not initialized")
}

func (s *WritersTestSuite) TestMarksWriter_Flush_NotInitialized() {
	outputPath := filepath.Join(s.tempDir, "marks.parquet")
	w := NewMarksWriter(outputPath)

	err := w.Flush()
	s.Error(err)
	s.Contains(err.Error(), "writer not initialized")
}

func (s *WritersTestSuite) TestMarksWriter_Flush_Success() {
	outputPath := filepath.Join(s.tempDir, "marks.parquet")
	w := NewMarksWriter(outputPath)

	err := w.Initialize()
	s.Require().NoError(err)
	defer w.Close()

	mark := types.Mark{
		MarketDataId: "md-flush",
		Color:        types.MarkColorBlue,
		Shape:        types.MarkShapeSquare,
		Level:        types.MarkLevelInfo,
		Title:        "Info Mark",
		Message:      "Testing flush",
		Category:     "test",
		Signal:       optional.None[types.Signal](),
	}

	err = w.Write(mark)
	s.Require().NoError(err)

	err = w.Flush()
	s.NoError(err)
	s.FileExists(outputPath)
}

func (s *WritersTestSuite) TestMarksWriter_Close_NotInitialized() {
	outputPath := filepath.Join(s.tempDir, "marks.parquet")
	w := NewMarksWriter(outputPath)

	err := w.Close()
	s.NoError(err)
}

func (s *WritersTestSuite) TestMarksWriter_GetMarkCount_NotInitialized() {
	outputPath := filepath.Join(s.tempDir, "marks.parquet")
	w := NewMarksWriter(outputPath)

	_, err := w.GetMarkCount()
	s.Error(err)
	s.Contains(err.Error(), "writer not initialized")
}

func (s *WritersTestSuite) TestMarksWriter_GetOutputPath() {
	outputPath := filepath.Join(s.tempDir, "marks.parquet")
	w := NewMarksWriter(outputPath)

	s.Equal(outputPath, w.GetOutputPath())
}

func (s *WritersTestSuite) TestMarksWriter_Persistence() {
	outputPath := filepath.Join(s.tempDir, "marks.parquet")

	// Write with first instance
	w1 := NewMarksWriter(outputPath)
	err := w1.Initialize()
	s.Require().NoError(err)

	mark := types.Mark{
		MarketDataId: "md-persist",
		Color:        types.MarkColorRed,
		Shape:        types.MarkShapeTriangle,
		Level:        types.MarkLevelWarning,
		Title:        "Warning",
		Message:      "Test persistence",
		Category:     "test",
		Signal:       optional.None[types.Signal](),
	}

	err = w1.Write(mark)
	s.Require().NoError(err)
	w1.Close()

	// Read with second instance
	w2 := NewMarksWriter(outputPath)
	err = w2.Initialize()
	s.Require().NoError(err)
	defer w2.Close()

	count, err := w2.GetMarkCount()
	s.Require().NoError(err)
	s.Equal(1, count)
}

// ============================================================================
// LogsWriter Edge Case Tests
// ============================================================================

func (s *WritersTestSuite) TestLogsWriter_Write_NotInitialized() {
	outputPath := filepath.Join(s.tempDir, "logs.parquet")
	w := NewLogsWriter(outputPath)

	entry := log.LogEntry{
		Timestamp: time.Now(),
		Symbol:    "BTCUSDT",
		Level:     types.LogLevelInfo,
		Message:   "Test log",
		Fields:    nil,
	}

	err := w.Write(entry)
	s.Error(err)
	s.Contains(err.Error(), "writer not initialized")
}

func (s *WritersTestSuite) TestLogsWriter_Flush_NotInitialized() {
	outputPath := filepath.Join(s.tempDir, "logs.parquet")
	w := NewLogsWriter(outputPath)

	err := w.Flush()
	s.Error(err)
	s.Contains(err.Error(), "writer not initialized")
}

func (s *WritersTestSuite) TestLogsWriter_Flush_Success() {
	outputPath := filepath.Join(s.tempDir, "logs.parquet")
	w := NewLogsWriter(outputPath)

	err := w.Initialize()
	s.Require().NoError(err)
	defer w.Close()

	entry := log.LogEntry{
		Timestamp: time.Now(),
		Symbol:    "BTCUSDT",
		Level:     types.LogLevelDebug,
		Message:   "Testing flush",
		Fields:    map[string]string{"key": "value"},
	}

	err = w.Write(entry)
	s.Require().NoError(err)

	err = w.Flush()
	s.NoError(err)
	s.FileExists(outputPath)
}

func (s *WritersTestSuite) TestLogsWriter_Close_NotInitialized() {
	outputPath := filepath.Join(s.tempDir, "logs.parquet")
	w := NewLogsWriter(outputPath)

	err := w.Close()
	s.NoError(err)
}

func (s *WritersTestSuite) TestLogsWriter_GetLogCount_NotInitialized() {
	outputPath := filepath.Join(s.tempDir, "logs.parquet")
	w := NewLogsWriter(outputPath)

	_, err := w.GetLogCount()
	s.Error(err)
	s.Contains(err.Error(), "writer not initialized")
}

func (s *WritersTestSuite) TestLogsWriter_GetOutputPath() {
	outputPath := filepath.Join(s.tempDir, "logs.parquet")
	w := NewLogsWriter(outputPath)

	s.Equal(outputPath, w.GetOutputPath())
}

func (s *WritersTestSuite) TestLogsWriter_Persistence() {
	outputPath := filepath.Join(s.tempDir, "logs.parquet")

	// Write with first instance
	w1 := NewLogsWriter(outputPath)
	err := w1.Initialize()
	s.Require().NoError(err)

	entry := log.LogEntry{
		Timestamp: time.Now(),
		Symbol:    "BTCUSDT",
		Level:     types.LogLevelError,
		Message:   "Test persistence",
		Fields:    map[string]string{"test": "persistence"},
	}

	err = w1.Write(entry)
	s.Require().NoError(err)
	w1.Close()

	// Read with second instance
	w2 := NewLogsWriter(outputPath)
	err = w2.Initialize()
	s.Require().NoError(err)
	defer w2.Close()

	count, err := w2.GetLogCount()
	s.Require().NoError(err)
	s.Equal(1, count)
}
