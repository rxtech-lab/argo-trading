package types

type TradeStats struct {
	TotalTrades       int
	WinningTrades     int
	LosingTrades      int
	WinRate           float64
	AverageProfitLoss float64
	TotalPnL          float64
	SharpeRatio       float64
	MaxDrawdown       float64
	// Additional statistics as needed
}
