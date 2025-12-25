package swiftargo

import (
	"context"
	"sync"
	"time"

	"github.com/rxtech-lab/argo-trading/pkg/marketdata"
)

type MarketDownloader struct {
	helper        MarketDownloaderHelper
	provider      string
	writer        string
	dataFolder    string
	polygonApiKey string

	// Cancellation support
	mu         sync.Mutex
	cancelFunc context.CancelFunc
}

type MarketDownloaderHelper interface {
	OnDownloadProgress(current, total float64, message string)
}

func NewMarketDownloader(helper MarketDownloaderHelper, provider string, writer string, dataFolder string, polygonApiKey string) *MarketDownloader {
	return &MarketDownloader{
		helper:        helper,
		provider:      provider,
		writer:        writer,
		dataFolder:    dataFolder,
		polygonApiKey: polygonApiKey,
		mu:            sync.Mutex{},
		cancelFunc:    nil,
	}
}

// Download downloads market data. This method is blocking.
// Can be cancelled by calling Cancel() from another goroutine.
func (m *MarketDownloader) Download(ticker string, from, to string, interval string) error {
	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Store cancel function with mutex protection
	m.mu.Lock()
	m.cancelFunc = cancel
	m.mu.Unlock()

	// Ensure we clean up the cancel function when done
	defer func() {
		m.mu.Lock()
		m.cancelFunc = nil
		m.mu.Unlock()
	}()

	client, err := marketdata.NewClient(marketdata.ClientConfig{
		ProviderType:  marketdata.ProviderType(m.provider),
		WriterType:    marketdata.WriterType(m.writer),
		DataPath:      m.dataFolder,
		PolygonApiKey: m.polygonApiKey,
	}, func(current, total float64, message string) {
		m.helper.OnDownloadProgress(current, total, message)
	})
	if err != nil {
		return err
	}

	fromTime, err := time.Parse(time.RFC3339, from)
	if err != nil {
		return err
	}

	toTime, err := time.Parse(time.RFC3339, to)
	if err != nil {
		return err
	}

	timespan := marketdata.Timespan(interval)

	err = client.Download(ctx, marketdata.DownloadParams{
		Ticker:     ticker,
		StartDate:  fromTime,
		EndDate:    toTime,
		Timespan:   timespan.Timespan(),
		Multiplier: timespan.Multiplier(),
	})
	if err != nil {
		return err
	}

	return nil
}

// Cancel cancels any in-progress download.
// This method is safe to call from any goroutine (e.g., Swift's main thread).
// Returns true if a download was cancelled, false if no download was in progress.
func (m *MarketDownloader) Cancel() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancelFunc != nil {
		m.cancelFunc()
		m.cancelFunc = nil

		return true
	}

	return false
}
