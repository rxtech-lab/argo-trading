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
		EnableLogging: false,
	}

	err = eng.Initialize(config)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	// streamingWriter and persistentDataSource are initialized lazily in Run(),
	// because they require the provider's interval which is not available at Initialize time.
	s.Nil(e.streamingWriter)
	s.Nil(e.persistentDataSource)
	s.Equal(tempDir, e.dataDir)
	s.Equal("binance", e.providerName)
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
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	s.NotNil(e.marketDataProvider)
}

func (s *LiveTradingEngineV1TestSuite) TestSetTradingProvider() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
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

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	err = e.preRunCheck()
	s.Error(err)
	s.Contains(err.Error(), "strategy not loaded")
}

func (s *LiveTradingEngineV1TestSuite) TestPreRunCheck_NoMarketDataProvider() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
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

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockProvider := mocks.NewMockProvider(s.ctrl)
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
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

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockProvider := mocks.NewMockProvider(s.ctrl)
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
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

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockProvider := mocks.NewMockProvider(s.ctrl)
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("").AnyTimes()
	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
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

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockProvider := mocks.NewMockProvider(s.ctrl)
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
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

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	err = e.initializeStrategy()
	s.NoError(err)
}

func (s *LiveTradingEngineV1TestSuite) TestInitializeStrategy_InitializeApiFails() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(errors.New("API init failed"))

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
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

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("", errors.New("version error"))

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
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

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	// Return an incompatible version (different major version)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("v0.1.0", nil)

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
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

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(errors.New("init failed"))

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
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

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(errors.New("API init failed"))

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockProvider := mocks.NewMockProvider(s.ctrl)
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
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

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	mockProvider := mocks.NewMockProvider(s.ctrl)
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	onStart := engine.OnEngineStartCallback(func(symbols []string, interval string, previousDataPath string) error {
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

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
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
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	mockProvider.EXPECT().Stream(gomock.Any()).Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	var dataCount int
	var mu sync.Mutex
	var startCalled bool
	var stopCalled bool
	var stopErr error

	onStart := engine.OnEngineStartCallback(func(symbols []string, interval string, previousDataPath string) error {
		mu.Lock()
		defer mu.Unlock()
		startCalled = true
		s.Equal([]string{"BTCUSDT"}, symbols)
		s.Equal("1m", interval)
		// previousDataPath will be empty string since persistence is not enabled in this test
		return nil
	})

	onData := engine.OnMarketDataCallback(func(_ string, data types.MarketData) error {
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

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
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
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	mockProvider.EXPECT().Stream(gomock.Any()).Return(createMockStream(testData, streamErrs))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
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

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
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
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	mockProvider.EXPECT().Stream(gomock.Any()).Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
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

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
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
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	mockProvider.EXPECT().Stream(gomock.Any()).Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	onData := engine.OnMarketDataCallback(func(_ string, data types.MarketData) error {
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

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
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
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	mockProvider.EXPECT().Stream(gomock.Any()).Return(stream)

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	var dataCount int
	var mu sync.Mutex

	onData := engine.OnMarketDataCallback(func(_ string, data types.MarketData) error {
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

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
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
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	mockProvider.EXPECT().Stream(gomock.Any()).Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
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

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
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
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	mockProvider.EXPECT().Stream(gomock.Any()).Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	// Empty callbacks - no panics should occur
	callbacks := engine.LiveTradingCallbacks{}

	err = eng.Run(context.Background(), callbacks)
	s.NoError(err)
}

// ============================================================================
// Run Tests with DataOutputPath
// ============================================================================

func (s *LiveTradingEngineV1TestSuite) TestRun_WithDataOutputPath() {
	// Create temp directory for data output
	tempDir, err := os.MkdirTemp("", "live-trading-data-output-test")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		EnableLogging: true, // Enable marks and logs
	})
	s.Require().NoError(err)

	// Set data output path to enable session management
	err = eng.SetDataOutputPath(tempDir)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)

	// Verify session manager was initialized
	s.NotNil(e.sessionManager)
	s.NotEmpty(e.sessionManager.GetRunID())

	// Verify writers were initialized
	s.NotNil(e.ordersWriter)
	s.NotNil(e.tradesWriter)
	s.NotNil(e.marksWriter)
	s.NotNil(e.logsWriter)

	// Verify stats tracker was initialized
	s.NotNil(e.statsTracker)

	// Verify prefetch manager was initialized
	s.NotNil(e.prefetchManager)

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
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	mockProvider.EXPECT().Stream(gomock.Any()).Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	callbacks := engine.LiveTradingCallbacks{}

	err = eng.Run(context.Background(), callbacks)
	s.NoError(err)

	// Verify run path exists
	runPath := e.sessionManager.GetCurrentRunPath()
	s.DirExists(runPath)

	// Verify market data persistence was initialized
	s.NotNil(e.streamingWriter)
	s.NotNil(e.persistentDataSource)

	// Verify market data parquet file was created
	parquetPath := e.streamingWriter.GetOutputPath()
	s.FileExists(parquetPath)
	s.Contains(parquetPath, runPath)
}

func (s *LiveTradingEngineV1TestSuite) TestRun_StatusUpdate_Stopped() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
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
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	mockProvider.EXPECT().Stream(gomock.Any()).Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	var statusUpdates []types.EngineStatus
	var mu sync.Mutex
	onStatusUpdate := engine.OnStatusUpdateCallback(func(status types.EngineStatus) error {
		mu.Lock()
		defer mu.Unlock()
		statusUpdates = append(statusUpdates, status)
		return nil
	})

	callbacks := engine.LiveTradingCallbacks{
		OnStatusUpdate: &onStatusUpdate,
	}

	err = eng.Run(context.Background(), callbacks)
	s.NoError(err)

	mu.Lock()
	defer mu.Unlock()
	// Should contain Running (no prefetch) and Stopped statuses
	s.Contains(statusUpdates, types.EngineStatusRunning)
	s.Contains(statusUpdates, types.EngineStatusStopped)
}

func (s *LiveTradingEngineV1TestSuite) TestRun_StatusUpdate_Running_NoPrefetch() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		// No DataOutputPath - no prefetch manager
	})
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	s.Nil(e.prefetchManager) // Prefetch manager should be nil

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil).Times(1)

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	now := time.Now()
	testData := []types.MarketData{
		createTestMarketData("BTCUSDT", now, 50000),
	}

	mockProvider := mocks.NewMockProvider(s.ctrl)
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	mockProvider.EXPECT().Stream(gomock.Any()).Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	var statusUpdates []types.EngineStatus
	var mu sync.Mutex
	onStatusUpdate := engine.OnStatusUpdateCallback(func(status types.EngineStatus) error {
		mu.Lock()
		defer mu.Unlock()
		statusUpdates = append(statusUpdates, status)
		return nil
	})

	callbacks := engine.LiveTradingCallbacks{
		OnStatusUpdate: &onStatusUpdate,
	}

	err = eng.Run(context.Background(), callbacks)
	s.NoError(err)

	mu.Lock()
	defer mu.Unlock()
	// Should get Running status on first data when no prefetch manager
	s.Contains(statusUpdates, types.EngineStatusRunning)
}

