package main

import "github.com/rxtech-lab/argo-trading/internal/types"

// MarketDataMsg carries new market data from the stream.
type MarketDataMsg struct {
	Data types.MarketData
}

// StreamErrorMsg indicates an error in the data stream.
type StreamErrorMsg struct {
	Err error
}

// StreamStartedMsg signals that streaming has begun.
type StreamStartedMsg struct{}
