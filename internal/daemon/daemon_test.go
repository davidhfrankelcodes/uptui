package daemon

// White-box tests: same package gives access to unexported functions.

import (
	"testing"
	"time"

	"uptui/internal/models"
)

// ── calcUptime ─────────────────────────────────────────────────────────────────

func TestCalcUptimeEmpty(t *testing.T) {
	got := calcUptime(nil, 24*time.Hour)
	if got != 0 {
		t.Errorf("empty history: uptime = %.2f, want 0", got)
	}
}

func TestCalcUptimeAllUp(t *testing.T) {
	now := time.Now()
	history := []models.Result{
		{Timestamp: now, Status: models.StatusUp},
		{Timestamp: now, Status: models.StatusUp},
		{Timestamp: now, Status: models.StatusUp},
	}
	got := calcUptime(history, 24*time.Hour)
	if got != 100 {
		t.Errorf("all up: uptime = %.2f, want 100", got)
	}
}

func TestCalcUptimeAllDown(t *testing.T) {
	now := time.Now()
	history := []models.Result{
		{Timestamp: now, Status: models.StatusDown},
		{Timestamp: now, Status: models.StatusDown},
	}
	got := calcUptime(history, 24*time.Hour)
	if got != 0 {
		t.Errorf("all down: uptime = %.2f, want 0", got)
	}
}

func TestCalcUptimeHalf(t *testing.T) {
	now := time.Now()
	history := []models.Result{
		{Timestamp: now, Status: models.StatusUp},
		{Timestamp: now, Status: models.StatusDown},
	}
	got := calcUptime(history, 24*time.Hour)
	if got != 50 {
		t.Errorf("half: uptime = %.2f, want 50", got)
	}
}

func TestCalcUptimeExcludesOldResults(t *testing.T) {
	now := time.Now()
	history := []models.Result{
		// Old result outside the window — should be ignored
		{Timestamp: now.Add(-48 * time.Hour), Status: models.StatusDown},
		// Recent result inside the window
		{Timestamp: now, Status: models.StatusUp},
	}
	got := calcUptime(history, 24*time.Hour)
	if got != 100 {
		t.Errorf("old result excluded: uptime = %.2f, want 100", got)
	}
}

func TestCalcUptimeAllOutsideWindow(t *testing.T) {
	now := time.Now()
	history := []models.Result{
		{Timestamp: now.Add(-48 * time.Hour), Status: models.StatusDown},
		{Timestamp: now.Add(-36 * time.Hour), Status: models.StatusDown},
	}
	got := calcUptime(history, 24*time.Hour)
	// All results are older than 24 h: total=0, returns 0 (not NaN)
	if got != 0 {
		t.Errorf("all outside window: uptime = %.2f, want 0", got)
	}
}

func TestCalcUptimeWindowBoundary(t *testing.T) {
	now := time.Now()
	// Exactly on the boundary (should be excluded because it's Before cutoff)
	boundary := now.Add(-24 * time.Hour).Add(-time.Millisecond)
	history := []models.Result{
		{Timestamp: boundary, Status: models.StatusDown},
		{Timestamp: now, Status: models.StatusUp},
	}
	got := calcUptime(history, 24*time.Hour)
	if got != 100 {
		t.Errorf("boundary: uptime = %.2f, want 100", got)
	}
}

func TestCalcUptimePendingNotCountedAsUp(t *testing.T) {
	now := time.Now()
	history := []models.Result{
		{Timestamp: now, Status: models.StatusUp},
		{Timestamp: now, Status: models.StatusPending},
	}
	got := calcUptime(history, 24*time.Hour)
	// 1 up out of 2 = 50%
	if got != 50 {
		t.Errorf("pending not up: uptime = %.2f, want 50", got)
	}
}

func TestCalcUptimeNarrowWindow(t *testing.T) {
	now := time.Now()
	history := []models.Result{
		{Timestamp: now.Add(-30 * time.Minute), Status: models.StatusDown},
		{Timestamp: now.Add(-5 * time.Minute), Status: models.StatusUp},
	}
	// 1h window — both results included
	got := calcUptime(history, time.Hour)
	if got != 50 {
		t.Errorf("1h window: uptime = %.2f, want 50", got)
	}

	// 10min window — only the most recent result falls in
	got = calcUptime(history, 10*time.Minute)
	if got != 100 {
		t.Errorf("10m window: uptime = %.2f, want 100", got)
	}
}
