package provider

import (
	"fmt"

	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

// GetStreamConfigSchema returns the JSON schema for a provider's streaming configuration.
func GetStreamConfigSchema(providerName string) (string, error) {
	switch ProviderType(providerName) {
	case ProviderPolygon:
		//nolint:exhaustruct // Empty struct is intentional for schema generation
		return strategy.ToJSONSchema(PolygonStreamConfig{})
	case ProviderBinance:
		//nolint:exhaustruct // Empty struct is intentional for schema generation
		return strategy.ToJSONSchema(BinanceStreamConfig{})
	default:
		return "", fmt.Errorf("unsupported market data provider: %s", providerName)
	}
}

// GetStreamKeychainFields returns the keychain field names for a provider's streaming configuration.
func GetStreamKeychainFields(providerName string) ([]string, error) {
	switch ProviderType(providerName) {
	case ProviderPolygon:
		//nolint:exhaustruct // Empty struct is intentional for field introspection
		return strategy.GetKeychainFields(PolygonStreamConfig{}), nil
	case ProviderBinance:
		//nolint:exhaustruct // Empty struct is intentional for field introspection
		return strategy.GetKeychainFields(BinanceStreamConfig{}), nil
	default:
		return nil, fmt.Errorf("unsupported market data provider: %s", providerName)
	}
}

// ParseStreamConfig parses a JSON configuration string for the given streaming provider.
func ParseStreamConfig(providerName string, jsonConfig string) (any, error) {
	switch ProviderType(providerName) {
	case ProviderPolygon:
		return ParsePolygonStreamConfig(jsonConfig)
	case ProviderBinance:
		return ParseBinanceStreamConfig(jsonConfig)
	default:
		return nil, fmt.Errorf("unsupported market data provider: %s", providerName)
	}
}
