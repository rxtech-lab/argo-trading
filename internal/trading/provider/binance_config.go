package tradingprovider

import (
	"encoding/json"

	"github.com/go-playground/validator/v10"
	"github.com/rxtech-lab/argo-trading/pkg/errors"
)

// BinanceProviderConfig contains configuration for Binance trading.
type BinanceProviderConfig struct {
	ApiKey    string `json:"apiKey" jsonschema:"title=API Key,description=Binance API key" validate:"required"`
	SecretKey string `json:"secretKey" jsonschema:"title=Secret Key,description=Binance API secret key" validate:"required"`
}

// Validate validates the BinanceProviderConfig struct.
func (c *BinanceProviderConfig) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return errors.Wrap(errors.ErrCodeInvalidParameter, "invalid binance provider config", err)
	}

	return nil
}

// parseBinanceConfig parses a JSON configuration string into a BinanceProviderConfig.
func parseBinanceConfig(jsonConfig string) (*BinanceProviderConfig, error) {
	var config BinanceProviderConfig
	if err := json.Unmarshal([]byte(jsonConfig), &config); err != nil {
		return nil, errors.Wrap(errors.ErrCodeInvalidParameter, "failed to parse binance config", err)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}
