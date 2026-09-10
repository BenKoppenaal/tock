package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorHighlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	colorSubtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	colorActive    = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	colorMuted     = lipgloss.Color("#666666")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorHighlight).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}).
			Background(colorSubtle).
			Padding(0, 1)

	activeEntryStyle = lipgloss.NewStyle().
				Foreground(colorActive).
				Bold(true)

	helpStyle = lipgloss.NewStyle().Foreground(colorMuted)

	timeGutterStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(6)

	blockColors = []lipgloss.Color{
		"#7D56F4",
		"#56A0D3",
		"#43BF6D",
		"#F4A556",
		"#D356A0",
		"#56D3C0",
	}
)
