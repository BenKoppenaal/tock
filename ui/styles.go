package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	colorHighlight = lipgloss.AdaptiveColor{Light: "#0369A1", Dark: "#0891B2"}
	colorSubtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	colorActive    = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	colorMuted     = lipgloss.Color("#666666")
	colorError     = lipgloss.Color("#ff5555")
	colorDim       = lipgloss.AdaptiveColor{Light: "#C0BBBC", Dark: "#555555"}

	// Text styles
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Padding(0, 1)
	helpStyle    = lipgloss.NewStyle().Foreground(colorMuted)
	labelStyle   = lipgloss.NewStyle().Foreground(colorMuted)
	errorStyle   = lipgloss.NewStyle().Foreground(colorError)
	confirmStyle = lipgloss.NewStyle().Foreground(colorError).Bold(true)

	// Tab bar
	tabActiveStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Margin(1, 1, 1, 2).
			Bold(true).
			Background(colorHighlight).
			Foreground(lipgloss.Color("#ffffff"))
	tabInactiveStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Margin(1, 1, 1, 2).
				Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"})
	tabTitleStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Margin(1, 2, 1, 1).
			Bold(true).
			Background(colorHighlight).
			Foreground(lipgloss.Color("#ffffff"))
	activeInfoStyle = lipgloss.NewStyle().
			Margin(1, 1, 1, 0).
			Foreground(colorActive)

	// Forms
	formBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorHighlight).
			Padding(1, 3).
			Width(50)
	formOuterStyle = lipgloss.NewStyle().Padding(4, 8)

	// Day view
	timeGutterStyle = lipgloss.NewStyle().Foreground(colorMuted).Width(6)
	blockColors     = []lipgloss.Color{
		"#0891B2",
		"#56A0D3",
		"#43BF6D",
		"#F4A556",
		"#D356A0",
		"#56D3C0",
	}
)
