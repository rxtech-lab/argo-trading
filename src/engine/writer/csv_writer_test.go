package writer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/sirily11/argo-trading-go/src/types"
	"github.com/stretchr/testify/assert"
)

func TestCSVWriter_WriteOrder(t *testing.T) {

	t.Run("test_write_order", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// Create a new CSVWriter
		writer, err := NewCSVWriter(tempDir)
		assert.NoError(t, err, "Failed to create CSVWriter")

		// Create a test order
		testTime := time.Now()
		testOrder := types.Order{
			Symbol:      "AAPL",
			OrderType:   types.OrderTypeBuy,
			Quantity:    10.5,
			Price:       150.75,
			Timestamp:   testTime,
			OrderID:     "test-order-123",
			IsCompleted: false,
			Reason: types.Reason{
				Reason:  types.OrderReasonBuySignal,
				Message: "Test buy signal",
			},
			StrategyName: "TestStrategy",
		}

		// Write the order
		err = writer.WriteOrder(testOrder)
		assert.NoError(t, err, "Failed to write order")

		// Close the writer to ensure all data is flushed
		err = writer.Close()
		assert.NoError(t, err, "Failed to close writer")

		// Verify the order was written correctly
		// Get the timestamp directory
		dirs, err := os.ReadDir(tempDir)
		assert.NoError(t, err, "Failed to read temp directory")
		assert.Equal(t, 1, len(dirs), "Should have one timestamp directory")

		// Read the orders.csv file
		ordersFilePath := filepath.Join(tempDir, dirs[0].Name(), "orders.csv")
		ordersFile, err := os.Open(ordersFilePath)
		assert.NoError(t, err, "Failed to open orders file")
		defer ordersFile.Close()

		// Parse the CSV file
		var orders []types.Order
		err = gocsv.UnmarshalFile(ordersFile, &orders)
		assert.NoError(t, err, "Failed to unmarshal orders CSV")

		// Verify we have one order
		assert.Equal(t, 1, len(orders), "Should have one order")

		// Verify the order data
		order := orders[0]
		assert.Equal(t, testOrder.Symbol, order.Symbol, "Symbol should match")
		assert.Equal(t, testOrder.OrderType, order.OrderType, "OrderType should match")
		assert.Equal(t, testOrder.Quantity, order.Quantity, "Quantity should match")
		assert.Equal(t, testOrder.Price, order.Price, "Price should match")
		assert.Equal(t, testOrder.Timestamp.Format(time.RFC3339), order.Timestamp.Format(time.RFC3339), "Timestamp should match")
		assert.Equal(t, testOrder.OrderID, order.OrderID, "OrderID should match")
		assert.Equal(t, testOrder.IsCompleted, order.IsCompleted, "IsCompleted should match")
		assert.Equal(t, testOrder.Reason.Reason, order.Reason.Reason, "Reason should match")
		assert.Equal(t, testOrder.Reason.Message, order.Reason.Message, "Reason message should match")
		assert.Equal(t, testOrder.StrategyName, order.StrategyName, "StrategyName should match")
	})
}

func TestCSVWriter_WriteTrade(t *testing.T) {
	t.Run("test_write_trade", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// Create a new CSVWriter
		writer, err := NewCSVWriter(tempDir)
		assert.NoError(t, err, "Failed to create CSVWriter")

		// Create a test order for the trade
		testTime := time.Now()
		testOrder := types.Order{
			Symbol:      "AAPL",
			OrderType:   types.OrderTypeBuy,
			Quantity:    10.5,
			Price:       150.75,
			Timestamp:   testTime,
			OrderID:     "test-order-123",
			IsCompleted: true,
			Reason: types.Reason{
				Reason:  types.OrderReasonBuySignal,
				Message: "Test buy signal",
			},
			StrategyName: "TestStrategy",
		}

		// Create a test trade
		executedTime := testTime.Add(time.Hour) // Executed 1 hour after order creation
		testTrade := types.Trade{
			Order:         testOrder,
			ExecutedAt:    executedTime,
			ExecutedQty:   10.0,
			ExecutedPrice: 151.25,
			Commission:    1.50,
			PnL:           5.0,
		}

		// Write the trade
		err = writer.WriteTrade(testTrade)
		assert.NoError(t, err, "Failed to write trade")

		// Close the writer to ensure all data is flushed
		err = writer.Close()
		assert.NoError(t, err, "Failed to close writer")

		// Verify the trade was written correctly
		// Get the timestamp directory
		dirs, err := os.ReadDir(tempDir)
		assert.NoError(t, err, "Failed to read temp directory")
		assert.Equal(t, 1, len(dirs), "Should have one timestamp directory")

		// Read the trades.csv file
		tradesFilePath := filepath.Join(tempDir, dirs[0].Name(), "trades.csv")
		tradesFile, err := os.Open(tradesFilePath)
		assert.NoError(t, err, "Failed to open trades file")
		defer tradesFile.Close()

		// Parse the CSV file
		var trades []types.Trade
		err = gocsv.UnmarshalFile(tradesFile, &trades)
		assert.NoError(t, err, "Failed to unmarshal trades CSV")

		// Verify we have one trade
		assert.Equal(t, 1, len(trades), "Should have one trade")

		// Verify the trade data
		trade := trades[0]
		assert.Equal(t, testTrade.Order.Symbol, trade.Order.Symbol, "Symbol should match")
		assert.Equal(t, testTrade.Order.OrderType, trade.Order.OrderType, "OrderType should match")
		assert.Equal(t, testTrade.ExecutedAt.Format(time.RFC3339), trade.ExecutedAt.Format(time.RFC3339), "ExecutedAt should match")
		assert.Equal(t, testTrade.ExecutedQty, trade.ExecutedQty, "ExecutedQty should match")
		assert.Equal(t, testTrade.ExecutedPrice, trade.ExecutedPrice, "ExecutedPrice should match")
		assert.Equal(t, testTrade.Commission, trade.Commission, "Commission should match")
		assert.Equal(t, testTrade.PnL, trade.PnL, "PnL should match")
		assert.Equal(t, testTrade.Order.StrategyName, trade.Order.StrategyName, "StrategyName should match")
	})
}
