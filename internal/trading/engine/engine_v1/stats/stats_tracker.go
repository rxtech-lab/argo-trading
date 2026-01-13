package stats

import (
	"sync"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"go.uber.org/zap"
)

// StatsAccumulator holds running statistics for trades.
type StatsAccumulator struct {
	TotalTrades   int
	WinningTrades int
	LosingTrades  int
	RealizedPnL   float64
	UnrealizedPnL float64
	TotalFees     float64
	MaxProfit     float64
	MaxLoss       float64
	MaxDrawdown   float64
	PeakPnL       float64
	HoldingTimes  []int // in seconds
}

// StatsTracker tracks live trading statistics in real-time.
type StatsTracker struct {
	symbols      []string
	runID        string
	sessionStart time.Time
	currentDate  string
	strategyInfo types.StrategyInfo

	// Daily accumulators (reset on date boundary)
	dailyStats *StatsAccumulator

	// Cumulative accumulators (from session start)
	cumulativeStats *StatsAccumulator

	// File paths for parquet files
	ordersFilePath     string
	tradesFilePath     string
	marksFilePath      string
	logsFilePath       string
	marketDataFilePath string

	// Stats output path
	statsOutputPath string

	mu     sync.Mutex
	logger *logger.Logger
}

// NewStatsTracker creates a new StatsTracker instance.
func NewStatsTracker(log *logger.Logger) *StatsTracker {
	return &StatsTracker{
		symbols:            nil,
		runID:              "",
		sessionStart:       time.Time{},
		currentDate:        "",
		strategyInfo:       types.StrategyInfo{}, //nolint:exhaustruct // initialized via Initialize()
		ordersFilePath:     "",
		tradesFilePath:     "",
		marksFilePath:      "",
		logsFilePath:       "",
		marketDataFilePath: "",
		statsOutputPath:    "",
		dailyStats:         newStatsAccumulator(),
		cumulativeStats:    newStatsAccumulator(),
		mu:                 sync.Mutex{},
		logger:             log,
	}
}

// newStatsAccumulator creates a new initialized StatsAccumulator.
func newStatsAccumulator() *StatsAccumulator {
	return &StatsAccumulator{
		TotalTrades:   0,
		WinningTrades: 0,
		LosingTrades:  0,
		RealizedPnL:   0,
		UnrealizedPnL: 0,
		TotalFees:     0,
		MaxProfit:     0,
		MaxLoss:       0,
		MaxDrawdown:   0,
		PeakPnL:       0,
		HoldingTimes:  make([]int, 0),
	}
}

// Initialize sets up the stats tracker with session information.
func (s *StatsTracker) Initialize(
	symbols []string,
	runID string,
	sessionStart time.Time,
	strategyInfo types.StrategyInfo,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.symbols = symbols
	s.runID = runID
	s.sessionStart = sessionStart
	s.currentDate = sessionStart.Format("2006-01-02")
	s.strategyInfo = strategyInfo

	s.logger.Info("Stats tracker initialized",
		zap.String("run_id", runID),
		zap.Strings("symbols", symbols),
	)
}

// SetFilePaths sets the paths for parquet files.
func (s *StatsTracker) SetFilePaths(ordersPath, tradesPath, marksPath, logsPath, marketDataPath, statsPath string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ordersFilePath = ordersPath
	s.tradesFilePath = tradesPath
	s.marksFilePath = marksPath
	s.logsFilePath = logsPath
	s.marketDataFilePath = marketDataPath
	s.statsOutputPath = statsPath
}

// RecordTrade records a trade and updates statistics.
func (s *StatsTracker) RecordTrade(trade types.Trade) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update both daily and cumulative stats
	s.updateAccumulator(s.dailyStats, trade)
	s.updateAccumulator(s.cumulativeStats, trade)

	s.logger.Debug("Trade recorded",
		zap.String("order_id", trade.Order.OrderID),
		zap.Float64("pnl", trade.PnL),
		zap.Int("total_trades", s.cumulativeStats.TotalTrades),
	)
}

// updateAccumulator updates a stats accumulator with a new trade.
//
//nolint:funcorder // helper method used by RecordTrade
func (s *StatsTracker) updateAccumulator(acc *StatsAccumulator, trade types.Trade) {
	acc.TotalTrades++
	acc.TotalFees += trade.Fee
	acc.RealizedPnL += trade.PnL

	// Track winning/losing trades
	if trade.PnL > 0 {
		acc.WinningTrades++
	} else if trade.PnL < 0 {
		acc.LosingTrades++
	}

	// Track max profit/loss
	if trade.PnL > acc.MaxProfit {
		acc.MaxProfit = trade.PnL
	}

	if trade.PnL < acc.MaxLoss {
		acc.MaxLoss = trade.PnL
	}

	// Track max drawdown
	if acc.RealizedPnL > acc.PeakPnL {
		acc.PeakPnL = acc.RealizedPnL
	}

	drawdown := acc.PeakPnL - acc.RealizedPnL
	if drawdown > acc.MaxDrawdown {
		acc.MaxDrawdown = drawdown
	}

	// Calculate holding time if we have order timestamp and executed time
	if !trade.Order.Timestamp.IsZero() && !trade.ExecutedAt.IsZero() {
		holdingTime := int(trade.ExecutedAt.Sub(trade.Order.Timestamp).Seconds())
		if holdingTime > 0 {
			acc.HoldingTimes = append(acc.HoldingTimes, holdingTime)
		}
	}
}

