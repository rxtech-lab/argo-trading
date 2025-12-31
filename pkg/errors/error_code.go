package errors

// ErrorCode represents a unique error code for identifying different error types.
type ErrorCode int

const (
	// ErrCodeUnknown represents a general unknown error (1-99 range).
	ErrCodeUnknown ErrorCode = 1

	// ErrCodeInvalidParameter indicates an invalid parameter was provided (100-199 range).
	ErrCodeInvalidParameter      ErrorCode = 100
	ErrCodeInvalidConfiguration  ErrorCode = 101
	ErrCodeInvalidExecuteOrder   ErrorCode = 102
	ErrCodeInvalidTakeProfit     ErrorCode = 103
	ErrCodeInvalidStopLoss       ErrorCode = 104
	ErrCodeInvalidOrder          ErrorCode = 105
	ErrCodeInsufficientData      ErrorCode = 106
	ErrCodeInvalidType           ErrorCode = 107
	ErrCodeInvalidPeriod         ErrorCode = 108
	ErrCodeMissingParameter      ErrorCode = 109
	ErrCodeInvalidVersion        ErrorCode = 110
	ErrCodeInvalidMultiplier     ErrorCode = 111
	ErrCodeInvalidThreshold      ErrorCode = 112
	ErrCodeInvalidStdDevPeriod   ErrorCode = 113
	ErrCodeInvalidDeadZone       ErrorCode = 114
	ErrCodeInvalidExplosionPower ErrorCode = 115
	ErrCodeInvalidLength         ErrorCode = 116
	ErrCodeInvalidFilterPeriod   ErrorCode = 117
	ErrCodeInvalidFilterType     ErrorCode = 118
	ErrCodeMarketDataRequired    ErrorCode = 119

	// ErrCodeDataNotFound indicates requested data was not found (200-299 range).
	ErrCodeDataNotFound          ErrorCode = 200
	ErrCodeDataSourceUnavailable ErrorCode = 201
	ErrCodeQueryFailed           ErrorCode = 202
	ErrCodeHistoricalDataFailed  ErrorCode = 203
	ErrCodeNoDataFound           ErrorCode = 204
	ErrCodeMarkerNotAvailable    ErrorCode = 205

	// ErrCodeIndicatorNotFound indicates a requested indicator was not found (300-399 range).
	ErrCodeIndicatorNotFound      ErrorCode = 300
	ErrCodeIndicatorAlreadyExists ErrorCode = 301
	ErrCodeIndicatorCalculation   ErrorCode = 302

	// ErrCodeStrategyNotLoaded indicates a strategy was not loaded (400-499 range).
	ErrCodeStrategyNotLoaded    ErrorCode = 400
	ErrCodeStrategyConfigError  ErrorCode = 401
	ErrCodeStrategyRuntimeError ErrorCode = 402
	ErrCodeUnsupportedStrategy  ErrorCode = 403
	ErrCodeVersionMismatch      ErrorCode = 404

	// ErrCodeOrderFailed indicates an order execution failed (500-599 range).
	ErrCodeOrderFailed       ErrorCode = 500
	ErrCodePositionNotFound  ErrorCode = 501
	ErrCodeMarketDataMissing ErrorCode = 502

	// ErrCodeBacktestStateNil indicates backtest state is nil (600-699 range).
	ErrCodeBacktestStateNil      ErrorCode = 600
	ErrCodeBacktestInitFailed    ErrorCode = 601
	ErrCodeBacktestConfigError   ErrorCode = 602
	ErrCodeBacktestDataPathError ErrorCode = 603
	ErrCodeBacktestNoStrategies  ErrorCode = 604
	ErrCodeBacktestNoConfigs     ErrorCode = 605
	ErrCodeBacktestNoDataPaths   ErrorCode = 606
	ErrCodeBacktestNoResultsDir  ErrorCode = 607
	ErrCodeBacktestNoDatasource  ErrorCode = 608

	// ErrCodeMarketDataFetchFailed indicates market data fetching failed (700-799 range).
	ErrCodeMarketDataFetchFailed ErrorCode = 700
	ErrCodeMarketDataWriteFailed ErrorCode = 701
	ErrCodeMarketDataParseFailed ErrorCode = 702
	ErrCodeInvalidTimespan       ErrorCode = 703
	ErrCodeInvalidProvider       ErrorCode = 704

	// ErrCodeCallbackFailed indicates a callback execution failed (800-899 range).
	ErrCodeCallbackFailed ErrorCode = 800
)
