package types

type IndicatorType string

const (
	IndicatorTypeRSI                   IndicatorType = "rsi"
	IndicatorTypeMACD                  IndicatorType = "macd"
	IndicatorTypeBollingerBands        IndicatorType = "bollinger_bands"
	IndicatorTypeStochasticOsciallator IndicatorType = "stochastic_oscillator"
	IndicatorTypeWilliamsR             IndicatorType = "williams_r"
	IndicatorTypeADX                   IndicatorType = "adx"
	IndicatorTypeCCI                   IndicatorType = "cci"
	IndicatorTypeAO                    IndicatorType = "ao"
	IndicatorTypeTrendStrength         IndicatorType = "trend_strength"
	IndicatorTypeRangeFilter           IndicatorType = "range_filter"
	IndicatorTypeEMA                   IndicatorType = "ema"
	IndicatorTypeWaddahAttar           IndicatorType = "waddah_attar"
	IndicatorTypeATR                   IndicatorType = "atr"
	IndicatorTypeMA                    IndicatorType = "ma"
)
