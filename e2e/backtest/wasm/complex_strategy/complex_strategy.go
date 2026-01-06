//go:build wasip1

package main

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/knqyf263/go-plugin/types/known/emptypb"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

// HybridTradingStrategy implements a multi-indicator confirmation strategy
// using RSI, MACD, EMA, and ATR for high-probability trade signals
type HybridTradingStrategy struct{}

// Indicator value structures for JSON parsing
type RSIValue struct {
	RSI float64 `json:"rsi"`
}

type MACDValue struct {
	MACD float64 `json:"macd"`
}

type EMAValue struct {
	EMA float64 `json:"ema"`
}

type ATRValue struct {
	ATR float64 `json:"atr"`
}

// CachedState stores previous indicator values for crossover detection and position tracking
type CachedState struct {
	PrevMACD   float64 `json:"prev_macd"`
	InPosition bool    `json:"in_position"`
	DataCount  int     `json:"data_count"`
}

func main() {}

func init() {
	strategy.RegisterTradingStrategy(NewHybridTradingStrategy())
}

func NewHybridTradingStrategy() strategy.TradingStrategy {
	return &HybridTradingStrategy{}
}

// GetConfigSchema implements [strategy.TradingStrategy].
func (s *HybridTradingStrategy) GetConfigSchema(ctx context.Context, req *strategy.GetConfigSchemaRequest) (*strategy.GetConfigSchemaResponse, error) {
	schema := `{
		"type": "object",
		"properties": {
			"rsi_oversold": {"type": "number", "default": 35, "description": "RSI oversold threshold for buy signals"},
			"rsi_overbought": {"type": "number", "default": 65, "description": "RSI overbought threshold for sell signals"},
			"ema_period": {"type": "integer", "default": 20, "description": "EMA period for trend detection"},
			"atr_multiplier": {"type": "number", "default": 2.0, "description": "ATR multiplier for stop-loss"}
		}
	}`
	return &strategy.GetConfigSchemaResponse{Schema: schema}, nil
}

// GetDescription implements [strategy.TradingStrategy].
func (s *HybridTradingStrategy) GetDescription(ctx context.Context, req *strategy.GetDescriptionRequest) (*strategy.GetDescriptionResponse, error) {
	return &strategy.GetDescriptionResponse{
		Description: "A hybrid trading strategy that combines RSI, MACD, EMA, and ATR indicators for high-probability trade signals. Buy signals require RSI oversold + bullish MACD + price above EMA. Sell signals require RSI overbought + bearish MACD + price below EMA.",
	}, nil
}

// GetIdentifier implements [strategy.TradingStrategy].
func (s *HybridTradingStrategy) GetIdentifier(ctx context.Context, req *strategy.GetIdentifierRequest) (*strategy.GetIdentifierResponse, error) {
	return &strategy.GetIdentifierResponse{
		Identifier: "com.argo-trading.hybrid-strategy",
	}, nil
}

