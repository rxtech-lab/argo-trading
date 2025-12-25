package marketdata

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/polygon-io/client-go/rest/models"
	"github.com/rxtech-lab/argo-trading/pkg/marketdata/provider"
	"github.com/rxtech-lab/argo-trading/pkg/marketdata/writer"
)

// ProviderType defines the type of market data provider.
type ProviderType string

const (
	ProviderPolygon ProviderType = "polygon"
	ProviderBinance ProviderType = "binance"
)

// WriterType defines the type of market data writer.
type WriterType string

const (
	WriterDuckDB WriterType = "duckdb"
)

// ClientConfig holds the configuration for the market data client.
type ClientConfig struct {
	ProviderType  ProviderType `validate:"required,oneof=polygon binance"`
	WriterType    WriterType   `validate:"required,oneof=duckdb"`
	DataPath      string       `validate:"required"`
	PolygonApiKey string       `validate:"required_if=ProviderType polygon"`
}

// DownloadParams holds the parameters for a market data download request.
type DownloadParams struct {
	Ticker     string          `validate:"required"`
	StartDate  time.Time       `validate:"required"`
	EndDate    time.Time       `validate:"required,gtfield=StartDate"`
	Multiplier int             `validate:"required,min=1"`
	Timespan   models.Timespan `validate:"required"`
}

// Client is the market data client responsible for downloading data from providers and storing it using writers.
type Client struct {
	provider   provider.Provider
	config     ClientConfig
	validate   *validator.Validate
	onProgress provider.OnDownloadProgress
}

// NewClient creates a new market data client with the given configuration.
func NewClient(config ClientConfig, onProgress provider.OnDownloadProgress) (*Client, error) {
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("invalid client configuration: %w", err)
	}

	var marketProvider provider.Provider

	var err error

	switch config.ProviderType {
	case ProviderPolygon:
		marketProvider, err = provider.NewPolygonClient(config.PolygonApiKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create Polygon client: %w", err)
		}
	case ProviderBinance:
		marketProvider, err = provider.NewBinanceClient()
		if err != nil {
			return nil, fmt.Errorf("failed to create Binance client: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", config.ProviderType)
	}

	return &Client{
		provider:   marketProvider,
		config:     config,
		validate:   validate,
		onProgress: onProgress,
	}, nil
}

// Download initiates a market data download with the given parameters.
// The context can be used to cancel the download operation.
func (c *Client) Download(ctx context.Context, params DownloadParams) error {
	// Validate download parameters
	if err := c.validate.Struct(params); err != nil {
		return fmt.Errorf("invalid download parameters: %w", err)
	}

	// Setup writer
	marketWriter, err := c.setupWriter(params)
	if err != nil {
		return fmt.Errorf("failed to setup writer: %w", err)
	}

	defer func() {
		if closer, ok := marketWriter.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				// Just log the error but don't fail the download operation
				fmt.Printf("Warning: failed to close writer: %v\n", err)
			}
		}
	}()

	// Configure provider with writer
	c.provider.ConfigWriter(marketWriter)

	// Execute download
	_, err = c.provider.Download(
		ctx,
		params.Ticker,
		params.StartDate,
		params.EndDate,
		params.Multiplier,
		params.Timespan,
		c.onProgress,
	)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	return nil
}

// setupWriter initializes the appropriate market data writer based on configuration.
func (c *Client) setupWriter(params DownloadParams) (writer.MarketDataWriter, error) {
	switch c.config.WriterType {
	case WriterDuckDB:
		// Construct filename: TICKER_START_END_MULTIPLIER_TIMESPAN.parquet
		outputFileName := fmt.Sprintf("%s_%s_%s_%d_%s.parquet",
			params.Ticker,
			params.StartDate.Format("2006-01-02"),
			params.EndDate.Format("2006-01-02"),
			params.Multiplier,
			params.Timespan)
		outputPath := filepath.Join(c.config.DataPath, outputFileName)

		// check if datapath exist. Otherwise, create it
		if _, err := os.Stat(c.config.DataPath); os.IsNotExist(err) {
			os.MkdirAll(c.config.DataPath, 0755)
		}

		// Create and initialize the DuckDB writer
		duckdbWriter := writer.NewDuckDBWriter(outputPath)

		err := duckdbWriter.Initialize()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize DuckDB writer at %s: %w", outputPath, err)
		}

		return duckdbWriter, nil
	default:
		return nil, fmt.Errorf("unsupported writer type: %s", c.config.WriterType)
	}
}
