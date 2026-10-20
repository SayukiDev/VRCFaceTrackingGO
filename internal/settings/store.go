// Package settings persists the application settings as a single typed
// JSON document with a debounced writer and a backup copy.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	fileName     = "LocalSettings.json"
	debounceTime = 300 * time.Millisecond
)

// Store holds the current Settings and writes them back to disk.
type Store struct {
	logger *zap.Logger
	path   string

	mu       sync.Mutex
	settings Settings
	timer    *time.Timer
}

// Open loads (or creates) the settings file inside dir. A corrupt file is
// replaced by its backup, or by the defaults when there is none. Fields
// missing from the file keep their default value.
func Open(dir string, logger *zap.Logger) (*Store, error) {
	s := &Store{
		logger: logger.Named("Settings"),
		path:   filepath.Join(dir, fileName),
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create settings directory: %w", err)
	}

	backup := s.path + ".bak"
	settings, err := readFile(s.path)
	if err == nil {
		_ = copyFile(s.path, backup)
	} else {
		if !errors.Is(err, os.ErrNotExist) {
			s.logger.Warn("Primary settings file was corrupt", zap.Error(err))
		}
		if settings, err = readFile(backup); err == nil {
			s.logger.Warn("Restored settings from backup")
		} else {
			s.logger.Warn("Restoring default settings")
			settings = Default()
		}
	}
	for _, err := range settings.sanitize() {
		s.logger.Warn("Invalid setting, reverting to default", zap.Error(err))
	}
	s.settings = settings
	if err := s.writeLocked(); err != nil {
		return nil, err
	}
	return s, nil
}

func readFile(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Settings{}, err
	}
	settings := Default()
	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, err
	}
	return settings, nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// Get returns a copy of the current settings.
func (s *Store) Get() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.settings
	c.Modules = slices.Clone(c.Modules)
	return c
}

// Update applies fn to the settings and schedules a debounced write. fn
// must not call back into the store.
func (s *Store) Update(fn func(*Settings)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.settings)
	if s.timer != nil {
		s.timer.Stop()
	}
	s.timer = time.AfterFunc(debounceTime, func() {
		if err := s.Flush(); err != nil {
			s.logger.Error("Failed to save settings", zap.Error(err))
		}
	})
}

// Flush writes pending changes to disk immediately.
func (s *Store) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	return s.writeLocked()
}

func (s *Store) writeLocked() error {
	data, err := json.MarshalIndent(s.settings, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return os.Rename(tmp, s.path)
}
