package go_runtime

import (
	"testing"

	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestNewGoRuntime(t *testing.T) {
	// Test creating a new GoRuntime
	runtime := NewGoRuntime(nil)
	assert.NotNil(t, runtime)

	// Verify it's a GoRuntime
	goRuntime, ok := runtime.(*GoRuntime)
	assert.True(t, ok)
	assert.Nil(t, goRuntime.strategy)
}

func TestGoRuntime_GetDescription_Panics(t *testing.T) {
	runtime := NewGoRuntime(nil)
	goRuntime := runtime.(*GoRuntime)

	assert.Panics(t, func() {
		goRuntime.GetDescription()
	})
}

func TestGoRuntime_Initialize_Panics(t *testing.T) {
	runtime := NewGoRuntime(nil)
	goRuntime := runtime.(*GoRuntime)

	assert.Panics(t, func() {
		goRuntime.Initialize("{}")
	})
}

func TestGoRuntime_Name_Panics(t *testing.T) {
	runtime := NewGoRuntime(nil)
	goRuntime := runtime.(*GoRuntime)

	assert.Panics(t, func() {
		goRuntime.Name()
	})
}

func TestGoRuntime_ProcessData_Panics(t *testing.T) {
	runtime := NewGoRuntime(nil)
	goRuntime := runtime.(*GoRuntime)

	assert.Panics(t, func() {
		goRuntime.ProcessData(types.MarketData{})
	})
}

func TestGoRuntime_InitializeApi_Panics(t *testing.T) {
	runtime := NewGoRuntime(nil)
	goRuntime := runtime.(*GoRuntime)

	assert.Panics(t, func() {
		goRuntime.InitializeApi(nil)
	})
}

func TestGoRuntime_GetConfigSchema_Panics(t *testing.T) {
	runtime := NewGoRuntime(nil)
	goRuntime := runtime.(*GoRuntime)

	assert.Panics(t, func() {
		goRuntime.GetConfigSchema()
	})
}

func TestGoRuntime_GetRuntimeEngineVersion_Panics(t *testing.T) {
	runtime := NewGoRuntime(nil)
	goRuntime := runtime.(*GoRuntime)

	assert.Panics(t, func() {
		goRuntime.GetRuntimeEngineVersion()
	})
}
