package wasm

import (
	"testing"
	"time"

	engine "github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/cache"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/commission_fee"
	"github.com/rxtech-lab/argo-trading/internal/indicator"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/runtime"
	"github.com/rxtech-lab/argo-trading/internal/trading"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

type StrategyTestSuite struct {
	suite.Suite
	runtime       runtime.StrategyRuntime
	cache         *cache.Cache
	tradingSystem trading.TradingSystem
	logger        *logger.Logger
	state         *engine.BacktestState
	commission    commission_fee.CommissionFee
}

// Test Suite
func (suite *StrategyTestSuite) SetupSuite() {
	logger, err := logger.NewLogger()
	suite.Require().NoError(err)
	suite.logger = logger
	suite.state = engine.NewBacktestState(suite.logger)
	suite.Require().NotNil(suite.state)
	suite.commission = commission_fee.NewZeroCommissionFee()
}

func (suite *StrategyTestSuite) TearDownSuite() {
	if suite.state != nil {
		suite.state.Cleanup()
	}
}

func (suite *StrategyTestSuite) SetupTest() {
	// Initialize cache
	cacheV1 := cache.NewCacheV1()
	suite.cache = &cacheV1

	// Initialize state
	err := suite.state.Initialize()
	suite.Require().NoError(err)

	// Create real trading system
	suite.tradingSystem = engine.NewBacktestTrading(*suite.state, 10000.0, suite.commission)

	// Initialize strategy
	suite.runtime, err = NewStrategyWasmRuntime("../../../examples/strategy/plugin.wasm", NewWasmStrategyApi(&runtime.RuntimeContext{
		Cache:             *suite.cache,
		TradingSystem:     suite.tradingSystem,
		IndicatorRegistry: indicator.NewIndicatorRegistry(),
	}))
	suite.Require().NoError(err)
}

func (suite *StrategyTestSuite) TearDownTest() {
	err := suite.state.Cleanup()
	suite.Require().NoError(err)
}

func (suite *StrategyTestSuite) TestConsecutiveUpCandles() {

	// First up candle
	data1 := types.MarketData{
		Symbol: "BTCUSDT",
		Open:   100.0,
		High:   110.0,
		Low:    95.0,
		Close:  105.0,
		Volume: 1000.0,
		Time:   time.Now(),
	}

	// Second up candle
	data2 := types.MarketData{
		Symbol: "BTCUSDT",
		Open:   105.0,
		High:   115.0,
		Low:    100.0,
		Close:  110.0,
		Volume: 1000.0,
		Time:   time.Now().Add(time.Minute),
	}
	// Update market data in trading system
	suite.tradingSystem.(*engine.BacktestTrading).UpdateCurrentMarketData(data1)

	// Process first candle (should just store in cache)
	err := suite.runtime.ProcessData(data1)
	suite.NoError(err)

	// Update market data in trading system
	suite.tradingSystem.(*engine.BacktestTrading).UpdateCurrentMarketData(data2)

	// Process second candle (should trigger buy)
	err = suite.runtime.ProcessData(data2)
	suite.NoError(err)

	// Verify that a buy order was placed
	position, err := suite.tradingSystem.GetPosition("BTCUSDT")
	suite.NoError(err)
	suite.Equal(1.0, position.Quantity)
}

func (suite *StrategyTestSuite) TestConsecutiveDownCandles() {
	// Create test context
	tradingSystem := suite.tradingSystem

	// First establish a position by buying
	buyOrder := types.ExecuteOrder{
		Symbol:    "BTCUSDT",
		Side:      types.PurchaseTypeBuy,
		OrderType: types.OrderTypeLimit,
		Quantity:  1,
		Price:     10000.0, // Adjusted price to match the test data
		Reason: types.Reason{
			Reason:  "test",
			Message: "Establish initial position",
		},
		StrategyName: "Test",
	}
	var err error
	tradingSystem.(*engine.BacktestTrading).UpdateCurrentMarketData(types.MarketData{
		Symbol: "BTCUSDT",
		Open:   10000.0,
		High:   11000.0,
		Low:    9500.0,
		Close:  10000.0,
		Volume: 1000.0,
		Time:   time.Now(),
	})
	tradingSystem.(*engine.BacktestTrading).UpdateBalance(100000.0)
	err = tradingSystem.PlaceOrder(buyOrder)
	suite.NoError(err)

	// Verify initial position
	var position types.Position
	position, err = suite.tradingSystem.GetPosition("BTCUSDT")
	suite.NoError(err)
	suite.Equal(1.0, position.Quantity)

	// First down candle
	data1 := types.MarketData{
		Symbol: "BTCUSDT",
		Open:   11000.0,
		High:   11500.0,
		Low:    10000.0,
		Close:  10500.0,
		Volume: 1000.0,
		Time:   time.Now(),
	}

	// Second down candle
	data2 := types.MarketData{
		Symbol: "BTCUSDT",
		Open:   10500.0,
		High:   11000.0,
		Low:    9500.0,
		Close:  10000.0,
		Volume: 1000.0,
		Time:   time.Now().Add(time.Minute),
	}
	// Update market data in trading system
	suite.tradingSystem.(*engine.BacktestTrading).UpdateCurrentMarketData(data1)

	// Process first candle (should just store in cache)
	err = suite.runtime.ProcessData(data1)
	suite.NoError(err)

	// Update market data in trading system
	suite.tradingSystem.(*engine.BacktestTrading).UpdateCurrentMarketData(data2)

	// Process second candle (should trigger sell)
	err = suite.runtime.ProcessData(data2)
	suite.NoError(err)

	// Verify that the position was sold
	position, err = suite.tradingSystem.GetPosition("BTCUSDT")
	suite.NoError(err)
	suite.Equal(0.0, position.Quantity)
}

func TestStrategySuite(t *testing.T) {
	suite.Run(t, new(StrategyTestSuite))
}