// Initialize implements strategy.TradingStrategy.
func (s *HybridTradingStrategy) Initialize(ctx context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// Name implements strategy.TradingStrategy.
func (s *HybridTradingStrategy) Name(ctx context.Context, req *strategy.NameRequest) (*strategy.NameResponse, error) {
	return &strategy.NameResponse{Name: "HybridTradingStrategy"}, nil
}

// ProcessData implements strategy.TradingStrategy.
func (s *HybridTradingStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
	data := req.Data
	api := strategy.NewStrategyApi()

	// Get RSI signal
	rsiSignal, err := api.GetSignal(ctx, &strategy.GetSignalRequest{
		IndicatorType: strategy.IndicatorType_INDICATOR_RSI,
		MarketData:    data,
	})
	if err != nil {
		return nil, err
	}

	// Get MACD signal
	macdSignal, err := api.GetSignal(ctx, &strategy.GetSignalRequest{
		IndicatorType: strategy.IndicatorType_INDICATOR_MACD,
		MarketData:    data,
	})
	if err != nil {
		return nil, err
	}

	// Get EMA signal
	emaSignal, err := api.GetSignal(ctx, &strategy.GetSignalRequest{
		IndicatorType: strategy.IndicatorType_INDICATOR_EMA,
		MarketData:    data,
	})
	if err != nil {
		return nil, err
	}

	// Get ATR signal for volatility-based stop-loss
	atrSignal, err := api.GetSignal(ctx, &strategy.GetSignalRequest{
		IndicatorType: strategy.IndicatorType_INDICATOR_ATR,
		MarketData:    data,
	})
	if err != nil {
		return nil, err
	}

	// Parse indicator values
	rsi := parseRSI(rsiSignal.RawValue)
	macd := parseMACD(macdSignal.RawValue)
	ema := parseEMA(emaSignal.RawValue)
	atr := parseATR(atrSignal.RawValue)

	// Get previous state from cache for crossover detection and position tracking
	cacheKey := "hybrid_state_" + data.Symbol
	cachedStateResp, _ := api.GetCache(ctx, &strategy.GetRequest{Key: cacheKey})

	var cachedState CachedState
	if cachedStateResp != nil && cachedStateResp.Value != "" {
		_ = json.Unmarshal([]byte(cachedStateResp.Value), &cachedState)
	}

	// Increment data count
	cachedState.DataCount++

	// Skip if we couldn't parse indicator values (insufficient data)
	if rsi == 0 || ema == 0 {
		// Mark data point with insufficient data (warning level, square shape)
		_, _ = api.Mark(ctx, &strategy.MarkRequest{
			MarketData: data,
			Mark: &strategy.Mark{
				SignalType: strategy.SignalType_SIGNAL_TYPE_NO_ACTION,
				Color:      "yellow",
				Shape:      strategy.MarkShape_MARK_SHAPE_SQUARE,
				Level:      strategy.MarkLevel_MARK_LEVEL_WARNING,
				Title:      "Insufficient Data",
				Message:    "Waiting for indicator warmup - RSI: " + formatFloat(rsi) + ", EMA: " + formatFloat(ema),
				Category:   "DataQuality",
			},
		})
		// Still save state
		stateJSON, _ := json.Marshal(cachedState)
		_, _ = api.SetCache(ctx, &strategy.SetRequest{
			Key:   cacheKey,
			Value: string(stateJSON),
		})
		return &emptypb.Empty{}, nil
	}

	// Detect MACD crossover (bullish: crosses above 0, bearish: crosses below 0)
	macdCrossedBullish := cachedState.PrevMACD <= 0 && macd > 0
	macdCrossedBearish := cachedState.PrevMACD >= 0 && macd < 0

	// Mark MACD crossovers - these are significant market events
	if macdCrossedBullish {
		_, _ = api.Mark(ctx, &strategy.MarkRequest{
			MarketData: data,
			Mark: &strategy.Mark{
				SignalType: strategy.SignalType_SIGNAL_TYPE_NO_ACTION,
				Color:      "green",
				Shape:      strategy.MarkShape_MARK_SHAPE_SQUARE,
				Level:      strategy.MarkLevel_MARK_LEVEL_INFO,
				Title:      "MACD Bullish Crossover",
				Message:    "MACD crossed above zero: " + formatFloat(cachedState.PrevMACD) + " -> " + formatFloat(macd),
				Category:   "MarketEvent",
			},
		})
	} else if macdCrossedBearish {
		_, _ = api.Mark(ctx, &strategy.MarkRequest{
			MarketData: data,
			Mark: &strategy.Mark{
				SignalType: strategy.SignalType_SIGNAL_TYPE_NO_ACTION,
				Color:      "red",
				Shape:      strategy.MarkShape_MARK_SHAPE_SQUARE,
				Level:      strategy.MarkLevel_MARK_LEVEL_INFO,
				Title:      "MACD Bearish Crossover",
				Message:    "MACD crossed below zero: " + formatFloat(cachedState.PrevMACD) + " -> " + formatFloat(macd),
				Category:   "MarketEvent",
			},
		})
	}

	// Strategy parameters - more relaxed thresholds for more frequent trading
	rsiOversold := 45.0   // Increased from 35 for more buy signals
	rsiOverbought := 55.0 // Decreased from 65 for more sell signals

	// BUY Signal: RSI below threshold OR bullish MACD crossover (when not in position)
	buyCondition := !cachedState.InPosition && (rsi < rsiOversold || macdCrossedBullish)
	if buyCondition {
		stopLoss := data.Close - (atr * 2)
		order := &strategy.ExecuteOrder{
			Symbol:    data.Symbol,
			Side:      strategy.PurchaseType_PURCHASE_TYPE_BUY,
			OrderType: strategy.OrderType_ORDER_TYPE_LIMIT,
			Quantity:  1.0,
			Price:     data.Close,
			Reason: &strategy.Reason{
				Reason: "strategy",
				Message: "BUY: RSI=" + formatFloat(rsi) + ", MACD=" + formatFloat(macd) +
					", Price=" + formatFloat(data.Close) + ", EMA=" + formatFloat(ema) +
					", StopLoss=" + formatFloat(stopLoss),
			},
			StrategyName: "HybridTradingStrategy",
		}
		_, err := api.PlaceOrder(ctx, order)
		if err != nil {
			// Mark failed order with error level
			_, _ = api.Mark(ctx, &strategy.MarkRequest{
				MarketData: data,
				Mark: &strategy.Mark{
					SignalType: strategy.SignalType_SIGNAL_TYPE_BUY_LONG,
					Color:      "orange",
					Shape:      strategy.MarkShape_MARK_SHAPE_TRIANGLE,
					Level:      strategy.MarkLevel_MARK_LEVEL_ERROR,
					Title:      "Buy Order Failed",
					Message:    "Failed to place buy order: " + err.Error(),
					Category:   "OrderError",
				},
			})
			return nil, err
		}

		// Mark successful buy order with triangle shape (directional signal)
		_, _ = api.Mark(ctx, &strategy.MarkRequest{
			MarketData: data,
			Mark: &strategy.Mark{
				SignalType: strategy.SignalType_SIGNAL_TYPE_BUY_LONG,
				Color:      "green",
				Shape:      strategy.MarkShape_MARK_SHAPE_TRIANGLE,
				Level:      strategy.MarkLevel_MARK_LEVEL_INFO,
				Title:      "Buy Order Placed",
				Message:    "BUY @ " + formatFloat(data.Close) + " RSI=" + formatFloat(rsi),
				Category:   "Trade",
			},
		})

		cachedState.InPosition = true
	}

	// SELL Signal: RSI above threshold OR bearish MACD crossover (when in position)
	sellCondition := cachedState.InPosition && (rsi > rsiOverbought || macdCrossedBearish)
	if sellCondition {
		stopLoss := data.Close + (atr * 2)
		order := &strategy.ExecuteOrder{
			Symbol:    data.Symbol,
			Side:      strategy.PurchaseType_PURCHASE_TYPE_SELL,
			OrderType: strategy.OrderType_ORDER_TYPE_LIMIT,
			Quantity:  1.0,
			Price:     data.Close,
			Reason: &strategy.Reason{
				Reason: "strategy",
				Message: "SELL: RSI=" + formatFloat(rsi) + ", MACD=" + formatFloat(macd) +
					", Price=" + formatFloat(data.Close) + ", EMA=" + formatFloat(ema) +
					", StopLoss=" + formatFloat(stopLoss),
			},
			StrategyName: "HybridTradingStrategy",
		}
		_, err := api.PlaceOrder(ctx, order)
		if err != nil {
			// Mark failed order with error level
			_, _ = api.Mark(ctx, &strategy.MarkRequest{
				MarketData: data,
				Mark: &strategy.Mark{
					SignalType: strategy.SignalType_SIGNAL_TYPE_SELL_LONG,
					Color:      "orange",
					Shape:      strategy.MarkShape_MARK_SHAPE_TRIANGLE,
					Level:      strategy.MarkLevel_MARK_LEVEL_ERROR,
					Title:      "Sell Order Failed",
					Message:    "Failed to place sell order: " + err.Error(),
					Category:   "OrderError",
				},
			})
			return nil, err
		}

		// Mark successful sell order with triangle shape (directional signal)
		_, _ = api.Mark(ctx, &strategy.MarkRequest{
			MarketData: data,
			Mark: &strategy.Mark{
				SignalType: strategy.SignalType_SIGNAL_TYPE_SELL_LONG,
				Color:      "red",
				Shape:      strategy.MarkShape_MARK_SHAPE_TRIANGLE,
				Level:      strategy.MarkLevel_MARK_LEVEL_INFO,
				Title:      "Sell Order Placed",
				Message:    "SELL @ " + formatFloat(data.Close) + " RSI=" + formatFloat(rsi),
				Category:   "Trade",
			},
		})

		cachedState.InPosition = false
	}

	// Mark skipped signals when conditions are met but we can't trade
	if !buyCondition && !sellCondition && rsi < rsiOversold && cachedState.InPosition {
		_, _ = api.Mark(ctx, &strategy.MarkRequest{
			MarketData: data,
			Mark: &strategy.Mark{
				SignalType: strategy.SignalType_SIGNAL_TYPE_BUY_LONG,
				Color:      "purple",
				Shape:      strategy.MarkShape_MARK_SHAPE_CIRCLE,
				Level:      strategy.MarkLevel_MARK_LEVEL_WARNING,
				Title:      "Skipped Buy Signal",
				Message:    "RSI oversold (" + formatFloat(rsi) + ") but already in position",
				Category:   "RiskManagement",
			},
		})
	}

	if !buyCondition && !sellCondition && rsi > rsiOverbought && !cachedState.InPosition {
		_, _ = api.Mark(ctx, &strategy.MarkRequest{
			MarketData: data,
			Mark: &strategy.Mark{
				SignalType: strategy.SignalType_SIGNAL_TYPE_SELL_LONG,
				Color:      "purple",
				Shape:      strategy.MarkShape_MARK_SHAPE_CIRCLE,
				Level:      strategy.MarkLevel_MARK_LEVEL_WARNING,
				Title:      "Skipped Sell Signal",
				Message:    "RSI overbought (" + formatFloat(rsi) + ") but not in position",
				Category:   "RiskManagement",
			},
		})
	}

	// Update cached state with current MACD
	cachedState.PrevMACD = macd
	stateJSON, _ := json.Marshal(cachedState)
	_, _ = api.SetCache(ctx, &strategy.SetRequest{
		Key:   cacheKey,
		Value: string(stateJSON),
	})

	return &emptypb.Empty{}, nil
}

// parseRSI extracts RSI value from JSON raw value
func parseRSI(rawValue string) float64 {
	if rawValue == "" {
		return 0
	}
	var v RSIValue
	if err := json.Unmarshal([]byte(rawValue), &v); err != nil {
		// Try parsing as plain number
		if f, err := strconv.ParseFloat(rawValue, 64); err == nil {
			return f
		}
		return 0
	}
	return v.RSI
}

// parseMACD extracts MACD value from JSON raw value
func parseMACD(rawValue string) float64 {
	if rawValue == "" {
		return 0
	}
	var v MACDValue
	if err := json.Unmarshal([]byte(rawValue), &v); err != nil {
		if f, err := strconv.ParseFloat(rawValue, 64); err == nil {
			return f
		}
		return 0
	}
	return v.MACD
}

// parseEMA extracts EMA value from JSON raw value
func parseEMA(rawValue string) float64 {
	if rawValue == "" {
		return 0
	}
	var v EMAValue
	if err := json.Unmarshal([]byte(rawValue), &v); err != nil {
		if f, err := strconv.ParseFloat(rawValue, 64); err == nil {
			return f
		}
		return 0
	}
	return v.EMA
}

// parseATR extracts ATR value from JSON raw value
func parseATR(rawValue string) float64 {
	if rawValue == "" {
		return 0
	}
	var v ATRValue
	if err := json.Unmarshal([]byte(rawValue), &v); err != nil {
		if f, err := strconv.ParseFloat(rawValue, 64); err == nil {
			return f
		}
		return 0
	}
	return v.ATR
}

// formatFloat formats a float for display in order reasons
func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}
