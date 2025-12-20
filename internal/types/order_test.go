package types

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/moznion/go-optional"
	"github.com/stretchr/testify/assert"
)

func TestExecuteOrderValidate(t *testing.T) {
	tests := []struct {
		name        string
		order       ExecuteOrder
		shouldError bool
	}{
		{
			name: "valid order",
			order: ExecuteOrder{
				ID:           uuid.New().String(),
				Symbol:       "BTC/USD",
				Side:         PurchaseTypeBuy,
				OrderType:    OrderTypeMarket,
				Reason:       Reason{Reason: "test", Message: "test"},
				Price:        100.0,
				StrategyName: "test-strategy",
				Quantity:     1.0,
				PositionType: PositionTypeLong,
				TakeProfit:   optional.None[ExecuteOrderTakeProfitOrStopLoss](),
				StopLoss:     optional.None[ExecuteOrderTakeProfitOrStopLoss](),
			},
			shouldError: false,
		},
		{
			name: "valid order with take profit",
			order: ExecuteOrder{
				ID:           uuid.New().String(),
				Symbol:       "BTC/USD",
				Side:         PurchaseTypeBuy,
				OrderType:    OrderTypeMarket,
				Reason:       Reason{Reason: "test", Message: "test"},
				Price:        100.0,
				StrategyName: "test-strategy",
				Quantity:     1.0,
				PositionType: PositionTypeLong,
				TakeProfit: optional.Some(ExecuteOrderTakeProfitOrStopLoss{
					Symbol:    "BTC/USD",
					Side:      PurchaseTypeSell,
					OrderType: OrderTypeLimit,
				}),
				StopLoss: optional.None[ExecuteOrderTakeProfitOrStopLoss](),
			},
			shouldError: false,
		},
		{
			name: "valid order with stop loss",
			order: ExecuteOrder{
				ID:           uuid.New().String(),
				Symbol:       "BTC/USD",
				Side:         PurchaseTypeBuy,
				OrderType:    OrderTypeMarket,
				Reason:       Reason{Reason: "test", Message: "test"},
				Price:        100.0,
				StrategyName: "test-strategy",
				Quantity:     1.0,
				PositionType: PositionTypeLong,
				TakeProfit:   optional.None[ExecuteOrderTakeProfitOrStopLoss](),
				StopLoss: optional.Some(ExecuteOrderTakeProfitOrStopLoss{
					Symbol:    "BTC/USD",
					Side:      PurchaseTypeSell,
					OrderType: OrderTypeMarket,
				}),
			},
			shouldError: false,
		},
		{
			name: "invalid order - empty ID",
			order: ExecuteOrder{
				ID:           "",
				Symbol:       "BTC/USD",
				Side:         PurchaseTypeBuy,
				OrderType:    OrderTypeMarket,
				Reason:       Reason{Reason: "test", Message: "test"},
				Price:        100.0,
				StrategyName: "test-strategy",
				Quantity:     1.0,
				PositionType: PositionTypeLong,
			},
			shouldError: true,
		},
		{
			name: "invalid order - empty symbol",
			order: ExecuteOrder{
				ID:           uuid.New().String(),
				Symbol:       "",
				Side:         PurchaseTypeBuy,
				OrderType:    OrderTypeMarket,
				Reason:       Reason{Reason: "test", Message: "test"},
				Price:        100.0,
				StrategyName: "test-strategy",
				Quantity:     1.0,
				PositionType: PositionTypeLong,
			},
			shouldError: true,
		},
		{
			name: "invalid order - empty side",
			order: ExecuteOrder{
				ID:           uuid.New().String(),
				Symbol:       "BTC/USD",
				Side:         "",
				OrderType:    OrderTypeMarket,
				Reason:       Reason{Reason: "test", Message: "test"},
				Price:        100.0,
				StrategyName: "test-strategy",
				Quantity:     1.0,
				PositionType: PositionTypeLong,
			},
			shouldError: true,
		},
		{
			name: "invalid order - invalid take profit",
			order: ExecuteOrder{
				ID:           uuid.New().String(),
				Symbol:       "BTC/USD",
				Side:         PurchaseTypeBuy,
				OrderType:    OrderTypeMarket,
				Reason:       Reason{Reason: "test", Message: "test"},
				Price:        100.0,
				StrategyName: "test-strategy",
				Quantity:     1.0,
				PositionType: PositionTypeLong,
				TakeProfit: optional.Some(ExecuteOrderTakeProfitOrStopLoss{
					Symbol:    "", // Empty symbol
					Side:      PurchaseTypeSell,
					OrderType: OrderTypeLimit,
				}),
			},
			shouldError: true,
		},
		{
			name: "invalid order - invalid stop loss",
			order: ExecuteOrder{
				ID:           uuid.New().String(),
				Symbol:       "BTC/USD",
				Side:         PurchaseTypeBuy,
				OrderType:    OrderTypeMarket,
				Reason:       Reason{Reason: "test", Message: "test"},
				Price:        100.0,
				StrategyName: "test-strategy",
				Quantity:     1.0,
				PositionType: PositionTypeLong,
				TakeProfit:   optional.None[ExecuteOrderTakeProfitOrStopLoss](),
				StopLoss: optional.Some(ExecuteOrderTakeProfitOrStopLoss{
					Symbol:    "", // Empty symbol
					Side:      PurchaseTypeSell,
					OrderType: OrderTypeMarket,
				}),
			},
			shouldError: true,
		},
		{
			name: "invalid order - negative price",
			order: ExecuteOrder{
				ID:           uuid.New().String(),
				Symbol:       "BTC/USD",
				Side:         PurchaseTypeBuy,
				OrderType:    OrderTypeMarket,
				Reason:       Reason{Reason: "test", Message: "test"},
				Price:        -100.0,
				StrategyName: "test-strategy",
				Quantity:     1.0,
				PositionType: PositionTypeLong,
			},
			shouldError: true,
		},
		{
			name: "invalid order - zero quantity",
			order: ExecuteOrder{
				ID:           uuid.New().String(),
				Symbol:       "BTC/USD",
				Side:         PurchaseTypeBuy,
				OrderType:    OrderTypeMarket,
				Reason:       Reason{Reason: "test", Message: "test"},
				Price:        100.0,
				StrategyName: "test-strategy",
				Quantity:     0.0,
				PositionType: PositionTypeLong,
			},
			shouldError: true,
		},
		{
			name: "invalid order - invalid position type",
			order: ExecuteOrder{
				ID:           uuid.New().String(),
				Symbol:       "BTC/USD",
				Side:         PurchaseTypeBuy,
				OrderType:    OrderTypeMarket,
				Reason:       Reason{Reason: "test", Message: "test"},
				Price:        100.0,
				StrategyName: "test-strategy",
				Quantity:     1.0,
				PositionType: PositionType("INVALID"),
			},
			shouldError: true,
		},
		{
			name: "valid order with both take profit and stop loss",
			order: ExecuteOrder{
				ID:           uuid.New().String(),
				Symbol:       "BTC/USD",
				Side:         PurchaseTypeBuy,
				OrderType:    OrderTypeMarket,
				Reason:       Reason{Reason: "test", Message: "test"},
				Price:        100.0,
				StrategyName: "test-strategy",
				Quantity:     1.0,
				PositionType: PositionTypeLong,
				TakeProfit: optional.Some(ExecuteOrderTakeProfitOrStopLoss{
					Symbol:    "BTC/USD",
					Side:      PurchaseTypeSell,
					OrderType: OrderTypeLimit,
				}),
				StopLoss: optional.Some(ExecuteOrderTakeProfitOrStopLoss{
					Symbol:    "BTC/USD",
					Side:      PurchaseTypeSell,
					OrderType: OrderTypeMarket,
				}),
			},
			shouldError: false,
		},
		{
			name: "valid short order",
			order: ExecuteOrder{
				ID:           uuid.New().String(),
				Symbol:       "BTC/USD",
				Side:         PurchaseTypeSell,
				OrderType:    OrderTypeLimit,
				Reason:       Reason{Reason: "test", Message: "test"},
				Price:        100.0,
				StrategyName: "test-strategy",
				Quantity:     1.0,
				PositionType: PositionTypeShort,
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.order.Validate()
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOrderValidate(t *testing.T) {
	tests := []struct {
		name        string
		order       Order
		shouldError bool
	}{
		{
			name: "valid order",
			order: Order{
				OrderID:      uuid.New().String(),
				Symbol:       "BTC/USD",
				Side:         PurchaseTypeBuy,
				Quantity:     1.0,
				Price:        100.0,
				Timestamp:    time.Now(),
				IsCompleted:  false,
				Reason:       Reason{Reason: "test", Message: "test"},
				StrategyName: "test-strategy",
				Fee:          0.1,
				PositionType: PositionTypeLong,
			},
			shouldError: false,
		},
		{
			name: "invalid order - empty symbol",
			order: Order{
				OrderID:      uuid.New().String(),
				Symbol:       "",
				Side:         PurchaseTypeBuy,
				Quantity:     1.0,
				Price:        100.0,
				Timestamp:    time.Now(),
				IsCompleted:  false,
				Reason:       Reason{Reason: "test", Message: "test"},
				StrategyName: "test-strategy",
				Fee:          0.1,
				PositionType: PositionTypeLong,
			},
			shouldError: true,
		},
		{
			name: "invalid order - empty side",
			order: Order{
				OrderID:      uuid.New().String(),
				Symbol:       "BTC/USD",
				Side:         "",
				Quantity:     1.0,
				Price:        100.0,
				Timestamp:    time.Now(),
				IsCompleted:  false,
				Reason:       Reason{Reason: "test", Message: "test"},
				StrategyName: "test-strategy",
				Fee:          0.1,
				PositionType: PositionTypeLong,
			},
			shouldError: true,
		},
		{
			name: "invalid order - negative quantity",
			order: Order{
				OrderID:      uuid.New().String(),
				Symbol:       "BTC/USD",
				Side:         PurchaseTypeBuy,
				Quantity:     -1.0,
				Price:        100.0,
				Timestamp:    time.Now(),
				IsCompleted:  false,
				Reason:       Reason{Reason: "test", Message: "test"},
				StrategyName: "test-strategy",
				Fee:          0.1,
				PositionType: PositionTypeLong,
			},
			shouldError: true,
		},
		{
			name: "invalid order - negative price",
			order: Order{
				OrderID:      uuid.New().String(),
				Symbol:       "BTC/USD",
				Side:         PurchaseTypeBuy,
				Quantity:     1.0,
				Price:        -100.0,
				Timestamp:    time.Now(),
				IsCompleted:  false,
				Reason:       Reason{Reason: "test", Message: "test"},
				StrategyName: "test-strategy",
				Fee:          0.1,
				PositionType: PositionTypeLong,
			},
			shouldError: true,
		},
		{
			name: "invalid order - empty reason",
			order: Order{
				OrderID:      uuid.New().String(),
				Symbol:       "BTC/USD",
				Side:         PurchaseTypeBuy,
				Quantity:     1.0,
				Price:        100.0,
				Timestamp:    time.Now(),
				IsCompleted:  false,
				Reason:       Reason{Reason: "", Message: "test"},
				StrategyName: "test-strategy",
				Fee:          0.1,
				PositionType: PositionTypeLong,
			},
			shouldError: true,
		},
		{
			name: "invalid order - empty position type",
			order: Order{
				OrderID:      uuid.New().String(),
				Symbol:       "BTC/USD",
				Side:         PurchaseTypeBuy,
				Quantity:     1.0,
				Price:        100.0,
				Timestamp:    time.Now(),
				IsCompleted:  false,
				Reason:       Reason{Reason: "test", Message: "test"},
				StrategyName: "test-strategy",
				Fee:          0.1,
				PositionType: "",
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.order.Validate()
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
