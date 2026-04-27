package engine

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/types"
)

// applyTradeToCache updates the cached Position for a symbol to reflect a newly
// committed trade. Mirrors the bucket assignments in getPositionFromDB so cache
// reads stay in lockstep with what an SQL recomputation would return. Must be
// called only after the trade insert transaction commits successfully.
func (b *BacktestState) applyTradeToCache(order types.Order) {
	b.positionCacheMu.Lock()
	defer b.positionCacheMu.Unlock()

	pos, ok := b.positionCache[order.Symbol]
	if !ok {
		pos = newEmptyPosition(order.Symbol)
		pos.OpenTimestamp = order.Timestamp
		b.positionCache[order.Symbol] = pos
	} else if pos.OpenTimestamp.IsZero() || order.Timestamp.Before(pos.OpenTimestamp) {
		// Match SQL MIN(executed_at) for first trade time. Handles the case
		// where the cache was populated from a zero-position SQL fallback
		// before any trades existed for this symbol.
		pos.OpenTimestamp = order.Timestamp
	}

	qty := order.Quantity
	amount := qty * order.Price
	fee := order.Fee

	switch {
	case order.Side == types.PurchaseTypeBuy && order.PositionType == types.PositionTypeLong:
		pos.TotalLongInPositionQuantity += qty
		pos.TotalLongInPositionAmount += amount
		pos.TotalLongInFee += fee
	case order.Side == types.PurchaseTypeSell && order.PositionType == types.PositionTypeLong:
		pos.TotalLongOutPositionQuantity += qty
		pos.TotalLongOutPositionAmount += amount
		pos.TotalLongOutFee += fee
	case order.Side == types.PurchaseTypeSell && order.PositionType == types.PositionTypeShort:
		pos.TotalShortOutPositionQuantity += qty
		pos.TotalShortOutPositionAmount += amount
		pos.TotalShortOutFee += fee
	case order.Side == types.PurchaseTypeBuy && order.PositionType == types.PositionTypeShort:
		pos.TotalShortInPositionQuantity += qty
		pos.TotalShortInPositionAmount += amount
		pos.TotalShortInFee += fee
	}

	// Match SQL MAX(strategy_name) (alphabetical max across all trades).
	if order.StrategyName > pos.StrategyName {
		pos.StrategyName = order.StrategyName
	}

	pos.TotalLongPositionQuantity = pos.TotalLongInPositionQuantity - pos.TotalLongOutPositionQuantity
	pos.TotalShortPositionQuantity = pos.TotalShortInPositionQuantity - pos.TotalShortOutPositionQuantity
}

// resetPositionCache discards all cached positions. Called when the underlying
// trades table is truncated/recreated so cached values can't go stale.
func (b *BacktestState) resetPositionCache() {
	b.positionCacheMu.Lock()
	b.positionCache = make(map[string]*types.Position)
	b.positionCacheMu.Unlock()
}

// newEmptyPosition returns a Position with all aggregate fields zeroed for the
// given symbol. Used as the starting point for both cache fallback (SQL no-rows)
// and the first incremental update for a symbol.
func newEmptyPosition(symbol string) *types.Position {
	return &types.Position{
		Symbol:                        symbol,
		TotalLongPositionQuantity:     0,
		TotalShortPositionQuantity:    0,
		TotalLongInPositionQuantity:   0,
		TotalLongOutPositionQuantity:  0,
		TotalLongInPositionAmount:     0,
		TotalLongOutPositionAmount:    0,
		TotalShortInPositionQuantity:  0,
		TotalShortOutPositionQuantity: 0,
		TotalShortInPositionAmount:    0,
		TotalShortOutPositionAmount:   0,
		TotalLongInFee:                0,
		TotalLongOutFee:               0,
		TotalShortInFee:               0,
		TotalShortOutFee:              0,
		OpenTimestamp:                 time.Time{},
		StrategyName:                  "",
	}
}

