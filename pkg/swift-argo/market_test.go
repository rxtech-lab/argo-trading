package swiftargo

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestMarketDownloader_Cancel_NoDownloadInProgress(t *testing.T) {
	helper := &mockDownloaderHelper{}
	downloader := NewMarketDownloader(helper, "polygon", "duckdb", "/tmp", "test-key")

	// Cancel with no download in progress should return false
	cancelled := downloader.Cancel()
	assert.False(t, cancelled)
}

func TestMarketDownloader_Cancel_ThreadSafety(t *testing.T) {
	helper := &mockDownloaderHelper{}
	downloader := NewMarketDownloader(helper, "polygon", "duckdb", "/tmp", "test-key")

	// Simulate concurrent cancel calls - should not panic
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			downloader.Cancel()
		}()
	}

	wg.Wait()
	// If we get here without panic, the test passes
}

func TestNewMarketDownloader(t *testing.T) {
	helper := &mockDownloaderHelper{}
	downloader := NewMarketDownloader(helper, "polygon", "duckdb", "/tmp/data", "api-key-123")

	assert.NotNil(t, downloader)
	assert.Equal(t, "polygon", downloader.provider)
	assert.Equal(t, "duckdb", downloader.writer)
	assert.Equal(t, "/tmp/data", downloader.dataFolder)
	assert.Equal(t, "api-key-123", downloader.polygonApiKey)
}

func TestMarketDownloader_Download_InvalidFromDate(t *testing.T) {
	helper := &mockDownloaderHelper{}
	downloader := NewMarketDownloader(helper, "polygon", "duckdb", "/tmp", "test-key")

	// Invalid "from" date should return an error
	err := downloader.Download("AAPL", "invalid-date", "2024-01-01T00:00:00Z", "1d")
	assert.Error(t, err)
}

func TestMarketDownloader_Download_InvalidToDate(t *testing.T) {
	helper := &mockDownloaderHelper{}
	downloader := NewMarketDownloader(helper, "polygon", "duckdb", "/tmp", "test-key")

	// Invalid "to" date should return an error
	err := downloader.Download("AAPL", "2024-01-01T00:00:00Z", "invalid-date", "1d")
	assert.Error(t, err)
}

func TestMarketDownloader_Download_InvalidProvider(t *testing.T) {
	helper := &mockDownloaderHelper{}
	downloader := NewMarketDownloader(helper, "invalid-provider", "duckdb", "/tmp", "test-key")

	// Invalid provider should return an error when trying to create client
	err := downloader.Download("AAPL", "2024-01-01T00:00:00Z", "2024-01-02T00:00:00Z", "1d")
	assert.Error(t, err)
}
