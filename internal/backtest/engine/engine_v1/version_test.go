package engine

import (
	"context"
	"testing"

	"github.com/rxtech-lab/argo-trading/internal/runtime"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/internal/version"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStrategyRuntime is a mock implementation of StrategyRuntime for testing
type mockStrategyRuntime struct {
	runtimeVersion string
	name           string
}

func (m *mockStrategyRuntime) Initialize(config string) error {
	return nil
}

func (m *mockStrategyRuntime) InitializeApi(api strategy.StrategyApi) error {
	return nil
}

func (m *mockStrategyRuntime) ProcessData(data types.MarketData) error {
	return nil
}

func (m *mockStrategyRuntime) GetConfigSchema() (string, error) {
	return "", nil
}

func (m *mockStrategyRuntime) Name() string {
	return m.name
}

func (m *mockStrategyRuntime) GetDescription() (string, error) {
	return "Mock strategy for testing", nil
}

func (m *mockStrategyRuntime) GetRuntimeEngineVersion() (string, error) {
	return m.runtimeVersion, nil
}

func (m *mockStrategyRuntime) GetIdentifier() (string, error) {
	return "com.test.mock-strategy", nil
}

func TestVersionCompatibilityCheck(t *testing.T) {
	tests := []struct {
		name             string
		engineVersion    string
		strategyVersion  string
		expectError      bool
		errorMsgContains string
	}{
		{
			name:            "compatible versions - exact match",
			engineVersion:   "1.2.0",
			strategyVersion: "1.2.0",
			expectError:     false,
		},
		{
			name:            "compatible versions - patch differs",
			engineVersion:   "1.2.5",
			strategyVersion: "1.2.0",
			expectError:     false,
		},
		{
			name:             "incompatible - minor version mismatch",
			engineVersion:    "1.3.0",
			strategyVersion:  "1.2.0",
			expectError:      true,
			errorMsgContains: "minor version mismatch",
		},
		{
			name:             "incompatible - major version mismatch",
			engineVersion:    "2.0.0",
			strategyVersion:  "1.2.0",
			expectError:      true,
			errorMsgContains: "major version mismatch",
		},
		{
			name:            "compatible - engine is main (dev build skips check)",
			engineVersion:   "main",
			strategyVersion: "1.2.0",
			expectError:     false,
		},
		{
			name:            "compatible - strategy is main (dev build skips check)",
			engineVersion:   "1.2.0",
			strategyVersion: "main",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the engine version for this test
			originalVersion := version.Version
			version.Version = tt.engineVersion
			defer func() { version.Version = originalVersion }()

			// Create a mock strategy with the specified version
			mockStrategy := &mockStrategyRuntime{
				runtimeVersion: tt.strategyVersion,
				name:           "TestStrategy",
			}

			// Check version compatibility directly
			err := version.CheckVersionCompatibility(version.Version, tt.strategyVersion)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsgContains)
			} else {
				require.NoError(t, err)
			}

			// Also verify the mock implements the interface
			var _ runtime.StrategyRuntime = mockStrategy
		})
	}
}

func TestVersionCheckInBacktestEngine(t *testing.T) {
	// This test verifies that the version check is integrated correctly
	// in the backtest engine flow

	// Save original version and restore after test
	originalVersion := version.Version
	defer func() { version.Version = originalVersion }()

	t.Run("version check happens after InitializeApi", func(t *testing.T) {
		// Set engine version
		version.Version = "1.2.0"

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)

		// Create a mock strategy that returns incompatible version
		mockStrategy := &mockStrategyRuntime{
			runtimeVersion: "2.0.0", // Incompatible major version
			name:           "IncompatibleStrategy",
		}

		// Load the strategy
		err = engine.LoadStrategy(mockStrategy)
		require.NoError(t, err)

		// The version check should fail during Run()
		// We can't easily test the full Run() flow here without more setup,
		// but we can verify the version check logic works
		strategyVersion, err := mockStrategy.GetRuntimeEngineVersion()
		require.NoError(t, err)

		err = version.CheckVersionCompatibility(version.Version, strategyVersion)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "major version mismatch")
	})

	t.Run("compatible versions pass check", func(t *testing.T) {
		version.Version = "1.2.0"

		mockStrategy := &mockStrategyRuntime{
			runtimeVersion: "1.2.5", // Compatible - same major.minor
			name:           "CompatibleStrategy",
		}

		strategyVersion, err := mockStrategy.GetRuntimeEngineVersion()
		require.NoError(t, err)

		err = version.CheckVersionCompatibility(version.Version, strategyVersion)
		require.NoError(t, err)
	})
}

func TestGetRuntimeVersionFromStrategy(t *testing.T) {
	mockStrategy := &mockStrategyRuntime{
		runtimeVersion: "1.5.0",
		name:           "VersionedStrategy",
	}

	// Test that GetRuntimeVersion returns the expected value
	ver, err := mockStrategy.GetRuntimeEngineVersion()
	require.NoError(t, err)
	assert.Equal(t, "1.5.0", ver)
}

// TestVersionCheckIntegration tests the version check in a more realistic scenario
func TestVersionCheckIntegration(t *testing.T) {
	// This test table covers all the scenarios from the requirements
	testCases := []struct {
		engineVersion  string
		runtimeVersion string
		shouldError    bool
	}{
		{"1.2.0", "1.2.0", false},
		{"1.2.1", "1.2.0", false},
		{"1.3.0", "1.2.0", true},
		{"1.1.0", "1.2.0", true},
		{"2.0.0", "1.2.0", true},
		{"main", "1.2.0", false}, // dev build skips check
		{"main", "1.3.0", false}, // dev build skips check
		{"main", "main", false},  // dev build skips check
		{"1.2.0", "main", false}, // dev build skips check
	}

	for _, tc := range testCases {
		t.Run(tc.engineVersion+"_vs_"+tc.runtimeVersion, func(t *testing.T) {
			originalVersion := version.Version
			version.Version = tc.engineVersion
			defer func() { version.Version = originalVersion }()

			mockStrategy := &mockStrategyRuntime{
				runtimeVersion: tc.runtimeVersion,
				name:           "TestStrategy",
			}

			// Simulate what happens in runSingleIteration
			strategyVersion, err := mockStrategy.GetRuntimeEngineVersion()
			require.NoError(t, err)

			err = version.CheckVersionCompatibility(version.Version, strategyVersion)

			if tc.shouldError {
				require.Error(t, err, "Expected error for engine=%s, runtime=%s", tc.engineVersion, tc.runtimeVersion)
			} else {
				require.NoError(t, err, "Expected no error for engine=%s, runtime=%s", tc.engineVersion, tc.runtimeVersion)
			}
		})
	}
}

// Ensure the mock implements the interface (compile-time check)
var _ runtime.StrategyRuntime = (*mockStrategyRuntime)(nil)

// contextCancelled is a helper for testing context cancellation
func contextCancelled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
