package types

type TradeStats struct {
	TotalTrades       int     `yaml:"total_trades"`
	WinningTrades     int     `yaml:"winning_trades"`
	LosingTrades      int     `yaml:"losing_trades"`
	WinRate           float64 `yaml:"win_rate"`
	AverageProfitLoss float64 `yaml:"average_profit_loss"`
	TotalPnL          float64 `yaml:"total_pnl"`
	UnrealizedPnL     float64 `yaml:"unrealized_pnl"`
	RealizedPnL       float64 `yaml:"realized_pnl"`
	SharpeRatio       float64 `yaml:"sharpe_ratio"`
	MaxDrawdown       float64 `yaml:"max_drawdown"`
	TotalFees         float64 `yaml:"total_fees"`
	// Additional statistics as needed
}
