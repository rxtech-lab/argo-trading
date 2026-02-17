package marketdata

import (
	"fmt"

	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

// ProviderInfo contains metadata about a market data provider.
type ProviderInfo struct {
	Name         string `json:"name"`
	DisplayName  string `json:"displayName"`
	Description  string `json:"description"`
	RequiresAuth bool   `json:"requiresAuth"`
}

// providerRegistry holds metadata about all supported providers.
var providerRegistry = map[ProviderType]ProviderInfo{
	ProviderPolygon: {
		Name:         string(ProviderPolygon),
		DisplayName:  "Polygon.io",
		Description:  "US stock market data provider with real-time and historical OHLCV data",
		RequiresAuth: true,
	},
	ProviderBinance: {
		Name:         string(ProviderBinance),
		DisplayName:  "Binance",
		Description:  "Cryptocurrency exchange with extensive market data for crypto trading pairs",
		RequiresAuth: false,
	},
}

// GetSupportedProviders returns a list of all supported provider names.
func GetSupportedProviders() []string {
	providers := make([]string, 0, len(providerRegistry))
	for providerType := range providerRegistry {
		providers = append(providers, string(providerType))
	}

	return providers
}

// GetProviderInfo returns metadata for a specific provider.
func GetProviderInfo(providerName string) (ProviderInfo, error) {
	info, exists := providerRegistry[ProviderType(providerName)]
	if !exists {
		return ProviderInfo{}, fmt.Errorf("unsupported provider: %s", providerName)
	}

	return info, nil
}

// GetDownloadConfigSchema returns the JSON schema for a provider's download configuration.
func GetDownloadConfigSchema(providerName string) (string, error) {
	switch ProviderType(providerName) {
	case ProviderPolygon:
		//nolint:exhaustruct // Empty struct is intentional for schema generation
		return strategy.ToJSONSchema(PolygonDownloadConfig{})
	case ProviderBinance:
		//nolint:exhaustruct // Empty struct is intentional for schema generation
		return strategy.ToJSONSchema(BinanceDownloadConfig{})
	default:
		return "", fmt.Errorf("unsupported provider: %s", providerName)
	}
}

// GetDownloadKeychainFields returns the list of keychain field names for a provider's download configuration.
func GetDownloadKeychainFields(providerName string) ([]string, error) {
	switch ProviderType(providerName) {
	case ProviderPolygon:
		//nolint:exhaustruct // Empty struct is intentional for field introspection
		return strategy.GetKeychainFields(PolygonDownloadConfig{}), nil
	case ProviderBinance:
		//nolint:exhaustruct // Empty struct is intentional for field introspection
		return strategy.GetKeychainFields(BinanceDownloadConfig{}), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}
}

// ParseDownloadConfig parses a JSON configuration string for the given provider.
// Returns the parsed config as an interface{} which can be type-asserted to the specific config type.
func ParseDownloadConfig(providerName string, jsonConfig string) (interface{}, error) {
	switch ProviderType(providerName) {
	case ProviderPolygon:
		return ParsePolygonConfig(jsonConfig)
	case ProviderBinance:
		return ParseBinanceConfig(jsonConfig)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}
}
