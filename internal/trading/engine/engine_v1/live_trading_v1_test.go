package engine_v1

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	internalLog "github.com/rxtech-lab/argo-trading/internal/log"
	"github.com/rxtech-lab/argo-trading/internal/trading/engine"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/internal/version"
	"github.com/rxtech-lab/argo-trading/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// wasmFilePath is the relative path to the example strategy WASM file.
const wasmFilePath = "../../../../examples/strategy/strategy.wasm"

// LiveTradingEngineV1TestSuite is the test suite for LiveTradingEngineV1.
type LiveTradingEngineV1TestSuite struct {
	suite.Suite
	ctrl *gomock.Controller
}

// TestLiveTradingEngineV1 runs the test suite.
func TestLiveTradingEngineV1(t *testing.T) {
	suite.Run(t, new(LiveTradingEngineV1TestSuite))
}

// SetupTest runs before each test.
func (s *LiveTradingEngineV1TestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
}

// TearDownTest runs after each test.
func (s *LiveTradingEngineV1TestSuite) TearDownTest() {
	s.ctrl.Finish()
}

// createTestMarketData creates test market data with the given symbol and timestamp.
func createTestMarketData(symbol string, timestamp time.Time, price float64) types.MarketData {
	return types.MarketData{
		Id:     "test-id",
		Symbol: symbol,
		Time:   timestamp,
		Open:   price,
		High:   price + 1,
		Low:    price - 1,
		Close:  price,
		Volume: 1000,
	}
}

// createMockStream creates an iter.Seq2 that yields the given data points and errors.
func createMockStream(data []types.MarketData, streamErrs []error) iter.Seq2[types.MarketData, error] {
	return func(yield func(types.MarketData, error) bool) {
		for i, d := range data {
			var err error
			if i < len(streamErrs) {
				err = streamErrs[i]
			}
			if !yield(d, err) {
				return
			}
		}
	}
}

// ============================================================================
// Constructor Tests
// ============================================================================

func (s *LiveTradingEngineV1TestSuite) TestNewLiveTradingEngineV1() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)
	s.Require().NotNil(eng)

	// Type assert to access internal fields
	e, ok := eng.(*LiveTradingEngineV1)
	s.Require().True(ok)

	// Verify initial state
	s.False(e.initialized)
	s.Nil(e.strategy)
	s.Nil(e.marketDataProvider)
	s.Nil(e.tradingProvider)
	s.NotNil(e.cache)
	s.NotNil(e.log)
	s.Empty(e.dataDir)
	s.Empty(e.providerName)
}

func (s *LiveTradingEngineV1TestSuite) TestNewLiveTradingEngineV1WithPersistence() {
	dataDir := "/tmp/test-data"
	providerName := "binance"

	eng, err := NewLiveTradingEngineV1WithPersistence(dataDir, providerName)
	s.Require().NoError(err)
	s.Require().NotNil(eng)

	// Type assert to access internal fields
	e, ok := eng.(*LiveTradingEngineV1)
	s.Require().True(ok)

	// Verify persistence fields are set
	s.Equal(dataDir, e.dataDir)
	s.Equal(providerName, e.providerName)
	s.False(e.initialized)
}

// ============================================================================
// Initialize Tests
// ============================================================================

func (s *LiveTradingEngineV1TestSuite) TestInitialize_Success() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	config := engine.LiveTradingEngineConfig{
		Symbols:             []string{"BTCUSDT"},
		Interval:            "1m",
		MarketDataCacheSize: 500,
		EnableLogging:       false,
	}

	err = eng.Initialize(config)
	s.Require().NoError(err)

	// Verify initialized
	e := eng.(*LiveTradingEngineV1)
	s.True(e.initialized)
	s.NotNil(e.indicatorRegistry)
	s.NotNil(e.streamingDataSource)
	s.Equal(500, e.config.MarketDataCacheSize)
}

func (s *LiveTradingEngineV1TestSuite) TestInitialize_DefaultCacheSize() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	config := engine.LiveTradingEngineConfig{
		Symbols:             []string{"BTCUSDT"},
		Interval:            "1m",
		MarketDataCacheSize: 0, // Should use default
		EnableLogging:       false,
	}

	err = eng.Initialize(config)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	s.Equal(DefaultMarketDataCacheSize, e.config.MarketDataCacheSize)
}

