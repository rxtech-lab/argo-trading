// Package errors provides structured error handling with typed error codes.
//
// Error codes are organized into categories:
//   - General errors (1-99): Unknown and general errors
//   - Validation errors (100-199): Invalid parameters, missing data, type mismatches
//   - Data/Resource errors (200-299): Data not found, query failures, unavailable resources
//   - Indicator errors (300-399): Technical indicator calculation and lookup errors
//   - Strategy errors (400-499): Strategy loading, configuration, and runtime errors
//   - Trading errors (500-599): Order execution and position management errors
//   - Backtest errors (600-699): Backtesting engine and state errors
//   - Market data errors (700-799): Market data fetching and parsing errors
//   - Callback errors (800-899): Callback execution failures
//
// Usage:
//
//	// Create a new error
//	err := errors.New(errors.ErrCodeInvalidParameter, "invalid parameter value")
//
//	// Create a formatted error
//	err := errors.Newf(errors.ErrCodeDataNotFound, "data not found for symbol %s", symbol)
//
//	// Wrap an existing error
//	err := errors.Wrap(errors.ErrCodeQueryFailed, "failed to execute query", originalErr)
//
//	// Check error code
//	if errors.HasCode(err, errors.ErrCodeDataNotFound) { ... }
package errors

import (
	"errors"
	"fmt"
)

// Error represents a structured error with an error code and message.
type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

// New creates a new Error with the given code and message.
func New(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   nil,
	}
}

// Newf creates a new Error with the given code and formatted message.
func Newf(code ErrorCode, format string, args ...any) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   nil,
	}
}

// Wrap wraps an existing error with a new Error containing the given code and message.
func Wrap(code ErrorCode, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Wrapf wraps an existing error with a new Error containing the given code and formatted message.
func Wrapf(code ErrorCode, cause error, format string, args ...any) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   cause,
	}
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}

	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error cause.
func (e *Error) Unwrap() error {
	return e.Cause
}

// Is reports whether any error in err's chain matches target.
// This is a convenience wrapper around the standard errors.Is function.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target.
// This is a convenience wrapper around the standard errors.As function.
func As(err error, target any) bool {
	return errors.As(err, target)
}

// GetCode extracts the ErrorCode from an error if it's an *Error type.
// Returns ErrCodeUnknown if the error is not an *Error type.
func GetCode(err error) ErrorCode {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}

	return ErrCodeUnknown
}

// HasCode checks if an error has a specific ErrorCode.
func HasCode(err error, code ErrorCode) bool {
	return GetCode(err) == code
}

// InsufficientDataError represents an error when there is not enough data
// for a calculation (e.g., indicator calculations requiring a minimum period).
type InsufficientDataError struct {
	Required int    // Minimum data points required
	Actual   int    // Actual data points available
	Symbol   string // Optional: symbol context
	Message  string // Human-readable message
}

// NewInsufficientDataError creates a new InsufficientDataError.
func NewInsufficientDataError(required, actual int, symbol, message string) *InsufficientDataError {
	return &InsufficientDataError{
		Required: required,
		Actual:   actual,
		Symbol:   symbol,
		Message:  message,
	}
}

// NewInsufficientDataErrorf creates a new InsufficientDataError with a formatted message.
func NewInsufficientDataErrorf(required, actual int, symbol, format string, args ...any) *InsufficientDataError {
	return &InsufficientDataError{
		Required: required,
		Actual:   actual,
		Symbol:   symbol,
		Message:  fmt.Sprintf(format, args...),
	}
}

// Error implements the error interface.
func (e *InsufficientDataError) Error() string {
	return e.Message
}

// IsInsufficientDataError checks if an error is an InsufficientDataError.
// It uses errors.As to check the error chain.
func IsInsufficientDataError(err error) bool {
	var insufficientErr *InsufficientDataError

	return errors.As(err, &insufficientErr)
}
