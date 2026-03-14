package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Theme holds the color palette for the TUI.
type Theme struct {
	Name       string
	Up         lipgloss.Color
	Down       lipgloss.Color
	Pending    lipgloss.Color
	Paused     lipgloss.Color
	Muted      lipgloss.Color
	Title      lipgloss.Color
	Cursor     lipgloss.Color
	Border     lipgloss.Color
	Label      lipgloss.Color
	SelectedBg lipgloss.Color
}

// ThemeNames is the canonical list of built-in theme names.
var ThemeNames = []string{
	"default",
	"dracula",
	"nord",
	"solarized",
	"monokai",
	"gruvbox",
	"monochrome",
}

// DefaultTheme returns the ANSI default theme.
func DefaultTheme() Theme {
	return Theme{
		Name:       "default",
		Up:         lipgloss.Color("10"),
		Down:       lipgloss.Color("9"),
		Pending:    lipgloss.Color("11"),
		Paused:     lipgloss.Color("8"),
		Muted:      lipgloss.Color("240"),
		Title:      lipgloss.Color("14"),
		Cursor:     lipgloss.Color("12"),
		Border:     lipgloss.Color("236"),
		Label:      lipgloss.Color("7"),
		SelectedBg: lipgloss.Color("235"),
	}
}

var builtinThemes = map[string]Theme{
	"default": DefaultTheme(),
	"dracula": {
		Name:       "dracula",
		Up:         lipgloss.Color("#50fa7b"),
		Down:       lipgloss.Color("#ff5555"),
		Pending:    lipgloss.Color("#f1fa8c"),
		Paused:     lipgloss.Color("#6272a4"),
		Muted:      lipgloss.Color("#6272a4"),
		Title:      lipgloss.Color("#bd93f9"),
		Cursor:     lipgloss.Color("#8be9fd"),
		Border:     lipgloss.Color("#44475a"),
		Label:      lipgloss.Color("#f8f8f2"),
		SelectedBg: lipgloss.Color("#44475a"),
	},
	"nord": {
		Name:       "nord",
		Up:         lipgloss.Color("#a3be8c"),
		Down:       lipgloss.Color("#bf616a"),
		Pending:    lipgloss.Color("#ebcb8b"),
		Paused:     lipgloss.Color("#4c566a"),
		Muted:      lipgloss.Color("#4c566a"),
		Title:      lipgloss.Color("#88c0d0"),
		Cursor:     lipgloss.Color("#81a1c1"),
		Border:     lipgloss.Color("#3b4252"),
		Label:      lipgloss.Color("#d8dee9"),
		SelectedBg: lipgloss.Color("#2e3440"),
	},
	"solarized": {
		Name:       "solarized",
		Up:         lipgloss.Color("#859900"),
		Down:       lipgloss.Color("#dc322f"),
		Pending:    lipgloss.Color("#b58900"),
		Paused:     lipgloss.Color("#586e75"),
		Muted:      lipgloss.Color("#586e75"),
		Title:      lipgloss.Color("#2aa198"),
		Cursor:     lipgloss.Color("#268bd2"),
		Border:     lipgloss.Color("#073642"),
		Label:      lipgloss.Color("#839496"),
		SelectedBg: lipgloss.Color("#073642"),
	},
	"monokai": {
		Name:       "monokai",
		Up:         lipgloss.Color("#a9dc76"),
		Down:       lipgloss.Color("#ff6188"),
		Pending:    lipgloss.Color("#ffd866"),
		Paused:     lipgloss.Color("#727072"),
		Muted:      lipgloss.Color("#727072"),
		Title:      lipgloss.Color("#78dce8"),
		Cursor:     lipgloss.Color("#ab9df2"),
		Border:     lipgloss.Color("#221f22"),
		Label:      lipgloss.Color("#fcfcfa"),
		SelectedBg: lipgloss.Color("#2d2a2e"),
	},
	"gruvbox": {
		Name:       "gruvbox",
		Up:         lipgloss.Color("#b8bb26"),
		Down:       lipgloss.Color("#fb4934"),
		Pending:    lipgloss.Color("#fabd2f"),
		Paused:     lipgloss.Color("#665c54"),
		Muted:      lipgloss.Color("#928374"),
		Title:      lipgloss.Color("#8ec07c"),
		Cursor:     lipgloss.Color("#83a598"),
		Border:     lipgloss.Color("#3c3836"),
		Label:      lipgloss.Color("#ebdbb2"),
		SelectedBg: lipgloss.Color("#3c3836"),
	},
	"monochrome": {
		Name:       "monochrome",
		Up:         lipgloss.Color("15"),
		Down:       lipgloss.Color("8"),
		Pending:    lipgloss.Color("7"),
		Paused:     lipgloss.Color("8"),
		Muted:      lipgloss.Color("8"),
		Title:      lipgloss.Color("15"),
		Cursor:     lipgloss.Color("15"),
		Border:     lipgloss.Color("236"),
		Label:      lipgloss.Color("7"),
		SelectedBg: lipgloss.Color("235"),
	},
}

// ParseTheme returns the named theme. Empty string or "default" returns DefaultTheme.
// Returns an error for unknown names.
func ParseTheme(name string) (Theme, error) {
	if name == "" {
		return DefaultTheme(), nil
	}
	t, ok := builtinThemes[name]
	if !ok {
		return Theme{}, fmt.Errorf("unknown theme %q (available: %v)", name, ThemeNames)
	}
	return t, nil
}
