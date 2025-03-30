package types

import "time"

type SignalType string

const (
	// SignalTypeBuy is a signal that tells the strategy to buy
	SignalTypeBuy SignalType = "buy"
	// SignalTypeSell is a signal that tells the strategy to sell
	SignalTypeSell SignalType = "sell"
	// SignalTypeNoAction is a signal that tells the strategy to take no action
	SignalTypeNoAction SignalType = "no_action"
	// SignalTypeMultiEntry is a signal that tells the strategy to wait for other signals before taking action
	SignalTypeMultiEntry SignalType = "multi_entry"
	// SignalTypeMultiExit is a signal that tells the strategy to wait for other signals before taking action
	SignalTypeMultiExit SignalType = "multi_exit"
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
}
