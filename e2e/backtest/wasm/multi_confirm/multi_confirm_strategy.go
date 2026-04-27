//go:build wasip1

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/knqyf263/go-plugin/types/known/emptypb"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

type MultiConfirmStrategy struct {
	config Config
}

type Config struct {
	SlowEMAPeriod         int     `json:"slowEmaPeriod"`
	RSIBuyMin             float64 `json:"rsiBuyMin"`
	RSIBuyMax             float64 `json:"rsiBuyMax"`
	RSIOverboughtExit     float64 `json:"rsiOverboughtExit"`
	RequireMACDPos        bool    `json:"requireMacdPos"`
	RequireMACDRising     bool    `json:"requireMacdRising"`
	RequireEMARising      bool    `json:"requireEmaRising"`
	RequireBullishBar     bool    `json:"requireBullishBar"`
	VolumeAvgBars         int     `json:"volumeAvgBars"`
	VolumeMinRatio        float64 `json:"volumeMinRatio"`
	ATRStopMult           float64 `json:"atrStopMult"`
	ATRTakeMult           float64 `json:"atrTakeMult"`
	ATRMinPct             float64 `json:"atrMinPct"`
	BreakevenAtR          float64 `json:"breakevenAtR"`
	TrailAtR              float64 `json:"trailAtR"`
	TrailATRMult          float64 `json:"trailAtrMult"`
	RSIExitOnlyProfit     bool    `json:"rsiExitOnlyProfit"`
	ScratchAfterBars      int     `json:"scratchAfterBars"`
	ScratchPnLPct         float64 `json:"scratchPnlPct"`
	HardStopLossPct       float64 `json:"hardStopLossPct"`
	MinHoldBars           int     `json:"minHoldBars"`
	TrendBreakBufferPct   float64 `json:"trendBreakBufferPct"`
	TrendBreakConfirmBars int     `json:"trendBreakConfirmBars"`
	CooldownBars          int     `json:"cooldownBars"`
	LossCooldownMult      float64 `json:"lossCooldownMult"`
	TimeExitBars          int     `json:"timeExitBars"`
	BuyingPowerPct        float64 `json:"buyingPowerPct"`
	FixedNotional         float64 `json:"fixedNotional"`
}

var (
	inPosition   bool
	posQty       float64
	entryPrice   float64
	entryATR     float64
	tickCount    int
	lastTradeBar int
	barsAtEntry  int
	currentStop  float64
	peakHigh     float64

	prevRSI  float64
	prevMACD float64

	slowEMA      float64
	prevSlowEMA  float64
	slowEMAReady bool
	emaAlpha     float64

	warmupBar int

	volEMA      float64
	volEMAReady bool
	volWarmup   int
	volAlpha    float64

	currentCooldown  int
	lastTradeWasLoss bool

	consecBelowEMA int

	hadData bool
)

func main() {}

func init() {
	strategy.RegisterTradingStrategy(&MultiConfirmStrategy{})
}

func (s *MultiConfirmStrategy) GetConfigSchema(ctx context.Context, req *strategy.GetConfigSchemaRequest) (*strategy.GetConfigSchemaResponse, error) {
	schema, err := strategy.ToJSONSchema(Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to generate schema: %w", err)
	}
	return &strategy.GetConfigSchemaResponse{Schema: schema}, nil
}

func (s *MultiConfirmStrategy) GetDescription(ctx context.Context, req *strategy.GetDescriptionRequest) (*strategy.GetDescriptionResponse, error) {
	return &strategy.GetDescriptionResponse{
		Description: "EMA + MACD + RSI + Volume entry, ATR stops, scratch losers early, adaptive cooldown.",
	}, nil
}

func (s *MultiConfirmStrategy) GetIdentifier(ctx context.Context, req *strategy.GetIdentifierRequest) (*strategy.GetIdentifierResponse, error) {
	return &strategy.GetIdentifierResponse{
		Identifier: "com.argo-trading.multi-confirm",
	}, nil
}

