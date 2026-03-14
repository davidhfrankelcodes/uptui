package ipc_test

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"uptui/internal/ipc"
	"uptui/internal/models"
)

// ── mock handler ──────────────────────────────────────────────────────────────

type mockHandler struct {
	mu        sync.Mutex
	monitors  []*models.MonitorStatus
	pausedIDs []int
	resumedID []int
	deletedID []int
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
	m.ID = len(h.monitors) + 1
	ms := &models.MonitorStatus{Monitor: m, Status: models.StatusPending}
	h.monitors = append(h.monitors, ms)
	return ms, nil
}

func (h *mockHandler) DeleteMonitor(id int) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.deletedID = append(h.deletedID, id)
	for i, ms := range h.monitors {
		if ms.Monitor.ID == id {
			h.monitors = append(h.monitors[:i], h.monitors[i+1:]...)
			break
		}
	}
	return nil
}

func (h *mockHandler) PauseMonitor(id int) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pausedIDs = append(h.pausedIDs, id)
	return nil
}

func (h *mockHandler) ResumeMonitor(id int) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.resumedID = append(h.resumedID, id)
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
	if ms.Monitor.ID == 0 {
		t.Error("expected non-zero ID")
	}

	// Verify it appears in List
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

	if err := client.Delete(ms.Monitor.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	h.mu.Lock()
	deleted := h.deletedID
	h.mu.Unlock()

	if len(deleted) != 1 || deleted[0] != ms.Monitor.ID {
		t.Errorf("deletedID = %v, want [%d]", deleted, ms.Monitor.ID)
	}
}

func TestPause(t *testing.T) {
	client, h, cancel := startTestServer(t)
	defer cancel()

	ms, _ := client.Add(models.Monitor{Name: "x", Type: models.HTTP, Target: "http://x", Active: true})

	if err := client.Pause(ms.Monitor.ID); err != nil {
		t.Fatalf("Pause: %v", err)
	}

	h.mu.Lock()
	paused := h.pausedIDs
	h.mu.Unlock()

	if len(paused) != 1 || paused[0] != ms.Monitor.ID {
		t.Errorf("pausedIDs = %v, want [%d]", paused, ms.Monitor.ID)
	}
}

func TestResume(t *testing.T) {
	client, h, cancel := startTestServer(t)
	defer cancel()

	ms, _ := client.Add(models.Monitor{Name: "x", Type: models.HTTP, Target: "http://x", Active: true})

	if err := client.Resume(ms.Monitor.ID); err != nil {
		t.Fatalf("Resume: %v", err)
	}

	h.mu.Lock()
	resumed := h.resumedID
	h.mu.Unlock()

	if len(resumed) != 1 || resumed[0] != ms.Monitor.ID {
		t.Errorf("resumedID = %v, want [%d]", resumed, ms.Monitor.ID)
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
