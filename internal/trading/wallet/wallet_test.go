package wallet

import (
	"context"
	"errors"
	"testing"

	tradingprovider "github.com/rxtech-lab/argo-trading/internal/trading/provider"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/require"
)

// noopProvider satisfies tradingprovider.TradingSystemProvider for the methods
// the wallet never calls in these tests.
type noopProvider struct{}

func (noopProvider) PlaceOrder(types.ExecuteOrder) error                { return nil }
func (noopProvider) PlaceMultipleOrders([]types.ExecuteOrder) error     { return nil }
func (noopProvider) GetPositions() ([]types.Position, error)            { return nil, nil }
func (noopProvider) GetPosition(string) (types.Position, error)         { return types.Position{}, nil }
func (noopProvider) CancelOrder(string) error                           { return nil }
func (noopProvider) CancelAllOrders() error                             { return nil }
func (noopProvider) GetOrderStatus(string) (types.OrderStatus, error)   { return "", nil }
func (noopProvider) GetAccountInfo() (types.AccountInfo, error)         { return types.AccountInfo{}, nil }
func (noopProvider) GetAssets() ([]types.Asset, error)                  { return nil, nil }
func (noopProvider) GetPrices([]string) (map[string]float64, error)     { return nil, nil }
func (noopProvider) GetOpenOrders() ([]types.ExecuteOrder, error)       { return nil, nil }
func (noopProvider) GetTrades(types.TradeFilter) ([]types.Trade, error) { return nil, nil }
func (noopProvider) GetMaxBuyQuantity(string, float64) (float64, error) { return 0, nil }
func (noopProvider) GetMaxSellQuantity(string) (float64, error)         { return 0, nil }
func (noopProvider) CheckConnection(context.Context) error              { return nil }
func (noopProvider) SetOnStatusChange(tradingprovider.OnStatusChange)   {}

// fakeProvider satisfies just enough of TradingSystemProvider for wallet tests
// — the wallet only calls GetAccountInfo, GetAssets, GetPrices, and GetTrades.
type fakeProvider struct {
	account    types.AccountInfo
	accountErr error
	assets     []types.Asset
	assetsErr  error
	prices     map[string]float64
	pricesErr  error
	pricesReq  []string
	trades     []types.Trade
	tradesErr  error
	lastFilter types.TradeFilter
}

func newTestWallet(t *testing.T, p *fakeProvider) Wallet {
	t.Helper()

	w, err := New(Config{
		Provider: &walletProviderAdapter{inner: p},
	})
	require.NoError(t, err)

	return w
}

func TestGetSupportedBaseCurrencies(t *testing.T) {
	w := newTestWallet(t, &fakeProvider{})

	got := w.GetSupportedBaseCurrencies()
	require.Equal(t, []string{"USD"}, got)
}

func TestGetBalance_BuyingPower(t *testing.T) {
	p := &fakeProvider{account: types.AccountInfo{BuyingPower: 1234.5}}
	w := newTestWallet(t, p)

	bal, err := w.GetBalance(types.BalanceTypeBuyingPower, "")
	require.NoError(t, err)
	require.Equal(t, types.BalanceTypeBuyingPower, bal.Type)
	require.Equal(t, "USD", bal.BaseCurrency)
	require.InDelta(t, 1234.5, bal.Value, 1e-9)
}

func TestGetBalance_BuyingPower_PropagatesError(t *testing.T) {
	p := &fakeProvider{accountErr: errors.New("boom")}
	w := newTestWallet(t, p)

	_, err := w.GetBalance(types.BalanceTypeBuyingPower, "USD")
	require.ErrorContains(t, err, "boom")
}

func TestGetBalance_TotalSumsAssetValues(t *testing.T) {
	p := &fakeProvider{
		assets: []types.Asset{
			{Symbol: "USDT", Quantity: 100},
			{Symbol: "BTC", Quantity: 0.5},
			{Symbol: "DOGE", Quantity: 200}, // no price -> skipped
		},
		prices: map[string]float64{"BTCUSDT": 50000},
	}
	w := newTestWallet(t, p)

	bal, err := w.GetBalance(types.BalanceTypeBalance, "USD")
	require.NoError(t, err)
	require.Equal(t, types.BalanceTypeBalance, bal.Type)
	require.Equal(t, "USD", bal.BaseCurrency)
	require.InDelta(t, 100+0.5*50000, bal.Value, 1e-9)
}

func TestGetBalance_UnsupportedType(t *testing.T) {
	w := newTestWallet(t, &fakeProvider{})

	_, err := w.GetBalance("nope", "")
	require.ErrorContains(t, err, "unsupported balance type")
}

func TestGetBalance_UnsupportedCurrency(t *testing.T) {
	w := newTestWallet(t, &fakeProvider{})

	_, err := w.GetBalance(types.BalanceTypeBuyingPower, "EUR")
	require.ErrorContains(t, err, "unsupported base currency")
}