func (s *MultiConfirmStrategy) Initialize(ctx context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
	cfg := Config{
		SlowEMAPeriod:         200,
		RSIBuyMin:             45,
		RSIBuyMax:             60,
		RSIOverboughtExit:     72,
		RequireMACDPos:        true,
		RequireMACDRising:     true,
		RequireEMARising:      true,
		RequireBullishBar:     true,
		VolumeAvgBars:         30,
		VolumeMinRatio:        1.0,
		ATRStopMult:           1.0,
		ATRTakeMult:           4.0,
		ATRMinPct:             0.15,
		BreakevenAtR:          1.0,
		TrailAtR:              2.0,
		TrailATRMult:          2.0,
		RSIExitOnlyProfit:     true,
		ScratchAfterBars:      60,
		ScratchPnLPct:         -0.2,
		HardStopLossPct:       -0.7,
		MinHoldBars:           180,
		TrendBreakBufferPct:   0.0015,
		TrendBreakConfirmBars: 3,
		CooldownBars:          120,
		LossCooldownMult:      2.0,
		TimeExitBars:          240,
		BuyingPowerPct:        15,
		FixedNotional:         15000,
	}
	if req.Config != "" {
		if err := json.Unmarshal([]byte(req.Config), &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse configuration: %w", err)
		}
	}
	if cfg.BuyingPowerPct <= 0 || cfg.BuyingPowerPct > 100 {
		return nil, fmt.Errorf("buyingPowerPct must be in (0, 100], got %v", cfg.BuyingPowerPct)
	}
	if cfg.RSIBuyMin >= cfg.RSIBuyMax {
		return nil, fmt.Errorf("rsiBuyMin (%v) must be less than rsiBuyMax (%v)", cfg.RSIBuyMin, cfg.RSIBuyMax)
	}
	if cfg.RSIBuyMax >= cfg.RSIOverboughtExit {
		return nil, fmt.Errorf("rsiBuyMax (%v) must be less than rsiOverboughtExit (%v)", cfg.RSIBuyMax, cfg.RSIOverboughtExit)
	}
	if cfg.SlowEMAPeriod < 2 {
		return nil, fmt.Errorf("slowEmaPeriod must be >= 2")
	}
	if cfg.VolumeAvgBars < 2 {
		return nil, fmt.Errorf("volumeAvgBars must be >= 2")
	}
	if cfg.TrendBreakBufferPct < 0 {
		return nil, fmt.Errorf("trendBreakBufferPct must be >= 0, got %v", cfg.TrendBreakBufferPct)
	}
	if cfg.TrendBreakConfirmBars < 1 {
		return nil, fmt.Errorf("trendBreakConfirmBars must be >= 1, got %v", cfg.TrendBreakConfirmBars)
	}
	s.config = cfg
	emaAlpha = 2.0 / float64(cfg.SlowEMAPeriod+1)

	inPosition = false
	posQty = 0
	entryPrice = 0
	entryATR = 0
	tickCount = 0
	lastTradeBar = 0
	barsAtEntry = 0
	currentStop = 0
	peakHigh = 0
	prevRSI = 0
	prevMACD = 0
	slowEMA = 0
	prevSlowEMA = 0
	slowEMAReady = false
	warmupBar = 0
	volEMA = 0
	volEMAReady = false
	volWarmup = 0
	volAlpha = 2.0 / float64(cfg.VolumeAvgBars+1)
	currentCooldown = cfg.CooldownBars
	lastTradeWasLoss = false
	consecBelowEMA = 0
	hadData = false

	return &emptypb.Empty{}, nil
}

func (s *MultiConfirmStrategy) Name(ctx context.Context, req *strategy.NameRequest) (*strategy.NameResponse, error) {
	return &strategy.NameResponse{Name: "MultiConfirmStrategy"}, nil
}

