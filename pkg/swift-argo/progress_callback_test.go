package swiftargo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type progressEvent struct {
	current int
	total   int
}

func collectingEmitter() (emit func(current, total int) error, events *[]progressEvent) {
	captured := make([]progressEvent, 0)

	return func(current, total int) error {
		captured = append(captured, progressEvent{current: current, total: total})

		return nil
	}, &captured
}

// TestThrottledProgressCallback_FirstCallbackOfNewRunIsForwarded is the
// regression guard for the "stuck at 0" bug: when a second run begins with
// a different total, the first callback (current=0) must reach the UI so
// it sees the new total immediately. Previously this was suppressed by the
// step-based throttle because lastReported was reset to 0 on the same tick.
func TestThrottledProgressCallback_FirstCallbackOfNewRunIsForwarded(t *testing.T) {
	emit, events := collectingEmitter()
	cb := newThrottledProgressCallback(emit)

	// First run: small dataset, runs to completion.
	require.NoError(t, cb(0, 475))
	require.NoError(t, cb(475, 475))

	before := len(*events)

	// Second run: large dataset. progressStep = 1_000_000/200 = 5000, so
	// without the new-run bypass, (0, 1_000_000) would be throttled away
	// and the UI would keep showing the stale prior state.
	require.NoError(t, cb(0, 1_000_000))

	require.Greater(t, len(*events), before, "first callback of new run must be forwarded")
	last := (*events)[len(*events)-1]
	assert.Equal(t, 0, last.current)
	assert.Equal(t, 1_000_000, last.total, "UI must learn the new total on the first tick of the new run")
}

// TestThrottledProgressCallback_FinalBarAlwaysForwarded ensures the
// final progress update is never throttled out, so the UI always reaches
// 100%.
func TestThrottledProgressCallback_FinalBarAlwaysForwarded(t *testing.T) {
	emit, events := collectingEmitter()
	cb := newThrottledProgressCallback(emit)

	const total = 10_000
	for i := 0; i <= total; i++ {
		require.NoError(t, cb(i, total))
	}

	last := (*events)[len(*events)-1]
	assert.Equal(t, total, last.current)
	assert.Equal(t, total, last.total)
}

// TestThrottledProgressCallback_ThrottlesWithinRun ensures the throttle
// actually reduces callback volume to roughly the ≤200-update budget so
// we don't regress back to per-bar forwarding and flood the bridge.
func TestThrottledProgressCallback_ThrottlesWithinRun(t *testing.T) {
	emit, events := collectingEmitter()
	cb := newThrottledProgressCallback(emit)

	const total = 100_000
	for i := 0; i <= total; i++ {
		require.NoError(t, cb(i, total))
	}

	// Budget: ≤200 throttled updates + 1 final bar. Allow a small cushion
	// for the "first bar of new run" forwarding.
	assert.LessOrEqual(t, len(*events), 210, "should throttle to ~200 updates per run")
	assert.GreaterOrEqual(t, len(*events), 100, "should still emit periodic updates, not just the final bar")
}

// TestThrottledProgressCallback_SmallDatasetEmitsEveryBar verifies the
// progressStep floor of 1 — for datasets smaller than 200 bars, every bar
// should be forwarded (no starvation).
func TestThrottledProgressCallback_SmallDatasetEmitsEveryBar(t *testing.T) {
	emit, events := collectingEmitter()
	cb := newThrottledProgressCallback(emit)

	const total = 50
	for i := 0; i <= total; i++ {
		require.NoError(t, cb(i, total))
	}

	assert.Len(t, *events, total+1, "every bar (including the initial and final) should be forwarded for small datasets")
}

// TestThrottledProgressCallback_DifferentTotalsResetState guards against
// state leaking across runs — a second run with a smaller total must not
// inherit the first run's lastReported value.
func TestThrottledProgressCallback_DifferentTotalsResetState(t *testing.T) {
	emit, events := collectingEmitter()
	cb := newThrottledProgressCallback(emit)

	// Large run reaches a high lastReported.
	require.NoError(t, cb(0, 100_000))
	require.NoError(t, cb(50_000, 100_000))
	require.NoError(t, cb(100_000, 100_000))

	before := len(*events)

	// Smaller second run: must emit from bar 0.
	require.NoError(t, cb(0, 400))
	require.NoError(t, cb(1, 400))

	require.Greater(t, len(*events), before)
	first := (*events)[before]
	assert.Equal(t, 0, first.current)
	assert.Equal(t, 400, first.total)
}