func (s *LiveTradingEngineV1TestSuite) TestInitialize_WithLogging() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	config := engine.LiveTradingEngineConfig{
		Symbols:       []string{"BTCUSDT"},
		Interval:      "1m",
		EnableLogging: true,
	}

	err = eng.Initialize(config)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	s.NotNil(e.marker)
	s.NotNil(e.logStorage)
}

func (s *LiveTradingEngineV1TestSuite) TestInitialize_WithPersistence() {
	// Create temp directory for persistence
	tempDir, err := os.MkdirTemp("", "live-trading-test")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	eng, err := NewLiveTradingEngineV1WithPersistence(tempDir, "binance")
	s.Require().NoError(err)

	config := engine.LiveTradingEngineConfig{
		Symbols:       []string{"BTCUSDT"},
		Interval:      "1m",
		EnableLogging: false,
	}

	err = eng.Initialize(config)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	s.NotNil(e.streamingWriter)
	s.NotNil(e.persistentDataSource)
}

// ============================================================================
// LoadStrategy Tests
// ============================================================================

func (s *LiveTradingEngineV1TestSuite) TestLoadStrategy_Success() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	s.NotNil(e.strategy)
}

func (s *LiveTradingEngineV1TestSuite) TestLoadStrategyFromFile_Success() {
	// Check if WASM file exists
	absPath, err := filepath.Abs(wasmFilePath)
	if err != nil || !fileExists(absPath) {
		s.T().Skip("WASM file not found. Run 'cd examples/strategy && make build' first")
	}

	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.LoadStrategyFromFile(absPath)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	s.NotNil(e.strategy)
}

func (s *LiveTradingEngineV1TestSuite) TestLoadStrategyFromFile_FileNotFound() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.LoadStrategyFromFile("/nonexistent/path/strategy.wasm")
	s.Error(err)
}

func (s *LiveTradingEngineV1TestSuite) TestLoadStrategyFromBytes_Success() {
	// Check if WASM file exists
	absPath, err := filepath.Abs(wasmFilePath)
	if err != nil || !fileExists(absPath) {
		s.T().Skip("WASM file not found. Run 'cd examples/strategy && make build' first")
	}

	wasmBytes, err := os.ReadFile(absPath)
	s.Require().NoError(err)

	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.LoadStrategyFromBytes(wasmBytes)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	s.NotNil(e.strategy)
}

func (s *LiveTradingEngineV1TestSuite) TestLoadStrategyFromBytes_InvalidBytes() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	// Invalid WASM bytes - note: the wasm runtime doesn't validate bytes at load time,
	// it only stores them. Validation happens during InitializeApi.
	// So we test that it loads successfully but then fails during initialization.
	invalidBytes := []byte("not a valid wasm file")
	err = eng.LoadStrategyFromBytes(invalidBytes)
	// LoadStrategyFromBytes does not validate bytes immediately
	s.NoError(err)

	e := eng.(*LiveTradingEngineV1)
	s.NotNil(e.strategy)
}

// ============================================================================
// Setter Tests
// ============================================================================

func (s *LiveTradingEngineV1TestSuite) TestSetStrategyConfig() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	config := `{"symbol": "BTCUSDT"}`
	err = eng.SetStrategyConfig(config)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	s.Equal(config, e.strategyConfig)
}

func (s *LiveTradingEngineV1TestSuite) TestSetMarketDataProvider() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	mockProvider := mocks.NewMockProvider(s.ctrl)
	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	s.NotNil(e.marketDataProvider)
}

func (s *LiveTradingEngineV1TestSuite) TestSetTradingProvider() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	s.NotNil(e.tradingProvider)
}

// ============================================================================
// preRunCheck Tests
// ============================================================================

func (s *LiveTradingEngineV1TestSuite) TestPreRunCheck_NotInitialized() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	err = e.preRunCheck()
	s.Error(err)
	s.Contains(err.Error(), "not initialized")
}

func (s *LiveTradingEngineV1TestSuite) TestPreRunCheck_NoStrategy() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "1m",
	})
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	err = e.preRunCheck()
	s.Error(err)
	s.Contains(err.Error(), "strategy not loaded")
}

func (s *LiveTradingEngineV1TestSuite) TestPreRunCheck_NoMarketDataProvider() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "1m",
	})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	err = e.preRunCheck()
	s.Error(err)
	s.Contains(err.Error(), "market data provider not set")
}

