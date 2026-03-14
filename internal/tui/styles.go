package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorUp      = lipgloss.Color("10")  // bright green
	colorDown    = lipgloss.Color("9")   // bright red
	colorPending = lipgloss.Color("11")  // bright yellow
	colorPaused  = lipgloss.Color("8")   // dark gray
	colorMuted   = lipgloss.Color("240") // gray
	colorTitle   = lipgloss.Color("14")  // bright cyan
	colorCursor  = lipgloss.Color("12")  // bright blue
	colorBorder  = lipgloss.Color("236") // very dark gray
	colorLabel   = lipgloss.Color("7")   // light gray
)

var (
	styleTitle   = lipgloss.NewStyle().Bold(true).Foreground(colorTitle)
	styleUp      = lipgloss.NewStyle().Foreground(colorUp)
	styleDown    = lipgloss.NewStyle().Foreground(colorDown)
	stylePending = lipgloss.NewStyle().Foreground(colorPending)
	stylePaused  = lipgloss.NewStyle().Foreground(colorPaused)
	styleMuted   = lipgloss.NewStyle().Foreground(colorMuted)
	styleBold    = lipgloss.NewStyle().Bold(true)
	styleLabel   = lipgloss.NewStyle().Foreground(colorLabel)
	styleCursor  = lipgloss.NewStyle().Foreground(colorCursor).Bold(true)
	styleError   = lipgloss.NewStyle().Foreground(colorDown)
	styleHeader  = lipgloss.NewStyle().Bold(true).Foreground(colorMuted)
	styleBorder  = lipgloss.NewStyle().Foreground(colorBorder)

	styleSelectedRow = lipgloss.NewStyle().
				Background(lipgloss.Color("235")).
				Bold(true)

	styleKeyHint = lipgloss.NewStyle().
			Foreground(colorTitle).
			Bold(true)

	styleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)
)

func statusStyle(s string) lipgloss.Style {
	switch s {
	case "up":
		return styleUp
	case "down":
		return styleDown
	case "pending":
		return stylePending
	case "paused":
		return stylePaused
	default:
		return styleMuted
	}
}
