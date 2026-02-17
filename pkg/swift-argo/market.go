package swiftargo

import (
	"context"
	"fmt"
	"sync"

	"github.com/rxtech-lab/argo-trading/pkg/marketdata"
)

// MarketDownloader handles market data downloads for Swift consumers.
type MarketDownloader struct {
	helper MarketDownloaderHelper

	// Cancellation support
	mu         sync.Mutex
	cancelFunc context.CancelFunc
}

// MarketDownloaderHelper is the callback interface for download progress.
type MarketDownloaderHelper interface {
	OnDownloadProgress(current, total float64, message string)
}

// GetSupportedDownloadClients returns a StringCollection of all supported download client names.
// This follows the gomobile pattern where slices cannot be returned directly.
func GetSupportedDownloadClients() StringCollection {
	providers := marketdata.GetSupportedProviders()

	return &StringArray{items: providers}
}

// GetDownloadClientSchema returns the JSON schema for a specific download client.
// The providerName should be one of the values returned by GetSupportedDownloadClients().
// Returns empty string if the provider is not found.
func GetDownloadClientSchema(providerName string) string {
	schema, err := marketdata.GetDownloadConfigSchema(providerName)
	if err != nil {
		return ""
	}

	return schema
}

// GetDownloadClientKeychainFields returns the keychain field names for a download client's configuration.
// The providerName should be one of the values returned by GetSupportedDownloadClients().
// Returns nil if the provider is not found.
func GetDownloadClientKeychainFields(providerName string) StringCollection {
	fields, err := marketdata.GetDownloadKeychainFields(providerName)
	if err != nil || len(fields) == 0 {
		return nil
	}

	return &StringArray{items: fields}
}

// NewMarketDownloader creates a new MarketDownloader with the given helper for progress callbacks.
func NewMarketDownloader(helper MarketDownloaderHelper) *MarketDownloader {
	return &MarketDownloader{
		helper:     helper,
		mu:         sync.Mutex{},
		cancelFunc: nil,
	}
}

// DownloadWithConfig downloads market data using a JSON configuration.
// The configJSON must conform to the schema returned by GetDownloadClientSchema() for the given provider.
// The dataFolder parameter specifies the directory path where downloaded data will be saved.
// This method is blocking. Can be cancelled by calling Cancel() from another goroutine.
func (m *MarketDownloader) DownloadWithConfig(providerName string, configJSON string, dataFolder string) error {
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

	// Create progress callback
	onProgress := func(current, total float64, message string) {
		if m.helper != nil {
			m.helper.OnDownloadProgress(current, total, message)
		}
	}

	// Parse config and create client based on provider type
	switch marketdata.ProviderType(providerName) {
	case marketdata.ProviderPolygon:
		config, err := marketdata.ParsePolygonConfig(configJSON)
		if err != nil {
			return fmt.Errorf("failed to parse polygon config: %w", err)
		}

		client, params, err := marketdata.NewClientFromPolygonConfig(config, dataFolder, onProgress)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		return client.Download(ctx, params)

	case marketdata.ProviderBinance:
		config, err := marketdata.ParseBinanceConfig(configJSON)
		if err != nil {
			return fmt.Errorf("failed to parse binance config: %w", err)
		}

		client, params, err := marketdata.NewClientFromBinanceConfig(config, dataFolder, onProgress)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		return client.Download(ctx, params)

	default:
		return fmt.Errorf("unsupported provider: %s", providerName)
	}
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