func (s *LiveTradingEngineV1TestSuite) TestPreRunCheck_NoTradingProvider() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "1m",
	})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockProvider := mocks.NewMockProvider(s.ctrl)
	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	err = e.preRunCheck()
	s.Error(err)
	s.Contains(err.Error(), "trading provider not set")
}

func (s *LiveTradingEngineV1TestSuite) TestPreRunCheck_NoSymbols() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:  []string{}, // Empty symbols
		Interval: "1m",
	})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockProvider := mocks.NewMockProvider(s.ctrl)
	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	err = e.preRunCheck()
	s.Error(err)
	s.Contains(err.Error(), "no symbols configured")
}

func (s *LiveTradingEngineV1TestSuite) TestPreRunCheck_NoInterval() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "", // Empty interval
	})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockProvider := mocks.NewMockProvider(s.ctrl)
	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	err = e.preRunCheck()
	s.Error(err)
	s.Contains(err.Error(), "no interval configured")
}

func (s *LiveTradingEngineV1TestSuite) TestPreRunCheck_Success() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "1m",
	})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockProvider := mocks.NewMockProvider(s.ctrl)
	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	err = e.preRunCheck()
	s.NoError(err)
}

// ============================================================================
// initializeStrategy Tests
// ============================================================================

func (s *LiveTradingEngineV1TestSuite) TestInitializeStrategy_Success() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "1m",
	})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	err = e.initializeStrategy()
	s.NoError(err)
}

func (s *LiveTradingEngineV1TestSuite) TestInitializeStrategy_InitializeApiFails() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "1m",
	})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(errors.New("API init failed"))

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	err = e.initializeStrategy()
	s.Error(err)
	s.Contains(err.Error(), "failed to initialize strategy API")
}

func (s *LiveTradingEngineV1TestSuite) TestInitializeStrategy_GetRuntimeVersionFails() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "1m",
	})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("", errors.New("version error"))

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	err = e.initializeStrategy()
	s.Error(err)
	s.Contains(err.Error(), "failed to get strategy runtime version")
}

func (s *LiveTradingEngineV1TestSuite) TestInitializeStrategy_VersionMismatch() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "1m",
	})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	// Return an incompatible version (different major version)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("v0.1.0", nil)

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	err = e.initializeStrategy()
	s.Error(err)
	s.Contains(err.Error(), "version mismatch")
}

func (s *LiveTradingEngineV1TestSuite) TestInitializeStrategy_InitializeFails() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "1m",
	})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(errors.New("init failed"))

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	err = e.initializeStrategy()
	s.Error(err)
	s.Contains(err.Error(), "failed to initialize strategy")
}

// ============================================================================
// Run Tests
// ============================================================================

func (s *LiveTradingEngineV1TestSuite) TestRun_PreRunCheckFails() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	// Don't initialize - preRunCheck should fail
	var stopErr error
	onStop := engine.OnEngineStopCallback(func(err error) {
		stopErr = err
	})

	callbacks := engine.LiveTradingCallbacks{
		OnEngineStop: &onStop,
	}

	err = eng.Run(context.Background(), callbacks)
	s.Error(err)
	s.Contains(err.Error(), "not initialized")
	s.NotNil(stopErr)
}

func (s *LiveTradingEngineV1TestSuite) TestRun_InitializeStrategyFails() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "1m",
	})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(errors.New("API init failed"))

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockProvider := mocks.NewMockProvider(s.ctrl)
	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	var stopErr error
	onStop := engine.OnEngineStopCallback(func(err error) {
		stopErr = err
	})

	callbacks := engine.LiveTradingCallbacks{
		OnEngineStop: &onStop,
	}

	err = eng.Run(context.Background(), callbacks)
	s.Error(err)
	s.Contains(err.Error(), "failed to initialize strategy API")
	s.NotNil(stopErr)
}

func (s *LiveTradingEngineV1TestSuite) TestRun_OnEngineStartCallbackError() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "1m",
	})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockProvider := mocks.NewMockProvider(s.ctrl)
	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	onStart := engine.OnEngineStartCallback(func(symbols []string, interval string) error {
		return errors.New("start callback failed")
	})

	callbacks := engine.LiveTradingCallbacks{
		OnEngineStart: &onStart,
	}

	err = eng.Run(context.Background(), callbacks)
	s.Error(err)
	s.Contains(err.Error(), "OnEngineStart callback failed")
}

