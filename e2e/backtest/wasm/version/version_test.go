package main

import (
	"testing"

	"github.com/rxtech-lab/argo-trading/internal/runtime/wasm"
	"github.com/rxtech-lab/argo-trading/internal/version"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// VersionTestSuite tests the engine version functionality
type VersionTestSuite struct {
	suite.Suite
}

func TestVersionTestSuite(t *testing.T) {
	suite.Run(t, new(VersionTestSuite))
}

// TestGetEngineVersion tests that GetRuntimeEngineVersion returns the correct version
func (s *VersionTestSuite) TestGetEngineVersion() {
	// Use an existing WASM file that was built with the new version export
	wasmPath := "../sma/sma_plugin.wasm"

	runtime, err := wasm.NewStrategyWasmRuntime(wasmPath)
	require.NoError(s.T(), err)

	// Initialize the strategy API (required before calling GetRuntimeEngineVersion)
	err = runtime.InitializeApi(nil)
	require.NoError(s.T(), err)

	// Get the engine version
	engineVersion, err := runtime.GetRuntimeEngineVersion()
	require.NoError(s.T(), err)

	// The version should match the version that was compiled into the WASM binary
	// In development, this is typically "main"
	require.NotEmpty(s.T(), engineVersion, "engine version should not be empty")

	// Verify it matches what we expect (the version from internal/version)
	expectedVersion := version.GetVersion()
	require.Equal(s.T(), expectedVersion, engineVersion, "engine version should match the compiled version")
}

// TestGetEngineVersionWithoutInit tests that GetRuntimeEngineVersion returns error when not initialized
func (s *VersionTestSuite) TestGetEngineVersionWithoutInit() {
	wasmPath := "../sma/sma_plugin.wasm"

	runtime, err := wasm.NewStrategyWasmRuntime(wasmPath)
	require.NoError(s.T(), err)

	// Try to get version without initializing - should fail
	_, err = runtime.GetRuntimeEngineVersion()
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "strategy is not initialized")
}
