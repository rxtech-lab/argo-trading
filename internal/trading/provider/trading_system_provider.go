package tradingprovider

import (
	"fmt"

	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

type TradingSystemProvider interface {
	// PlaceOrder places a single order
	PlaceOrder(order types.ExecuteOrder) error
	// PlaceMultipleOrders places multiple orders
	PlaceMultipleOrders(orders []types.ExecuteOrder) error
	// GetPositions returns the current positions
	GetPositions() ([]types.Position, error)
	// GetPosition returns the current position for a symbol
	GetPosition(symbol string) (types.Position, error)
	// CancelOrder cancels an order
	CancelOrder(orderID string) error
	// CancelAllOrders cancels all orders
	CancelAllOrders() error
	// GetOrderStatus returns the status of an order
	GetOrderStatus(orderID string) (types.OrderStatus, error)
	// GetAccountInfo returns the current account state including balance, equity, and P&L
	GetAccountInfo() (types.AccountInfo, error)
	// GetOpenOrders returns all pending/open orders that have not been executed yet
	GetOpenOrders() ([]types.ExecuteOrder, error)
	// GetTrades returns executed trades with optional filtering
	GetTrades(filter types.TradeFilter) ([]types.Trade, error)
	// GetMaxBuyQuantity returns the maximum quantity that can be bought at the given price.
	// It takes into account the current balance and commission fees.
	GetMaxBuyQuantity(symbol string, price float64) (float64, error)
	// GetMaxSellQuantity returns the maximum quantity that can be sold for a symbol.
	// This is the total long position quantity for the symbol.
	GetMaxSellQuantity(symbol string) (float64, error)
}

type ProviderType string

const (
	ProviderBinancePaper ProviderType = "binance-paper"
	ProviderBinanceLive  ProviderType = "binance-live"
)

type ProviderInfo struct {
	Name           string `json:"name"`
	DisplayName    string `json:"displayName"`
	Description    string `json:"description"`
	IsPaperTrading bool   `json:"isPaperTrading"`
}

var providerRegistry = map[ProviderType]ProviderInfo{
	ProviderBinancePaper: {
		Name:           string(ProviderBinancePaper),
		DisplayName:    "Binance Testnet",
		Description:    "Binance testnet for paper trading cryptocurrency without real funds",
		IsPaperTrading: true,
	},
	ProviderBinanceLive: {
		Name:           string(ProviderBinanceLive),
		DisplayName:    "Binance Live",
		Description:    "Binance live environment for real-funds cryptocurrency trading",
		IsPaperTrading: false,
	},
}

func GetSupportedProviders() []string {
	providers := make([]string, 0, len(providerRegistry))
	for providerType := range providerRegistry {
		providers = append(providers, string(providerType))
	}

	return providers
}

// GetProviderInfo returns metadata for a specific trading provider.
func GetProviderInfo(providerName string) (ProviderInfo, error) {
	info, exists := providerRegistry[ProviderType(providerName)]
	if !exists {
		return ProviderInfo{}, fmt.Errorf("unsupported trading provider: %s", providerName)
	}

	return info, nil
}

// GetProviderConfigSchema returns the JSON schema for a provider's configuration.
func GetProviderConfigSchema(providerName string) (string, error) {
	switch ProviderType(providerName) {
	case ProviderBinancePaper, ProviderBinanceLive:
		return strategy.ToJSONSchema(BinanceProviderConfig{
			ApiKey:    "",
			SecretKey: "",
			BaseURL:   "",
		})
	default:
		return "", fmt.Errorf("unsupported trading provider: %s", providerName)
	}
}

// ParseProviderConfig parses a JSON configuration string for the given provider.
func ParseProviderConfig(providerName string, jsonConfig string) (any, error) {
	switch ProviderType(providerName) {
	case ProviderBinancePaper, ProviderBinanceLive:
		return parseBinanceConfig(jsonConfig)
	default:
		return nil, fmt.Errorf("unsupported trading provider: %s", providerName)
	}
}

// NewTradingSystemProvider creates a new trading system provider based on the provider type.
func NewTradingSystemProvider(providerType ProviderType, config any) (TradingSystemProvider, error) {
	switch providerType {
	case ProviderBinancePaper:
		cfg, ok := config.(*BinanceProviderConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for binance paper provider")
		}

		return NewBinanceTradingSystemProvider(*cfg, true) // useTestnet=true

	case ProviderBinanceLive:
		cfg, ok := config.(*BinanceProviderConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for binance live provider")
		}

		return NewBinanceTradingSystemProvider(*cfg, false) // useTestnet=false

	default:
		return nil, fmt.Errorf("unsupported trading provider: %s", providerType)
	}
}
