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

// CachedState stores previous indicator values for crossover detection
type CachedState struct {
	PrevMACD float64 `json:"prev_macd"`
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

	// Skip if we couldn't parse indicator values (insufficient data)
	if rsi == 0 || ema == 0 {
		return &emptypb.Empty{}, nil
	}

	// Get previous MACD from cache for crossover detection
	cacheKey := "hybrid_state_" + data.Symbol
	cachedStateResp, _ := api.GetCache(ctx, &strategy.GetRequest{Key: cacheKey})

	var cachedState CachedState
	if cachedStateResp != nil && cachedStateResp.Value != "" {
		json.Unmarshal([]byte(cachedStateResp.Value), &cachedState)
	}

	// Detect MACD crossover (bullish: crosses above 0, bearish: crosses below 0)
	macdCrossedBullish := cachedState.PrevMACD <= 0 && macd > 0
	macdCrossedBearish := cachedState.PrevMACD >= 0 && macd < 0

	// Strategy parameters (moderate risk)
	rsiOversold := 35.0
	rsiOverbought := 65.0

	// BUY Signal: RSI oversold + bullish MACD + price above EMA (uptrend)
	buyCondition := rsi < rsiOversold && (macd > 0 || macdCrossedBullish) && data.Close > ema
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
				Message: "BUY: RSI=" + formatFloat(rsi) + " (oversold<" + formatFloat(rsiOversold) + "), " +
					"MACD=" + formatFloat(macd) + " (bullish), " +
					"Price=" + formatFloat(data.Close) + " > EMA=" + formatFloat(ema) + ", " +
					"StopLoss=" + formatFloat(stopLoss),
			},
			StrategyName: "HybridTradingStrategy",
		}
		_, err := api.PlaceOrder(ctx, order)
		if err != nil {
			return nil, err
		}
	}

	// SELL Signal: RSI overbought + bearish MACD + price below EMA (downtrend)
	sellCondition := rsi > rsiOverbought && (macd < 0 || macdCrossedBearish) && data.Close < ema
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
				Message: "SELL: RSI=" + formatFloat(rsi) + " (overbought>" + formatFloat(rsiOverbought) + "), " +
					"MACD=" + formatFloat(macd) + " (bearish), " +
					"Price=" + formatFloat(data.Close) + " < EMA=" + formatFloat(ema) + ", " +
					"StopLoss=" + formatFloat(stopLoss),
			},
			StrategyName: "HybridTradingStrategy",
		}
		_, err := api.PlaceOrder(ctx, order)
		if err != nil {
			return nil, err
		}
	}

	// Update cached state with current MACD
	cachedState.PrevMACD = macd
	stateJSON, _ := json.Marshal(cachedState)
	api.SetCache(ctx, &strategy.SetRequest{
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
