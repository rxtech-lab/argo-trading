package wasm

import (
	"testing"

	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// WasmUnitTestSuite tests the wasm package without requiring a compiled wasm file
type WasmUnitTestSuite struct {
	suite.Suite
}

func TestWasmUnitSuite(t *testing.T) {
	suite.Run(t, new(WasmUnitTestSuite))
}

func (suite *WasmUnitTestSuite) TestNewStrategyWasmRuntime_FileNotFound() {
	// Test creating a runtime with a non-existent file path
	runtime, err := NewStrategyWasmRuntime("/nonexistent/path/plugin.wasm")
	suite.Error(err)
	suite.Nil(runtime)
	suite.Contains(err.Error(), "file does not exist")
}

func (suite *WasmUnitTestSuite) TestNewStrategyWasmRuntimeFromBytes_EmptyBytes() {
	// Test creating a runtime with empty bytes (should succeed initially)
	runtime, err := NewStrategyWasmRuntimeFromBytes([]byte{})
	suite.NoError(err)
	suite.NotNil(runtime)
}

func (suite *WasmUnitTestSuite) TestStrategyWasmRuntime_NameWithoutInit() {
	// Test Name() returns empty string when strategy is not initialized
	runtime, err := NewStrategyWasmRuntimeFromBytes([]byte{0x00, 0x61, 0x73, 0x6d})
	suite.NoError(err)

	wasmRuntime := runtime.(*StrategyWasmRuntime)
	name := wasmRuntime.Name()
	suite.Empty(name)
}

func (suite *WasmUnitTestSuite) TestStrategyWasmRuntime_GetDescriptionWithoutInit() {
	// Test GetDescription() returns error when strategy is not initialized
	runtime, err := NewStrategyWasmRuntimeFromBytes([]byte{0x00, 0x61, 0x73, 0x6d})
	suite.NoError(err)

	wasmRuntime := runtime.(*StrategyWasmRuntime)
	description, err := wasmRuntime.GetDescription()
	suite.Error(err)
	suite.Contains(err.Error(), "strategy is not initialized")
	suite.Empty(description)
}

func (suite *WasmUnitTestSuite) TestStrategyWasmRuntime_InitializeWithoutApi() {
	// Test Initialize() returns error when strategy is not initialized
	runtime, err := NewStrategyWasmRuntimeFromBytes([]byte{0x00, 0x61, 0x73, 0x6d})
	suite.NoError(err)

	wasmRuntime := runtime.(*StrategyWasmRuntime)
	err = wasmRuntime.Initialize("{}")
	suite.Error(err)
	suite.Contains(err.Error(), "strategy is not initialized")
}

func (suite *WasmUnitTestSuite) TestStrategyWasmRuntime_ProcessDataWithoutInit() {
	// Test ProcessData() returns error when strategy is not initialized
	runtime, err := NewStrategyWasmRuntimeFromBytes([]byte{0x00, 0x61, 0x73, 0x6d})
	suite.NoError(err)

	wasmRuntime := runtime.(*StrategyWasmRuntime)
	err = wasmRuntime.ProcessData(types.MarketData{Symbol: "TEST"})
	suite.Error(err)
	suite.Contains(err.Error(), "strategy is not initialized")
}

func (suite *WasmUnitTestSuite) TestStrategyWasmRuntime_GetRuntimeEngineVersionWithoutInit() {
	// Test GetRuntimeEngineVersion() returns error when strategy is not initialized
	runtime, err := NewStrategyWasmRuntimeFromBytes([]byte{0x00, 0x61, 0x73, 0x6d})
	suite.NoError(err)

	wasmRuntime := runtime.(*StrategyWasmRuntime)
	version, err := wasmRuntime.GetRuntimeEngineVersion()
	suite.Error(err)
	suite.Contains(err.Error(), "strategy is not initialized")
	suite.Empty(version)
}

func TestStrategyWasmRuntime_LoadPluginBothSet(t *testing.T) {
	// Test that loadPlugin returns an error when both wasmFilePath and wasmBytes are set
	runtime := &StrategyWasmRuntime{
		wasmFilePath: "/some/path.wasm",
		wasmBytes:    []byte{0x00, 0x61, 0x73, 0x6d},
	}

	_, err := runtime.loadPlugin(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "both wasmFilePath and wasmBytes are set")
}

func TestStrategyWasmRuntime_LoadPluginNeitherSet(t *testing.T) {
	// Test that loadPlugin returns an error when neither wasmFilePath nor wasmBytes are set
	runtime := &StrategyWasmRuntime{
		wasmFilePath: "",
		wasmBytes:    nil,
	}

	_, err := runtime.loadPlugin(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "either wasmFilePath or wasmBytes must be set")
}

func TestNewStrategyWasmRuntimeFromBytes_ValidBytes(t *testing.T) {
	// Test creating a runtime with valid wasm magic number bytes
	wasmMagic := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	runtime, err := NewStrategyWasmRuntimeFromBytes(wasmMagic)
	assert.NoError(t, err)
	assert.NotNil(t, runtime)
}