func TestGetAssets_NoConversionRequested(t *testing.T) {
	p := &fakeProvider{assets: []types.Asset{{Symbol: "USDT", Quantity: 10}}}
	w := newTestWallet(t, p)

	got, err := w.GetAssets("")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Empty(t, got[0].BaseCurrency)
	require.Nil(t, got[0].BaseCurrencyValue)
	require.Empty(t, p.pricesReq, "no conversion -> no GetPrices call")
}

func TestGetAssets_USDConversion(t *testing.T) {
	p := &fakeProvider{
		assets: []types.Asset{
			{Symbol: "USDT", Quantity: 50},
			{Symbol: "BTC", Quantity: 2},
			{Symbol: "USDC", Quantity: 25},
			{Symbol: "ZZZ", Quantity: 1}, // no API price
		},
		prices: map[string]float64{
			"BTCUSDT":  30000,
			"USDCUSDT": 0.9999,
		},
	}
	w := newTestWallet(t, p)

	got, err := w.GetAssets("USD")
	require.NoError(t, err)
	require.Len(t, got, 4)

	byName := map[string]types.Asset{}
	for _, a := range got {
		byName[a.Symbol] = a
	}

	// USDT is the quote unit -> value == quantity (no API call).
	require.NotNil(t, byName["USDT"].BaseCurrencyValue)
	require.InDelta(t, 50, *byName["USDT"].BaseCurrencyValue, 1e-9)

	require.NotNil(t, byName["BTC"].BaseCurrencyValue)
	require.InDelta(t, 60000, *byName["BTC"].BaseCurrencyValue, 1e-9)

	// USDC uses the actual API price (no longer assumed = 1).
	require.NotNil(t, byName["USDC"].BaseCurrencyValue)
	require.InDelta(t, 25*0.9999, *byName["USDC"].BaseCurrencyValue, 1e-9)

	require.Nil(t, byName["ZZZ"].BaseCurrencyValue, "no broker price -> nil value")

	require.ElementsMatch(t, []string{"BTCUSDT", "USDCUSDT", "ZZZUSDT"}, p.pricesReq,
		"wallet batches all non-quote symbols into a single GetPrices call")
}

func TestGetAssets_PricesErrorPropagates(t *testing.T) {
	p := &fakeProvider{
		assets:    []types.Asset{{Symbol: "BTC", Quantity: 1}},
		pricesErr: errors.New("rate limited"),
	}
	w := newTestWallet(t, p)

	_, err := w.GetAssets("USD")
	require.ErrorContains(t, err, "rate limited")
}

func TestGetAssets_OnlyQuoteAssetSkipsAPI(t *testing.T) {
	p := &fakeProvider{
		assets: []types.Asset{{Symbol: "USDT", Quantity: 7}},
	}
	w := newTestWallet(t, p)

	got, err := w.GetAssets("USD")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.NotNil(t, got[0].BaseCurrencyValue)
	require.InDelta(t, 7, *got[0].BaseCurrencyValue, 1e-9)
	require.Empty(t, p.pricesReq, "only quote asset -> no GetPrices call")
}

func TestGetAssets_CaseInsensitiveCurrency(t *testing.T) {
	p := &fakeProvider{assets: []types.Asset{{Symbol: "USDT", Quantity: 1}}}
	w := newTestWallet(t, p)

	got, err := w.GetAssets("usd")
	require.NoError(t, err)
	require.Equal(t, "USD", got[0].BaseCurrency)
}

func TestGetHistoricalOrders_PassthroughFilter(t *testing.T) {
	wantTrades := []types.Trade{{ExecutedQty: 1.0}}
	p := &fakeProvider{trades: wantTrades}
	w := newTestWallet(t, p)

	got, err := w.GetHistoricalOrders(types.TradeFilter{Symbol: "BTCUSDT", Limit: 5})
	require.NoError(t, err)
	require.Equal(t, wantTrades, got)
}

// walletProviderAdapter wraps fakeProvider into the full
// tradingprovider.TradingSystemProvider interface by embedding noopProvider for
// methods the wallet never calls.
type walletProviderAdapter struct {
	noopProvider
	inner *fakeProvider
}

func (a *walletProviderAdapter) GetAccountInfo() (types.AccountInfo, error) {
	return a.inner.account, a.inner.accountErr
}
func (a *walletProviderAdapter) GetAssets() ([]types.Asset, error) {
	return a.inner.assets, a.inner.assetsErr
}
func (a *walletProviderAdapter) GetPrices(symbols []string) (map[string]float64, error) {
	a.inner.pricesReq = append([]string(nil), symbols...)
	return a.inner.prices, a.inner.pricesErr
}
func (a *walletProviderAdapter) GetTrades(filter types.TradeFilter) ([]types.Trade, error) {
	a.inner.lastFilter = filter
	return a.inner.trades, a.inner.tradesErr
}
