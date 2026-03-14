package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"uptui/internal/models"
)

const maxHistory = 500

type dbData struct {
	Monitors []models.Monitor          `json:"monitors"`
	History  map[int][]models.Result   `json:"history"`
	NextID   int                       `json:"next_id"`
}

type Store struct {
	mu   sync.RWMutex
	path string
	data dbData
}

func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	s := &Store{
		path: filepath.Join(dir, "db.json"),
		data: dbData{
			History: make(map[int][]models.Result),
			NextID:  1,
		},
	}
	b, err := os.ReadFile(s.path)
	if err == nil {
		_ = json.Unmarshal(b, &s.data)
		if s.data.History == nil {
			s.data.History = make(map[int][]models.Result)
		}
		if s.data.NextID < 1 {
			s.data.NextID = 1
		}
	}
	return s, nil
}

func (s *Store) save() error {
	b, err := json.Marshal(s.data)
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) AddMonitor(m models.Monitor) (models.Monitor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m.ID = s.data.NextID
	s.data.NextID++
	s.data.Monitors = append(s.data.Monitors, m)
	return m, s.save()
}

func (s *Store) DeleteMonitor(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, m := range s.data.Monitors {
		if m.ID == id {
			s.data.Monitors = append(s.data.Monitors[:i], s.data.Monitors[i+1:]...)
			delete(s.data.History, id)
			return s.save()
		}
	}
	return nil
}

func (s *Store) SetMonitorActive(id int, active bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, m := range s.data.Monitors {
		if m.ID == id {
			s.data.Monitors[i].Active = active
			return s.save()
		}
	}
	return nil
}

func (s *Store) AddResult(monitorID int, r models.Result) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	history := s.data.History[monitorID]
	history = append(history, r)
	if len(history) > maxHistory {
		history = history[len(history)-maxHistory:]
	}
	s.data.History[monitorID] = history
	return s.save()
}

func (s *Store) GetMonitors() []models.Monitor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.Monitor, len(s.data.Monitors))
	copy(out, s.data.Monitors)
	return out
}

func (s *Store) GetHistory(monitorID int) []models.Result {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h := s.data.History[monitorID]
	out := make([]models.Result, len(h))
	copy(out, h)
	return out
}
