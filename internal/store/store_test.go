package store_test

import (
	"testing"
	"time"

	"uptui/internal/models"
	"uptui/internal/store"
)

func newStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	return s
}

func TestAddMonitor(t *testing.T) {
	s := newStore(t)

	m, err := s.AddMonitor(models.Monitor{
		Name:     "my-site",
		Type:     models.HTTP,
		Target:   "https://example.com",
		Interval: 60,
		Active:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if m.ID == 0 {
		t.Error("expected non-zero ID")
	}

	all := s.GetMonitors()
	if len(all) != 1 {
		t.Fatalf("len(monitors) = %d, want 1", len(all))
	}
	if all[0].Name != "my-site" {
		t.Errorf("name = %q, want %q", all[0].Name, "my-site")
	}
}

func TestIDsIncrement(t *testing.T) {
	s := newStore(t)

	a, _ := s.AddMonitor(models.Monitor{Name: "a", Type: models.HTTP, Target: "http://a", Active: true})
	b, _ := s.AddMonitor(models.Monitor{Name: "b", Type: models.HTTP, Target: "http://b", Active: true})

	if b.ID <= a.ID {
		t.Errorf("IDs not incrementing: a=%d b=%d", a.ID, b.ID)
	}
}

func TestDeleteMonitor(t *testing.T) {
	s := newStore(t)

	m, _ := s.AddMonitor(models.Monitor{Name: "x", Type: models.HTTP, Target: "http://x", Active: true})
	s.AddResult(m.ID, models.Result{Timestamp: time.Now(), Status: models.StatusUp, Latency: 10})

	if err := s.DeleteMonitor(m.ID); err != nil {
		t.Fatal(err)
	}

	if got := s.GetMonitors(); len(got) != 0 {
		t.Errorf("monitors after delete: %d, want 0", len(got))
	}
	// history should also be gone
	if hist := s.GetHistory(m.ID); len(hist) != 0 {
		t.Errorf("history after delete: %d, want 0", len(hist))
	}
}

func TestDeleteNonexistent(t *testing.T) {
	s := newStore(t)
	// Should not error
	if err := s.DeleteMonitor(999); err != nil {
		t.Errorf("delete nonexistent: unexpected error: %v", err)
	}
}

func TestSetMonitorActive(t *testing.T) {
	s := newStore(t)
	m, _ := s.AddMonitor(models.Monitor{Name: "x", Type: models.HTTP, Target: "http://x", Active: true})

	if err := s.SetMonitorActive(m.ID, false); err != nil {
		t.Fatal(err)
	}

	all := s.GetMonitors()
	if all[0].Active {
		t.Error("expected Active=false after SetMonitorActive(false)")
	}

	s.SetMonitorActive(m.ID, true)
	all = s.GetMonitors()
	if !all[0].Active {
		t.Error("expected Active=true after SetMonitorActive(true)")
	}
}

func TestAddAndGetResult(t *testing.T) {
	s := newStore(t)
	m, _ := s.AddMonitor(models.Monitor{Name: "x", Type: models.HTTP, Target: "http://x", Active: true})

	r := models.Result{Timestamp: time.Now(), Status: models.StatusUp, Latency: 42, Message: "HTTP 200"}
	if err := s.AddResult(m.ID, r); err != nil {
		t.Fatal(err)
	}

	hist := s.GetHistory(m.ID)
	if len(hist) != 1 {
		t.Fatalf("len(history) = %d, want 1", len(hist))
	}
	if hist[0].Latency != 42 {
		t.Errorf("latency = %d, want 42", hist[0].Latency)
	}
	if hist[0].Status != models.StatusUp {
		t.Errorf("status = %q, want up", hist[0].Status)
	}
}

func TestHistoryMaxSize(t *testing.T) {
	s := newStore(t)
	m, _ := s.AddMonitor(models.Monitor{Name: "x", Type: models.HTTP, Target: "http://x", Active: true})

	for i := 0; i < 510; i++ {
		s.AddResult(m.ID, models.Result{
			Timestamp: time.Now(),
			Status:    models.StatusUp,
			Latency:   i,
		})
	}

	hist := s.GetHistory(m.ID)
	if len(hist) > 500 {
		t.Errorf("history length %d exceeds cap of 500", len(hist))
	}
	// Most recent results should be at the end
	if hist[len(hist)-1].Latency != 509 {
		t.Errorf("last result latency = %d, want 509", hist[len(hist)-1].Latency)
	}
}

func TestGetHistoryUnknownMonitor(t *testing.T) {
	s := newStore(t)
	hist := s.GetHistory(999)
	if hist == nil {
		t.Error("expected non-nil slice for unknown monitor")
	}
	if len(hist) != 0 {
		t.Errorf("expected empty history for unknown monitor, got %d", len(hist))
	}
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()

	// Write data
	s1, _ := store.New(dir)
	m, _ := s1.AddMonitor(models.Monitor{Name: "persist", Type: models.TCP, Target: "localhost:5432", Active: true})
	s1.AddResult(m.ID, models.Result{Timestamp: time.Now(), Status: models.StatusUp, Latency: 7})

	// Reload from same dir
	s2, err := store.New(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	monitors := s2.GetMonitors()
	if len(monitors) != 1 {
		t.Fatalf("after reload: len(monitors) = %d, want 1", len(monitors))
	}
	if monitors[0].Name != "persist" {
		t.Errorf("after reload: name = %q, want %q", monitors[0].Name, "persist")
	}
	if monitors[0].ID != m.ID {
		t.Errorf("after reload: ID = %d, want %d", monitors[0].ID, m.ID)
	}

	hist := s2.GetHistory(m.ID)
	if len(hist) != 1 || hist[0].Latency != 7 {
		t.Errorf("after reload: history = %v, want [{Latency:7}]", hist)
	}
}

func TestGetMonitorsReturnsCopy(t *testing.T) {
	s := newStore(t)
	s.AddMonitor(models.Monitor{Name: "x", Type: models.HTTP, Target: "http://x", Active: true})

	a := s.GetMonitors()
	a[0].Name = "mutated"

	b := s.GetMonitors()
	if b[0].Name == "mutated" {
		t.Error("GetMonitors does not return a copy; mutation affected internal state")
	}
}
