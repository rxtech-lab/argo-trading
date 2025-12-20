package indicator

import (
	"testing"

	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

// mockIndicator is a simple mock indicator for testing the registry
type mockIndicator struct {
	name types.IndicatorType
}

func newMockIndicator(name types.IndicatorType) *mockIndicator {
	return &mockIndicator{name: name}
}

func (m *mockIndicator) GetSignal(marketData types.MarketData, ctx IndicatorContext) (types.Signal, error) {
	return types.Signal{}, nil
}

func (m *mockIndicator) Name() types.IndicatorType {
	return m.name
}

func (m *mockIndicator) RawValue(params ...any) (float64, error) {
	return 0, nil
}

func (m *mockIndicator) Config(params ...any) error {
	return nil
}

type RegistryTestSuite struct {
	suite.Suite
}

func TestRegistrySuite(t *testing.T) {
	suite.Run(t, new(RegistryTestSuite))
}

func (suite *RegistryTestSuite) TestNewIndicatorRegistry() {
	registry := NewIndicatorRegistry()
	suite.NotNil(registry)
}

func (suite *RegistryTestSuite) TestRegisterIndicator() {
	registry := NewIndicatorRegistry()

	indicator := newMockIndicator(types.IndicatorTypeRSI)
	err := registry.RegisterIndicator(indicator)
	suite.NoError(err)

	// Verify the indicator is registered
	retrieved, err := registry.GetIndicator(types.IndicatorTypeRSI)
	suite.NoError(err)
	suite.Equal(indicator, retrieved)
}

func (suite *RegistryTestSuite) TestRegisterIndicatorDuplicate() {
	registry := NewIndicatorRegistry()

	indicator1 := newMockIndicator(types.IndicatorTypeRSI)
	indicator2 := newMockIndicator(types.IndicatorTypeRSI)

	err := registry.RegisterIndicator(indicator1)
	suite.NoError(err)

	// Trying to register another indicator with the same name should fail
	err = registry.RegisterIndicator(indicator2)
	suite.Error(err)
	suite.Contains(err.Error(), "already registered")
}

func (suite *RegistryTestSuite) TestGetIndicatorNotFound() {
	registry := NewIndicatorRegistry()

	_, err := registry.GetIndicator(types.IndicatorTypeRSI)
	suite.Error(err)
	suite.Contains(err.Error(), "not found")
}

func (suite *RegistryTestSuite) TestListIndicators() {
	registry := NewIndicatorRegistry()

	// Empty registry should return empty list
	indicators := registry.ListIndicators()
	suite.Empty(indicators)

	// Register some indicators
	registry.RegisterIndicator(newMockIndicator(types.IndicatorTypeRSI))
	registry.RegisterIndicator(newMockIndicator(types.IndicatorTypeMACD))
	registry.RegisterIndicator(newMockIndicator(types.IndicatorTypeEMA))

	// Should now have 3 indicators
	indicators = registry.ListIndicators()
	suite.Len(indicators, 3)
	suite.Contains(indicators, types.IndicatorTypeRSI)
	suite.Contains(indicators, types.IndicatorTypeMACD)
	suite.Contains(indicators, types.IndicatorTypeEMA)
}

func (suite *RegistryTestSuite) TestRemoveIndicator() {
	registry := NewIndicatorRegistry()

	// Register an indicator
	indicator := newMockIndicator(types.IndicatorTypeRSI)
	err := registry.RegisterIndicator(indicator)
	suite.NoError(err)

	// Remove it
	err = registry.RemoveIndicator(types.IndicatorTypeRSI)
	suite.NoError(err)

	// Should no longer be found
	_, err = registry.GetIndicator(types.IndicatorTypeRSI)
	suite.Error(err)
}

func (suite *RegistryTestSuite) TestRemoveIndicatorNotFound() {
	registry := NewIndicatorRegistry()

	// Trying to remove a non-existent indicator should fail
	err := registry.RemoveIndicator(types.IndicatorTypeRSI)
	suite.Error(err)
	suite.Contains(err.Error(), "not found")
}

func (suite *RegistryTestSuite) TestConcurrentAccess() {
	registry := NewIndicatorRegistry()

	// Test concurrent registration
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			indicatorType := types.IndicatorType(string(rune('A' + idx)))
			indicator := newMockIndicator(indicatorType)
			registry.RegisterIndicator(indicator)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have 10 indicators
	indicators := registry.ListIndicators()
	suite.Len(indicators, 10)
}