// getPositionFromDB recomputes a Position from the trades table via SQL.
func (b *BacktestState) getPositionFromDB(symbol string) (types.Position, error) {
	// Extended CTEs to calculate both long (buy/sell) and short (sell/buy) position fields
	query := `
    WITH long_buy_trades AS (
       SELECT
          SUM(executed_qty) as total_long_in_qty,
          SUM(commission) as total_long_in_fee,
          SUM(executed_qty * executed_price) as total_long_in_amount
       FROM trades
       WHERE symbol = ? AND order_type = ? AND position_type = ?
    ),
    long_sell_trades AS (
       SELECT
          SUM(executed_qty) as total_long_out_qty,
          SUM(commission) as total_long_out_fee,
          SUM(executed_qty * executed_price) as total_long_out_amount
       FROM trades
       WHERE symbol = ? AND order_type = ? AND position_type = ?
    ),
    short_sell_trades AS (
       SELECT
          SUM(executed_qty) as total_out_short_qty,
          SUM(commission) as total_short_out_fee,
          SUM(executed_qty * executed_price) as total_short_out_amount
       FROM trades
       WHERE symbol = ? AND order_type = ? AND position_type = ?
    ),
    short_buy_trades AS (
       SELECT
          SUM(executed_qty) as total_short_in_qty,
          SUM(commission) as total_short_in_fee,
          SUM(executed_qty * executed_price) as total_short_in_amount
       FROM trades
       WHERE symbol = ? AND order_type = ? AND position_type = ?
    ),
    first_trade AS (
       SELECT
          MIN(executed_at) as first_trade_time
       FROM trades
       WHERE symbol = ?
    )
    SELECT
       ? as symbol,
       COALESCE(b.total_long_in_qty, 0) - COALESCE(s.total_long_out_qty, 0) as quantity,
       COALESCE(b.total_long_in_qty, 0) as total_in_long_position_quantity,
       COALESCE(s.total_long_out_qty, 0) as total_out_long_position_quantity,
       COALESCE(b.total_long_in_amount, 0) as total_in_long_position_amount,
       COALESCE(s.total_long_out_amount, 0) as total_out_long_position_amount,
       COALESCE(b.total_long_in_fee, 0) as total_long_in_fee,
       COALESCE(s.total_long_out_fee, 0)  as total_long_out_fee,
       COALESCE(sb.total_short_in_fee, 0) as total_short_in_fee,
       COALESCE(ss.total_short_out_fee, 0)  as total_short_out_fee,
       ft.first_trade_time as open_timestamp,
       MAX(t.strategy_name) as strategy_name,
       COALESCE(sb.total_short_in_qty, 0) as total_in_short_position_quantity,
       COALESCE(ss.total_out_short_qty, 0) as total_out_short_position_quantity,
       COALESCE(sb.total_short_in_amount, 0) as total_in_short_position_amount,
       COALESCE(ss.total_short_out_amount, 0) as total_out_short_position_amount,
       COALESCE(sb.total_short_in_qty, 0) - COALESCE(ss.total_out_short_qty, 0) as short_quantity
    FROM trades t
    LEFT JOIN long_buy_trades b ON 1=1
    LEFT JOIN long_sell_trades s ON 1=1
    LEFT JOIN short_sell_trades ss ON 1=1
    LEFT JOIN short_buy_trades sb ON 1=1
    CROSS JOIN first_trade ft
    WHERE t.symbol = ?
    GROUP BY b.total_long_in_qty, s.total_long_out_qty, b.total_long_in_amount, s.total_long_out_amount, b.total_long_in_fee, s.total_long_out_fee, sb.total_short_in_fee, ss.total_short_out_fee, ss.total_out_short_qty, sb.total_short_in_qty, ss.total_short_out_amount, sb.total_short_in_amount, ss.total_short_out_fee, sb.total_short_in_fee, ft.first_trade_time
    `

	args := []interface{}{
		symbol, types.PurchaseTypeBuy, types.PositionTypeLong, // long_buy_trades
		symbol, types.PurchaseTypeSell, types.PositionTypeLong, // long_sell_trades
		symbol, types.PurchaseTypeSell, types.PositionTypeShort, // short_sell_trades
		symbol, types.PurchaseTypeBuy, types.PositionTypeShort, // short_buy_trades
		symbol, // first_trade CTE symbol parameter
		symbol, // symbol for select
		symbol, // symbol for WHERE
	}

	var position types.Position
	err := b.db.QueryRow(query, args...).Scan(
		&position.Symbol,
		&position.TotalLongPositionQuantity,
		&position.TotalLongInPositionQuantity,
		&position.TotalLongOutPositionQuantity,
		&position.TotalLongInPositionAmount,
		&position.TotalLongOutPositionAmount,
		&position.TotalLongInFee,
		&position.TotalLongOutFee,
		&position.TotalShortInFee,
		&position.TotalShortOutFee,
		&position.OpenTimestamp,
		&position.StrategyName,
		&position.TotalShortInPositionQuantity,
		&position.TotalShortOutPositionQuantity,
		&position.TotalShortInPositionAmount,
		&position.TotalShortOutPositionAmount,
		&position.TotalShortPositionQuantity,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return *newEmptyPosition(symbol), nil
	}

	if err != nil {
		return types.Position{}, fmt.Errorf("failed to query position: %w", err)
	}

	return position, nil
}