func (s *LiveTradingEngineV1TestSuite) TestRun_SuccessfulExecution() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "1m",
	})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil).Times(3)

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	// Create test data
	now := time.Now()
	testData := []types.MarketData{
		createTestMarketData("BTCUSDT", now, 50000),
		createTestMarketData("BTCUSDT", now.Add(time.Minute), 50100),
		createTestMarketData("BTCUSDT", now.Add(2*time.Minute), 50200),
	}

	mockProvider := mocks.NewMockProvider(s.ctrl)
	mockProvider.EXPECT().Stream(gomock.Any(), []string{"BTCUSDT"}, "1m").Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	var dataCount int
	var mu sync.Mutex
	var startCalled bool
	var stopCalled bool
	var stopErr error

	onStart := engine.OnEngineStartCallback(func(symbols []string, interval string) error {
		mu.Lock()
		defer mu.Unlock()
		startCalled = true
		s.Equal([]string{"BTCUSDT"}, symbols)
		s.Equal("1m", interval)
		return nil
	})

	onData := engine.OnMarketDataCallback(func(data types.MarketData) error {
		mu.Lock()
		defer mu.Unlock()
		dataCount++
		return nil
	})

	onStop := engine.OnEngineStopCallback(func(err error) {
		mu.Lock()
		defer mu.Unlock()
		stopCalled = true
		stopErr = err
	})

	callbacks := engine.LiveTradingCallbacks{
		OnEngineStart: &onStart,
		OnMarketData:  &onData,
		OnEngineStop:  &onStop,
	}

	err = eng.Run(context.Background(), callbacks)
	s.NoError(err)

	mu.Lock()
	defer mu.Unlock()
	s.True(startCalled)
	s.True(stopCalled)
	s.Nil(stopErr)
	s.Equal(3, dataCount)
}

func (s *LiveTradingEngineV1TestSuite) TestRun_StreamError_NonFatal() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "1m",
	})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)
	// ProcessData should still be called for the successful data points
	mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil).Times(2)

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	// Create test data with an error in the middle
	now := time.Now()
	testData := []types.MarketData{
		createTestMarketData("BTCUSDT", now, 50000),
		{}, // Empty data, error will be returned
		createTestMarketData("BTCUSDT", now.Add(2*time.Minute), 50200),
	}
	streamErrs := []error{nil, errors.New("stream error"), nil}

	mockProvider := mocks.NewMockProvider(s.ctrl)
	mockProvider.EXPECT().Stream(gomock.Any(), []string{"BTCUSDT"}, "1m").Return(createMockStream(testData, streamErrs))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	var errorCount int
	var mu sync.Mutex

	onError := engine.OnErrorCallback(func(err error) {
		mu.Lock()
		defer mu.Unlock()
		errorCount++
	})

	callbacks := engine.LiveTradingCallbacks{
		OnError: &onError,
	}

	err = eng.Run(context.Background(), callbacks)
	s.NoError(err) // Non-fatal errors don't cause Run to fail

	mu.Lock()
	defer mu.Unlock()
	s.Equal(1, errorCount)
}

func (s *LiveTradingEngineV1TestSuite) TestRun_StrategyError_NonFatal() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "1m",
	})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)
	// ProcessData returns error on second call
	mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil).Times(1)
	mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(errors.New("strategy error")).Times(1)
	mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil).Times(1)

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	now := time.Now()
	testData := []types.MarketData{
		createTestMarketData("BTCUSDT", now, 50000),
		createTestMarketData("BTCUSDT", now.Add(time.Minute), 50100),
		createTestMarketData("BTCUSDT", now.Add(2*time.Minute), 50200),
	}

	mockProvider := mocks.NewMockProvider(s.ctrl)
	mockProvider.EXPECT().Stream(gomock.Any(), []string{"BTCUSDT"}, "1m").Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	var strategyErrorCount int
	var mu sync.Mutex

	onStrategyError := engine.OnStrategyErrorCallback(func(data types.MarketData, err error) {
		mu.Lock()
		defer mu.Unlock()
		strategyErrorCount++
	})

	callbacks := engine.LiveTradingCallbacks{
		OnStrategyError: &onStrategyError,
	}

	err = eng.Run(context.Background(), callbacks)
	s.NoError(err) // Non-fatal errors don't cause Run to fail

	mu.Lock()
	defer mu.Unlock()
	s.Equal(1, strategyErrorCount)
}

