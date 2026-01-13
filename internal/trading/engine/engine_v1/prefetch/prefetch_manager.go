package prefetch

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/polygon-io/client-go/rest/models"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/trading/engine"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/marketdata/provider"
	"github.com/rxtech-lab/argo-trading/pkg/marketdata/writer"
	"go.uber.org/zap"
)

// PrefetchManager handles historical data prefetching and gap filling.
type PrefetchManager struct {
	config           engine.PrefetchConfig
	provider         provider.Provider
	streamingWriter  *writer.StreamingDuckDBWriter
	logger           *logger.Logger
	onStatusUpdate   *engine.OnStatusUpdateCallback
	parquetPath      string
	interval         string
	gapToleranceUnit time.Duration
}

// NewPrefetchManager creates a new PrefetchManager instance.
func NewPrefetchManager(log *logger.Logger) *PrefetchManager {
	return &PrefetchManager{
		config:           engine.PrefetchConfig{}, //nolint:exhaustruct // zero value is fine
		provider:         nil,
		streamingWriter:  nil,
		logger:           log,
		onStatusUpdate:   nil,
		parquetPath:      "",
		interval:         "",
		gapToleranceUnit: time.Minute,
	}
}

// Initialize sets up the prefetch manager with required components.
func (p *PrefetchManager) Initialize(
	config engine.PrefetchConfig,
	prov provider.Provider,
	streamingWriter *writer.StreamingDuckDBWriter,
	interval string,
	onStatusUpdate *engine.OnStatusUpdateCallback,
) {
	p.config = config
	p.provider = prov
	p.streamingWriter = streamingWriter
	p.onStatusUpdate = onStatusUpdate
	p.parquetPath = streamingWriter.GetOutputPath()
	p.interval = interval
	p.gapToleranceUnit = parseIntervalDuration(interval)
}

// parseIntervalDuration converts interval string to time.Duration.
func parseIntervalDuration(interval string) time.Duration {
	switch interval {
	case "1s":
		return time.Second
	case "1m":
		return time.Minute
	case "3m":
		return 3 * time.Minute
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "30m":
		return 30 * time.Minute
	case "1h":
		return time.Hour
	case "2h":
		return 2 * time.Hour
	case "4h":
		return 4 * time.Hour
	case "6h":
		return 6 * time.Hour
	case "8h":
		return 8 * time.Hour
	case "12h":
		return 12 * time.Hour
	case "1d":
		return 24 * time.Hour
	case "3d":
		return 72 * time.Hour
	case "1w":
		return 168 * time.Hour
	default:
		return time.Minute
	}
}

// intervalToTimespan converts interval string to Polygon Timespan.
func intervalToTimespan(interval string) models.Timespan {
	switch interval {
	case "1s":
		return models.Second
	case "1m", "3m", "5m", "15m", "30m":
		return models.Minute
	case "1h", "2h", "4h", "6h", "8h", "12h":
		return models.Hour
	case "1d", "3d":
		return models.Day
	case "1w":
		return models.Week
	default:
		return models.Minute
	}
}

// intervalToMultiplier extracts the multiplier from interval string.
func intervalToMultiplier(interval string) int {
	switch interval {
	case "1s", "1m", "1h", "1d", "1w":
		return 1
	case "3m", "3d":
		return 3
	case "5m":
		return 5
	case "15m":
		return 15
	case "30m":
		return 30
	case "2h":
		return 2
	case "4h":
		return 4
	case "6h":
		return 6
	case "8h":
		return 8
	case "12h":
		return 12
	default:
		return 1
	}
}

// emitStatus sends a status update callback.
//
//nolint:funcorder // helper method used by exported methods
func (p *PrefetchManager) emitStatus(status types.EngineStatus) {
	if p.onStatusUpdate != nil {
		_ = (*p.onStatusUpdate)(status)
	}
}

// ExecutePrefetch downloads historical data for the specified symbols.
// This is Phase 1 of the prefetch process.
func (p *PrefetchManager) ExecutePrefetch(ctx context.Context, symbols []string) error {
	if !p.config.Enabled {
		p.logger.Info("Prefetch is disabled, skipping")

		return nil
	}

	p.emitStatus(types.EngineStatusPrefetching)
	p.logger.Info("Starting historical data prefetch")

	// Calculate start time
	startTime := p.calculateStartTime()

	p.logger.Info("Prefetch parameters",
		zap.Time("start_time", startTime),
		zap.Time("end_time", time.Now()),
		zap.Strings("symbols", symbols),
		zap.String("interval", p.interval),
	)

	// Download historical data for each symbol
	for _, symbol := range symbols {
		if err := p.downloadSymbolData(ctx, symbol, startTime); err != nil {
			p.logger.Warn("Failed to prefetch data for symbol",
				zap.String("symbol", symbol),
				zap.Error(err),
			)
			// Continue with other symbols
			continue
		}
	}

	p.logger.Info("Prefetch completed")

	return nil
}

