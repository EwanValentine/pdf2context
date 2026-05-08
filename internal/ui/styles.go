package ui

import "github.com/charmbracelet/lipgloss"

var (
	// headerStyle renders the application title in bold indigo/purple.
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99"))

	// boxStyle renders a panel with a rounded border.
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			PaddingLeft(1).
			PaddingRight(1)

	// successStyle is used for successful completions.
	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("76"))

	// errorStyle is used for errors.
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	// mutedStyle renders less-important text in grey.
	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	// statLabelStyle renders statistic labels in dim text.
	statLabelStyle = lipgloss.NewStyle().
			Faint(true)

	// statValueStyle renders statistic values in bold.
	statValueStyle = lipgloss.NewStyle().
			Bold(true)
)