func (s *LiveTradingEngineV1TestSuite) TestRun_StatsUpdate_Success() {
	// Create temp directory for data output
	tempDir, err := os.MkdirTemp("", "live-trading-stats-test")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
	s.Require().NoError(err)

	// Enable stats tracker via SetDataOutputPath
	err = eng.SetDataOutputPath(tempDir)
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
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	mockProvider.EXPECT().Stream(gomock.Any()).Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	var statsUpdates []types.LiveTradeStats
	var mu sync.Mutex
	onStatsUpdate := engine.OnStatsUpdateCallback(func(stats types.LiveTradeStats) error {
		mu.Lock()
		defer mu.Unlock()
		statsUpdates = append(statsUpdates, stats)
		return nil
	})

	callbacks := engine.LiveTradingCallbacks{
		OnStatsUpdate: &onStatsUpdate,
	}

	err = eng.Run(context.Background(), callbacks)
	s.NoError(err)

	mu.Lock()
	defer mu.Unlock()
	// Should have received stats updates (one per market data point)
	s.Equal(2, len(statsUpdates))
	// Verify stats contain proper data
	for _, stats := range statsUpdates {
		s.NotEmpty(stats.ID)
		s.Equal([]string{"BTCUSDT"}, stats.Symbols)
	}
}

func (s *LiveTradingEngineV1TestSuite) TestRun_StatsUpdate_Error_Continues() {
	// Create temp directory for data output
	tempDir, err := os.MkdirTemp("", "live-trading-stats-error-test")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
	s.Require().NoError(err)

	err = eng.SetDataOutputPath(tempDir)
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
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	mockProvider.EXPECT().Stream(gomock.Any()).Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	callCount := 0
	onStatsUpdate := engine.OnStatsUpdateCallback(func(stats types.LiveTradeStats) error {
		callCount++
		return errors.New("stats callback error")
	})

	callbacks := engine.LiveTradingCallbacks{
		OnStatsUpdate: &onStatsUpdate,
	}

	// Should not fail - stats callback errors are logged but don't stop execution
	err = eng.Run(context.Background(), callbacks)
	s.NoError(err)

	// Callback should have been called for each data point
	s.Equal(2, callCount)
}

