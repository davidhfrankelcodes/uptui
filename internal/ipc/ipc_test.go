package ipc_test

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"uptui/internal/ipc"
	"uptui/internal/models"
)

// ── mock handler ──────────────────────────────────────────────────────────────

type mockHandler struct {
	mu           sync.Mutex
	monitors     []*models.MonitorStatus
	pausedNames  []string
	resumedNames []string
	deletedNames []string
	editedOld    []string
	reloaded     int
}

func (h *mockHandler) GetAllStatus() []*models.MonitorStatus {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]*models.MonitorStatus, len(h.monitors))
	copy(out, h.monitors)
	return out
}

func (h *mockHandler) AddMonitor(m models.Monitor) (*models.MonitorStatus, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ms := &models.MonitorStatus{Monitor: m, Status: models.StatusPending}
	h.monitors = append(h.monitors, ms)
	return ms, nil
}

func (h *mockHandler) DeleteMonitor(name string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.deletedNames = append(h.deletedNames, name)
	for i, ms := range h.monitors {
		if ms.Monitor.Name == name {
			h.monitors = append(h.monitors[:i], h.monitors[i+1:]...)
			break
		}
	}
	return nil
}

func (h *mockHandler) PauseMonitor(name string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pausedNames = append(h.pausedNames, name)
	return nil
}

func (h *mockHandler) ResumeMonitor(name string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.resumedNames = append(h.resumedNames, name)
	return nil
}

func (h *mockHandler) EditMonitor(oldName string, m models.Monitor) (*models.MonitorStatus, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.editedOld = append(h.editedOld, oldName)
	for _, ms := range h.monitors {
		if ms.Monitor.Name == oldName {
			ms.Monitor = m
			return ms, nil
		}
	}
	ms := &models.MonitorStatus{Monitor: m, Status: models.StatusPending}
	h.monitors = append(h.monitors, ms)
	return ms, nil
}

func (h *mockHandler) Reload() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.reloaded++
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func startTestServer(t *testing.T) (*ipc.Client, *mockHandler, context.CancelFunc) {
	t.Helper()

	// Grab a free port then release it; the server will re-bind.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()

	h := &mockHandler{}
	srv := ipc.NewServer(addr, h)
	ctx, cancel := context.WithCancel(context.Background())
	go srv.Listen(ctx) //nolint:errcheck

	// Poll until the server is ready (up to 2 s)
	client := ipc.NewClient(addr)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if client.Ping() {
			return client, h, cancel
		}
		time.Sleep(20 * time.Millisecond)
	}

	cancel()
	t.Fatal("IPC server did not start within 2 s")
	return nil, nil, nil
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestListEmpty(t *testing.T) {
	client, _, cancel := startTestServer(t)
	defer cancel()

	monitors, err := client.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(monitors) != 0 {
		t.Errorf("len = %d, want 0", len(monitors))
	}
}

func TestAdd(t *testing.T) {
	client, h, cancel := startTestServer(t)
	defer cancel()

	m := models.Monitor{
		Name:     "my-service",
		Type:     models.HTTP,
		Target:   "https://example.com",
		Interval: 60,
		Timeout:  30,
		Active:   true,
	}
	ms, err := client.Add(m)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if ms == nil {
		t.Fatal("Add returned nil MonitorStatus")
	}
	if ms.Monitor.Name != "my-service" {
		t.Errorf("name = %q, want %q", ms.Monitor.Name, "my-service")
	}

	h.mu.Lock()
	count := len(h.monitors)
	h.mu.Unlock()
	if count != 1 {
		t.Errorf("handler has %d monitors, want 1", count)
	}
}

func TestListAfterAdd(t *testing.T) {
	client, _, cancel := startTestServer(t)
	defer cancel()

	client.Add(models.Monitor{Name: "a", Type: models.HTTP, Target: "http://a", Active: true})
	client.Add(models.Monitor{Name: "b", Type: models.HTTP, Target: "http://b", Active: true})

	monitors, err := client.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(monitors) != 2 {
		t.Errorf("len = %d, want 2", len(monitors))
	}
}

func TestDelete(t *testing.T) {
	client, h, cancel := startTestServer(t)
	defer cancel()

	ms, _ := client.Add(models.Monitor{Name: "x", Type: models.HTTP, Target: "http://x", Active: true})

	if err := client.Delete(ms.Monitor.Name); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	h.mu.Lock()
	deleted := h.deletedNames
	h.mu.Unlock()

	if len(deleted) != 1 || deleted[0] != ms.Monitor.Name {
		t.Errorf("deletedNames = %v, want [%q]", deleted, ms.Monitor.Name)
	}
}

