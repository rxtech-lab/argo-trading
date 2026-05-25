package types

// Asset represents a single asset holding reported by the broker.
type Asset struct {
	// Symbol is the asset ticker as reported by the broker (e.g. "BTC", "USDT").
	Symbol string `json:"symbol" yaml:"symbol"`
	// Quantity is the total balance for the asset (free + locked).
	Quantity float64 `json:"quantity" yaml:"quantity"`
	// BaseCurrency is the currency the asset is valued in. Empty when no
	// conversion was requested.
	BaseCurrency string `json:"base_currency,omitempty" yaml:"base_currency,omitempty"`
	// BaseCurrencyValue is the asset's value expressed in BaseCurrency. Nil
	// when conversion was not requested or no price was available.
	BaseCurrencyValue *float64 `json:"base_currency_value,omitempty" yaml:"base_currency_value,omitempty"`
}

// BalanceType selects which balance figure to return from Wallet.GetBalance.
type BalanceType string

const (
	// BalanceTypeBuyingPower returns the cash currently available for new orders.
	BalanceTypeBuyingPower BalanceType = "buying_power"
	// BalanceTypeBalance returns the combined value of all assets (cash + holdings)
	// expressed in the requested base currency.
	BalanceTypeBalance BalanceType = "balance"
)

// Balance is the response payload for Wallet.GetBalance.
type Balance struct {
	Type         BalanceType `json:"type" yaml:"type"`
	Value        float64     `json:"value" yaml:"value"`
	BaseCurrency string      `json:"base_currency" yaml:"base_currency"`
}