func (s *LiveTradingEngineV1TestSuite) TestRun_WritesMarks() {
	// Create temp directory for data output
	tempDir, err := os.MkdirTemp("", "live-trading-marks-test")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		EnableLogging: true, // Enable marks
	})
	s.Require().NoError(err)

	err = eng.SetDataOutputPath(tempDir)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	s.NotNil(e.marker)
	s.NotNil(e.marksWriter)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)
	// When ProcessData is called, add a mark
	mockStrategy.EXPECT().ProcessData(gomock.Any()).DoAndReturn(func(data types.MarketData) error {
		// Add a mark via the marker
		mark := types.Mark{
			MarketDataId: data.Id,
			Color:        types.MarkColorGreen,
			Shape:        types.MarkShapeCircle,
			Level:        types.MarkLevelInfo,
			Title:        "Test Mark",
		}
		_ = e.marker.Mark(data, mark)
		return nil
	}).Times(1)

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	now := time.Now()
	testData := []types.MarketData{
		createTestMarketData("BTCUSDT", now, 50000),
	}

	mockProvider := mocks.NewMockProvider(s.ctrl)
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	mockProvider.EXPECT().Stream(gomock.Any()).Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	callbacks := engine.LiveTradingCallbacks{}

	err = eng.Run(context.Background(), callbacks)
	s.NoError(err)

	// Verify marks file was created
	marksPath := filepath.Join(e.sessionManager.GetCurrentRunPath(), "marks.parquet")
	s.FileExists(marksPath)
}

func (s *LiveTradingEngineV1TestSuite) TestRun_WritesLogs() {
	// Create temp directory for data output
	tempDir, err := os.MkdirTemp("", "live-trading-logs-test")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		EnableLogging: true, // Enable logs
	})
	s.Require().NoError(err)

	err = eng.SetDataOutputPath(tempDir)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	s.NotNil(e.logStorage)
	s.NotNil(e.logsWriter)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)
	// When ProcessData is called, add a log entry
	mockStrategy.EXPECT().ProcessData(gomock.Any()).DoAndReturn(func(data types.MarketData) error {
		// Add a log via the logStorage
		logEntry := internalLog.LogEntry{
			Timestamp: data.Time,
			Symbol:    data.Symbol,
			Level:     types.LogLevelInfo,
			Message:   "Test log message",
		}
		_ = e.logStorage.Log(logEntry)
		return nil
	}).Times(1)

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	now := time.Now()
	testData := []types.MarketData{
		createTestMarketData("BTCUSDT", now, 50000),
	}

	mockProvider := mocks.NewMockProvider(s.ctrl)
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	mockProvider.EXPECT().Stream(gomock.Any()).Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	callbacks := engine.LiveTradingCallbacks{}

	err = eng.Run(context.Background(), callbacks)
	s.NoError(err)

	// Verify logs file was created
	logsPath := filepath.Join(e.sessionManager.GetCurrentRunPath(), "logs.parquet")
	s.FileExists(logsPath)
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
	s.Contains(schema, "market_data_cache_size")
	s.Contains(schema, "enable_logging")
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
// OnProviderStatusChange Callback Tests
// ============================================================================

func (s *LiveTradingEngineV1TestSuite) TestRun_OnProviderStatusChangeCallback_Success() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil).Times(1)

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	// Create test data
	now := time.Now()
	testData := []types.MarketData{
		createTestMarketData("BTCUSDT", now, 50000),
	}

	mockProvider := mocks.NewMockProvider(s.ctrl)
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	mockProvider.EXPECT().Stream(gomock.Any()).Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	var statusUpdateReceived bool
	var receivedStatus types.ProviderStatusUpdate
	var mu sync.Mutex

	onStatusChange := engine.OnProviderStatusChangeCallback(func(status types.ProviderStatusUpdate) error {
		mu.Lock()
		defer mu.Unlock()
		statusUpdateReceived = true
		receivedStatus = status
		return nil
	})

	callbacks := engine.LiveTradingCallbacks{
		OnProviderStatusChange: &onStatusChange,
	}

	err = eng.Run(context.Background(), callbacks)
	s.NoError(err)

	mu.Lock()
	defer mu.Unlock()
	// Callback should have been invoked because we set up the provider callbacks
	s.True(statusUpdateReceived)
	// After successful connection, trading status should be connected
	s.Equal(types.ProviderStatusConnected, receivedStatus.TradingStatus)
}

