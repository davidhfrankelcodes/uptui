package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"uptui/internal/config"
	"uptui/internal/models"
)

func TestLoadMissingFile(t *testing.T) {
	monitors, err := config.Load(filepath.Join(t.TempDir(), "nonexistent.toml"))
	if err != nil {
		t.Fatalf("Load(missing): unexpected error: %v", err)
	}
	if monitors != nil {
		t.Errorf("Load(missing): expected nil slice, got %v", monitors)
	}
}

func TestLoadEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "monitors.toml")
	os.WriteFile(path, []byte(""), 0644)

	monitors, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load(empty): %v", err)
	}
	if len(monitors) != 0 {
		t.Errorf("Load(empty): len = %d, want 0", len(monitors))
	}
}

func TestLoadValidTOML(t *testing.T) {
	toml := `
[[monitor]]
name     = "GitHub"
type     = "http"
target   = "https://github.com"
interval = 30

[[monitor]]
name   = "Postgres"
type   = "tcp"
target = "localhost:5432"
active = false
`
	path := filepath.Join(t.TempDir(), "monitors.toml")
	os.WriteFile(path, []byte(toml), 0644)

	monitors, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(monitors) != 2 {
		t.Fatalf("len = %d, want 2", len(monitors))
	}

	gh := monitors[0]
	if gh.Name != "GitHub" {
		t.Errorf("name = %q, want GitHub", gh.Name)
	}
	if gh.Type != models.HTTP {
		t.Errorf("type = %q, want http", gh.Type)
	}
	if gh.Target != "https://github.com" {
		t.Errorf("target = %q", gh.Target)
	}
	if gh.Interval != 30 {
		t.Errorf("interval = %d, want 30", gh.Interval)
	}
	if !gh.Active {
		t.Error("GitHub should be active")
	}

	pg := monitors[1]
	if pg.Name != "Postgres" {
		t.Errorf("name = %q, want Postgres", pg.Name)
	}
	if pg.Active {
		t.Error("Postgres should be inactive (active = false)")
	}
}

func TestLoadDefaultsApplied(t *testing.T) {
	toml := `
[[monitor]]
name   = "test"
type   = "http"
target = "http://example.com"
`
	path := filepath.Join(t.TempDir(), "monitors.toml")
	os.WriteFile(path, []byte(toml), 0644)

	monitors, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(monitors) != 1 {
		t.Fatalf("len = %d, want 1", len(monitors))
	}
	m := monitors[0]
	if m.Interval != 60 {
		t.Errorf("default interval = %d, want 60", m.Interval)
	}
	if m.Timeout != 30 {
		t.Errorf("default timeout = %d, want 30", m.Timeout)
	}
	if !m.Active {
		t.Error("default active should be true")
	}
}

func TestLoadSkipsBlankEntries(t *testing.T) {
	toml := `
[[monitor]]
name   = ""
type   = "http"
target = "http://example.com"

[[monitor]]
name   = "valid"
type   = "http"
target = ""

[[monitor]]
name   = "ok"
type   = "tcp"
target = "localhost:9000"
`
	path := filepath.Join(t.TempDir(), "monitors.toml")
	os.WriteFile(path, []byte(toml), 0644)

	monitors, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(monitors) != 1 {
		t.Errorf("len = %d, want 1 (entries with blank name/target should be skipped)", len(monitors))
	}
	if monitors[0].Name != "ok" {
		t.Errorf("name = %q, want ok", monitors[0].Name)
	}
}

func TestLoadInvalidTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "monitors.toml")
	os.WriteFile(path, []byte("this is not [ valid toml !!!"), 0644)

	_, err := config.Load(path)
	if err == nil {
		t.Error("Load(invalid): expected error, got nil")
	}
}

func TestSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "monitors.toml")

	want := []models.Monitor{
		{Name: "Alpha", Type: models.HTTP, Target: "https://alpha.com", Interval: 60, Timeout: 30, Active: true},
		{Name: "Beta", Type: models.TCP, Target: "localhost:5432", Interval: 15, Timeout: 5, Active: false},
	}

	if err := config.Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load after Save: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i].Name != want[i].Name {
			t.Errorf("[%d] name = %q, want %q", i, got[i].Name, want[i].Name)
		}
		if got[i].Target != want[i].Target {
			t.Errorf("[%d] target = %q, want %q", i, got[i].Target, want[i].Target)
		}
		if got[i].Active != want[i].Active {
			t.Errorf("[%d] active = %v, want %v", i, got[i].Active, want[i].Active)
		}
		if got[i].Interval != want[i].Interval {
			t.Errorf("[%d] interval = %d, want %d", i, got[i].Interval, want[i].Interval)
		}
	}
}

