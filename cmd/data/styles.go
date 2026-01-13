package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Style definitions.
var (
	// TitleStyle for headers.
	TitleStyle = lipgloss.NewStyle().Bold(true)

	// HelpStyle for help text.
	HelpStyle = lipgloss.NewStyle().Faint(true)

	// ErrorStyle for error messages.
	ErrorStyle = lipgloss.NewStyle().Bold(true)
)

// FormatPriceWithColor formats a price with indicator based on comparison with previous price.
func FormatPriceWithColor(current, previous float64) string {
	priceStr := fmt.Sprintf("%.4f", current)

	if previous == 0 {
		return priceStr
	}

	if current > previous {
		return priceStr + " ▲"
	} else if current < previous {
		return priceStr + " ▼"
	}

	return priceStr
}
