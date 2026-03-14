package daemon

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"uptui/internal/checker"
	"uptui/internal/config"
	"uptui/internal/ipc"
	"uptui/internal/models"
	"uptui/internal/store"
)

type monitorState struct {
	ms     models.MonitorStatus
	cancel context.CancelFunc
}

type Daemon struct {
	store      *store.Store
	configPath string
	mu         sync.RWMutex
	state      map[string]*monitorState
	rootCtx    context.Context
}

func New(s *store.Store, configPath string) *Daemon {
	return &Daemon{
		store:      s,
		configPath: configPath,
		state:      make(map[string]*monitorState),
	}
}

// Run loads config, starts monitor goroutines, a config watcher, and the IPC server.
// Blocks until ctx is cancelled.
func (d *Daemon) Run(ctx context.Context, addr string) error {
	d.rootCtx = ctx

	monitors, err := config.Load(d.configPath)
	if err != nil {
		log.Printf("config load: %v", err)
	}
	// Single-threaded startup: reconcile without holding d.mu
	d.reconcileLocked(monitors)

	go d.watchConfig(ctx)

	srv := ipc.NewServer(addr, d)
	return srv.Listen(ctx)
}

// watchConfig polls the config file mtime every 5 s and reconciles on change.
func (d *Daemon) watchConfig(ctx context.Context) {
	var lastMtime time.Time
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := os.Stat(d.configPath)
			if err != nil {
				continue
			}
			mtime := info.ModTime()
			if mtime.Equal(lastMtime) {
				continue
			}
			lastMtime = mtime

			monitors, err := config.Load(d.configPath)
			if err != nil {
				log.Printf("config reload: %v", err)
				continue
			}
			d.mu.Lock()
			d.reconcileLocked(monitors)
			d.mu.Unlock()
		}
	}
}

// reconcileLocked synchronises d.state with the desired monitor list.
// Must be called either before the IPC server starts (no lock needed) or
// with d.mu write-locked.
func (d *Daemon) reconcileLocked(desired []models.Monitor) {
	desiredSet := make(map[string]models.Monitor, len(desired))
	for _, m := range desired {
		desiredSet[m.Name] = m
	}

	// Remove monitors that are no longer in config
	for name, st := range d.state {
		if _, ok := desiredSet[name]; !ok {
			st.cancel()
			delete(d.state, name)
		}
	}

	// Add or update monitors
	for _, m := range desired {
		existing, exists := d.state[m.Name]
		if !exists {
			history := d.store.GetHistory(m.Name)
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

			mctx, mcancel := context.WithCancel(d.rootCtx)
			d.state[m.Name] = &monitorState{ms: ms, cancel: mcancel}
			if m.Active {
				go d.runMonitor(mctx, m)
			} else {
				mcancel()
				d.state[m.Name].cancel = func() {}
			}
		} else {
			old := existing.ms.Monitor
			changed := old.Target != m.Target || old.Type != m.Type ||
				old.Interval != m.Interval || old.Timeout != m.Timeout
			activeChanged := old.Active != m.Active

			if changed {
				existing.cancel()
				existing.ms.Monitor = m
				if m.Active {
					mctx, mcancel := context.WithCancel(d.rootCtx)
					existing.cancel = mcancel
					go d.runMonitor(mctx, m)
				} else {
					existing.ms.Status = models.StatusPaused
					existing.cancel = func() {}
				}
			} else if activeChanged {
				if m.Active {
					existing.ms.Monitor.Active = true
					existing.ms.Status = models.StatusPending
					mctx, mcancel := context.WithCancel(d.rootCtx)
					existing.cancel = mcancel
					go d.runMonitor(mctx, m)
				} else {
					existing.cancel()
					existing.ms.Monitor.Active = false
					existing.ms.Status = models.StatusPaused
					existing.cancel = func() {}
				}
			}
		}
	}
}

func (d *Daemon) runMonitor(ctx context.Context, m models.Monitor) {
	interval := time.Duration(m.Interval) * time.Second
	if interval <= 0 {
		interval = 60 * time.Second
	}

	d.doCheck(ctx, m)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.mu.RLock()
			st, ok := d.state[m.Name]
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

	if err := d.store.AddResult(m.Name, result); err != nil {
		log.Printf("store result: %v", err)
	}

	d.mu.Lock()
	st, ok := d.state[m.Name]
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

// currentMonitors returns a sorted snapshot of all monitor configs from state.
func (d *Daemon) currentMonitors() []models.Monitor {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]models.Monitor, 0, len(d.state))
	for _, st := range d.state {
		out = append(out, st.ms.Monitor)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

// ── IPC handlers ───────────────────────────────────────────────────────────────

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
		return out[i].Monitor.Name < out[j].Monitor.Name
	})
	return out
}

