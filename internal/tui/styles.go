package tui

import "github.com/charmbracelet/lipgloss"

// Styles holds all composed lipgloss styles for the TUI.
type Styles struct {
	Title      lipgloss.Style
	Up         lipgloss.Style
	Down       lipgloss.Style
	Pending    lipgloss.Style
	Paused     lipgloss.Style
	Muted      lipgloss.Style
	Bold       lipgloss.Style
	Label      lipgloss.Style
	Cursor     lipgloss.Style
	Error      lipgloss.Style
	Header     lipgloss.Style
	Border     lipgloss.Style
	SelectedRow lipgloss.Style
	KeyHint    lipgloss.Style
	Box        lipgloss.Style
}

// NewStyles builds a Styles set from the given Theme.
func NewStyles(t Theme) Styles {
	return Styles{
		Title:   lipgloss.NewStyle().Bold(true).Foreground(t.Title),
		Up:      lipgloss.NewStyle().Foreground(t.Up),
		Down:    lipgloss.NewStyle().Foreground(t.Down),
		Pending: lipgloss.NewStyle().Foreground(t.Pending),
		Paused:  lipgloss.NewStyle().Foreground(t.Paused),
		Muted:   lipgloss.NewStyle().Foreground(t.Muted),
		Bold:    lipgloss.NewStyle().Bold(true),
		Label:   lipgloss.NewStyle().Foreground(t.Label),
		Cursor:  lipgloss.NewStyle().Foreground(t.Cursor).Bold(true),
		Error:   lipgloss.NewStyle().Foreground(t.Down),
		Header:  lipgloss.NewStyle().Bold(true).Foreground(t.Muted),
		Border:  lipgloss.NewStyle().Foreground(t.Border),
		SelectedRow: lipgloss.NewStyle().
			Background(t.SelectedBg).
			Bold(true),
		KeyHint: lipgloss.NewStyle().
			Foreground(t.Title).
			Bold(true),
		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border).
			Padding(0, 1),
	}
}

// StatusStyle returns the style for a given status string.
func (s Styles) StatusStyle(status string) lipgloss.Style {
	switch status {
	case "up":
		return s.Up
	case "down":
		return s.Down
	case "pending":
		return s.Pending
	case "paused":
		return s.Paused
	default:
		return s.Muted
	}
}
