package wasm

import (
	"os"
	"testing"

	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/runtime"
	"github.com/rxtech-lab/argo-trading/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type StrategyTestSuite struct {
	suite.Suite
	runtime               runtime.StrategyRuntime
	mockCache             *mocks.MockCache
	mockTradingSystem     *mocks.MockTradingSystem
	logger                *logger.Logger
	ctrl                  *gomock.Controller
	mockIndicatorRegistry *mocks.MockIndicatorRegistry
}

// Test Suite
func (suite *StrategyTestSuite) SetupSuite() {
	logger, err := logger.NewLogger()
	suite.Require().NoError(err)
	suite.logger = logger
}

func (suite *StrategyTestSuite) TearDownSuite() {
	// Nothing to clean up at the suite level
}

func (suite *StrategyTestSuite) SetupTest() {
	// Create controller for mocks
	suite.ctrl = gomock.NewController(suite.T())

	// Create mock objects
	suite.mockCache = mocks.NewMockCache(suite.ctrl)
	suite.mockTradingSystem = mocks.NewMockTradingSystem(suite.ctrl)
	suite.mockIndicatorRegistry = mocks.NewMockIndicatorRegistry(suite.ctrl)

	// Initialize strategy runtime with mocks
	// Note: WASM strategies can be challenging to test with mocks since they
	// run in an isolated environment
	var err error
	suite.runtime, err = NewStrategyWasmRuntime("../../../examples/strategy/plugin.wasm")
	if err != nil {
		suite.T().Skip("Skipping test: wasm file not found - run 'make build' in examples/strategy first")
	}

	err = suite.runtime.InitializeApi(NewWasmStrategyApi(&runtime.RuntimeContext{
		Cache:             suite.mockCache,
		TradingSystem:     suite.mockTradingSystem,
		IndicatorRegistry: suite.mockIndicatorRegistry,
	}))
	suite.Require().NoError(err)
}

func (suite *StrategyTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestStrategySuite(t *testing.T) {
	suite.Run(t, new(StrategyTestSuite))
}

// TestInitializeApiWithPath tests the InitializeApi function with a file path
func (suite *StrategyTestSuite) TestInitializeApiWithPath() {
	// Create a new runtime with path
	testRuntime, err := NewStrategyWasmRuntime("../../../examples/strategy/plugin.wasm")
	suite.Require().NoError(err)

	// Initialize the API with the proper context type
	err = testRuntime.InitializeApi(NewWasmStrategyApi(&runtime.RuntimeContext{
		Cache:             suite.mockCache,
		TradingSystem:     suite.mockTradingSystem,
		IndicatorRegistry: suite.mockIndicatorRegistry,
	}))
	suite.Require().NoError(err)

	// Verify that name can be retrieved (which means initialization worked)
	name := testRuntime.Name()
	suite.Require().NotEmpty(name)
}

// TestInitializeApiWithBytes tests the InitializeApi function with bytes
func (suite *StrategyTestSuite) TestInitializeApiWithBytes() {
	// Read the wasm file into bytes
	wasmBytes, err := os.ReadFile("../../../examples/strategy/plugin.wasm")
	suite.Require().NoError(err)
	suite.Require().NotEmpty(wasmBytes)

	// Create a new runtime with bytes
	testRuntime, err := NewStrategyWasmRuntimeFromBytes(wasmBytes)
	suite.Require().NoError(err)

	// Initialize the API with the proper context type
	err = testRuntime.InitializeApi(NewWasmStrategyApi(&runtime.RuntimeContext{
		Cache:             suite.mockCache,
		TradingSystem:     suite.mockTradingSystem,
		IndicatorRegistry: suite.mockIndicatorRegistry,
	}))
	suite.Require().NoError(err)

	// Verify that name can be retrieved (which means initialization worked)
	name := testRuntime.Name()
	suite.Require().NotEmpty(name)
}

func (suite *StrategyTestSuite) TestBothNotSet() {
	testRuntime, err := NewStrategyWasmRuntimeFromBytes([]byte{})
	suite.Require().NoError(err)

	err = testRuntime.InitializeApi(NewWasmStrategyApi(&runtime.RuntimeContext{
		Cache:             suite.mockCache,
		TradingSystem:     suite.mockTradingSystem,
		IndicatorRegistry: suite.mockIndicatorRegistry,
	}))
	suite.Require().Error(err)
}