func (s *LiveTradingEngineV1TestSuite) TestRun_OnMarketDataCallbackError() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "1m",
	})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	now := time.Now()
	testData := []types.MarketData{
		createTestMarketData("BTCUSDT", now, 50000),
	}

	mockProvider := mocks.NewMockProvider(s.ctrl)
	mockProvider.EXPECT().Stream(gomock.Any(), []string{"BTCUSDT"}, "1m").Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	onData := engine.OnMarketDataCallback(func(data types.MarketData) error {
		return errors.New("callback error")
	})

	callbacks := engine.LiveTradingCallbacks{
		OnMarketData: &onData,
	}

	err = eng.Run(context.Background(), callbacks)
	s.Error(err)
	s.Contains(err.Error(), "OnMarketData callback failed")
}

func (s *LiveTradingEngineV1TestSuite) TestRun_ContextCancellation() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "1m",
	})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil).AnyTimes()

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	// Create stream that generates data continuously
	ctx, cancel := context.WithCancel(context.Background())

	// Generate data until context is cancelled
	stream := func(yield func(types.MarketData, error) bool) {
		now := time.Now()
		i := 0
		for {
			select {
			case <-ctx.Done():
				return
			default:
				data := createTestMarketData("BTCUSDT", now.Add(time.Duration(i)*time.Minute), 50000+float64(i))
				if !yield(data, nil) {
					return
				}
				i++
				if i > 100 { // Safety limit
					return
				}
			}
		}
	}

	mockProvider := mocks.NewMockProvider(s.ctrl)
	mockProvider.EXPECT().Stream(gomock.Any(), []string{"BTCUSDT"}, "1m").Return(stream)

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	var dataCount int
	var mu sync.Mutex

	onData := engine.OnMarketDataCallback(func(data types.MarketData) error {
		mu.Lock()
		defer mu.Unlock()
		dataCount++
		if dataCount >= 3 {
			cancel() // Cancel after 3 data points
		}
		return nil
	})

	callbacks := engine.LiveTradingCallbacks{
		OnMarketData: &onData,
	}

	err = eng.Run(ctx, callbacks)
	s.Error(err)
	s.Equal(context.Canceled, err)
}

func (s *LiveTradingEngineV1TestSuite) TestRun_WithPersistence() {
	// Create temp directory for persistence
	tempDir, err := os.MkdirTemp("", "live-trading-run-test")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	eng, err := NewLiveTradingEngineV1WithPersistence(tempDir, "binance")
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "1m",
	})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil).Times(2)

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	now := time.Now()
	testData := []types.MarketData{
		createTestMarketData("BTCUSDT", now, 50000),
		createTestMarketData("BTCUSDT", now.Add(time.Minute), 50100),
	}

	mockProvider := mocks.NewMockProvider(s.ctrl)
	mockProvider.EXPECT().Stream(gomock.Any(), []string{"BTCUSDT"}, "1m").Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	var stopCalled bool
	var mu sync.Mutex

	onStop := engine.OnEngineStopCallback(func(err error) {
		mu.Lock()
		defer mu.Unlock()
		stopCalled = true
	})

	callbacks := engine.LiveTradingCallbacks{
		OnEngineStop: &onStop,
	}

	err = eng.Run(context.Background(), callbacks)
	s.NoError(err)

	mu.Lock()
	defer mu.Unlock()
	s.True(stopCalled)

	// Verify parquet file was created
	e := eng.(*LiveTradingEngineV1)
	parquetPath := filepath.Join(tempDir, "stream_data_binance_1m.parquet")
	s.True(fileExists(parquetPath), "Parquet file should exist at %s", parquetPath)
	s.NotNil(e.persistentDataSource)
}

func (s *LiveTradingEngineV1TestSuite) TestRun_AllCallbacksNil() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "1m",
	})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil).Times(2)

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	now := time.Now()
	testData := []types.MarketData{
		createTestMarketData("BTCUSDT", now, 50000),
		createTestMarketData("BTCUSDT", now.Add(time.Minute), 50100),
	}

	mockProvider := mocks.NewMockProvider(s.ctrl)
	mockProvider.EXPECT().Stream(gomock.Any(), []string{"BTCUSDT"}, "1m").Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	// Empty callbacks - no panics should occur
	callbacks := engine.LiveTradingCallbacks{}

	err = eng.Run(context.Background(), callbacks)
	s.NoError(err)
}

