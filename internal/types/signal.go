package types

import "time"

type SignalType string

const (
	// SignalTypeBuyLong is a signal that tells the strategy to buy
	SignalTypeBuyLong SignalType = "buy_long"
	// SignalTypeSellLong is a signal that tells the strategy to sell
	SignalTypeSellLong SignalType = "sell_long"
	// SignalTypeBuyShort is a signal that tells the strategy to buy
	SignalTypeBuyShort SignalType = "buy_short"
	// SignalTypeSellShort is a signal that tells the strategy to sell
	SignalTypeSellShort SignalType = "sell_short"
	// SignalTypeNoAction is a signal that tells the strategy to take no action
	SignalTypeNoAction SignalType = "no_action"
	// SignalTypeClosePosition is a signal that tells the strategy to close a position
	SignalTypeClosePosition SignalType = "close_position"
	// SignalTypeWait is a signal that tells the strategy to wait more signal to confirm the entry
	SignalTypeWait SignalType = "wait"
	// SignalTypeAbort is a signal that tells the strategy to abort the current operation
	SignalTypeAbort SignalType = "abort"
)

type Signal struct {
	// Time is the time of the signal
	Time time.Time
	// Type is the type of the signal
	Type SignalType
	// Name is the name of the signal
	Name string
	// Reason is the reason for the signal
	Reason string
	// RawValue is the raw value of the signal
	RawValue any
	// Symbol is the symbol of the signal
	Symbol string
	// Indicator is the indicator that generated the signal
	Indicator IndicatorType
}
