// Package wallet exposes a read-only facade over a TradingSystemProvider that
// returns live balance, asset holdings, and historical orders straight from the
// broker. Used by the iOS bridge so the wallet UI does not have to read parquet
// files written by the engine.
package wallet

import (
	"fmt"
	"strings"

	tradingprovider "github.com/rxtech-lab/argo-trading/internal/trading/provider"
	"github.com/rxtech-lab/argo-trading/internal/types"
)

// BaseCurrencyUSD is the only base currency supported today. Conversions for
// other values are rejected with an error.
const BaseCurrencyUSD = "USD"

// quoteSymbolUSD is the broker-side quote ticker that maps to USD. Binance
// quotes USD-denominated pairs against USDT, so converting an asset to USD
// means looking up the `<asset>USDT` ticker price.
const quoteSymbolUSD = "USDT"

// supportedBaseCurrencies is the static list returned by GetSupportedBaseCurrencies.
var supportedBaseCurrencies = []string{BaseCurrencyUSD}

// Wallet is the read-only API exposed to the UI.
type Wallet interface {
	// GetBalance returns either the cash buying power or the combined value of
	// all assets, expressed in baseCurrency. Pass "" to default to USD.
	GetBalance(balanceType types.BalanceType, baseCurrency string) (types.Balance, error)

	// GetAssets returns every non-zero asset holding reported by the broker. When
	// baseCurrency is non-empty, each asset is populated with its converted value
	// (nil when no price is available from the broker for that pair).
	GetAssets(baseCurrency string) ([]types.Asset, error)

	// GetHistoricalOrders returns executed trades from the broker (not from the
	// local parquet writers).
	GetHistoricalOrders(filter types.TradeFilter) ([]types.Trade, error)

	// GetSupportedBaseCurrencies lists the base currencies accepted by
	// GetBalance/GetAssets.
	GetSupportedBaseCurrencies() []string
}

// Config holds the dependencies for a Wallet.
type Config struct {
	// Provider is the trading provider whose broker is queried for live data.
	// Required.
	Provider tradingprovider.TradingSystemProvider
}

type wallet struct {
	provider tradingprovider.TradingSystemProvider
}

// New builds a Wallet. Returns an error if Provider is nil.
func New(cfg Config) (Wallet, error) {
	if cfg.Provider == nil {
		return nil, fmt.Errorf("wallet: provider is required")
	}

	return &wallet{
		provider: cfg.Provider,
	}, nil
}

// GetSupportedBaseCurrencies implements Wallet.
func (w *wallet) GetSupportedBaseCurrencies() []string {
	out := make([]string, len(supportedBaseCurrencies))
	copy(out, supportedBaseCurrencies)

	return out
}

// GetBalance implements Wallet.
func (w *wallet) GetBalance(balanceType types.BalanceType, baseCurrency string) (types.Balance, error) {
	normalized, err := normalizeBaseCurrency(baseCurrency)
	if err != nil {
		return types.Balance{}, err
	}

	switch balanceType {
	case types.BalanceTypeBuyingPower:
		info, err := w.provider.GetAccountInfo()
		if err != nil {
			return types.Balance{}, err
		}

		return types.Balance{
			Type:         types.BalanceTypeBuyingPower,
			Value:        info.BuyingPower,
			BaseCurrency: normalized,
		}, nil
	case types.BalanceTypeBalance:
		assets, err := w.GetAssets(normalized)
		if err != nil {
			return types.Balance{}, err
		}

		var total float64
		for _, asset := range assets {
			if asset.BaseCurrencyValue != nil {
				total += *asset.BaseCurrencyValue
			}
		}

		return types.Balance{
			Type:         types.BalanceTypeBalance,
			Value:        total,
			BaseCurrency: normalized,
		}, nil
	default:
		return types.Balance{}, fmt.Errorf("wallet: unsupported balance type %q", balanceType)
	}
}

// GetAssets implements Wallet.
func (w *wallet) GetAssets(baseCurrency string) ([]types.Asset, error) {
	assets, err := w.provider.GetAssets()
	if err != nil {
		return nil, err
	}

	if baseCurrency == "" {
		return assets, nil
	}

	normalized, err := normalizeBaseCurrency(baseCurrency)
	if err != nil {
		return nil, err
	}

	quote := quoteSymbolFor(normalized)

	// Build the broker-side pair list for every asset that needs pricing.
	// Assets whose symbol already equals the quote (e.g. USDT when pricing in
	// USD) skip the API call — there is no `<quote><quote>` ticker.
	pairs := make([]string, 0, len(assets))
	pairBySymbol := make(map[string]string, len(assets))

	for _, asset := range assets {
		upper := strings.ToUpper(asset.Symbol)
		if upper == quote {
			continue
		}

		pair := upper + quote
		pairBySymbol[upper] = pair
		pairs = append(pairs, pair)
	}

	prices := map[string]float64{}

	if len(pairs) > 0 {
		var pricesErr error

		prices, pricesErr = w.provider.GetPrices(pairs)
		if pricesErr != nil {
			return nil, pricesErr
		}
	}

	for i := range assets {
		assets[i].BaseCurrency = normalized
		assets[i].BaseCurrencyValue = valueFor(assets[i].Symbol, assets[i].Quantity, quote, pairBySymbol, prices)
	}

	return assets, nil
}

// GetHistoricalOrders implements Wallet.
func (w *wallet) GetHistoricalOrders(filter types.TradeFilter) ([]types.Trade, error) {
	return w.provider.GetTrades(filter)
}

// valueFor returns the asset's value in the base currency. The base currency
// is identified by its broker-side quote symbol (e.g. "USDT" for USD). Returns
// nil when the broker has no price for the constructed pair.
func valueFor(symbol string, quantity float64, quote string, pairBySymbol map[string]string, prices map[string]float64) *float64 {
	upper := strings.ToUpper(symbol)

	// The asset is already denominated in the quote currency — no API price
	// exists for `<quote><quote>`, so the value is the quantity itself.
	if upper == quote {
		v := quantity

		return &v
	}

	pair, ok := pairBySymbol[upper]
	if !ok {
		return nil
	}

	price, ok := prices[pair]
	if !ok || price <= 0 {
		return nil
	}

	v := quantity * price

	return &v
}

// quoteSymbolFor returns the broker-side ticker that represents baseCurrency.
// baseCurrency is assumed to already be normalized (e.g. "USD").
func quoteSymbolFor(baseCurrency string) string {
	if baseCurrency == BaseCurrencyUSD {
		return quoteSymbolUSD
	}

	return baseCurrency
}

// normalizeBaseCurrency uppercases the input and rejects unsupported values.
// An empty input defaults to USD.
func normalizeBaseCurrency(baseCurrency string) (string, error) {
	if baseCurrency == "" {
		return BaseCurrencyUSD, nil
	}

	upper := strings.ToUpper(baseCurrency)
	for _, supported := range supportedBaseCurrencies {
		if upper == supported {
			return upper, nil
		}
	}

	return "", fmt.Errorf("wallet: unsupported base currency %q (supported: %s)",
		baseCurrency, strings.Join(supportedBaseCurrencies, ", "))
}
