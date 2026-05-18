package engine_v1

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	internalLog "github.com/rxtech-lab/argo-trading/internal/log"
	"github.com/rxtech-lab/argo-trading/internal/trading/engine"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/internal/version"
	"github.com/rxtech-lab/argo-trading/mocks"
	strategypb "github.com/rxtech-lab/argo-trading/pkg/strategy"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// countParquetRows opens the parquet file with DuckDB and returns the row count.
func countParquetRows(path string) (int, error) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var count int
	row := db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM read_parquet('%s')`, path))
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// wasmFilePath is the relative path to the example strategy WASM file.
const wasmFilePath = "../../../../examples/strategy/strategy.wasm"

// logEveryTickWasmPath is a tiny WASM strategy that emits one INFO log per
// ProcessData call. Used by the live-trading e2e test that verifies every
// strategy log lands in logs.parquet through the real WASM → gRPC → host →
// LogStorage → tick-loop → writer pipeline. Build with
// `cd e2e/backtest/wasm && make build`.
const logEveryTickWasmPath = "../../../../e2e/backtest/wasm/log_every_tick/log_every_tick_plugin.wasm"

// logInDeferWasmPath is a WASM strategy that emits a DEBUG log from inside a
// deferred function at ProcessData exit. Real strategies (e.g. MultiConfirm)
// use this pattern for HOLD/no-action lines, and zap drops DEBUG entries
// from the running.log — so logs.parquet is the only way to verify those
// logs are persisted. The e2e test guards against silent regressions in
// either the WASM defer path or the host's level-agnostic LogStorage write.
const logInDeferWasmPath = "../../../../e2e/backtest/wasm/log_in_defer/log_in_defer_plugin.wasm"

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

// TestRun_StrategyLogsViaApiArePersisted is a regression test for a bug where the
// engine bound the WASM strategy API to one RuntimeContext at init time but
// mutated CurrentMarketData on a *different* context inside Run(). The host's
// Log handler gates persistence on `runtimeContext.CurrentMarketData != nil`,
// so the strategy's api.Log() calls silently dropped on the floor and
// logs.parquet stayed empty for the entire session.
//
// This test exercises the real strategy → host API path: it captures the
// StrategyApi the engine passes to InitializeApi, then has the mock strategy
// invoke api.Log() from inside ProcessData (the same path a WASM strategy
// takes through gRPC). If the engine ever reverts to allocating a separate
// context for the tick loop, CurrentMarketData will be nil on the bound
// context and this test will fail.
func (s *LiveTradingEngineV1TestSuite) TestRun_StrategyLogsViaApiArePersisted() {
	tempDir, err := os.MkdirTemp("", "live-trading-strategy-api-logs-test")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{EnableLogging: true})
	s.Require().NoError(err)

	err = eng.SetDataOutputPath(tempDir)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	s.Require().NotNil(e.logStorage)

	// Capture the StrategyApi the engine binds to the strategy at init time —
	// this is the same object a real WASM strategy receives via go-plugin.
	var capturedAPI strategypb.StrategyApi
	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).DoAndReturn(func(api strategypb.StrategyApi) error {
		capturedAPI = api
		return nil
	})
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().ProcessData(gomock.Any()).DoAndReturn(func(data types.MarketData) error {
		s.Require().NotNil(capturedAPI, "engine must bind StrategyApi before ProcessData")
		_, logErr := capturedAPI.Log(context.Background(), &strategypb.LogRequest{
			Level:   strategypb.LogLevel_LOG_LEVEL_INFO,
			Message: "strategy log via api",
		})
		return logErr
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

	err = eng.Run(context.Background(), engine.LiveTradingCallbacks{})
	s.Require().NoError(err)

	// The strategy's api.Log() call must have flowed all the way through to
	// LogStorage. If CurrentMarketData was nil on the bound context (the bug),
	// the host handler silently skips persistence and logs stays empty.
	logs, err := e.logStorage.GetLogs()
	s.Require().NoError(err)
	s.Require().Len(logs, 1, "strategy log via api.Log() should be persisted to LogStorage")
	s.Equal("strategy log via api", logs[0].Message)
	s.Equal("BTCUSDT", logs[0].Symbol)
	s.Equal(types.LogLevelInfo, logs[0].Level)
	s.Equal(now.Unix(), logs[0].Timestamp.Unix())
}

// TestRun_LogsAndMarksNotDuplicatedAcrossTicks is a regression test for a bug
// where every tick re-wrote the entire in-memory log/mark buffer to the
// parquet writers. LiveTradingLog.GetLogs() and LiveTradingMarker.GetMarks()
// return the full cumulative buffer (the strategy may query GetMarkers via the
// host API), but the parquet writers are append-only. The tick loop must
// track a cursor and only persist entries appended since the previous tick.
//
// Repro: a strategy emits one log + one mark on its first ProcessData call
// only (e.g. an "X strategy started" startup log gated by tickCount == 0).
// After streaming N bars, the parquet files should contain exactly 1 row each,
// not N rows.
func (s *LiveTradingEngineV1TestSuite) TestRun_LogsAndMarksNotDuplicatedAcrossTicks() {
	tempDir, err := os.MkdirTemp("", "live-trading-no-dup-test")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{EnableLogging: true})
	s.Require().NoError(err)

	err = eng.SetDataOutputPath(tempDir)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	s.Require().NotNil(e.logStorage)
	s.Require().NotNil(e.marker)
	s.Require().NotNil(e.logsWriter)
	s.Require().NotNil(e.marksWriter)

	const totalBars = 5

	// Emit one log + one mark only on the very first bar — mirrors the
	// `if tickCount == 0` pattern real strategies use for startup logs.
	callCount := 0
	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().ProcessData(gomock.Any()).DoAndReturn(func(data types.MarketData) error {
		if callCount == 0 {
			_ = e.logStorage.Log(internalLog.LogEntry{
				Timestamp: data.Time,
				Symbol:    data.Symbol,
				Level:     types.LogLevelInfo,
				Message:   "startup log",
			})
			_ = e.marker.Mark(data, types.Mark{
				MarketDataId: data.Id,
				Color:        types.MarkColorGreen,
				Shape:        types.MarkShapeCircle,
				Level:        types.MarkLevelInfo,
				Title:        "startup mark",
			})
		}
		callCount++
		return nil
	}).Times(totalBars)

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	now := time.Now()
	testData := make([]types.MarketData, totalBars)
	for i := 0; i < totalBars; i++ {
		testData[i] = createTestMarketData("BTCUSDT", now.Add(time.Duration(i)*time.Minute), 50000+float64(i))
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

	err = eng.Run(context.Background(), engine.LiveTradingCallbacks{})
	s.Require().NoError(err)

	// The in-memory buffers should still hold exactly one entry — the host API
	// remains a cumulative view so strategies that call GetMarkers see prior marks.
	logs, err := e.logStorage.GetLogs()
	s.Require().NoError(err)
	s.Len(logs, 1, "logStorage should still hold the one log the strategy emitted")

	marks, err := e.marker.GetMarks()
	s.Require().NoError(err)
	s.Len(marks, 1, "marker should still hold the one mark the strategy emitted")

	// The parquet files must contain exactly one row each, not totalBars rows.
	// Before the fix, the tick loop re-wrote every cumulative entry on every
	// bar — so logs.parquet ended up with totalBars copies of the same row.
	runPath := e.sessionManager.GetCurrentRunPath()
	logsPath := filepath.Join(runPath, "logs.parquet")
	s.Require().FileExists(logsPath)
	logsRows, err := countParquetRows(logsPath)
	s.Require().NoError(err)
	s.Equal(1, logsRows, "logs.parquet must contain exactly one row, not one-per-tick duplicates")

	marksPath := filepath.Join(runPath, "marks.parquet")
	s.Require().FileExists(marksPath)
	marksRows, err := countParquetRows(marksPath)
	s.Require().NoError(err)
	s.Equal(1, marksRows, "marks.parquet must contain exactly one row, not one-per-tick duplicates")
}

// TestRun_StrategyLogsEveryTickAllPersisted is the symmetric counterpart to
// TestRun_LogsAndMarksNotDuplicatedAcrossTicks. It guards the other direction
// of the cursor fix: when a strategy logs on every ProcessData call, every
// log must land in logs.parquet exactly once (N ticks → N rows). A naive
// drain-on-write fix would have collapsed this to whichever subset survived
// races between Log() and the writer; a cursor-stuck-at-0 fix would have
// kept duplicating. Only an advancing cursor produces N distinct rows.
//
// Uses the host's StrategyApi (the same path WASM strategies take through
// gRPC), so the test covers the real Log → LogStorage → tick-loop → writer
// chain end to end.
func (s *LiveTradingEngineV1TestSuite) TestRun_StrategyLogsEveryTickAllPersisted() {
	tempDir, err := os.MkdirTemp("", "live-trading-log-every-tick-test")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{EnableLogging: true})
	s.Require().NoError(err)

	err = eng.SetDataOutputPath(tempDir)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	s.Require().NotNil(e.logStorage)
	s.Require().NotNil(e.logsWriter)

	const totalBars = 7

	// Capture the StrategyApi the engine binds at init, then route every
	// ProcessData call through api.Log() — the same path a real WASM strategy
	// takes via gRPC. The message embeds the bar index so we can assert
	// every distinct bar's log made it through.
	var capturedAPI strategypb.StrategyApi
	callIdx := 0
	mockStrategy := mocks.NewMockStrategyRuntime(s.ctrl)
	mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
	mockStrategy.EXPECT().InitializeApi(gomock.Any()).DoAndReturn(func(api strategypb.StrategyApi) error {
		capturedAPI = api
		return nil
	})
	mockStrategy.EXPECT().GetRuntimeEngineVersion().Return(version.Version, nil)
	mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil)
	mockStrategy.EXPECT().ProcessData(gomock.Any()).DoAndReturn(func(data types.MarketData) error {
		s.Require().NotNil(capturedAPI, "engine must bind StrategyApi before ProcessData")
		_, logErr := capturedAPI.Log(context.Background(), &strategypb.LogRequest{
			Level:   strategypb.LogLevel_LOG_LEVEL_INFO,
			Message: fmt.Sprintf("tick %d", callIdx),
		})
		callIdx++
		return logErr
	}).Times(totalBars)

	err = eng.LoadStrategy(mockStrategy)
	s.Require().NoError(err)

	now := time.Now()
	testData := make([]types.MarketData, totalBars)
	for i := 0; i < totalBars; i++ {
		testData[i] = createTestMarketData("BTCUSDT", now.Add(time.Duration(i)*time.Minute), 50000+float64(i))
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

	err = eng.Run(context.Background(), engine.LiveTradingCallbacks{})
	s.Require().NoError(err)

	// In-memory buffer should hold all N logs (cumulative view preserved).
	logs, err := e.logStorage.GetLogs()
	s.Require().NoError(err)
	s.Require().Len(logs, totalBars, "logStorage should hold one log per tick")

	// And the parquet file should have exactly N rows — not 1 (drain bug),
	// not N*(N+1)/2 (re-write-all bug). Verify the messages match the
	// per-tick payloads to rule out N copies of the same row.
	runPath := e.sessionManager.GetCurrentRunPath()
	logsPath := filepath.Join(runPath, "logs.parquet")
	s.Require().FileExists(logsPath)

	rows, err := countParquetRows(logsPath)
	s.Require().NoError(err)
	s.Equal(totalBars, rows, "logs.parquet must contain one row per tick")

	db, err := sql.Open("duckdb", ":memory:")
	s.Require().NoError(err)
	defer db.Close()

	res, err := db.Query(fmt.Sprintf(`SELECT message FROM read_parquet('%s') ORDER BY timestamp ASC`, logsPath))
	s.Require().NoError(err)
	defer res.Close()

	messages := []string{}
	for res.Next() {
		var msg string
		s.Require().NoError(res.Scan(&msg))
		messages = append(messages, msg)
	}
	s.Require().NoError(res.Err())

	expected := make([]string, totalBars)
	for i := 0; i < totalBars; i++ {
		expected[i] = fmt.Sprintf("tick %d", i)
	}
	s.Equal(expected, messages, "every tick's log must be persisted exactly once with the original payload")
}

// TestRun_LogEveryTickWasmStrategy_E2E loads the real log_every_tick WASM
// strategy and streams N bars through a mock provider. The strategy emits one
// log per ProcessData call; the test asserts logs.parquet contains exactly N
// distinct rows with the expected per-bar payloads.
//
// This is the full end-to-end pipeline: real WASM module instantiated via
// wazero → go-plugin gRPC marshal/unmarshal → host StrategyApi.Log handler →
// LiveTradingLog buffer → tick-loop cursor → LogsWriter → parquet. The
// previously-added mock-strategy tests cover the host-side half; this one
// also covers the WASM-side half.
func (s *LiveTradingEngineV1TestSuite) TestRun_LogEveryTickWasmStrategy_E2E() {
	absPath, err := filepath.Abs(logEveryTickWasmPath)
	if err != nil || !fileExists(absPath) {
		s.T().Skipf("log_every_tick WASM not found at %s. Run `cd e2e/backtest/wasm && make build`.", absPath)
	}

	tempDir, err := os.MkdirTemp("", "live-trading-log-every-tick-e2e")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{EnableLogging: true})
	s.Require().NoError(err)

	err = eng.SetDataOutputPath(tempDir)
	s.Require().NoError(err)

	err = eng.LoadStrategyFromFile(absPath)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	s.Require().NotNil(e.logStorage)
	s.Require().NotNil(e.logsWriter)

	const totalBars = 6
	now := time.Now().Truncate(time.Minute)
	testData := make([]types.MarketData, totalBars)
	for i := 0; i < totalBars; i++ {
		testData[i] = createTestMarketData("BTCUSDT", now.Add(time.Duration(i)*time.Minute), 50000+float64(i))
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

	err = eng.Run(context.Background(), engine.LiveTradingCallbacks{})
	s.Require().NoError(err)

	// In-memory buffer should hold one log per bar.
	logs, err := e.logStorage.GetLogs()
	s.Require().NoError(err)
	s.Require().Len(logs, totalBars, "logStorage should hold one log per bar")

	// And the parquet file should have exactly N rows with the per-bar payload
	// the WASM strategy emitted. This is what the user originally reported as
	// "logs.parquet only has one log even we log everytime" — that bug would
	// surface here as either 1 row (drain-too-eagerly) or N*(N+1)/2 rows
	// (re-write-all). The cursor fix produces exactly N.
	runPath := e.sessionManager.GetCurrentRunPath()
	logsPath := filepath.Join(runPath, "logs.parquet")
	s.Require().FileExists(logsPath)

	rows, err := countParquetRows(logsPath)
	s.Require().NoError(err)
	s.Equal(totalBars, rows, "logs.parquet must contain exactly one row per bar")

	db, err := sql.Open("duckdb", ":memory:")
	s.Require().NoError(err)
	defer db.Close()

	res, err := db.Query(fmt.Sprintf(`SELECT message FROM read_parquet('%s') ORDER BY timestamp ASC`, logsPath))
	s.Require().NoError(err)
	defer res.Close()

	messages := []string{}
	for res.Next() {
		var msg string
		s.Require().NoError(res.Scan(&msg))
		messages = append(messages, msg)
	}
	s.Require().NoError(res.Err())

	expected := make([]string, totalBars)
	for i := 0; i < totalBars; i++ {
		expected[i] = fmt.Sprintf("tick BTCUSDT close=%.4f", 50000+float64(i))
	}
	s.Equal(expected, messages, "every WASM strategy log must reach logs.parquet exactly once")
}

// TestRun_DebugLogFromDefer_E2E verifies a real WASM strategy that emits a
// DEBUG log from inside a deferred function at ProcessData exit. zap drops
// DEBUG, so the running.log won't see these entries — logs.parquet is the
// only persistent record. The test asserts N bars produce N debug rows.
//
// This guards two regressions:
//   - Host's StrategyApi.Log silently dropping DEBUG before LogStorage write
//     (would leave parquet empty even when WASM did emit DEBUG)
//   - WASM/wazero/go-plugin breaking deferred host calls (the BUY/SELL paths
//     log inline so they would still work; the HOLD path uses defer and
//     would silently disappear)
func (s *LiveTradingEngineV1TestSuite) TestRun_DebugLogFromDefer_E2E() {
	absPath, err := filepath.Abs(logInDeferWasmPath)
	if err != nil || !fileExists(absPath) {
		s.T().Skipf("log_in_defer WASM not found at %s. Run `cd e2e/backtest/wasm && make build`.", absPath)
	}

	tempDir, err := os.MkdirTemp("", "live-trading-debug-defer-e2e")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	eng, err := NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{EnableLogging: true})
	s.Require().NoError(err)

	err = eng.SetDataOutputPath(tempDir)
	s.Require().NoError(err)

	err = eng.LoadStrategyFromFile(absPath)
	s.Require().NoError(err)

	e := eng.(*LiveTradingEngineV1)
	s.Require().NotNil(e.logStorage)
	s.Require().NotNil(e.logsWriter)

	const totalBars = 5
	now := time.Now().Truncate(time.Minute)
	testData := make([]types.MarketData, totalBars)
	for i := 0; i < totalBars; i++ {
		testData[i] = createTestMarketData("BTCUSDT", now.Add(time.Duration(i)*time.Minute), 50000+float64(i))
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

	err = eng.Run(context.Background(), engine.LiveTradingCallbacks{})
	s.Require().NoError(err)

	// In-memory buffer should hold one DEBUG entry per bar.
	logs, err := e.logStorage.GetLogs()
	s.Require().NoError(err)
	s.Require().Len(logs, totalBars, "logStorage should hold one DEBUG entry per bar")
	for i, l := range logs {
		s.Equal(types.LogLevelDebug, l.Level, "entry %d must be DEBUG", i)
	}

	// Parquet must persist all DEBUG rows — zap drops them but LogStorage
	// (and therefore the parquet writer) is level-agnostic.
	runPath := e.sessionManager.GetCurrentRunPath()
	logsPath := filepath.Join(runPath, "logs.parquet")
	s.Require().FileExists(logsPath)

	rows, err := countParquetRows(logsPath)
	s.Require().NoError(err)
	s.Equal(totalBars, rows, "logs.parquet must contain one DEBUG row per bar")

	db, err := sql.Open("duckdb", ":memory:")
	s.Require().NoError(err)
	defer db.Close()

	res, err := db.Query(fmt.Sprintf(
		`SELECT level, message FROM read_parquet('%s') ORDER BY timestamp ASC`,
		logsPath,
	))
	s.Require().NoError(err)
	defer res.Close()

	type row struct{ level, message string }
	rowsRead := []row{}
	for res.Next() {
		var r row
		s.Require().NoError(res.Scan(&r.level, &r.message))
		rowsRead = append(rowsRead, r)
	}
	s.Require().NoError(res.Err())

	for i, r := range rowsRead {
		s.Equal("debug", r.level, "parquet row %d must be debug level", i)
		s.Equal(fmt.Sprintf("defer HOLD symbol=BTCUSDT close=%.4f", 50000+float64(i)), r.message,
			"parquet row %d must carry the deferred log's payload", i)
	}
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
	// CheckConnection fails - trading provider is not connected
	mockTrading.EXPECT().CheckConnection(gomock.Any()).Return(errors.New("connection failed")).AnyTimes()
	err = eng.SetTradingProvider(mockTrading)
	s.Require().NoError(err)

	var mu sync.Mutex
	var statusUpdates []types.ProviderStatusUpdate

	onStatusChange := engine.OnProviderStatusChangeCallback(func(update types.ProviderStatusUpdate) error {
		mu.Lock()
		defer mu.Unlock()
		statusUpdates = append(statusUpdates, update)
		return nil
	})

	callbacks := engine.LiveTradingCallbacks{
		OnProviderStatusChange: &onStatusChange,
	}

	// Precheck failure aborts Run and returns the wrapped error.
	err = eng.Run(context.Background(), callbacks)
	s.Require().Error(err)
	s.Contains(err.Error(), "trading provider precheck failed")

	// Force-emitted disconnected status should still reach the callback so the UI
	// can render the failure state.
	mu.Lock()
	defer mu.Unlock()
	s.GreaterOrEqual(len(statusUpdates), 1)
}

func (s *LiveTradingEngineV1TestSuite) TestRun_TradingProviderConnectionFails_CallsOnError() {
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

	// Precheck failure aborts Run and surfaces the wrapped error via both the
	// return value and the OnError callback.
	err = eng.Run(context.Background(), callbacks)
	s.Require().Error(err)
	s.Contains(err.Error(), "trading provider precheck failed")
	s.Contains(err.Error(), "invalid API key")

	mu.Lock()
	defer mu.Unlock()
	s.Require().NotNil(errorReceived, "OnError should have been called when CheckConnection fails")
	s.Contains(errorReceived.Error(), "trading provider precheck failed")
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
