package swiftargo

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
)

type TradingTestSuite struct {
	suite.Suite
}

func TestTradingTestSuite(t *testing.T) {
	suite.Run(t, new(TradingTestSuite))
}

// mockTradingHelper implements TradingEngineHelper for testing.
type mockTradingHelper struct {
	mu                   sync.Mutex
	startCalled          bool
	stopCalled           bool
	marketDataCalls      int
	orderPlacedCalls     int
	orderFilledCalls     int
	errorCalls           int
	strategyErrors       int
	lastSymbols          []string
	lastInterval         string
	lastPreviousDataPath string
	lastError            error
}

func (m *mockTradingHelper) OnEngineStart(symbols StringCollection, interval string, previousDataPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startCalled = true
	m.lastSymbols = make([]string, symbols.Size())
	for i := 0; i < symbols.Size(); i++ {
		m.lastSymbols[i] = symbols.Get(i)
	}
	m.lastInterval = interval
	m.lastPreviousDataPath = previousDataPath
	return nil
}

func (m *mockTradingHelper) OnEngineStop(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopCalled = true
	m.lastError = err
}

func (m *mockTradingHelper) OnMarketData(symbol string, timestamp int64, open, high, low, close, volume float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.marketDataCalls++
	return nil
}

func (m *mockTradingHelper) OnOrderPlaced(orderJSON string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.orderPlacedCalls++
	return nil
}

func (m *mockTradingHelper) OnOrderFilled(orderJSON string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.orderFilledCalls++
	return nil
}

func (m *mockTradingHelper) OnError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCalls++
	m.lastError = err
}

func (m *mockTradingHelper) OnStrategyError(symbol string, timestamp int64, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.strategyErrors++
	m.lastError = err
}

// Test GetSupportedTradingProviders
func (suite *TradingTestSuite) TestGetSupportedTradingProviders() {
	providers := GetSupportedTradingProviders()

	suite.NotNil(providers)
	suite.GreaterOrEqual(providers.Size(), 2)

	// Check that binance-paper and binance-live are in the list
	found := make(map[string]bool)
	for i := 0; i < providers.Size(); i++ {
		found[providers.Get(i)] = true
	}
	suite.True(found["binance-paper"], "should contain binance-paper")
	suite.True(found["binance-live"], "should contain binance-live")
}

// Test GetTradingProviderKeychainFields
func (suite *TradingTestSuite) TestGetTradingProviderKeychainFields_BinancePaper() {
	fields := GetTradingProviderKeychainFields("binance-paper")

	suite.NotNil(fields)
	suite.Equal(2, fields.Size())
	suite.Equal("apiKey", fields.Get(0))
	suite.Equal("secretKey", fields.Get(1))
}

func (suite *TradingTestSuite) TestGetTradingProviderKeychainFields_BinanceLive() {
	fields := GetTradingProviderKeychainFields("binance-live")

	suite.NotNil(fields)
	suite.Equal(2, fields.Size())
	suite.Equal("apiKey", fields.Get(0))
	suite.Equal("secretKey", fields.Get(1))
}

func (suite *TradingTestSuite) TestGetTradingProviderKeychainFields_InvalidProvider() {
	fields := GetTradingProviderKeychainFields("invalid")

	suite.Nil(fields)
}

// Test GetTradingProviderSchema
func (suite *TradingTestSuite) TestGetTradingProviderSchema_BinancePaper() {
	schema := GetTradingProviderSchema("binance-paper")

	suite.NotEmpty(schema)

	// Verify it's valid JSON
	var schemaMap map[string]interface{}
	err := json.Unmarshal([]byte(schema), &schemaMap)
	suite.NoError(err)

	// Check properties exist
	properties, ok := schemaMap["properties"].(map[string]interface{})
	suite.True(ok)
	suite.Contains(properties, "apiKey")
	suite.Contains(properties, "secretKey")
}

func (suite *TradingTestSuite) TestGetTradingProviderSchema_BinanceLive() {
	schema := GetTradingProviderSchema("binance-live")

	suite.NotEmpty(schema)

	// Verify it's valid JSON
	var schemaMap map[string]interface{}
	err := json.Unmarshal([]byte(schema), &schemaMap)
	suite.NoError(err)

	// Check properties exist
	properties, ok := schemaMap["properties"].(map[string]interface{})
	suite.True(ok)
	suite.Contains(properties, "apiKey")
	suite.Contains(properties, "secretKey")
}

func (suite *TradingTestSuite) TestGetTradingProviderSchema_InvalidProvider() {
	schema := GetTradingProviderSchema("invalid")

	suite.Empty(schema)
}

// Test GetTradingProviderInfo
func (suite *TradingTestSuite) TestGetTradingProviderInfo_BinancePaper() {
	info, err := GetTradingProviderInfo("binance-paper")

	suite.NoError(err)
	suite.NotNil(info)
	suite.Equal("binance-paper", info.Name)
	suite.True(info.IsPaperTrading)
	suite.NotEmpty(info.DisplayName)
	suite.NotEmpty(info.Description)
}

