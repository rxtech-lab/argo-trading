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
