package swiftargo

import (
	"encoding/json"
	"fmt"

	"github.com/rxtech-lab/argo-trading/internal/trading/wallet"
	"github.com/rxtech-lab/argo-trading/internal/types"
)

// WalletBridge is the gomobile-friendly facade over the engine's wallet API.
// All methods return JSON strings because gomobile cannot return slices or
// struct values directly.
type WalletBridge struct {
	wallet wallet.Wallet
}

// Wallet returns a WalletBridge for the engine's currently configured trading
// provider. SetTradingProvider must have been called first; otherwise an error
// is returned. The bridge is safe to use both inside and outside Run().
func (t *TradingEngine) Wallet() (*WalletBridge, error) {
	w, err := t.engine.Wallet()
	if err != nil {
		return nil, err
	}

	return &WalletBridge{wallet: w}, nil
}

// GetBalanceJSON returns the JSON-encoded types.Balance.
//
//   - balanceType: "buying_power" or "balance".
//   - baseCurrency: ISO code (e.g. "USD"). Pass "" for the default (USD).
func (w *WalletBridge) GetBalanceJSON(balanceType string, baseCurrency string) (string, error) {
	balance, err := w.wallet.GetBalance(types.BalanceType(balanceType), baseCurrency)
	if err != nil {
		return "", err
	}

	out, err := json.Marshal(balance)
	if err != nil {
		return "", fmt.Errorf("failed to marshal balance: %w", err)
	}

	return string(out), nil
}

// GetAssetsJSON returns the JSON-encoded []types.Asset. When baseCurrency is
// non-empty each asset is populated with its value in that currency (nil when
// no price is available).
func (w *WalletBridge) GetAssetsJSON(baseCurrency string) (string, error) {
	assets, err := w.wallet.GetAssets(baseCurrency)
	if err != nil {
		return "", err
	}

	out, err := json.Marshal(assets)
	if err != nil {
		return "", fmt.Errorf("failed to marshal assets: %w", err)
	}

	return string(out), nil
}

// GetHistoricalOrdersJSON returns the JSON-encoded []types.Trade for orders
// matching filterJSON. filterJSON must conform to types.TradeFilter. Pass "{}"
// for an unfiltered query (subject to broker-specific requirements — e.g.
// Binance requires a symbol).
func (w *WalletBridge) GetHistoricalOrdersJSON(filterJSON string) (string, error) {
	var filter types.TradeFilter

	if filterJSON != "" {
		if err := json.Unmarshal([]byte(filterJSON), &filter); err != nil {
			return "", fmt.Errorf("failed to parse trade filter: %w", err)
		}
	}

	trades, err := w.wallet.GetHistoricalOrders(filter)
	if err != nil {
		return "", err
	}

	out, err := json.Marshal(trades)
	if err != nil {
		return "", fmt.Errorf("failed to marshal trades: %w", err)
	}

	return string(out), nil
}

// GetSupportedBaseCurrencies returns the list of base currencies accepted by
// GetBalance/GetAssets. Today this is just USD.
func GetSupportedBaseCurrencies() StringCollection {
	return &StringArray{items: []string{wallet.BaseCurrencyUSD}}
}
