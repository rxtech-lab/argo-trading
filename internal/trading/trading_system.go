package trading

import (
	tradingprovider "github.com/rxtech-lab/argo-trading/internal/trading/provider"
	marketdataProvider "github.com/rxtech-lab/argo-trading/pkg/marketdata/provider"
)

type TradingSystem struct {
	tradingProvider tradingprovider.TradingSystemProvider
	dataProvider    marketdataProvider.Provider
}

// NewTradingSystem creates a new TradingSystem with the given trading and data providers.
func NewTradingSystem(provider tradingprovider.TradingSystemProvider, dataProvider marketdataProvider.Provider) *TradingSystem {
	return &TradingSystem{
		tradingProvider: provider,
		dataProvider:    dataProvider,
	}
}
