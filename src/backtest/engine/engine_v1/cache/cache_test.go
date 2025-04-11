package cache

import (
	"testing"

	"github.com/moznion/go-optional"
	"github.com/stretchr/testify/suite"
)

// CacheTestSuite is a test suite for CacheV1
type CacheTestSuite struct {
	suite.Suite
	cache *CacheV1
}

// SetupTest runs before each test
func (suite *CacheTestSuite) SetupTest() {
	suite.cache = NewCacheV1().(*CacheV1)
}

// TestCacheSuite runs the test suite
func TestCacheSuite(t *testing.T) {
	suite.Run(t, new(CacheTestSuite))
}

// TestNewCacheV1 tests the creation of a new CacheV1 instance
func (suite *CacheTestSuite) TestNewCacheV1() {
	cache := NewCacheV1()
	suite.Require().NotNil(cache)
	suite.IsType(&CacheV1{}, cache)
}

// TestReset tests the Reset functionality
func (suite *CacheTestSuite) TestReset() {
	// Set some initial state
	initialState := RangeFilterState{
		Initialized: true,
		PrevFilt:    100.0,
		PrevSource:  100.0,
		Upward:      1.0,
		Downward:    1.0,
		Symbol:      "BTCUSDT",
	}
	suite.cache.RangeFilterState = optional.Some(initialState)
	suite.cache.otherData = map[string]any{
		"test": "value",
	}

	// Reset the cache
	suite.cache.Reset()

	// Verify the cache is reset
	suite.True(suite.cache.RangeFilterState.IsNone())
	suite.Empty(suite.cache.otherData)
}

// TestSetAndGet tests the Set and Get functionality
func (suite *CacheTestSuite) TestSetAndGet() {
	// Test setting and getting a value
	key := "testKey"
	value := "testValue"
	suite.cache.Set(key, value)

	// Verify the value can be retrieved
	retrievedValue, exists := suite.cache.Get(key)
	suite.True(exists)
	suite.Equal(value, retrievedValue)

	// Test getting a non-existent key
	_, exists = suite.cache.Get("nonExistentKey")
	suite.False(exists)
}

// TestRangeFilterState tests the RangeFilterState functionality
func (suite *CacheTestSuite) TestRangeFilterState() {
	// Test initial state is None
	suite.True(suite.cache.RangeFilterState.IsNone())

	// Test setting RangeFilterState
	state := RangeFilterState{
		Initialized: true,
		PrevFilt:    100.0,
		PrevSource:  100.0,
		Upward:      1.0,
		Downward:    1.0,
		Symbol:      "BTCUSDT",
	}
	suite.cache.RangeFilterState = optional.Some(state)

	// Verify the state is set correctly
	suite.True(suite.cache.RangeFilterState.IsSome())
	retrievedState := suite.cache.RangeFilterState.Unwrap()
	suite.Equal(state, retrievedState)
}
