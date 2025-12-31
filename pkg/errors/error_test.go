package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ErrorTestSuite struct {
	suite.Suite
}

func TestErrorSuite(t *testing.T) {
	suite.Run(t, new(ErrorTestSuite))
}

func (suite *ErrorTestSuite) TestNewError() {
	err := New(ErrCodeInvalidParameter, "invalid parameter")
	suite.NotNil(err)
	suite.Equal(ErrCodeInvalidParameter, err.Code)
	suite.Equal("invalid parameter", err.Message)
	suite.Nil(err.Cause)
}

func (suite *ErrorTestSuite) TestNewfError() {
	err := Newf(ErrCodeInvalidParameter, "invalid parameter: %s", "test")
	suite.NotNil(err)
	suite.Equal(ErrCodeInvalidParameter, err.Code)
	suite.Equal("invalid parameter: test", err.Message)
	suite.Nil(err.Cause)
}

func (suite *ErrorTestSuite) TestWrapError() {
	cause := errors.New("underlying error")
	err := Wrap(ErrCodeDataNotFound, "data not found", cause)
	suite.NotNil(err)
	suite.Equal(ErrCodeDataNotFound, err.Code)
	suite.Equal("data not found", err.Message)
	suite.Equal(cause, err.Cause)
}

func (suite *ErrorTestSuite) TestWrapfError() {
	cause := errors.New("underlying error")
	err := Wrapf(ErrCodeDataNotFound, cause, "data not found for symbol: %s", "AAPL")
	suite.NotNil(err)
	suite.Equal(ErrCodeDataNotFound, err.Code)
	suite.Equal("data not found for symbol: AAPL", err.Message)
	suite.Equal(cause, err.Cause)
}

func (suite *ErrorTestSuite) TestErrorString() {
	err := New(ErrCodeInvalidParameter, "invalid parameter")
	suite.Equal("[100] invalid parameter", err.Error())
}

func (suite *ErrorTestSuite) TestErrorStringWithCause() {
	cause := errors.New("underlying error")
	err := Wrap(ErrCodeDataNotFound, "data not found", cause)
	suite.Equal("[200] data not found: underlying error", err.Error())
}

func (suite *ErrorTestSuite) TestUnwrap() {
	cause := errors.New("underlying error")
	err := Wrap(ErrCodeDataNotFound, "data not found", cause)
	suite.Equal(cause, err.Unwrap())
}

func (suite *ErrorTestSuite) TestUnwrapNil() {
	err := New(ErrCodeInvalidParameter, "invalid parameter")
	suite.Nil(err.Unwrap())
}

func (suite *ErrorTestSuite) TestGetCode() {
	err := New(ErrCodeInvalidParameter, "invalid parameter")
	suite.Equal(ErrCodeInvalidParameter, GetCode(err))
}

func (suite *ErrorTestSuite) TestGetCodeFromWrapped() {
	cause := New(ErrCodeDataNotFound, "data not found")
	err := Wrap(ErrCodeIndicatorNotFound, "indicator not found", cause)
	// GetCode should return the outermost error's code
	suite.Equal(ErrCodeIndicatorNotFound, GetCode(err))
}

func (suite *ErrorTestSuite) TestGetCodeFromNonArgoError() {
	err := errors.New("standard error")
	suite.Equal(ErrCodeUnknown, GetCode(err))
}

func (suite *ErrorTestSuite) TestHasCode() {
	err := New(ErrCodeInvalidParameter, "invalid parameter")
	suite.True(HasCode(err, ErrCodeInvalidParameter))
	suite.False(HasCode(err, ErrCodeDataNotFound))
}

func (suite *ErrorTestSuite) TestIsError() {
	cause := errors.New("underlying error")
	err := Wrap(ErrCodeDataNotFound, "data not found", cause)
	suite.True(Is(err, cause))
}

func (suite *ErrorTestSuite) TestAsError() {
	err := New(ErrCodeInvalidParameter, "invalid parameter")
	var argoErr *Error
	suite.True(As(err, &argoErr))
	suite.Equal(ErrCodeInvalidParameter, argoErr.Code)
}

func (suite *ErrorTestSuite) TestErrorCodeValues() {
	// Verify some key error codes have expected values
	suite.Equal(ErrorCode(1), ErrCodeUnknown)
	suite.Equal(ErrorCode(100), ErrCodeInvalidParameter)
	suite.Equal(ErrorCode(200), ErrCodeDataNotFound)
	suite.Equal(ErrorCode(300), ErrCodeIndicatorNotFound)
	suite.Equal(ErrorCode(400), ErrCodeStrategyNotLoaded)
	suite.Equal(ErrorCode(500), ErrCodeOrderFailed)
	suite.Equal(ErrorCode(600), ErrCodeBacktestStateNil)
	suite.Equal(ErrorCode(700), ErrCodeMarketDataFetchFailed)
	suite.Equal(ErrorCode(800), ErrCodeCallbackFailed)
}