// ============================================================================
// GetConfigSchema Tests
// ============================================================================

func (s *LiveTradingEngineV1TestSuite) TestGetConfigSchema() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	schema, err := eng.GetConfigSchema()
	s.Require().NoError(err)
	s.NotEmpty(schema)

	// Verify it's valid JSON
	var result map[string]interface{}
	err = json.Unmarshal([]byte(schema), &result)
	s.NoError(err)

	// Verify it contains expected fields
	s.Contains(schema, "symbols")
	s.Contains(schema, "interval")
}

// ============================================================================
// LiveTradingMarker Tests
// ============================================================================

func (s *LiveTradingEngineV1TestSuite) TestNewLiveTradingMarker() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	marker, err := NewLiveTradingMarker(e.log)
	s.Require().NoError(err)
	s.NotNil(marker)

	m := marker.(*LiveTradingMarker)
	s.Empty(m.marks)
}

func (s *LiveTradingEngineV1TestSuite) TestLiveTradingMarker_Mark() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	marker, err := NewLiveTradingMarker(e.log)
	s.Require().NoError(err)

	marketData := createTestMarketData("BTCUSDT", time.Now(), 50000)
	mark := types.Mark{
		MarketDataId: "test-id",
		Color:        types.MarkColorGreen,
		Shape:        types.MarkShapeCircle,
		Level:        types.MarkLevelInfo,
		Title:        "Test Mark",
		Message:      "Test message",
		Category:     "test",
	}

	err = marker.Mark(marketData, mark)
	s.NoError(err)

	m := marker.(*LiveTradingMarker)
	s.Len(m.marks, 1)
	s.Equal("Test Mark", m.marks[0].Title)
}

func (s *LiveTradingEngineV1TestSuite) TestLiveTradingMarker_GetMarks() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	marker, err := NewLiveTradingMarker(e.log)
	s.Require().NoError(err)

	// Add multiple marks
	marketData := createTestMarketData("BTCUSDT", time.Now(), 50000)
	for i := 0; i < 3; i++ {
		mark := types.Mark{
			MarketDataId: "test-id",
			Color:        types.MarkColorGreen,
			Shape:        types.MarkShapeCircle,
			Level:        types.MarkLevelInfo,
			Title:        "Mark",
		}
		err = marker.Mark(marketData, mark)
		s.NoError(err)
	}

	marks, err := marker.GetMarks()
	s.NoError(err)
	s.Len(marks, 3)
}

// ============================================================================
// LiveTradingLog Tests
// ============================================================================

func (s *LiveTradingEngineV1TestSuite) TestNewLiveTradingLog() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	logStorage, err := NewLiveTradingLog(e.log)
	s.Require().NoError(err)
	s.NotNil(logStorage)

	l := logStorage.(*LiveTradingLog)
	s.Empty(l.logs)
}

func (s *LiveTradingEngineV1TestSuite) TestLiveTradingLog_Log() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	logStorage, err := NewLiveTradingLog(e.log)
	s.Require().NoError(err)

	entry := internalLog.LogEntry{
		Timestamp: time.Now(),
		Symbol:    "BTCUSDT",
		Level:     types.LogLevelInfo,
		Message:   "Test log message",
		Fields:    map[string]string{"key": "value"},
	}

	err = logStorage.Log(entry)
	s.NoError(err)

	l := logStorage.(*LiveTradingLog)
	s.Len(l.logs, 1)
	s.Equal("Test log message", l.logs[0].Message)
}

func (s *LiveTradingEngineV1TestSuite) TestLiveTradingLog_GetLogs() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	logStorage, err := NewLiveTradingLog(e.log)
	s.Require().NoError(err)

	// Add multiple log entries
	for i := 0; i < 3; i++ {
		entry := internalLog.LogEntry{
			Timestamp: time.Now(),
			Symbol:    "BTCUSDT",
			Level:     types.LogLevelInfo,
			Message:   "Log message",
		}
		err = logStorage.Log(entry)
		s.NoError(err)
	}

	logs, err := logStorage.GetLogs()
	s.NoError(err)
	s.Len(logs, 3)
}

// ============================================================================
// Helper Functions
// ============================================================================

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
