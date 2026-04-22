---
title: Logging from Strategies
description: How to emit log messages from a trading strategy using the Log API
---

# Logging from Strategies

Strategies run as WebAssembly plugins and cannot write directly to the host's
stdout/stderr or filesystem. To emit log output, use the `Log` method on the
strategy API. Log entries are forwarded to the host where they are written to
the configured logger and, during backtests, persisted to log storage so they
can be reviewed alongside trades.

## Quick Start

Get the strategy API and call `Log` with a level and message:

```go
import (
    "context"

    "github.com/rxtech-lab/argo-trading/pkg/strategy"
)

func (s *MyStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
    api := strategy.NewStrategyApi()

    _, _ = api.Log(ctx, &strategy.LogRequest{
        Level:   strategy.LogLevel_LOG_LEVEL_INFO,
        Message: "Processing data for " + req.Data.Symbol,
    })

    return &emptypb.Empty{}, nil
}
```

`Log` returns `(*emptypb.Empty, error)`. The error is only non-nil if the
host call itself fails (for example, if the host is unavailable). Logging
errors are not fatal to the strategy and can typically be ignored with
`_, _ = api.Log(...)`.

## Log Levels

Use the `strategy.LogLevel_*` constants to indicate severity. Pick the level
that best matches the message so log filters and downstream tooling work
correctly.

| Constant | When to use |
|----------|-------------|
| `LogLevel_LOG_LEVEL_DEBUG` | Detailed diagnostic information useful while developing or debugging a strategy. Usually filtered out in production. |
| `LogLevel_LOG_LEVEL_INFO`  | Normal lifecycle events, such as a signal being generated or a position being opened. |
| `LogLevel_LOG_LEVEL_WARN`  | Unexpected but recoverable situations, such as an indicator returning insufficient data. |
| `LogLevel_LOG_LEVEL_ERROR` | Errors that prevented the strategy from doing what it intended, such as a failed order placement. |

## Structured Fields

In addition to a free-form `Message`, you can attach structured key/value
metadata via the `Fields` map. These are forwarded as logger fields on the host
and stored alongside the message, which makes them easier to query and filter
than embedding values into the message string.

```go
_, _ = api.Log(ctx, &strategy.LogRequest{
    Level:   strategy.LogLevel_LOG_LEVEL_INFO,
    Message: "Buy signal generated",
    Fields: map[string]string{
        "symbol":    data.Symbol,
        "price":     fmt.Sprintf("%.2f", data.Close),
        "rsi":       fmt.Sprintf("%.2f", rsiValue),
        "reason":    "rsi_oversold",
    },
})
```

Both `Message` and `Fields` are optional individually, but at least a message
is recommended so logs remain readable.

## Examples

### Logging an Error from a Failed Operation

```go
positions, err := api.GetPositions(ctx, &strategy.GetPositionsRequest{})
if err != nil {
    _, _ = api.Log(ctx, &strategy.LogRequest{
        Level:   strategy.LogLevel_LOG_LEVEL_ERROR,
        Message: "Failed to fetch positions",
        Fields: map[string]string{
            "error": err.Error(),
        },
    })
    return nil, err
}
```

### Debug-Level Tracing of Indicator Values

```go
signal, err := api.GetSignal(ctx, &strategy.GetSignalRequest{
    IndicatorType: strategy.IndicatorType_INDICATOR_TYPE_RSI,
})
if err == nil {
    _, _ = api.Log(ctx, &strategy.LogRequest{
        Level:   strategy.LogLevel_LOG_LEVEL_DEBUG,
        Message: "RSI signal evaluated",
        Fields: map[string]string{
            "symbol": data.Symbol,
            "signal": signal.Signal.String(),
        },
    })
}
```

### Warning When Skipping a Trade

```go
if availableCash < requiredCash {
    _, _ = api.Log(ctx, &strategy.LogRequest{
        Level:   strategy.LogLevel_LOG_LEVEL_WARN,
        Message: "Skipping order due to insufficient cash",
        Fields: map[string]string{
            "symbol":    data.Symbol,
            "required":  fmt.Sprintf("%.2f", requiredCash),
            "available": fmt.Sprintf("%.2f", availableCash),
        },
    })
    return &emptypb.Empty{}, nil
}
```

## Where Log Entries Go

When the host receives a log request it does two things:

1. Forwards the message and fields to the host's structured logger
   (zap), using the matching log method (`Debug`, `Info`, `Warn`, `Error`).
2. During backtests, persists the entry to log storage along with the current
   market data's timestamp and symbol, so logs can be correlated with the bar
   that was being processed when they were emitted.

If no host logger or log storage is configured, the call is silently dropped;
strategies do not need to check for this case.

## Best Practices

- Prefer structured `Fields` over string concatenation for values you may want
  to filter or aggregate later (symbols, prices, indicator values, error
  messages).
- Keep `Message` short and descriptive; put variable data in `Fields`.
- Use `DEBUG` liberally during development and `INFO`/`WARN`/`ERROR` for
  events that matter in production runs.
- Avoid logging on every tick at `INFO` or higher in tight loops; it can
  produce very large log volumes during long backtests.
- Ignore the error return from `Log` unless you specifically need to react to
  host transport failures.