// calculateStartTime determines the prefetch start time based on config.
//
//nolint:funcorder // helper method used by exported methods
func (p *PrefetchManager) calculateStartTime() time.Time {
	if p.config.StartTimeType == "date" {
		return p.config.StartTime
	}

	// Default to "days" mode
	return time.Now().AddDate(0, 0, -p.config.Days)
}

// downloadSymbolData downloads historical data for a single symbol.
//
//nolint:funcorder // helper method used by exported methods
func (p *PrefetchManager) downloadSymbolData(ctx context.Context, symbol string, startTime time.Time) error {
	p.logger.Info("Downloading historical data",
		zap.String("symbol", symbol),
		zap.Time("start", startTime),
	)

	// Configure the provider to write to the streaming writer
	p.provider.ConfigWriter(p.streamingWriter)

	// Download the data
	_, err := p.provider.Download(
		ctx,
		symbol,
		startTime,
		time.Now(),
		intervalToMultiplier(p.interval),
		intervalToTimespan(p.interval),
		nil, // No progress callback
	)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	return nil
}

// GetLastStoredTimestamp returns the timestamp of the last stored data point.
func (p *PrefetchManager) GetLastStoredTimestamp(symbol string) (time.Time, error) {
	// Check if parquet file exists
	if _, err := os.Stat(p.parquetPath); os.IsNotExist(err) {
		return time.Time{}, fmt.Errorf("parquet file does not exist: %s", p.parquetPath)
	}

	// Open DuckDB to query the parquet file
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to open DuckDB: %w", err)
	}

	defer db.Close()

	// Query for the last timestamp
	query := fmt.Sprintf(`
		SELECT MAX(time) as last_time
		FROM read_parquet('%s')
		WHERE symbol = '%s'
	`, p.parquetPath, symbol)

	var lastTime sql.NullTime

	err = db.QueryRow(query).Scan(&lastTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to query last timestamp: %w", err)
	}

	if !lastTime.Valid {
		return time.Time{}, fmt.Errorf("no data found for symbol: %s", symbol)
	}

	return lastTime.Time, nil
}

// DetectGap checks if there's a gap between stored data and stream data.
// Returns the gap duration. If gap is less than tolerance, returns 0.
func (p *PrefetchManager) DetectGap(firstStreamTime time.Time, symbol string) (time.Duration, error) {
	lastStored, err := p.GetLastStoredTimestamp(symbol)
	if err != nil {
		// No stored data - this is fine, gap filling will handle it
		return 0, nil
	}

	gap := firstStreamTime.Sub(lastStored)
	tolerance := 2 * p.gapToleranceUnit

	if gap <= tolerance {
		p.logger.Debug("Gap within tolerance, no fill needed",
			zap.Duration("gap", gap),
			zap.Duration("tolerance", tolerance),
		)

		return 0, nil
	}

	p.logger.Info("Gap detected",
		zap.Time("last_stored", lastStored),
		zap.Time("first_stream", firstStreamTime),
		zap.Duration("gap", gap),
	)

	return gap, nil
}

// FillGap downloads and stores data to fill the gap.
// This is Phase 3 of the prefetch process.
func (p *PrefetchManager) FillGap(ctx context.Context, symbol string, from time.Time, to time.Time) error {
	p.emitStatus(types.EngineStatusGapFilling)

	p.logger.Info("Filling gap",
		zap.String("symbol", symbol),
		zap.Time("from", from),
		zap.Time("to", to),
	)

	// Configure the provider to write to the streaming writer
	p.provider.ConfigWriter(p.streamingWriter)

	// Download the gap data
	_, err := p.provider.Download(
		ctx,
		symbol,
		from,
		to,
		intervalToMultiplier(p.interval),
		intervalToTimespan(p.interval),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to fill gap: %w", err)
	}

	p.logger.Info("Gap filled successfully")

	return nil
}

// HandleStreamStart processes the transition from prefetch to live streaming.
// It detects gaps and fills them before live trading begins.
// This combines Phases 2 and 3.
func (p *PrefetchManager) HandleStreamStart(ctx context.Context, firstStreamTime time.Time, symbols []string) error {
	if !p.config.Enabled {
		return nil
	}

	for _, symbol := range symbols {
		gap, err := p.DetectGap(firstStreamTime, symbol)
		if err != nil {
			p.logger.Warn("Failed to detect gap",
				zap.String("symbol", symbol),
				zap.Error(err),
			)

			continue
		}

		if gap > 0 {
			lastStored, _ := p.GetLastStoredTimestamp(symbol)

			if err := p.FillGap(ctx, symbol, lastStored, firstStreamTime); err != nil {
				p.logger.Warn("Failed to fill gap",
					zap.String("symbol", symbol),
					zap.Error(err),
				)
				// Continue - we can trade with incomplete data
			}
		}
	}

	// Transition to running status
	p.emitStatus(types.EngineStatusRunning)

	return nil
}

// IsEnabled returns whether prefetch is enabled.
func (p *PrefetchManager) IsEnabled() bool {
	return p.config.Enabled
}

// GetConfig returns the prefetch configuration.
func (p *PrefetchManager) GetConfig() engine.PrefetchConfig {
	return p.config
}
