package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"uptui/internal/models"
)

const maxHistory = 500

type historyData struct {
	History map[string][]models.Result `json:"history"`
}

type Store struct {
	mu   sync.RWMutex
	path string
	data historyData
}

func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	s := &Store{
		path: filepath.Join(dir, "history.json"),
		data: historyData{
			History: make(map[string][]models.Result),
		},
	}
	b, err := os.ReadFile(s.path)
	if err == nil {
		_ = json.Unmarshal(b, &s.data)
		if s.data.History == nil {
			s.data.History = make(map[string][]models.Result)
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

// DeleteHistory removes all stored results for name.
func (s *Store) DeleteHistory(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.History, name)
	_ = s.save()
}

// RenameHistory moves history from oldName to newName.
func (s *Store) RenameHistory(oldName, newName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if h, ok := s.data.History[oldName]; ok {
		s.data.History[newName] = h
		delete(s.data.History, oldName)
		_ = s.save()
	}
}

// AddResult appends a check result for the named monitor, capping at maxHistory.
func (s *Store) AddResult(name string, r models.Result) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	history := s.data.History[name]
	history = append(history, r)
	if len(history) > maxHistory {
		history = history[len(history)-maxHistory:]
	}
	s.data.History[name] = history
	return s.save()
}

// GetHistory returns a copy of stored results for name (empty slice if none).
func (s *Store) GetHistory(name string) []models.Result {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h := s.data.History[name]
	out := make([]models.Result, len(h))
	copy(out, h)
	return out
}
