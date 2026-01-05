package marketdata

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
)

// BaseDownloadConfig contains common fields for all download configurations.
type BaseDownloadConfig struct {
	Ticker    string `json:"ticker" jsonschema:"title=Ticker,description=The trading symbol to download data for (e.g. SPY or BTCUSDT),required" validate:"required"`
	StartDate string `json:"startDate" jsonschema:"title=Start Date,description=Start date,format=date,required" validate:"required"`
	EndDate   string `json:"endDate" jsonschema:"title=End Date,description=End date,format=date,required" validate:"required"`
	Interval  string `json:"interval" jsonschema:"title=Interval,description=Data interval,required,enum=1s,enum=1m,enum=3m,enum=5m,enum=15m,enum=30m,enum=1h,enum=2h,enum=4h,enum=6h,enum=8h,enum=12h,enum=1d,enum=3d,enum=1w,enum=1M" validate:"required,oneof=1s 1m 3m 5m 15m 30m 1h 2h 4h 6h 8h 12h 1d 3d 1w 1M"`
}

// PolygonDownloadConfig contains configuration for downloading from Polygon.io.
type PolygonDownloadConfig struct {
	BaseDownloadConfig

	ApiKey string `json:"apiKey" jsonschema:"title=API Key,description=Polygon.io API key for authentication,required" validate:"required"`
}

// BinanceDownloadConfig contains configuration for downloading from Binance.
// Binance public market data API does not require authentication.
type BinanceDownloadConfig struct {
	BaseDownloadConfig
}

// Validate validates the BaseDownloadConfig fields.
func (c *BaseDownloadConfig) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Validate date formats
	if _, err := time.Parse(time.RFC3339, c.StartDate); err != nil {
		return fmt.Errorf("invalid startDate format, expected RFC3339: %w", err)
	}

	if _, err := time.Parse(time.RFC3339, c.EndDate); err != nil {
		return fmt.Errorf("invalid endDate format, expected RFC3339: %w", err)
	}

	return nil
}

// Validate validates the PolygonDownloadConfig.
func (c *PolygonDownloadConfig) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	return c.BaseDownloadConfig.Validate()
}

// Validate validates the BinanceDownloadConfig.
func (c *BinanceDownloadConfig) Validate() error {
	return c.BaseDownloadConfig.Validate()
}

// ToDownloadParams converts a BaseDownloadConfig to DownloadParams.
func (c *BaseDownloadConfig) ToDownloadParams() (DownloadParams, error) {
	startDate, err := time.Parse(time.RFC3339, c.StartDate)
	if err != nil {
		return DownloadParams{}, fmt.Errorf("failed to parse startDate: %w", err)
	}

	endDate, err := time.Parse(time.RFC3339, c.EndDate)
	if err != nil {
		return DownloadParams{}, fmt.Errorf("failed to parse endDate: %w", err)
	}

	timespan := Timespan(c.Interval)

	return DownloadParams{
		Ticker:     c.Ticker,
		StartDate:  startDate,
		EndDate:    endDate,
		Multiplier: timespan.Multiplier(),
		Timespan:   timespan.Timespan(),
	}, nil
}

// ToClientConfig converts a PolygonDownloadConfig to ClientConfig.
func (c *PolygonDownloadConfig) ToClientConfig(dataPath string) ClientConfig {
	return ClientConfig{
		ProviderType:  ProviderPolygon,
		WriterType:    WriterDuckDB,
		DataPath:      dataPath,
		PolygonApiKey: c.ApiKey,
	}
}

// ToClientConfig converts a BinanceDownloadConfig to ClientConfig.
func (c *BinanceDownloadConfig) ToClientConfig(dataPath string) ClientConfig {
	return ClientConfig{
		ProviderType:  ProviderBinance,
		WriterType:    WriterDuckDB,
		DataPath:      dataPath,
		PolygonApiKey: "",
	}
}

// ParsePolygonConfig parses JSON into a PolygonDownloadConfig.
func ParsePolygonConfig(jsonConfig string) (*PolygonDownloadConfig, error) {
	var config PolygonDownloadConfig
	if err := json.Unmarshal([]byte(jsonConfig), &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// ParseBinanceConfig parses JSON into a BinanceDownloadConfig.
func ParseBinanceConfig(jsonConfig string) (*BinanceDownloadConfig, error) {
	var config BinanceDownloadConfig
	if err := json.Unmarshal([]byte(jsonConfig), &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}
