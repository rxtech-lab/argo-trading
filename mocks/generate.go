package mocks

//go:generate mockgen -destination=./mock_trading.go -package=mocks github.com/rxtech-lab/argo-trading/internal/trading TradingSystem
//go:generate mockgen -destination=./mock_indicator.go -package=mocks github.com/rxtech-lab/argo-trading/internal/indicator Indicator
//go:generate mockgen -destination=./mock_datasource.go -package=mocks github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource DataSource
//go:generate mockgen -destination=./mock_cache.go -package=mocks github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/cache Cache
//go:generate mockgen -destination=./mock_marker.go -package=mocks github.com/rxtech-lab/argo-trading/internal/marker Marker
//go:generate mockgen -destination=./mock_indicator_registry.go -package=mocks github.com/rxtech-lab/argo-trading/internal/indicator IndicatorRegistry
//go:generate mockgen -destination=./mock_strategy_runtime.go -package=mocks github.com/rxtech-lab/argo-trading/internal/runtime StrategyRuntime
