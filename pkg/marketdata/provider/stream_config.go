package provider

import (
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
)

// BaseStreamConfig contains common fields for all streaming market data configurations.
type BaseStreamConfig struct {
	Symbols  []string `json:"symbols" jsonschema:"title=Symbols,description=List of symbols to stream (e.g. BTCUSDT or SPY),required" validate:"required,min=1"`
	Interval string   `json:"interval" jsonschema:"title=Interval,description=Candlestick interval for streaming data,required,enum=1s,enum=1m,enum=3m,enum=5m,enum=15m,enum=30m,enum=1h,enum=2h,enum=4h,enum=6h,enum=8h,enum=12h,enum=1d,enum=3d,enum=1w,enum=1M" validate:"required,oneof=1s 1m 3m 5m 15m 30m 1h 2h 4h 6h 8h 12h 1d 3d 1w 1M"`
}

// PolygonStreamConfig contains configuration for Polygon.io streaming market data.
type PolygonStreamConfig struct {
	BaseStreamConfig

	ApiKey string `json:"apiKey" jsonschema:"title=API Key,description=Polygon.io API key for authentication,required" keychain:"true" validate:"required"`
}

// BinanceStreamConfig contains configuration for Binance streaming market data.
type BinanceStreamConfig struct {
	BaseStreamConfig
}

// Validate validates the BaseStreamConfig fields.
func (c *BaseStreamConfig) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	return nil
}

// Validate validates the PolygonStreamConfig.
func (c *PolygonStreamConfig) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	return c.BaseStreamConfig.Validate()
}

// Validate validates the BinanceStreamConfig.
func (c *BinanceStreamConfig) Validate() error {
	return c.BaseStreamConfig.Validate()
}

// ParsePolygonStreamConfig parses JSON into a PolygonStreamConfig.
func ParsePolygonStreamConfig(jsonConfig string) (*PolygonStreamConfig, error) {
	var config PolygonStreamConfig
	if err := json.Unmarshal([]byte(jsonConfig), &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// ParseBinanceStreamConfig parses JSON into a BinanceStreamConfig.
func ParseBinanceStreamConfig(jsonConfig string) (*BinanceStreamConfig, error) {
	var config BinanceStreamConfig
	if err := json.Unmarshal([]byte(jsonConfig), &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}
