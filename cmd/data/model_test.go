package main

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestNewModel(t *testing.T) {
	m := NewModel()

	assert.Equal(t, StateProviderSelect, m.state)
	assert.NotNil(t, m.marketData)
	assert.NotNil(t, m.prevPrices)
	assert.Empty(t, m.symbols)
	assert.Empty(t, m.interval)
}

func TestParseSymbols(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single symbol",
			input:    "BTCUSDT",
			expected: []string{"BTCUSDT"},
		},
		{
			name:     "multiple symbols",
			input:    "BTCUSDT,ETHUSDT,BNBUSDT",
			expected: []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"},
		},
		{
			name:     "with spaces",
			input:    "BTCUSDT, ETHUSDT , BNBUSDT",
			expected: []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"},
		},
		{
			name:     "lowercase",
			input:    "btcusdt,ethusdt",
			expected: []string{"BTCUSDT", "ETHUSDT"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "only commas",
			input:    ",,,",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSymbols(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProviderSelection(t *testing.T) {
	m := NewModel()
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	// Wait for provider list to render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Binance"))
	}, teatest.WithDuration(2*time.Second))

	// Send Enter to select provider
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify state changed to symbol input
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Enter Symbols"))
	}, teatest.WithDuration(2*time.Second))

	err := tm.Quit()
	assert.NoError(t, err)
}

func TestSymbolInput(t *testing.T) {
	m := NewModel()
	m.state = StateSymbolInput
	m.symbolInput.Focus()

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	// Wait for symbol input view
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Enter Symbols"))
	}, teatest.WithDuration(2*time.Second))

	// Type symbols
	tm.Type("BTCUSDT,ETHUSDT")

	// Wait for typed text to appear
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("BTCUSDT"))
	}, teatest.WithDuration(2*time.Second))

	// Press Enter to confirm
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify state changed to interval selection
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Select Interval"))
	}, teatest.WithDuration(2*time.Second))

	err := tm.Quit()
	assert.NoError(t, err)
}

func TestIntervalSelection(t *testing.T) {
	m := NewModel()
	m.state = StateIntervalSelect
	m.symbols = []string{"BTCUSDT"}

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	// Wait for interval selection view
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Select Interval")) &&
			bytes.Contains(bts, []byte("1m"))
	}, teatest.WithDuration(2*time.Second))

	err := tm.Quit()
	assert.NoError(t, err)
}

func TestStateTransitions(t *testing.T) {
	t.Run("Esc from symbol input goes back to provider select", func(t *testing.T) {
		m := NewModel()
		m.state = StateSymbolInput

		tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

		// Wait for symbol input view
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Enter Symbols"))
		}, teatest.WithDuration(2*time.Second))

		// Press Esc
		tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

		// Verify we're back at provider selection
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select Data Provider"))
		}, teatest.WithDuration(2*time.Second))

		err := tm.Quit()
		assert.NoError(t, err)
	})

	t.Run("Esc from interval select goes back to symbol input", func(t *testing.T) {
		m := NewModel()
		m.state = StateIntervalSelect
		m.symbols = []string{"BTCUSDT"}

		tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

		// Wait for interval selection view
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select Interval"))
		}, teatest.WithDuration(2*time.Second))

		// Press Esc
		tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

		// Verify we're back at symbol input
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Enter Symbols"))
		}, teatest.WithDuration(2*time.Second))

		err := tm.Quit()
		assert.NoError(t, err)
	})

	t.Run("Esc from data display clears symbols and goes to symbol input", func(t *testing.T) {
		m := NewModel()
		m.state = StateDataDisplay
		m.symbols = []string{"BTCUSDT", "ETHUSDT"}
		m.interval = "1m"
		m.marketData = map[string]types.MarketData{
			"BTCUSDT": {Symbol: "BTCUSDT", Close: 67000.0},
		}
		m.prevPrices = map[string]float64{"BTCUSDT": 66500.0}

		// Simulate pressing Esc
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		updatedModel := newModel.(Model)

		// Verify state changed to symbol input
		assert.Equal(t, StateSymbolInput, updatedModel.state)
		// Verify symbols are cleared
		assert.Nil(t, updatedModel.symbols)
		// Verify interval is cleared
		assert.Empty(t, updatedModel.interval)
		// Verify market data is cleared
		assert.Empty(t, updatedModel.marketData)
		// Verify previous prices are cleared
		assert.Empty(t, updatedModel.prevPrices)
	})
}

func TestDataDisplay(t *testing.T) {
	m := NewModel()
	m.state = StateDataDisplay
	m.symbols = []string{"BTCUSDT"}
	m.interval = "1m"
	m.marketData = map[string]types.MarketData{
		"BTCUSDT": {
			Symbol: "BTCUSDT",
			Open:   67000.0,
			High:   67500.0,
			Low:    66500.0,
			Close:  67234.5,
			Volume: 1234.56,
			Time:   time.Now(),
		},
	}
	m.dataTable = UpdateTableRows(m.dataTable, m.marketData, m.prevPrices)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 30))

	// Wait for data display view with market data
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("BTCUSDT")) &&
			bytes.Contains(bts, []byte("Live Data"))
	}, teatest.WithDuration(2*time.Second))

	err := tm.Quit()
	assert.NoError(t, err)
}

func TestQuitBehavior(t *testing.T) {
	t.Run("ctrl+c quits from any state", func(t *testing.T) {
		m := NewModel()
		tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

		// Send ctrl+c
		tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})

		// Wait for program to finish
		tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
	})

	t.Run("q quits from provider select", func(t *testing.T) {
		m := NewModel()
		tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

		// Wait for view to render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Binance"))
		}, teatest.WithDuration(2*time.Second))

		// Send q
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

		// Wait for program to finish
		tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
	})
}

func TestMarketDataMessage(t *testing.T) {
	m := NewModel()
	m.state = StateDataDisplay
	m.symbols = []string{"BTCUSDT"}
	m.interval = "1m"

	// Simulate receiving market data
	msg := MarketDataMsg{
		Data: types.MarketData{
			Symbol: "BTCUSDT",
			Open:   67000.0,
			High:   67500.0,
			Low:    66500.0,
			Close:  67234.5,
			Volume: 1234.56,
			Time:   time.Now(),
		},
	}

	newModel, _ := m.Update(msg)
	updatedModel := newModel.(Model)

	assert.Contains(t, updatedModel.marketData, "BTCUSDT")
	assert.Equal(t, 67234.5, updatedModel.marketData["BTCUSDT"].Close)
}

func TestPriceColorFormatting(t *testing.T) {
	tests := []struct {
		name     string
		current  float64
		previous float64
		contains string
	}{
		{
			name:     "price up shows green arrow",
			current:  100.0,
			previous: 90.0,
			contains: "▲",
		},
		{
			name:     "price down shows red arrow",
			current:  90.0,
			previous: 100.0,
			contains: "▼",
		},
		{
			name:     "same price no arrow",
			current:  100.0,
			previous: 100.0,
			contains: "100.0000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatPriceWithColor(tt.current, tt.previous)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestWindowResize(t *testing.T) {
	m := NewModel()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	newModel, _ := m.Update(msg)
	updatedModel := newModel.(Model)

	assert.Equal(t, 120, updatedModel.width)
	assert.Equal(t, 40, updatedModel.height)
}
