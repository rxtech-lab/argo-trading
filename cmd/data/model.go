package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/marketdata/provider"
)

// Application states.
const (
	StateProviderSelect = iota
	StateSymbolInput
	StateIntervalSelect
	StateDataDisplay
)

// Model is the main Bubble Tea model for the real-time data CLI.
type Model struct {
	state        int
	providerList list.Model
	symbolInput  textinput.Model
	intervalList list.Model
	dataTable    table.Model
	marketData   map[string]types.MarketData
	prevPrices   map[string]float64
	symbols      []string
	interval     string
	err          error
	width        int
	height       int

	// Streaming control
	streamCtx    context.Context
	streamCancel context.CancelFunc
	program      *tea.Program
}

// NewModel creates a new Model with initial state.
func NewModel() Model {
	return Model{
		state:        StateProviderSelect,
		providerList: NewProviderList(),
		symbolInput:  NewSymbolInput(),
		intervalList: NewIntervalList(),
		dataTable:    NewDataTable(),
		marketData:   make(map[string]types.MarketData),
		prevPrices:   make(map[string]float64),
	}
}

// SetProgram sets the tea.Program reference for sending messages from goroutines.
func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.streamCancel != nil {
				m.streamCancel()
			}
			return m, tea.Quit
		case "q":
			// Only quit on 'q' if not in text input mode
			if m.state != StateSymbolInput {
				if m.streamCancel != nil {
					m.streamCancel()
				}
				return m, tea.Quit
			}
		case "esc":
			return m.handleEsc()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.providerList.SetSize(msg.Width, msg.Height-4)
		m.intervalList.SetSize(msg.Width, msg.Height-4)
		m.dataTable.SetWidth(msg.Width)
		m.dataTable.SetHeight(msg.Height - 6)
		return m, nil

	case MarketDataMsg:
		// Store previous price for color coding
		if existing, ok := m.marketData[msg.Data.Symbol]; ok {
			m.prevPrices[msg.Data.Symbol] = existing.Close
		}
		m.marketData[msg.Data.Symbol] = msg.Data
		m.dataTable = UpdateTableRows(m.dataTable, m.marketData, m.prevPrices)
		return m, nil

	case StreamErrorMsg:
		m.err = msg.Err
		return m, nil

	case StreamStartedMsg:
		m.state = StateDataDisplay
		return m, nil
	}

	// Delegate to state-specific update
	switch m.state {
	case StateProviderSelect:
		return m.updateProviderSelect(msg)
	case StateSymbolInput:
		return m.updateSymbolInput(msg)
	case StateIntervalSelect:
		return m.updateIntervalSelect(msg)
	case StateDataDisplay:
		return m.updateDataDisplay(msg)
	}

	return m, nil
}

func (m Model) handleEsc() (tea.Model, tea.Cmd) {
	switch m.state {
	case StateSymbolInput:
		m.state = StateProviderSelect
	case StateIntervalSelect:
		m.state = StateSymbolInput
		m.symbolInput.Focus()
	case StateDataDisplay:
		// Stop streaming and clear watched symbols
		if m.streamCancel != nil {
			m.streamCancel()
			m.streamCancel = nil
		}
		m.marketData = make(map[string]types.MarketData)
		m.prevPrices = make(map[string]float64)
		m.symbols = nil
		m.interval = ""
		m.err = nil
		m.symbolInput.Reset()
		m.symbolInput.Focus()
		m.state = StateSymbolInput
		return m, textinput.Blink
	}
	return m, nil
}

func (m Model) updateProviderSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if item, ok := m.providerList.SelectedItem().(listItem); ok {
				_ = item.name // Provider selected (currently only Binance)
				m.state = StateSymbolInput
				m.symbolInput.Focus()
				return m, textinput.Blink
			}
		}
	}

	var cmd tea.Cmd
	m.providerList, cmd = m.providerList.Update(msg)
	return m, cmd
}

func (m Model) updateSymbolInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			symbols := ParseSymbols(m.symbolInput.Value())
			if len(symbols) > 0 {
				m.symbols = symbols
				m.state = StateIntervalSelect
				m.symbolInput.Blur()
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.symbolInput, cmd = m.symbolInput.Update(msg)
	return m, cmd
}

func (m Model) updateIntervalSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if item, ok := m.intervalList.SelectedItem().(listItem); ok {
				m.interval = item.name
				m.state = StateDataDisplay
				return m, m.startStreaming()
			}
		}
	}

	var cmd tea.Cmd
	m.intervalList, cmd = m.intervalList.Update(msg)
	return m, cmd
}

func (m Model) updateDataDisplay(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.dataTable, cmd = m.dataTable.Update(msg)
	return m, cmd
}

// startStreaming returns a command that starts the market data stream.
func (m *Model) startStreaming() tea.Cmd {
	return func() tea.Msg {
		if m.program == nil {
			return StreamErrorMsg{Err: fmt.Errorf("program not set")}
		}

		ctx, cancel := context.WithCancel(context.Background())
		m.streamCtx = ctx
		m.streamCancel = cancel

		go streamMarketData(m.program, ctx, m.symbols, m.interval)

		return StreamStartedMsg{}
	}
}

// streamMarketData streams market data from Binance and sends messages to the program.
func streamMarketData(p *tea.Program, ctx context.Context, symbols []string, interval string) {
	client, err := provider.NewMarketDataProvider(provider.ProviderBinance, nil)
	if err != nil {
		p.Send(StreamErrorMsg{Err: err})
		return
	}

	for data, err := range client.Stream(ctx, symbols, interval) {
		if err != nil {
			p.Send(StreamErrorMsg{Err: err})
			continue
		}
		p.Send(MarketDataMsg{Data: data})
	}
}

// View implements tea.Model.
func (m Model) View() string {
	var s strings.Builder

	switch m.state {
	case StateProviderSelect:
		s.WriteString(TitleStyle.Render("Argo Trading - Real-Time Data"))
		s.WriteString("\n\n")
		s.WriteString(m.providerList.View())
		s.WriteString("\n")
		s.WriteString(HelpStyle.Render("Press Enter to select, q to quit"))

	case StateSymbolInput:
		s.WriteString(TitleStyle.Render("Enter Symbols"))
		s.WriteString("\n\n")
		s.WriteString("Enter comma-separated symbols (e.g., BTCUSDT,ETHUSDT):\n\n")
		s.WriteString(m.symbolInput.View())
		s.WriteString("\n\n")
		s.WriteString(HelpStyle.Render("Press Enter to confirm, Esc to go back"))

	case StateIntervalSelect:
		s.WriteString(TitleStyle.Render("Select Interval"))
		s.WriteString("\n\n")
		s.WriteString(m.intervalList.View())
		s.WriteString("\n")
		s.WriteString(HelpStyle.Render("Press Enter to select, Esc to go back"))

	case StateDataDisplay:
		s.WriteString(TitleStyle.Render(fmt.Sprintf("Live Data - Binance (%s)", m.interval)))
		s.WriteString("\n\n")

		if m.err != nil {
			s.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
			s.WriteString("\n\n")
		}

		if len(m.marketData) == 0 {
			s.WriteString("Waiting for data...\n")
		} else {
			s.WriteString(m.dataTable.View())
		}

		s.WriteString("\n")
		s.WriteString(HelpStyle.Render(fmt.Sprintf("q: quit | Esc: back | Streaming: %s", strings.Join(m.symbols, ", "))))
	}

	return s.String()
}