func (suite *TradingTestSuite) TestGetTradingProviderInfo_BinanceLive() {
	info, err := GetTradingProviderInfo("binance-live")

	suite.NoError(err)
	suite.NotNil(info)
	suite.Equal("binance-live", info.Name)
	suite.False(info.IsPaperTrading)
	suite.NotEmpty(info.DisplayName)
	suite.NotEmpty(info.Description)
}

func (suite *TradingTestSuite) TestGetTradingProviderInfo_InvalidProvider() {
	info, err := GetTradingProviderInfo("invalid")

	suite.Error(err)
	suite.Nil(info)
}

// Test GetLiveTradingEngineConfigSchema
func (suite *TradingTestSuite) TestGetLiveTradingEngineConfigSchema() {
	schema := GetLiveTradingEngineConfigSchema()

	suite.NotEmpty(schema)

	// Verify it's valid JSON
	var schemaMap map[string]any
	err := json.Unmarshal([]byte(schema), &schemaMap)
	suite.NoError(err)

	// Should NOT have $ref at top level (DoNotReference: true)
	suite.NotContains(schemaMap, "$ref")
	suite.NotContains(schemaMap, "$defs")

	// Check that properties are inlined at top level
	properties, ok := schemaMap["properties"].(map[string]any)
	suite.True(ok, "schema should have top-level properties")
	suite.Contains(properties, "market_data_cache_size")
	suite.Contains(properties, "enable_logging")
	suite.Contains(properties, "prefetch")

	// Verify prefetch is inlined (not a $ref)
	prefetch, ok := properties["prefetch"].(map[string]any)
	suite.True(ok)
	suite.NotContains(prefetch, "$ref")
	prefetchProps, ok := prefetch["properties"].(map[string]any)
	suite.True(ok, "prefetch should have inlined properties")
	suite.Contains(prefetchProps, "enabled")
	suite.Contains(prefetchProps, "start_time_type")
	suite.Contains(prefetchProps, "days")
}

// Test GetSupportedMarketDataProviders
func (suite *TradingTestSuite) TestGetSupportedMarketDataProviders() {
	providers := GetSupportedMarketDataProviders()

	suite.NotNil(providers)
	suite.GreaterOrEqual(providers.Size(), 2)

	// Check that binance and polygon are in the list
	found := make(map[string]bool)
	for i := 0; i < providers.Size(); i++ {
		found[providers.Get(i)] = true
	}
	suite.True(found["binance"], "should contain binance")
	suite.True(found["polygon"], "should contain polygon")
}

// Test NewTradingEngine
func (suite *TradingTestSuite) TestNewTradingEngine() {
	helper := &mockTradingHelper{}
	eng, err := NewTradingEngine(helper)

	suite.NoError(err)
	suite.NotNil(eng)
	suite.NotNil(eng.helper)
	suite.NotNil(eng.engine)
}

func (suite *TradingTestSuite) TestNewTradingEngine_NilHelper() {
	eng, err := NewTradingEngine(nil)

	suite.NoError(err)
	suite.NotNil(eng)
	suite.Nil(eng.helper)
}

// Test Initialize
func (suite *TradingTestSuite) TestInitialize_ValidConfig() {
	helper := &mockTradingHelper{}
	eng, err := NewTradingEngine(helper)
	suite.NoError(err)

	configJSON := `{
		"market_data_cache_size": 1000,
		"enable_logging": false
	}`

	err = eng.Initialize(configJSON)
	suite.NoError(err)
}

func (suite *TradingTestSuite) TestInitialize_InvalidJSON() {
	helper := &mockTradingHelper{}
	eng, err := NewTradingEngine(helper)
	suite.NoError(err)

	err = eng.Initialize(`{invalid json}`)
	suite.Error(err)
	suite.Contains(err.Error(), "failed to parse engine config")
}

func (suite *TradingTestSuite) TestInitialize_EmptySymbols() {
	helper := &mockTradingHelper{}
	eng, err := NewTradingEngine(helper)
	suite.NoError(err)

	configJSON := `{}`

	err = eng.Initialize(configJSON)
	// Should succeed - validation happens at Run time
	suite.NoError(err)
}

// Test SetTradingProvider
func (suite *TradingTestSuite) TestSetTradingProvider_InvalidProvider() {
	helper := &mockTradingHelper{}
	eng, err := NewTradingEngine(helper)
	suite.NoError(err)

	err = eng.SetTradingProvider("invalid", `{}`)
	suite.Error(err)
}

func (suite *TradingTestSuite) TestSetTradingProvider_InvalidJSON() {
	helper := &mockTradingHelper{}
	eng, err := NewTradingEngine(helper)
	suite.NoError(err)

	err = eng.SetTradingProvider("binance-paper", `{invalid json}`)
	suite.Error(err)
	suite.Contains(err.Error(), "failed to parse")
}

