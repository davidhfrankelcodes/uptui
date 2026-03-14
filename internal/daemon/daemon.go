package daemon

import (
	"context"
	"log"
	"sort"
	"sync"
	"time"

	"uptui/internal/checker"
	"uptui/internal/ipc"
	"uptui/internal/models"
	"uptui/internal/store"
)

type monitorState struct {
	ms      models.MonitorStatus
	cancel  context.CancelFunc
}

type Daemon struct {
	store  *store.Store
	mu     sync.RWMutex
	state  map[int]*monitorState
}

func New(s *store.Store) *Daemon {
	return &Daemon{
		store: s,
		state: make(map[int]*monitorState),
	}
}

// Run starts all monitor goroutines and the IPC server. Blocks until ctx is cancelled.
func (d *Daemon) Run(ctx context.Context, addr string) error {
	monitors := d.store.GetMonitors()
	for _, m := range monitors {
		history := d.store.GetHistory(m.ID)
		ms := models.MonitorStatus{
			Monitor: m,
			Status:  models.StatusPending,
			History: history,
		}
		if len(history) > 0 {
			last := history[len(history)-1]
			ms.Status = last.Status
			ms.Latency = last.Latency
			ms.LastCheck = last.Timestamp
		}
		if !m.Active {
			ms.Status = models.StatusPaused
		}
		ms.Uptime24h = calcUptime(history, 24*time.Hour)

		mctx, mcancel := context.WithCancel(ctx)
		d.state[m.ID] = &monitorState{ms: ms, cancel: mcancel}

		if m.Active {
			go d.runMonitor(mctx, m)
		}
	}

	srv := ipc.NewServer(addr, d)
	return srv.Listen(ctx)
}

func (d *Daemon) runMonitor(ctx context.Context, m models.Monitor) {
	interval := time.Duration(m.Interval) * time.Second
	if interval <= 0 {
		interval = 60 * time.Second
	}

	// Check immediately on start
	d.doCheck(ctx, m)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Re-read monitor in case it was updated
			d.mu.RLock()
			st, ok := d.state[m.ID]
			if ok {
				m = st.ms.Monitor
			}
			d.mu.RUnlock()
			if !ok || !m.Active {
				return
			}
			d.doCheck(ctx, m)
		}
	}
}

func (d *Daemon) doCheck(ctx context.Context, m models.Monitor) {
	result := checker.Check(ctx, m)

	if err := d.store.AddResult(m.ID, result); err != nil {
		log.Printf("store result: %v", err)
	}

	d.mu.Lock()
	st, ok := d.state[m.ID]
	if ok {
		st.ms.Status = result.Status
		st.ms.Latency = result.Latency
		st.ms.LastCheck = result.Timestamp
		st.ms.History = append(st.ms.History, result)
		if len(st.ms.History) > 500 {
			st.ms.History = st.ms.History[len(st.ms.History)-500:]
		}
		st.ms.Uptime24h = calcUptime(st.ms.History, 24*time.Hour)
	}
	d.mu.Unlock()
}

// GetAllStatus implements ipc.Handler.
func (d *Daemon) GetAllStatus() []*models.MonitorStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]*models.MonitorStatus, 0, len(d.state))
	for _, st := range d.state {
		cp := st.ms
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Monitor.ID < out[j].Monitor.ID
	})
	return out
}

// AddMonitor implements ipc.Handler.
func (d *Daemon) AddMonitor(m models.Monitor) (*models.MonitorStatus, error) {
	saved, err := d.store.AddMonitor(m)
	if err != nil {
		return nil, err
	}

	ms := models.MonitorStatus{
		Monitor: saved,
		Status:  models.StatusPending,
	}

	ctx := context.Background()
	mctx, mcancel := context.WithCancel(ctx)

	d.mu.Lock()
	d.state[saved.ID] = &monitorState{ms: ms, cancel: mcancel}
	d.mu.Unlock()

	if saved.Active {
		go d.runMonitor(mctx, saved)
	} else {
		mcancel()
	}

	cp := ms
	return &cp, nil
}

// DeleteMonitor implements ipc.Handler.
func (d *Daemon) DeleteMonitor(id int) error {
	d.mu.Lock()
	if st, ok := d.state[id]; ok {
		st.cancel()
		delete(d.state, id)
	}
	d.mu.Unlock()
	return d.store.DeleteMonitor(id)
}

// PauseMonitor implements ipc.Handler.
func (d *Daemon) PauseMonitor(id int) error {
	d.mu.Lock()
	if st, ok := d.state[id]; ok {
		st.cancel() // stops the running goroutine
		st.ms.Monitor.Active = false
		st.ms.Status = models.StatusPaused
		st.cancel = func() {} // no-op until resumed
	}
	d.mu.Unlock()
	return d.store.SetMonitorActive(id, false)
}

// ResumeMonitor implements ipc.Handler.
func (d *Daemon) ResumeMonitor(id int) error {
	if err := d.store.SetMonitorActive(id, true); err != nil {
		return err
	}

	d.mu.Lock()
	st, ok := d.state[id]
	if ok {
		st.ms.Monitor.Active = true
		st.ms.Status = models.StatusPending
		mctx, mcancel := context.WithCancel(context.Background())
		st.cancel = mcancel
		m := st.ms.Monitor
		go d.runMonitor(mctx, m)
	}
	d.mu.Unlock()
	return nil
}

func calcUptime(history []models.Result, window time.Duration) float64 {
	if len(history) == 0 {
		return 0
	}
	cutoff := time.Now().Add(-window)
	var total, up int
	for _, r := range history {
		if r.Timestamp.Before(cutoff) {
			continue
		}
		total++
		if r.Status == models.StatusUp {
			up++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(up) / float64(total) * 100
}