func (s *LiveTradingEngineV1TestSuite) TestRun_OnProviderStatusChangeCallback_Nil() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil).Times(1)

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	// Create test data
	now := time.Now()
	testData := []types.MarketData{
		createTestMarketData("BTCUSDT", now, 50000),
	}

	mockProvider := mocks.NewMockProvider(s.ctrl)
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	mockProvider.EXPECT().Stream(gomock.Any()).Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	// Test with nil OnProviderStatusChange callback - should not panic
	callbacks := engine.LiveTradingCallbacks{
		OnProviderStatusChange: nil,
	}

	err = eng.Run(context.Background(), callbacks)
	s.NoError(err)
}

func (s *LiveTradingEngineV1TestSuite) TestRun_OnProviderStatusChangeCallback_Error() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil).AnyTimes()

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	// Create test data
	now := time.Now()
	testData := []types.MarketData{
		createTestMarketData("BTCUSDT", now, 50000),
	}

	mockProvider := mocks.NewMockProvider(s.ctrl)
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	mockProvider.EXPECT().Stream(gomock.Any()).Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(nil).AnyTimes()
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	var callCount int
	var mu sync.Mutex

	// Callback that returns an error
	onStatusChange := engine.OnProviderStatusChangeCallback(func(status types.ProviderStatusUpdate) error {
		mu.Lock()
		defer mu.Unlock()
		callCount++
		return errors.New("status callback error")
	})

	callbacks := engine.LiveTradingCallbacks{
		OnProviderStatusChange: &onStatusChange,
	}

	// The run should complete without error since status callback errors are non-fatal
	err = eng.Run(context.Background(), callbacks)
	s.NoError(err)

	mu.Lock()
	defer mu.Unlock()
	s.GreaterOrEqual(callCount, 1)
}

func (s *LiveTradingEngineV1TestSuite) TestRun_OnProviderStatusChangeCallback_TradingConnectionFails() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil).AnyTimes()

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	// Create test data
	now := time.Now()
	testData := []types.MarketData{
		createTestMarketData("BTCUSDT", now, 50000),
	}

	mockProvider := mocks.NewMockProvider(s.ctrl)
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	mockProvider.EXPECT().Stream(gomock.Any()).Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	// CheckConnection fails - trading provider is not connected
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(errors.New("connection failed")).AnyTimes()
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	var mu sync.Mutex

	onStatusChange := engine.OnProviderStatusChangeCallback(func(_ types.ProviderStatusUpdate) error {
		mu.Lock()
		defer mu.Unlock()
		return nil
	})

	callbacks := engine.LiveTradingCallbacks{
		OnProviderStatusChange: &onStatusChange,
	}

	err = eng.Run(context.Background(), callbacks)
	s.NoError(err)

	// When CheckConnection fails, the status stays disconnected (initial value),
	// so the callback is NOT called for trading status since no status change occurred.
	// However, callback may still be called from market data status changes.
	// This test verifies that the engine does not crash when CheckConnection fails.
}

func (s *LiveTradingEngineV1TestSuite) TestRun_TradingProviderConnectionFails_CallsOnError() {
	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{})
	s.Require().NoError(err)

	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil).AnyTimes()

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	// Create test data
	now := time.Now()
	testData := []types.MarketData{
		createTestMarketData("BTCUSDT", now, 50000),
	}

	mockProvider := mocks.NewMockProvider(s.ctrl)
	mockProvider.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockProvider.EXPECT().GetSymbols().Return([]string{"BTCUSDT"}).AnyTimes()
	mockProvider.EXPECT().GetInterval().Return("1m").AnyTimes()
	mockProvider.EXPECT().Stream(gomock.Any()).Return(createMockStream(testData, nil))

	err = eng.SetMarketDataProvider(mockProvider)
	s.Require().NoError(err)

	mockTrading := mocks.NewMockTradingSystemProvider(s.ctrl)
	mockTrading.EXPECT().SetOnStatusChange(gomock.Any()).AnyTimes()
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(errors.New("invalid API key")).AnyTimes()
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	var errorReceived error
	var mu sync.Mutex

	onError := engine.OnErrorCallback(func(err error) {
		mu.Lock()
		defer mu.Unlock()
		errorReceived = err
	})

	callbacks := engine.LiveTradingCallbacks{
		OnError: &onError,
	}

	err = eng.Run(context.Background(), callbacks)
	s.NoError(err)

	mu.Lock()
	defer mu.Unlock()
	s.Require().NotNil(errorReceived, "OnError should have been called when CheckConnection fails")
	s.Contains(errorReceived.Error(), "trading provider connection check failed")
	s.Contains(errorReceived.Error(), "invalid API key")
}

// ============================================================================
// Helper Functions
// ============================================================================

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
