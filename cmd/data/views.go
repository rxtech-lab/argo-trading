package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/rxtech-lab/argo-trading/internal/types"
)

// listItem implements list.Item interface for provider and interval lists.
type listItem struct {
	name        string
	description string
}

func (i listItem) Title() string       { return i.name }
func (i listItem) Description() string { return i.description }
func (i listItem) FilterValue() string { return i.name }

// NewProviderList creates a new list for provider selection.
func NewProviderList() list.Model {
	items := []list.Item{
		listItem{name: "Binance", description: "Binance cryptocurrency exchange (WebSocket)"},
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true

	l := list.New(items, delegate, 0, 0)
	l.Title = "Select Data Provider"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	return l
}

// NewIntervalList creates a new list for interval selection.
func NewIntervalList() list.Model {
	items := []list.Item{
		listItem{name: "1s", description: "1 second candles"},
		listItem{name: "1m", description: "1 minute candles"},
		listItem{name: "3m", description: "3 minute candles"},
		listItem{name: "5m", description: "5 minute candles"},
		listItem{name: "15m", description: "15 minute candles"},
		listItem{name: "30m", description: "30 minute candles"},
		listItem{name: "1h", description: "1 hour candles"},
		listItem{name: "2h", description: "2 hour candles"},
		listItem{name: "4h", description: "4 hour candles"},
		listItem{name: "6h", description: "6 hour candles"},
		listItem{name: "8h", description: "8 hour candles"},
		listItem{name: "12h", description: "12 hour candles"},
		listItem{name: "1d", description: "1 day candles"},
		listItem{name: "3d", description: "3 day candles"},
		listItem{name: "1w", description: "1 week candles"},
		listItem{name: "1M", description: "1 month candles"},
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true

	l := list.New(items, delegate, 0, 0)
	l.Title = "Select Interval"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	return l
}

// NewApiKeyInput creates a new text input for API key entry.
func NewApiKeyInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "your-api-key"
	ti.CharLimit = 128
	ti.Width = 70
	ti.Prompt = "> "

	return ti
}

// NewSecretKeyInput creates a new text input for secret key entry.
func NewSecretKeyInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "your-secret-key"
	ti.EchoMode = textinput.EchoPassword
	ti.CharLimit = 128
	ti.Width = 70
	ti.Prompt = "> "

	return ti
}

// NewSymbolInput creates a new text input for symbol entry.
func NewSymbolInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "BTCUSDT,ETHUSDT,BNBUSDT"
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 50
	ti.Prompt = "> "

	return ti
}

// ParseSymbols parses comma-separated symbols into a slice.
func ParseSymbols(input string) []string {
	parts := strings.Split(input, ",")
	symbols := make([]string, 0, len(parts))

	for _, p := range parts {
		s := strings.TrimSpace(strings.ToUpper(p))
		if s != "" {
			symbols = append(symbols, s)
		}
	}

	return symbols
}

// NewDataTable creates a new table for displaying market data.
func NewDataTable() table.Model {
	columns := []table.Column{
		{Title: "Symbol", Width: 12},
		{Title: "Price", Width: 18},
		{Title: "Open", Width: 14},
		{Title: "High", Width: 14},
		{Title: "Low", Width: 14},
		{Title: "Volume", Width: 16},
		{Title: "Time", Width: 10},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	t.SetStyles(s)

	return t
}

// UpdateTableRows updates the table with the latest market data.
func UpdateTableRows(t table.Model, data map[string]types.MarketData, prevPrices map[string]float64) table.Model {
	// Sort symbols for consistent ordering
	symbols := make([]string, 0, len(data))
	for symbol := range data {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)

	rows := make([]table.Row, 0, len(data))

	for _, symbol := range symbols {
		md := data[symbol]
		prevPrice := prevPrices[symbol]

		rows = append(rows, table.Row{
			symbol,
			FormatPriceWithColor(md.Close, prevPrice),
			fmt.Sprintf("%.4f", md.Open),
			fmt.Sprintf("%.4f", md.High),
			fmt.Sprintf("%.4f", md.Low),
			fmt.Sprintf("%.2f", md.Volume),
			md.Time.Format("15:04:05"),
		})
	}

	t.SetRows(rows)

	return t
}
