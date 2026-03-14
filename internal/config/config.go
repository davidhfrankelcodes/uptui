package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"uptui/internal/models"
)

const defaultInterval = 60
const defaultTimeout = 30

// tomlMonitor is the on-disk representation of a single monitor entry.
type tomlMonitor struct {
	Name     string `toml:"name"`
	Type     string `toml:"type"`
	Target   string `toml:"target"`
	Interval int    `toml:"interval"`
	Timeout  int    `toml:"timeout"`
	Active   *bool  `toml:"active"` // nil → true; explicit false → paused
}

type tomlFile struct {
	Monitor []tomlMonitor `toml:"monitor"`
}

// Load reads monitors from path. Returns empty (nil) slice if the file does not exist.
func Load(path string) ([]models.Monitor, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var f tomlFile
	if _, err := toml.Decode(string(b), &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	out := make([]models.Monitor, 0, len(f.Monitor))
	for _, tm := range f.Monitor {
		if tm.Name == "" || tm.Target == "" {
			continue
		}
		m := models.Monitor{
			Name:     tm.Name,
			Type:     models.MonitorType(tm.Type),
			Target:   tm.Target,
			Interval: tm.Interval,
			Timeout:  tm.Timeout,
			Active:   true,
		}
		if m.Type == "" {
			m.Type = models.HTTP
		}
		if m.Interval < 10 {
			m.Interval = defaultInterval
		}
		if m.Timeout <= 0 {
			m.Timeout = defaultTimeout
		}
		if tm.Active != nil && !*tm.Active {
			m.Active = false
		}
		out = append(out, m)
	}
	return out, nil
}

// Save writes monitors to path using a compact hand-rolled TOML encoder.
// Interval is omitted when 60 (default); Timeout when 30 (default).
// Active is omitted when true; written as `active = false` when false.
func Save(path string, monitors []models.Monitor) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	var buf bytes.Buffer
	for i, m := range monitors {
		if i > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString("[[monitor]]\n")
		buf.WriteString(fmt.Sprintf("name     = %q\n", m.Name))
		buf.WriteString(fmt.Sprintf("type     = %q\n", string(m.Type)))
		buf.WriteString(fmt.Sprintf("target   = %q\n", m.Target))
		if m.Interval != defaultInterval {
			buf.WriteString(fmt.Sprintf("interval = %d\n", m.Interval))
		}
		if m.Timeout != defaultTimeout {
			buf.WriteString(fmt.Sprintf("timeout  = %d\n", m.Timeout))
		}
		if !m.Active {
			buf.WriteString("active   = false\n")
		}
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
