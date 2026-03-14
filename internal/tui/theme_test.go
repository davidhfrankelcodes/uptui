package tui

import (
	"testing"
)

func TestDefaultTheme(t *testing.T) {
	th := DefaultTheme()
	if th.Name != "default" {
		t.Errorf("Name = %q, want default", th.Name)
	}
	if th.Up == "" {
		t.Error("Up color should be non-empty")
	}
	if th.Down == "" {
		t.Error("Down color should be non-empty")
	}
}

func TestParseThemeAllBuiltins(t *testing.T) {
	for _, name := range ThemeNames {
		th, err := ParseTheme(name)
		if err != nil {
			t.Errorf("ParseTheme(%q): unexpected error: %v", name, err)
		}
		if th.Name != name {
			t.Errorf("ParseTheme(%q).Name = %q", name, th.Name)
		}
	}
}

func TestParseThemeInvalid(t *testing.T) {
	_, err := ParseTheme("neon-rainbow")
	if err == nil {
		t.Error("ParseTheme(invalid): expected error, got nil")
	}
}

func TestParseThemeEmpty(t *testing.T) {
	th, err := ParseTheme("")
	if err != nil {
		t.Errorf("ParseTheme(empty): unexpected error: %v", err)
	}
	if th.Name != "default" {
		t.Errorf("ParseTheme(empty).Name = %q, want default", th.Name)
	}
}

func TestNewStylesDoesNotPanic(t *testing.T) {
	for _, name := range ThemeNames {
		th, _ := ParseTheme(name)
		s := NewStyles(th)
		// Verify at least one style is non-zero by checking it renders
		rendered := s.Title.Render("x")
		if rendered == "" {
			t.Errorf("NewStyles(%q).Title rendered empty string", name)
		}
	}
}

func TestStatusStyleAllStatuses(t *testing.T) {
	s := NewStyles(DefaultTheme())
	statuses := []string{"up", "down", "pending", "paused", "", "unknown"}
	for _, st := range statuses {
		style := s.StatusStyle(st)
		// Just verify we get a usable style (renders without empty result)
		rendered := style.Render("●")
		if rendered == "" {
			t.Errorf("StatusStyle(%q) rendered empty string", st)
		}
	}
}