func TestPause(t *testing.T) {
	client, h, cancel := startTestServer(t)
	defer cancel()

	ms, _ := client.Add(models.Monitor{Name: "x", Type: models.HTTP, Target: "http://x", Active: true})

	if err := client.Pause(ms.Monitor.Name); err != nil {
		t.Fatalf("Pause: %v", err)
	}

	h.mu.Lock()
	paused := h.pausedNames
	h.mu.Unlock()

	if len(paused) != 1 || paused[0] != ms.Monitor.Name {
		t.Errorf("pausedNames = %v, want [%q]", paused, ms.Monitor.Name)
	}
}

func TestResume(t *testing.T) {
	client, h, cancel := startTestServer(t)
	defer cancel()

	ms, _ := client.Add(models.Monitor{Name: "x", Type: models.HTTP, Target: "http://x", Active: true})

	if err := client.Resume(ms.Monitor.Name); err != nil {
		t.Fatalf("Resume: %v", err)
	}

	h.mu.Lock()
	resumed := h.resumedNames
	h.mu.Unlock()

	if len(resumed) != 1 || resumed[0] != ms.Monitor.Name {
		t.Errorf("resumedNames = %v, want [%q]", resumed, ms.Monitor.Name)
	}
}

func TestEdit(t *testing.T) {
	client, h, cancel := startTestServer(t)
	defer cancel()

	client.Add(models.Monitor{Name: "original", Type: models.HTTP, Target: "http://x", Active: true})

	updated := models.Monitor{Name: "renamed", Type: models.HTTP, Target: "http://x", Interval: 30, Active: true}
	ms, err := client.Edit("original", updated)
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if ms.Monitor.Name != "renamed" {
		t.Errorf("edited name = %q, want renamed", ms.Monitor.Name)
	}

	h.mu.Lock()
	edited := h.editedOld
	h.mu.Unlock()

	if len(edited) != 1 || edited[0] != "original" {
		t.Errorf("editedOld = %v, want [original]", edited)
	}
}

func TestReload(t *testing.T) {
	client, h, cancel := startTestServer(t)
	defer cancel()

	if err := client.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	h.mu.Lock()
	reloaded := h.reloaded
	h.mu.Unlock()

	if reloaded != 1 {
		t.Errorf("reloaded = %d, want 1", reloaded)
	}
}

func TestPingTrue(t *testing.T) {
	client, _, cancel := startTestServer(t)
	defer cancel()

	if !client.Ping() {
		t.Error("Ping() = false, want true")
	}
}

func TestPingFalse(t *testing.T) {
	client := ipc.NewClient("127.0.0.1:19997") // nothing listening here
	if client.Ping() {
		t.Error("Ping() = true, want false (nothing listening)")
	}
}

// TestListLargePayload verifies that List() succeeds when the response exceeds
// 64 KB (the old bufio.Scanner limit). 50 monitors × 500 history entries
// produces several MB of JSON.
func TestListLargePayload(t *testing.T) {
	client, h, cancel := startTestServer(t)
	defer cancel()

	history := make([]models.Result, 500)
	now := time.Now()
	for i := range history {
		history[i] = models.Result{
			Timestamp: now.Add(-time.Duration(i) * time.Minute),
			Status:    models.StatusUp,
			Latency:   42 + i,
			Message:   "HTTP 200",
		}
	}

	h.mu.Lock()
	for i := 0; i < 50; i++ {
		h.monitors = append(h.monitors, &models.MonitorStatus{
			Monitor: models.Monitor{
				Name:     fmt.Sprintf("monitor-%03d", i),
				Type:     models.HTTP,
				Target:   fmt.Sprintf("https://example-%03d.com", i),
				Interval: 60,
				Timeout:  30,
				Active:   true,
			},
			Status:  models.StatusUp,
			History: history,
		})
	}
	h.mu.Unlock()

	monitors, err := client.List()
	if err != nil {
		t.Fatalf("List with large payload: %v", err)
	}
	if len(monitors) != 50 {
		t.Errorf("got %d monitors, want 50", len(monitors))
	}
}

func TestConcurrentClients(t *testing.T) {
	client, _, cancel := startTestServer(t)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.List()
			if err != nil {
				t.Errorf("concurrent List: %v", err)
			}
		}()
	}
	wg.Wait()
}
