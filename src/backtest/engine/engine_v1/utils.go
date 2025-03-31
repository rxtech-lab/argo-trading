package engine

import (
	"fmt"
	"path/filepath"

	"github.com/sirily11/argo-trading-go/src/backtest/engine/engine_v1/commission_fee"
	"github.com/sirily11/argo-trading-go/src/backtest/engine/engine_v1/datasource"
	"github.com/sirily11/argo-trading-go/src/logger"
	s "github.com/sirily11/argo-trading-go/src/strategy"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

// runKey represents a unique key for a backtest run
type runKey struct {
	strategy, configPath, dataPath string
}

// createProgressBars creates progress bars for each combination of strategy, config, and data path
func createProgressBars(p *mpb.Progress, strategies []s.TradingStrategy, strategyConfigPaths []string, dataPaths []string, log *logger.Logger) (map[runKey]*mpb.Bar, error) {
	bars := make(map[runKey]*mpb.Bar)

	for _, strategy := range strategies {
		for _, configPath := range strategyConfigPaths {
			for _, dataPath := range dataPaths {
				// Initialize datasource to get count
				datasource, err := datasource.NewDataSource(":memory:", log)
				if err != nil {
					return nil, fmt.Errorf("failed to create data source: %w", err)
				}
				if err := datasource.Initialize(dataPath); err != nil {
					return nil, fmt.Errorf("failed to initialize data source: %w", err)
				}
				count, err := datasource.Count()
				if err != nil {
					return nil, fmt.Errorf("failed to get count of data source: %w", err)
				}

				key := runKey{
					strategy:   strategy.Name(),
					configPath: configPath,
					dataPath:   dataPath,
				}

				bars[key] = p.AddBar(int64(count),
					mpb.PrependDecorators(
						decor.Name(fmt.Sprintf("%s - %s", strategy.Name(), filepath.Base(dataPath)),
							decor.WC{W: len(strategy.Name()) + len(filepath.Base(dataPath)) + 3}),
					),
					mpb.AppendDecorators(
						decor.Percentage(),
						decor.OnComplete(
							decor.AverageETA(decor.ET_STYLE_GO),
							"done",
						),
					),
				)
			}
		}
	}

	return bars, nil
}

// Calculate the maximum quantity that can be bought with the given balance. Returns the quantity in integer
func CalculateMaxQuantity(balance float64, price float64, commissionFee commission_fee.CommissionFee) int {
	// Handle edge cases
	if price <= 0 || balance <= 0 {
		return 0
	}

	// Calculate the maximum quantity that can be bought with the given balance
	// We need to account for both the price and commission fee
	// We use binary search to find the maximum quantity that fits within the balance
	// The total cost should be: quantity * price + commissionFee.Calculate(quantity) <= balance
	left := 0
	right := int(balance / price) // Upper bound is balance/price
	maxQty := 0

	for left <= right {
		mid := (left + right) / 2
		totalCost := float64(mid)*price + commissionFee.Calculate(float64(mid))
		if totalCost <= balance {
			maxQty = mid
			left = mid + 1
		} else {
			right = mid - 1
		}
	}

	return maxQty
}
