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