// AddMonitor implements ipc.Handler.
func (d *Daemon) AddMonitor(m models.Monitor) (*models.MonitorStatus, error) {
	if m.Interval < 10 {
		m.Interval = defaultInterval
	}
	if m.Timeout <= 0 {
		m.Timeout = defaultTimeout
	}

	d.mu.RLock()
	_, exists := d.state[m.Name]
	d.mu.RUnlock()
	if exists {
		return nil, fmt.Errorf("monitor %q already exists", m.Name)
	}

	ms := models.MonitorStatus{Monitor: m, Status: models.StatusPending}
	if !m.Active {
		ms.Status = models.StatusPaused
	}

	mctx, mcancel := context.WithCancel(d.rootCtx)

	d.mu.Lock()
	// Re-check under write lock
	if _, exists = d.state[m.Name]; exists {
		d.mu.Unlock()
		mcancel()
		return nil, fmt.Errorf("monitor %q already exists", m.Name)
	}
	d.state[m.Name] = &monitorState{ms: ms, cancel: mcancel}
	d.mu.Unlock()

	if m.Active {
		go d.runMonitor(mctx, m)
	} else {
		mcancel()
		d.mu.Lock()
		d.state[m.Name].cancel = func() {}
		d.mu.Unlock()
	}

	monitors := d.currentMonitors()
	if err := config.Save(d.configPath, monitors); err != nil {
		return nil, err
	}

	cp := ms
	return &cp, nil
}

// DeleteMonitor implements ipc.Handler.
func (d *Daemon) DeleteMonitor(name string) error {
	d.mu.Lock()
	if st, ok := d.state[name]; ok {
		st.cancel()
		delete(d.state, name)
	}
	d.mu.Unlock()

	d.store.DeleteHistory(name)

	monitors := d.currentMonitors()
	return config.Save(d.configPath, monitors)
}

// PauseMonitor implements ipc.Handler.
func (d *Daemon) PauseMonitor(name string) error {
	d.mu.Lock()
	if st, ok := d.state[name]; ok {
		st.cancel()
		st.ms.Monitor.Active = false
		st.ms.Status = models.StatusPaused
		st.cancel = func() {}
	}
	d.mu.Unlock()

	monitors := d.currentMonitors()
	return config.Save(d.configPath, monitors)
}

// ResumeMonitor implements ipc.Handler.
func (d *Daemon) ResumeMonitor(name string) error {
	d.mu.Lock()
	st, ok := d.state[name]
	if ok {
		st.ms.Monitor.Active = true
		st.ms.Status = models.StatusPending
		mctx, mcancel := context.WithCancel(d.rootCtx)
		st.cancel = mcancel
		m := st.ms.Monitor
		go d.runMonitor(mctx, m)
	}
	d.mu.Unlock()

	monitors := d.currentMonitors()
	return config.Save(d.configPath, monitors)
}

// EditMonitor implements ipc.Handler.
func (d *Daemon) EditMonitor(oldName string, m models.Monitor) (*models.MonitorStatus, error) {
	if m.Interval < 10 {
		m.Interval = defaultInterval
	}
	if m.Timeout <= 0 {
		m.Timeout = defaultTimeout
	}

	d.mu.Lock()
	st, ok := d.state[oldName]
	if !ok {
		d.mu.Unlock()
		return nil, fmt.Errorf("monitor %q not found", oldName)
	}

	// Cancel the running goroutine for the old monitor
	st.cancel()
	delete(d.state, oldName)

	// Move history if renamed
	if oldName != m.Name {
		d.store.RenameHistory(oldName, m.Name)
	}

	history := d.store.GetHistory(m.Name)
	ms := models.MonitorStatus{
		Monitor:   m,
		Status:    models.StatusPending,
		History:   history,
		Uptime24h: calcUptime(history, 24*time.Hour),
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

	mctx, mcancel := context.WithCancel(d.rootCtx)
	newSt := &monitorState{ms: ms, cancel: mcancel}
	d.state[m.Name] = newSt
	d.mu.Unlock()

	if m.Active {
		go d.runMonitor(mctx, m)
	} else {
		mcancel()
		d.mu.Lock()
		newSt.cancel = func() {}
		d.mu.Unlock()
	}

	monitors := d.currentMonitors()
	if err := config.Save(d.configPath, monitors); err != nil {
		return nil, err
	}

	cp := ms
	return &cp, nil
}

// Reload implements ipc.Handler. Forces a re-read of the config file.
func (d *Daemon) Reload() error {
	monitors, err := config.Load(d.configPath)
	if err != nil {
		return err
	}
	d.mu.Lock()
	d.reconcileLocked(monitors)
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

const defaultInterval = 60
const defaultTimeout = 30
