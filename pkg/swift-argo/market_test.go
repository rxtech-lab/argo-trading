package swiftargo

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
)

type MarketTestSuite struct {
	suite.Suite
}

func TestMarketTestSuite(t *testing.T) {
	suite.Run(t, new(MarketTestSuite))
}

// mockDownloaderHelper implements MarketDownloaderHelper for testing.
type mockDownloaderHelper struct {
	mu            sync.Mutex
	progressCalls []struct {
		current float64
		total   float64
		message string
	}
}

func (m *mockDownloaderHelper) OnDownloadProgress(current, total float64, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.progressCalls = append(m.progressCalls, struct {
		current float64
		total   float64
		message string
	}{current, total, message})
}

func (suite *MarketTestSuite) TestGetSupportedDownloadClients() {
	clients := GetSupportedDownloadClients()

	suite.NotNil(clients)
	suite.GreaterOrEqual(clients.Size(), 2)

	// Check that polygon and binance are in the list
	found := make(map[string]bool)
	for i := 0; i < clients.Size(); i++ {
		found[clients.Get(i)] = true
	}
	suite.True(found["polygon"], "should contain polygon")
	suite.True(found["binance"], "should contain binance")
}

func (suite *MarketTestSuite) TestGetDownloadClientSchema_Polygon() {
	schema := GetDownloadClientSchema("polygon")

	suite.NotEmpty(schema)

	// Verify it's valid JSON
	var schemaMap map[string]interface{}
	err := json.Unmarshal([]byte(schema), &schemaMap)
	suite.NoError(err)

	// Check properties exist
	properties, ok := schemaMap["properties"].(map[string]interface{})
	suite.True(ok)
	suite.Contains(properties, "ticker")
	suite.Contains(properties, "apiKey")
}

func (suite *MarketTestSuite) TestGetDownloadClientSchema_Binance() {
	schema := GetDownloadClientSchema("binance")

	suite.NotEmpty(schema)

	// Verify it's valid JSON
	var schemaMap map[string]interface{}
	err := json.Unmarshal([]byte(schema), &schemaMap)
	suite.NoError(err)

	// Check properties exist
	properties, ok := schemaMap["properties"].(map[string]interface{})
	suite.True(ok)
	suite.Contains(properties, "ticker")
	suite.NotContains(properties, "apiKey") // Binance doesn't need API key
}

func (suite *MarketTestSuite) TestGetDownloadClientSchema_InvalidProvider() {
	schema := GetDownloadClientSchema("invalid")

	suite.Empty(schema)
}

func (suite *MarketTestSuite) TestNewMarketDownloader() {
	helper := &mockDownloaderHelper{}
	downloader := NewMarketDownloader(helper)

	suite.NotNil(downloader)
	suite.NotNil(downloader.helper)
}

func (suite *MarketTestSuite) TestNewMarketDownloader_NilHelper() {
	downloader := NewMarketDownloader(nil)

	suite.NotNil(downloader)
	suite.Nil(downloader.helper)
}

func (suite *MarketTestSuite) TestDownloadWithConfig_InvalidProvider() {
	helper := &mockDownloaderHelper{}
	downloader := NewMarketDownloader(helper)

	err := downloader.DownloadWithConfig("invalid", `{}`)

	suite.Error(err)
	suite.Contains(err.Error(), "unsupported provider")
}

func (suite *MarketTestSuite) TestDownloadWithConfig_InvalidJSON() {
	helper := &mockDownloaderHelper{}
	downloader := NewMarketDownloader(helper)

	err := downloader.DownloadWithConfig("polygon", `{invalid json}`)

	suite.Error(err)
	suite.Contains(err.Error(), "failed to parse")
}

func (suite *MarketTestSuite) TestDownloadWithConfig_MissingRequiredFields_Polygon() {
	helper := &mockDownloaderHelper{}
	downloader := NewMarketDownloader(helper)

	// Missing apiKey
	jsonConfig := `{
		"ticker": "SPY",
		"startDate": "2024-01-01T00:00:00Z",
		"endDate": "2024-12-31T23:59:59Z",
		"interval": "1d",
		"dataPath": "/tmp/data"
	}`

	err := downloader.DownloadWithConfig("polygon", jsonConfig)

	suite.Error(err)
}

func (suite *MarketTestSuite) TestDownloadWithConfig_MissingRequiredFields_Binance() {
	helper := &mockDownloaderHelper{}
	downloader := NewMarketDownloader(helper)

	// Missing ticker
	jsonConfig := `{
		"startDate": "2024-01-01T00:00:00Z",
		"endDate": "2024-12-31T23:59:59Z",
		"interval": "1h",
		"dataPath": "/tmp/data"
	}`

	err := downloader.DownloadWithConfig("binance", jsonConfig)

	suite.Error(err)
}

func (suite *MarketTestSuite) TestCancel_NoDownloadInProgress() {
	helper := &mockDownloaderHelper{}
	downloader := NewMarketDownloader(helper)

	cancelled := downloader.Cancel()

	suite.False(cancelled)
}

func (suite *MarketTestSuite) TestCancelThreadSafety() {
	helper := &mockDownloaderHelper{}
	downloader := NewMarketDownloader(helper)

	// Test that Cancel is safe to call concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			downloader.Cancel()
		}()
	}
	wg.Wait()
	// If we get here without panic, test passes
}

func (suite *MarketTestSuite) TestStringCollectionInterface() {
	clients := GetSupportedDownloadClients()

	// Test Get with out of bounds index
	suite.Empty(clients.Get(-1))
	suite.Empty(clients.Get(100))

	// Test Size
	suite.Greater(clients.Size(), 0)
}