func (suite *TradingTestSuite) TestSetTradingProvider_MissingRequiredFields() {
	helper := &mockTradingHelper{}
	eng, err := NewTradingEngine(helper)
	suite.NoError(err)

	// Missing apiKey and secretKey
	err = eng.SetTradingProvider("binance-paper", `{}`)
	suite.Error(err)
}

// Test SetMarketDataProvider
func (suite *TradingTestSuite) TestSetMarketDataProvider_Binance() {
	helper := &mockTradingHelper{}
	eng, err := NewTradingEngine(helper)
	suite.NoError(err)

	// Binance requires symbols and interval
	err = eng.SetMarketDataProvider("binance", `{"symbols": ["BTCUSDT"], "interval": "1m"}`)
	suite.NoError(err)
}

func (suite *TradingTestSuite) TestSetMarketDataProvider_PolygonMissingApiKey() {
	helper := &mockTradingHelper{}
	eng, err := NewTradingEngine(helper)
	suite.NoError(err)

	// Polygon requires apiKey, symbols, and interval
	err = eng.SetMarketDataProvider("polygon", `{"symbols": ["SPY"], "interval": "1m"}`)
	suite.Error(err)
	suite.Contains(err.Error(), "ApiKey")
}

func (suite *TradingTestSuite) TestSetMarketDataProvider_PolygonEmptyApiKey() {
	helper := &mockTradingHelper{}
	eng, err := NewTradingEngine(helper)
	suite.NoError(err)

	err = eng.SetMarketDataProvider("polygon", `{"symbols": ["SPY"], "interval": "1m", "apiKey": ""}`)
	suite.Error(err)
	suite.Contains(err.Error(), "ApiKey")
}

func (suite *TradingTestSuite) TestSetMarketDataProvider_InvalidProvider() {
	helper := &mockTradingHelper{}
	eng, err := NewTradingEngine(helper)
	suite.NoError(err)

	err = eng.SetMarketDataProvider("invalid", `{}`)
	suite.Error(err)
	suite.Contains(err.Error(), "unsupported")
}

// Test SetWasm
func (suite *TradingTestSuite) TestSetWasm_InvalidPath() {
	helper := &mockTradingHelper{}
	eng, err := NewTradingEngine(helper)
	suite.NoError(err)

	err = eng.SetWasm("/nonexistent/strategy.wasm")
	suite.Error(err)
}

// Test SetStrategyConfig
func (suite *TradingTestSuite) TestSetStrategyConfig() {
	helper := &mockTradingHelper{}
	eng, err := NewTradingEngine(helper)
	suite.NoError(err)

	err = eng.SetStrategyConfig(`{"fastPeriod": 10, "slowPeriod": 20}`)
	suite.NoError(err)
}

// Test Cancel
func (suite *TradingTestSuite) TestCancel_NoRunInProgress() {
	helper := &mockTradingHelper{}
	eng, err := NewTradingEngine(helper)
	suite.NoError(err)

	cancelled := eng.Cancel()
	suite.False(cancelled)
}

func (suite *TradingTestSuite) TestCancelThreadSafety() {
	helper := &mockTradingHelper{}
	eng, err := NewTradingEngine(helper)
	suite.NoError(err)

	// Test that Cancel is safe to call concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			eng.Cancel()
		}()
	}
	wg.Wait()
	// If we get here without panic, test passes
}

// Test createCallbacks with nil helper
func (suite *TradingTestSuite) TestCreateCallbacks_NilHelper() {
	eng, err := NewTradingEngine(nil)
	suite.NoError(err)

	callbacks := eng.createCallbacks()

	// All callbacks should be nil when helper is nil
	suite.Nil(callbacks.OnEngineStart)
	suite.Nil(callbacks.OnEngineStop)
	suite.Nil(callbacks.OnMarketData)
	suite.Nil(callbacks.OnOrderPlaced)
	suite.Nil(callbacks.OnOrderFilled)
	suite.Nil(callbacks.OnError)
	suite.Nil(callbacks.OnStrategyError)
}

// Test createCallbacks with valid helper
func (suite *TradingTestSuite) TestCreateCallbacks_ValidHelper() {
	helper := &mockTradingHelper{}
	eng, err := NewTradingEngine(helper)
	suite.NoError(err)

	callbacks := eng.createCallbacks()

	// All callbacks should be non-nil when helper is provided
	suite.NotNil(callbacks.OnEngineStart)
	suite.NotNil(callbacks.OnEngineStop)
	suite.NotNil(callbacks.OnMarketData)
	suite.NotNil(callbacks.OnOrderPlaced)
	suite.NotNil(callbacks.OnOrderFilled)
	suite.NotNil(callbacks.OnError)
	suite.NotNil(callbacks.OnStrategyError)
}
