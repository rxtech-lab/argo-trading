package provider

import (
	"context"
	"fmt"
	"iter"
	"time"

	"github.com/polygon-io/client-go/rest/models"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/marketdata/writer"
)

// ProviderType defines the type of market data provider.
type ProviderType string

const (
	ProviderPolygon ProviderType = "polygon"
	ProviderBinance ProviderType = "binance"
)

type OnDownloadProgress = func(current float64, total float64, message string)

type Provider interface {
	// ConfigWriter configures the writer for the provider
	// Writer is used to write the market data to the database.
	// It could be a file, a database, etc.
	ConfigWriter(writer writer.MarketDataWriter)
	// Download downloads the data for the given ticker and date range.
	// The context can be used to cancel the download operation.
	// example:
	// Download(ctx, "AAPL", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2020, 1, 31, 0, 0, 0, 0, time.UTC), 1, models.TimespanMinute, onProgress)
	Download(ctx context.Context, ticker string, startDate time.Time, endDate time.Time, multiplier int, timespan models.Timespan, onProgress OnDownloadProgress) (path string, err error)
	// Stream returns an iterator that yields realtime market data via WebSocket.
	// Uses Go 1.23+ iter.Seq2 pattern for streaming data.
	// The iterator yields MarketData and error pairs. Cancel the context to stop streaming.
	Stream(ctx context.Context, symbols []string, interval string) iter.Seq2[types.MarketData, error]
}

// NewMarketDataProvider creates a new market data provider based on the provider type.
func NewMarketDataProvider(providerType ProviderType, config any) (Provider, error) {
	switch providerType {
	case ProviderBinance:
		return NewBinanceClient()
	case ProviderPolygon:
		apiKey, ok := config.(string)
		if !ok {
			return nil, fmt.Errorf("polygon provider requires API key string config")
		}

		return NewPolygonClient(apiKey)
	default:
		return nil, fmt.Errorf("unsupported market data provider: %s", providerType)
	}
}