func (s *MultiConfirmStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
	data := req.Data
	api := strategy.NewStrategyApi()

	prevSlowEMA = slowEMA
	if !slowEMAReady {
		warmupBar++
		slowEMA = (slowEMA*float64(warmupBar-1) + data.Close) / float64(warmupBar)
		if warmupBar >= s.config.SlowEMAPeriod {
			slowEMAReady = true
		}
	} else {
		slowEMA = emaAlpha*data.Close + (1-emaAlpha)*slowEMA
	}

	if slowEMAReady {
		if data.Close < slowEMA {
			consecBelowEMA++
		} else {
			consecBelowEMA = 0
		}
	}

	if !volEMAReady {
		volWarmup++
		if volWarmup == 1 {
			volEMA = data.Volume
		} else {
			volEMA = (volEMA*float64(volWarmup-1) + data.Volume) / float64(volWarmup)
		}
		if volWarmup >= s.config.VolumeAvgBars {
			volEMAReady = true
		}
	} else {
		volEMA = volAlpha*data.Volume + (1-volAlpha)*volEMA
	}

	rsi := getSignal(ctx, api, data, strategy.IndicatorType_INDICATOR_RSI, "rsi")
	if rsi == 0 {
		return &emptypb.Empty{}, nil
	}
	macd := getSignal(ctx, api, data, strategy.IndicatorType_INDICATOR_MACD, "macd")
	atr := getSignal(ctx, api, data, strategy.IndicatorType_INDICATOR_ATR, "atr")

	tickCount++

	if !slowEMAReady || atr == 0 {
		prevRSI = rsi
		prevMACD = macd
		hadData = true
		return &emptypb.Empty{}, nil
	}
	if !hadData {
		prevRSI = rsi
		prevMACD = macd
		hadData = true
		return &emptypb.Empty{}, nil
	}

	// In backtest, every order goes through the engine on the same goroutine
	// that's calling us — local state can't drift from broker state, so we
	// skip the per-bar GetPosition reconcile that dominated profiling
	// (~50% of CPU was an SQL aggregation over the full trades table).
	// For live trading where the broker can fill out-of-band, restore the
	// reconcile but throttle it (e.g. every 200 bars) instead of every bar.

	barsSinceTrade := tickCount - lastTradeBar
	barsHeld := tickCount - barsAtEntry

	if !inPosition && barsSinceTrade > currentCooldown {
		uptrend := data.Close > slowEMA
		emaRising := !s.config.RequireEMARising || slowEMA > prevSlowEMA
		macdPos := !s.config.RequireMACDPos || macd > 0
		macdRising := !s.config.RequireMACDRising || macd > prevMACD
		rsiCrossUp := prevRSI < s.config.RSIBuyMin && rsi >= s.config.RSIBuyMin
		macdCrossUp := prevMACD < 0 && macd >= 0
		entryTrigger := rsiCrossUp || macdCrossUp
		rsiNotOverextended := rsi < s.config.RSIBuyMax
		atrPct := (atr / data.Close) * 100
		hasVolatility := atrPct >= s.config.ATRMinPct
		bullishBar := !s.config.RequireBullishBar || data.Close > data.Open

		volPass := true
		if s.config.VolumeMinRatio > 0 {
			if !volEMAReady || volEMA <= 0 {
				volPass = false
			} else {
				volPass = data.Volume >= volEMA*s.config.VolumeMinRatio
			}
		}

		if uptrend && emaRising && macdPos && macdRising && entryTrigger && rsiNotOverextended && hasVolatility && bullishBar && volPass {
			acct, err := api.GetAccountInfo(ctx, &emptypb.Empty{})
			if err != nil {
				return nil, err
			}
			var notional float64
			if s.config.FixedNotional > 0 {
				notional = s.config.FixedNotional
				if notional > acct.BuyingPower {
					notional = acct.BuyingPower
				}
			} else {
				notional = acct.BuyingPower * (s.config.BuyingPowerPct / 100.0)
			}
			if notional <= 0 || data.Close <= 0 {
				prevRSI = rsi
				prevMACD = macd
				return &emptypb.Empty{}, nil
			}
			qty := notional / data.Close
			if qty <= 0 {
				prevRSI = rsi
				prevMACD = macd
				return &emptypb.Empty{}, nil
			}

			trig := "rsi_x"
			if !rsiCrossUp && macdCrossUp {
				trig = "macd_x"
			}
			buyMsg := fmt.Sprintf("BUY trig=%s rsi=%.1f macd=%.3f atrPct=%.3f ema=%.0f close=%.0f",
				trig, rsi, macd, atrPct, slowEMA, data.Close)

			_, err = api.PlaceOrder(ctx, &strategy.ExecuteOrder{
				Symbol:    data.Symbol,
				Side:      strategy.PurchaseType_PURCHASE_TYPE_BUY,
				OrderType: strategy.OrderType_ORDER_TYPE_MARKET,
				Quantity:  qty,
				Price:     data.Close,
				Reason: &strategy.Reason{
					Reason:  "strategy",
					Message: buyMsg,
				},
				StrategyName: "MultiConfirmStrategy",
			})
			if err != nil {
				return nil, err
			}
			inPosition = true
			posQty = qty
			entryPrice = data.Close
			entryATR = atr
			lastTradeBar = tickCount
			barsAtEntry = tickCount
			currentStop = data.Close - atr*s.config.ATRStopMult
			peakHigh = data.High

			prevRSI = rsi
			prevMACD = macd
			return &emptypb.Empty{}, nil
		}
	}

	if inPosition {
		if data.High > peakHigh {
			peakHigh = data.High
		}

		initialRisk := entryATR * s.config.ATRStopMult
		unrealizedGain := data.Close - entryPrice

		if s.config.BreakevenAtR > 0 && unrealizedGain >= initialRisk*s.config.BreakevenAtR {
			if currentStop < entryPrice {
				currentStop = entryPrice
			}
		}
		if s.config.TrailAtR > 0 && unrealizedGain >= initialRisk*s.config.TrailAtR {
			trailStop := peakHigh - entryATR*s.config.TrailATRMult
			if trailStop > currentStop {
				currentStop = trailStop
			}
		}

		takePrice := entryPrice + entryATR*s.config.ATRTakeMult
		pnlPct := (data.Close - entryPrice) / entryPrice * 100

		stopLoss := data.Low <= currentStop
		takeProfit := data.High >= takePrice
		rsiExit := rsi >= s.config.RSIOverboughtExit && barsHeld >= 3 &&
			(!s.config.RSIExitOnlyProfit || data.Close > entryPrice*1.001)
		belowWithBuffer := data.Close < slowEMA*(1-s.config.TrendBreakBufferPct)
		trendExit := barsHeld >= s.config.MinHoldBars && belowWithBuffer && consecBelowEMA >= s.config.TrendBreakConfirmBars
		timeExit := barsHeld >= s.config.TimeExitBars
		scratchExit := s.config.ScratchAfterBars > 0 &&
			barsHeld >= s.config.ScratchAfterBars &&
			pnlPct < s.config.ScratchPnLPct
		hardStop := s.config.HardStopLossPct < 0 && pnlPct <= s.config.HardStopLossPct

		if stopLoss || takeProfit || rsiExit || trendExit || timeExit || scratchExit || hardStop {
			exitPrice := data.Close
			reason := "time"
			if stopLoss {
				exitPrice = currentStop
				if exitPrice > data.Close {
					exitPrice = data.Close
				}
				if currentStop >= entryPrice {
					reason = "TRAIL_STOP"
				} else {
					reason = "ATR_STOP"
				}
			} else if takeProfit {
				exitPrice = takePrice
				if exitPrice < data.Close {
					exitPrice = data.Close
				}
				reason = "ATR_TAKE"
			} else if rsiExit {
				reason = "RSI_overbought"
			} else if trendExit {
				reason = "trend_break"
			} else if scratchExit {
				reason = "SCRATCH"
			} else if hardStop {
				reason = "HARD_STOP"
			}
			msg := fmt.Sprintf("SELL %s bars=%d pnl%%=%.2f rsi=%.1f entry=%.0f exit=%.0f stop=%.0f peak=%.0f",
				reason, barsHeld, pnlPct, rsi, entryPrice, exitPrice, currentStop, peakHigh)

			_, err := api.PlaceOrder(ctx, &strategy.ExecuteOrder{
				Symbol:    data.Symbol,
				Side:      strategy.PurchaseType_PURCHASE_TYPE_SELL,
				OrderType: strategy.OrderType_ORDER_TYPE_MARKET,
				Quantity:  posQty,
				Price:     exitPrice,
				Reason: &strategy.Reason{
					Reason:  "strategy",
					Message: msg,
				},
				StrategyName: "MultiConfirmStrategy",
			})
			if err != nil {
				return nil, err
			}
			realizedPnL := exitPrice - entryPrice
			lastTradeWasLoss = realizedPnL < 0
			if lastTradeWasLoss && s.config.LossCooldownMult > 1 {
				currentCooldown = int(float64(s.config.CooldownBars) * s.config.LossCooldownMult)
			} else {
				currentCooldown = s.config.CooldownBars
			}
			inPosition = false
			posQty = 0
			lastTradeBar = tickCount
		}
	}

	prevRSI = rsi
	prevMACD = macd

	return &emptypb.Empty{}, nil
}

func getSignal(ctx context.Context, api strategy.StrategyApi, data *strategy.MarketData, t strategy.IndicatorType, key string) float64 {
	resp, err := api.GetSignal(ctx, &strategy.GetSignalRequest{
		IndicatorType: t,
		MarketData:    data,
	})
	if err != nil || resp == nil {
		return 0
	}
	return parseValue(resp.RawValue, key)
}

func parseValue(rawValue string, key string) float64 {
	if rawValue == "" {
		return 0
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(rawValue), &v); err != nil {
		if f, err := strconv.ParseFloat(rawValue, 64); err == nil {
			return f
		}
		return 0
	}
	if val, ok := v[key]; ok {
		if f, ok := val.(float64); ok {
			return f
		}
	}
	return 0
}
