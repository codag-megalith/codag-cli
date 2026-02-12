package ui

import "github.com/charmbracelet/lipgloss"

var (
	Green  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	Red    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	Yellow = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	Cyan   = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	Dim    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	Bold   = lipgloss.NewStyle().Bold(true)

	CodeBlockStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(0, 1)
)
