package types

// Mark a point in time with a signal and a reason
type Mark struct {
	// Signal is the signal that was generated
	Signal Signal
	// Reason is the reason for the signal
	Reason string
}
