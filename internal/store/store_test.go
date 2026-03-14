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

func TestAddAndGetResult(t *testing.T) {
	s := newStore(t)

	r := models.Result{Timestamp: time.Now(), Status: models.StatusUp, Latency: 42, Message: "HTTP 200"}
	if err := s.AddResult("my-site", r); err != nil {
		t.Fatal(err)
	}

	hist := s.GetHistory("my-site")
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

	for i := 0; i < 510; i++ {
		s.AddResult("x", models.Result{
			Timestamp: time.Now(),
			Status:    models.StatusUp,
			Latency:   i,
		})
	}

	hist := s.GetHistory("x")
	if len(hist) > 500 {
		t.Errorf("history length %d exceeds cap of 500", len(hist))
	}
	if hist[len(hist)-1].Latency != 509 {
		t.Errorf("last result latency = %d, want 509", hist[len(hist)-1].Latency)
	}
}

func TestGetHistoryUnknownMonitor(t *testing.T) {
	s := newStore(t)
	hist := s.GetHistory("nonexistent")
	if hist == nil {
		t.Error("expected non-nil slice for unknown monitor")
	}
	if len(hist) != 0 {
		t.Errorf("expected empty history, got %d items", len(hist))
	}
}

func TestDeleteHistory(t *testing.T) {
	s := newStore(t)

	s.AddResult("target", models.Result{Timestamp: time.Now(), Status: models.StatusUp, Latency: 10})
	s.DeleteHistory("target")

	hist := s.GetHistory("target")
	if len(hist) != 0 {
		t.Errorf("history after delete: %d items, want 0", len(hist))
	}
}

func TestDeleteHistoryNonexistent(t *testing.T) {
	s := newStore(t)
	// Should not panic or error
	s.DeleteHistory("nobody")
}

func TestRenameHistory(t *testing.T) {
	s := newStore(t)

	s.AddResult("old", models.Result{Timestamp: time.Now(), Status: models.StatusUp, Latency: 7})
	s.AddResult("old", models.Result{Timestamp: time.Now(), Status: models.StatusDown, Latency: 0})

	s.RenameHistory("old", "new")

	if hist := s.GetHistory("old"); len(hist) != 0 {
		t.Errorf("old history should be gone, got %d items", len(hist))
	}
	hist := s.GetHistory("new")
	if len(hist) != 2 {
		t.Errorf("new history = %d items, want 2", len(hist))
	}
	if hist[0].Latency != 7 {
		t.Errorf("first result latency = %d, want 7", hist[0].Latency)
	}
}

func TestRenameHistoryNonexistent(t *testing.T) {
	s := newStore(t)
	// Should not panic or error
	s.RenameHistory("nobody", "newname")
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()

	s1, _ := store.New(dir)
	s1.AddResult("persist", models.Result{Timestamp: time.Now(), Status: models.StatusUp, Latency: 7})

	s2, err := store.New(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	hist := s2.GetHistory("persist")
	if len(hist) != 1 || hist[0].Latency != 7 {
		t.Errorf("after reload: history = %v, want [{Latency:7}]", hist)
	}
}

func TestGetHistoryReturnsCopy(t *testing.T) {
	s := newStore(t)
	s.AddResult("x", models.Result{Timestamp: time.Now(), Status: models.StatusUp, Latency: 5})

	a := s.GetHistory("x")
	a[0].Latency = 999

	b := s.GetHistory("x")
	if b[0].Latency == 999 {
		t.Error("GetHistory does not return a copy; mutation affected internal state")
	}
}