func TestSaveDefaultsOmitted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "monitors.toml")

	monitors := []models.Monitor{
		{Name: "test", Type: models.HTTP, Target: "http://x.com", Interval: 60, Timeout: 30, Active: true},
	}
	if err := config.Save(path, monitors); err != nil {
		t.Fatalf("Save: %v", err)
	}

	b, _ := os.ReadFile(path)
	content := string(b)

	// interval=60 and timeout=30 are defaults, should be omitted
	if contains(content, "interval") {
		t.Error("default interval should be omitted from saved TOML")
	}
	if contains(content, "timeout") {
		t.Error("default timeout should be omitted from saved TOML")
	}
	// active=true is default, should be omitted
	if contains(content, "active") {
		t.Error("active=true should be omitted from saved TOML")
	}
}

func TestSaveActiveFalseWritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "monitors.toml")

	monitors := []models.Monitor{
		{Name: "paused", Type: models.HTTP, Target: "http://x.com", Interval: 60, Timeout: 30, Active: false},
	}
	if err := config.Save(path, monitors); err != nil {
		t.Fatalf("Save: %v", err)
	}

	b, _ := os.ReadFile(path)
	if !contains(string(b), "active") {
		t.Error("active = false should be written to TOML")
	}
}

func TestSaveAtomicWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "monitors.toml")

	monitors := []models.Monitor{
		{Name: "x", Type: models.HTTP, Target: "http://x.com", Interval: 60, Timeout: 30, Active: true},
	}
	if err := config.Save(path, monitors); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Temporary file should be gone after Save
	tmp := path + ".tmp"
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Error("tmp file should not exist after successful Save")
	}
}

func TestSaveEmptyList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "monitors.toml")

	if err := config.Save(path, nil); err != nil {
		t.Fatalf("Save(nil): %v", err)
	}

	monitors, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load after Save(nil): %v", err)
	}
	if len(monitors) != 0 {
		t.Errorf("len = %d after saving empty list, want 0", len(monitors))
	}
}

// ── Settings tests ────────────────────────────────────────────────────────────

func TestLoadSettingsMissingFile(t *testing.T) {
	s, err := config.LoadSettings(filepath.Join(t.TempDir(), "settings.toml"))
	if err != nil {
		t.Fatalf("LoadSettings(missing): unexpected error: %v", err)
	}
	if s.Theme != "default" {
		t.Errorf("Theme = %q, want default", s.Theme)
	}
}

func TestLoadSettingsValidTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.toml")
	os.WriteFile(path, []byte("theme = \"nord\"\n"), 0644)

	s, err := config.LoadSettings(path)
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if s.Theme != "nord" {
		t.Errorf("Theme = %q, want nord", s.Theme)
	}
}

func TestLoadSettingsEmptyTheme(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.toml")
	os.WriteFile(path, []byte(""), 0644)

	s, err := config.LoadSettings(path)
	if err != nil {
		t.Fatalf("LoadSettings(empty): %v", err)
	}
	if s.Theme != "default" {
		t.Errorf("Theme = %q, want default for empty file", s.Theme)
	}
}

func TestSaveSettingsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.toml")

	if err := config.SaveSettings(path, config.Settings{Theme: "dracula"}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	s, err := config.LoadSettings(path)
	if err != nil {
		t.Fatalf("LoadSettings after Save: %v", err)
	}
	if s.Theme != "dracula" {
		t.Errorf("Theme = %q, want dracula", s.Theme)
	}
}

func TestSaveSettingsDefaultOmitsThemeKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.toml")

	if err := config.SaveSettings(path, config.Settings{Theme: "default"}); err != nil {
		t.Fatalf("SaveSettings(default): %v", err)
	}

	b, _ := os.ReadFile(path)
	if contains(string(b), "theme") {
		t.Error("saving default theme should produce empty file (no theme key)")
	}
}

func contains(s, sub string) bool {
	return strings.Contains(s, sub)
}
