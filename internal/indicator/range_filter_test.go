package indicator

import (
	"testing"

	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/cache"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

type RangeFilterTestSuite struct {
	suite.Suite
}

func TestRangeFilterSuite(t *testing.T) {
	suite.Run(t, new(RangeFilterTestSuite))
}

func (suite *RangeFilterTestSuite) TestNewRangeFilter() {
	rf := NewRangeFilter()
	suite.NotNil(rf)

	// Cast to *RangeFilter to check default values
	rfImpl := rf.(*RangeFilter)
	suite.Equal(100, rfImpl.period)
	suite.Equal(3.0, rfImpl.multiplier)
}

func (suite *RangeFilterTestSuite) TestName() {
	rf := NewRangeFilter()
	suite.Equal(types.IndicatorTypeRangeFilter, rf.Name())
}

func (suite *RangeFilterTestSuite) TestConfigValid() {
	rf := NewRangeFilter()
	rfImpl := rf.(*RangeFilter)

	err := rf.Config(50, 2.5)
	suite.NoError(err)
	suite.Equal(50, rfImpl.period)
	suite.Equal(2.5, rfImpl.multiplier)
}

func (suite *RangeFilterTestSuite) TestConfigInvalidParamCount() {
	rf := NewRangeFilter()

	// Too few params
	err := rf.Config(50)
	suite.Error(err)
	suite.Contains(err.Error(), "expects 2 parameters")

	// Too many params
	err = rf.Config(50, 2.5, "extra")
	suite.Error(err)
}

func (suite *RangeFilterTestSuite) TestConfigInvalidPeriodType() {
	rf := NewRangeFilter()
	err := rf.Config("invalid", 2.5)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for period")
}

func (suite *RangeFilterTestSuite) TestConfigInvalidPeriodValue() {
	rf := NewRangeFilter()

	err := rf.Config(0, 2.5)
	suite.Error(err)
	suite.Contains(err.Error(), "period must be a positive integer")

	err = rf.Config(-10, 2.5)
	suite.Error(err)
}

func (suite *RangeFilterTestSuite) TestConfigInvalidMultiplierType() {
	rf := NewRangeFilter()
	err := rf.Config(50, "invalid")
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for multiplier")
}

func (suite *RangeFilterTestSuite) TestConfigInvalidMultiplierValue() {
	rf := NewRangeFilter()

	err := rf.Config(50, 0.0)
	suite.Error(err)
	suite.Contains(err.Error(), "multiplier must be a positive number")

	err = rf.Config(50, -1.5)
	suite.Error(err)
}

func (suite *RangeFilterTestSuite) TestRawValueInvalidParams() {
	rf := NewRangeFilter()

	// Too few params
	_, err := rf.RawValue()
	suite.Error(err)
	suite.Contains(err.Error(), "requires at least 3 parameters")

	_, err = rf.RawValue("symbol")
	suite.Error(err)

	_, err = rf.RawValue("symbol", "not-a-time")
	suite.Error(err)

	// Invalid first param type
	_, err = rf.RawValue(123, nil, nil)
	suite.Error(err)
	suite.Contains(err.Error(), "first parameter must be of type string")

	// Invalid second param type
	_, err = rf.RawValue("symbol", "not-a-time", nil)
	suite.Error(err)
	suite.Contains(err.Error(), "second parameter must be of type time.Time")
}

func (suite *RangeFilterTestSuite) TestCalculateFilterValue() {
	rf := &RangeFilter{
		period:     100,
		multiplier: 3.0,
	}

	// Test when src > prevFilt and (src - smrng) < prevFilt
	// Should return prevFilt
	result := rf.calculateFilterValue(105.0, 100.0, 10.0)
	suite.Equal(100.0, result)

	// Test when src > prevFilt and (src - smrng) >= prevFilt
	// Should return src - smrng
	result = rf.calculateFilterValue(120.0, 100.0, 10.0)
	suite.Equal(110.0, result)

	// Test when src <= prevFilt and (src + smrng) > prevFilt
	// Should return prevFilt
	result = rf.calculateFilterValue(95.0, 100.0, 10.0)
	suite.Equal(100.0, result)

	// Test when src <= prevFilt and (src + smrng) <= prevFilt
	// Should return src + smrng
	result = rf.calculateFilterValue(80.0, 100.0, 10.0)
	suite.Equal(90.0, result)
}

func (suite *RangeFilterTestSuite) TestCalculateTrend() {
	rf := &RangeFilter{}

	// Test upward trend
	value := cache.RangeFilterState{Upward: 5, Downward: 0}
	upward, downward := rf.calculateTrend(110.0, 100.0, value)
	suite.Equal(6.0, upward)
	suite.Equal(0.0, downward)

	// Test downward trend
	value = cache.RangeFilterState{Upward: 0, Downward: 3}
	upward, downward = rf.calculateTrend(90.0, 100.0, value)
	suite.Equal(0.0, upward)
	suite.Equal(4.0, downward)

	// Test no change
	value = cache.RangeFilterState{Upward: 2, Downward: 0}
	upward, downward = rf.calculateTrend(100.0, 100.0, value)
	suite.Equal(2.0, upward)
	suite.Equal(0.0, downward)
}

func (suite *RangeFilterTestSuite) TestHandleInitialization() {
	rf := &RangeFilter{}

	// Test with valid source value
	value := cache.RangeFilterState{}
	result := RangeFilterData{}

	filt, smrng, upward, downward, resultOut := rf.handleInitialization(100.0, value, result)
	suite.Equal(100.0, filt)
	suite.Equal(0.0, smrng)
	suite.Equal(0.0, upward)
	suite.Equal(0.0, downward)
	suite.True(resultOut.initialized)
}

func (suite *RangeFilterTestSuite) TestRangeFilterDataStruct() {
	data := RangeFilterData{
		filt:        100.0,
		smrng:       5.0,
		prevFilt:    98.0,
		upward:      3.0,
		downward:    0.0,
		initialized: true,
	}

	suite.Equal(100.0, data.filt)
	suite.Equal(5.0, data.smrng)
	suite.Equal(98.0, data.prevFilt)
	suite.Equal(3.0, data.upward)
	suite.Equal(0.0, data.downward)
	suite.True(data.initialized)
}
