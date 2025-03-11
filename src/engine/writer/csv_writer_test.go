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
