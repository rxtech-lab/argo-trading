package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/moznion/go-optional"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yamlv3 "gopkg.in/yaml.v3"

	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/commission_fee"
)

// TestConfigToYAMLNode verifies that a BacktestEngineV1Config is converted to
// a YAML node whose re-marshaled form contains the expected scalar fields and
// renders optional time fields as readable timestamps.
func TestConfigToYAMLNode(t *testing.T) {
	startTime := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	cfg := BacktestEngineV1Config{
		InitialCapital:       12345.67,
		Broker:               commission_fee.BrokerInteractiveBroker,
		StartTime:            optional.Some(startTime),
		EndTime:              optional.None[time.Time](),
		DecimalPrecision:     2,
		MarketDataCacheSize:  500,
		PortfolioCalculation: PortfolioCalculationFIFO,
	}

	node, err := configToYAMLNode(cfg)
	require.NoError(t, err)
	require.NotNil(t, node)
	assert.NotEqual(t, yamlv3.DocumentNode, node.Kind, "document wrapper should be unwrapped")

	out, err := yamlv3.Marshal(node)
	require.NoError(t, err)

	rendered := string(out)
	assert.Contains(t, rendered, "initial_capital: 12345.67")
	assert.Contains(t, rendered, "decimal_precision: 2")
	assert.Contains(t, rendered, "market_data_cache_size: 500")
	assert.Contains(t, rendered, "portfolio_calculation: fifo")
	assert.Contains(t, rendered, "start_time: 2024-01-02T03:04:05Z")
	assert.NotContains(t, rendered, "end_time:", "None optional time fields should be omitted")
}

// TestStrategyConfigToYAMLNode verifies the strategy config string is parsed
// into a structured YAML node, and that empty input yields a nil node so the
// field is omitted from stats.yaml.
func TestStrategyConfigToYAMLNode(t *testing.T) {
	t.Run("structured", func(t *testing.T) {
		node, err := strategyConfigToYAMLNode("threshold: 0.25\nlookback: 14\n")
		require.NoError(t, err)
		require.NotNil(t, node)

		out, err := yamlv3.Marshal(node)
		require.NoError(t, err)

		rendered := string(out)
		assert.Contains(t, rendered, "threshold: 0.25")
		assert.Contains(t, rendered, "lookback: 14")
	})

	t.Run("empty", func(t *testing.T) {
		node, err := strategyConfigToYAMLNode("   \n")
		require.NoError(t, err)
		assert.Nil(t, node)
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := strategyConfigToYAMLNode(":\n  - [unbalanced")
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "strategy config"))
	})
}