// SetUnrealizedPnL updates the unrealized PnL for current positions.
func (s *StatsTracker) SetUnrealizedPnL(unrealizedPnL float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.dailyStats.UnrealizedPnL = unrealizedPnL
	s.cumulativeStats.UnrealizedPnL = unrealizedPnL
}

// HandleDateBoundary handles the transition to a new date.
// Resets daily stats while keeping cumulative stats.
func (s *StatsTracker) HandleDateBoundary(newDate string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldDate := s.currentDate
	s.currentDate = newDate

	// Reset daily stats
	s.dailyStats = newStatsAccumulator()

	s.logger.Info("Date boundary handled, daily stats reset",
		zap.String("old_date", oldDate),
		zap.String("new_date", newDate),
	)
}

// GetDailyStats returns the current daily statistics.
func (s *StatsTracker) GetDailyStats() types.LiveTradeStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.buildLiveTradeStats(s.dailyStats, s.currentDate)
}

// GetCumulativeStats returns the cumulative statistics from session start.
func (s *StatsTracker) GetCumulativeStats() types.LiveTradeStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.buildLiveTradeStats(s.cumulativeStats, s.sessionStart.Format("2006-01-02"))
}

// buildLiveTradeStats builds a LiveTradeStats from an accumulator.
//
//nolint:funcorder // helper method used by GetDailyStats, GetCumulativeStats, WriteStatsYAML
func (s *StatsTracker) buildLiveTradeStats(acc *StatsAccumulator, date string) types.LiveTradeStats {
	// Calculate win rate
	winRate := 0.0
	if acc.TotalTrades > 0 {
		winRate = float64(acc.WinningTrades) / float64(acc.TotalTrades)
	}

	// Calculate holding time stats
	holdingTime := types.TradeHoldingTime{
		Min: 0,
		Max: 0,
		Avg: 0,
	}

	if len(acc.HoldingTimes) > 0 {
		minTime := acc.HoldingTimes[0]
		maxTime := acc.HoldingTimes[0]
		totalTime := 0

		for _, t := range acc.HoldingTimes {
			totalTime += t
			if t < minTime {
				minTime = t
			}

			if t > maxTime {
				maxTime = t
			}
		}

		holdingTime.Min = minTime
		holdingTime.Max = maxTime
		holdingTime.Avg = totalTime / len(acc.HoldingTimes)
	}

	return types.LiveTradeStats{
		ID:           s.runID,
		Date:         date,
		SessionStart: s.sessionStart,
		LastUpdated:  time.Now(),
		Symbols:      s.symbols,
		TradeResult: types.TradeResult{
			NumberOfTrades:        acc.TotalTrades,
			NumberOfWinningTrades: acc.WinningTrades,
			NumberOfLosingTrades:  acc.LosingTrades,
			WinRate:               winRate,
			MaxDrawdown:           acc.MaxDrawdown,
		},
		TradePnl: types.TradePnl{
			RealizedPnL:   acc.RealizedPnL,
			UnrealizedPnL: acc.UnrealizedPnL,
			TotalPnL:      acc.RealizedPnL + acc.UnrealizedPnL,
			MaximumLoss:   acc.MaxLoss,
			MaximumProfit: acc.MaxProfit,
		},
		TradeHoldingTime:   holdingTime,
		TotalFees:          acc.TotalFees,
		OrdersFilePath:     s.ordersFilePath,
		TradesFilePath:     s.tradesFilePath,
		MarksFilePath:      s.marksFilePath,
		LogsFilePath:       s.logsFilePath,
		MarketDataFilePath: s.marketDataFilePath,
		Strategy:           s.strategyInfo,
	}
}

// WriteStatsYAML writes the current cumulative stats to the stats.yaml file.
func (s *StatsTracker) WriteStatsYAML() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.statsOutputPath == "" {
		return nil // No output path configured
	}

	stats := s.buildLiveTradeStats(s.cumulativeStats, s.currentDate)

	return types.WriteLiveTradeStats(s.statsOutputPath, stats)
}

// GetStatsOutputPath returns the stats output path.
func (s *StatsTracker) GetStatsOutputPath() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.statsOutputPath
}

// GetCurrentDate returns the current date.
func (s *StatsTracker) GetCurrentDate() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.currentDate
}

// GetRunID returns the run ID.
func (s *StatsTracker) GetRunID() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.runID
}
