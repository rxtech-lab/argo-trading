package swiftargo_test

import (
	"sync"
	"testing"

	swiftargo "github.com/rxtech-lab/argo-trading/pkg/swift-argo"
	"github.com/stretchr/testify/assert"
)

func TestGetBacktestEngineConfigSchema(t *testing.T) {
	schema := swiftargo.GetBacktestEngineConfigSchema()
	assert.NotEmpty(t, schema)
}

func TestGetBacktestEngineVersion(t *testing.T) {
	version := swiftargo.GetBacktestEngineVersion()
	assert.NotEmpty(t, version)
}

// mockArgoHelper implements ArgoHelper for testing.
type mockArgoHelper struct {
	mu                  sync.Mutex
	backtestStartCalls  []struct{ totalStrategies, totalConfigs, totalDataFiles int }
	backtestEndCalls    []error
	strategyStartCalls  []struct{ strategyIndex int; strategyName string; totalStrategies int }
	strategyEndCalls    []struct{ strategyIndex int; strategyName string }
	runStartCalls       []struct{ runID, configName, dataFilePath string; configIndex, dataFileIndex, totalDataPoints int }
	runEndCalls         []struct{ configIndex, dataFileIndex int; configName, dataFilePath, resultFolderPath string }
	processDataCalls    []struct{ current, total int }
}

func (m *mockArgoHelper) OnBacktestStart(totalStrategies int, totalConfigs int, totalDataFiles int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.backtestStartCalls = append(m.backtestStartCalls, struct{ totalStrategies, totalConfigs, totalDataFiles int }{totalStrategies, totalConfigs, totalDataFiles})
	return nil
}

func (m *mockArgoHelper) OnBacktestEnd(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.backtestEndCalls = append(m.backtestEndCalls, err)
}

func (m *mockArgoHelper) OnStrategyStart(strategyIndex int, strategyName string, totalStrategies int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.strategyStartCalls = append(m.strategyStartCalls, struct{ strategyIndex int; strategyName string; totalStrategies int }{strategyIndex, strategyName, totalStrategies})
	return nil
}

func (m *mockArgoHelper) OnStrategyEnd(strategyIndex int, strategyName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.strategyEndCalls = append(m.strategyEndCalls, struct{ strategyIndex int; strategyName string }{strategyIndex, strategyName})
}

func (m *mockArgoHelper) OnRunStart(runID string, configIndex int, configName string, dataFileIndex int, dataFilePath string, totalDataPoints int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runStartCalls = append(m.runStartCalls, struct{ runID, configName, dataFilePath string; configIndex, dataFileIndex, totalDataPoints int }{runID, configName, dataFilePath, configIndex, dataFileIndex, totalDataPoints})
	return nil
}

func (m *mockArgoHelper) OnRunEnd(configIndex int, configName string, dataFileIndex int, dataFilePath string, resultFolderPath string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runEndCalls = append(m.runEndCalls, struct{ configIndex, dataFileIndex int; configName, dataFilePath, resultFolderPath string }{configIndex, dataFileIndex, configName, dataFilePath, resultFolderPath})
}

func (m *mockArgoHelper) OnProcessData(current int, total int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processDataCalls = append(m.processDataCalls, struct{ current, total int }{current, total})
	return nil
}

func TestNewArgo(t *testing.T) {
	helper := &mockArgoHelper{}
	argo, err := swiftargo.NewArgo(helper)
	assert.NoError(t, err)
	assert.NotNil(t, argo)
}

func TestArgo_Cancel_NoRunInProgress(t *testing.T) {
	helper := &mockArgoHelper{}
	argo, err := swiftargo.NewArgo(helper)
	assert.NoError(t, err)

	// Cancel with no run in progress should return false
	cancelled := argo.Cancel()
	assert.False(t, cancelled)
}

func TestArgo_Cancel_ThreadSafety(t *testing.T) {
	helper := &mockArgoHelper{}
	argo, err := swiftargo.NewArgo(helper)
	assert.NoError(t, err)

	// Simulate concurrent cancel calls - should not panic
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			argo.Cancel()
		}()
	}

	wg.Wait()
	// If we get here without panic, the test passes
}

func TestArgo_SetDataPath_ValidPath(t *testing.T) {
	helper := &mockArgoHelper{}
	argo, err := swiftargo.NewArgo(helper)
	assert.NoError(t, err)

	// Setting a valid path pattern should work
	err = argo.SetDataPath("/tmp/*.parquet")
	assert.NoError(t, err)
}

// stringCollectionMock implements StringCollection for testing
type stringCollectionMock struct {
	items []string
}

func (s *stringCollectionMock) Add(str string) swiftargo.StringCollection {
	s.items = append(s.items, str)
	return s
}

func (s *stringCollectionMock) Get(i int) string {
	if i < 0 || i >= len(s.items) {
		return ""
	}
	return s.items[i]
}

func (s *stringCollectionMock) Size() int {
	return len(s.items)
}

func TestArgo_SetConfigContent_EmptyCollection(t *testing.T) {
	helper := &mockArgoHelper{}
	argo, err := swiftargo.NewArgo(helper)
	assert.NoError(t, err)

	// Empty collection should work
	configs := &stringCollectionMock{items: []string{}}
	err = argo.SetConfigContent(configs)
	assert.NoError(t, err)
}

func TestArgo_SetConfigContent_WithConfigs(t *testing.T) {
	helper := &mockArgoHelper{}
	argo, err := swiftargo.NewArgo(helper)
	assert.NoError(t, err)

	// Collection with configs should work
	configs := &stringCollectionMock{items: []string{
		`{"name": "config1"}`,
		`{"name": "config2"}`,
	}}
	err = argo.SetConfigContent(configs)
	assert.NoError(t, err)
}
