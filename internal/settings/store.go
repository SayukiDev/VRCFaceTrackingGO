// Package settings persists application settings as a single JSON document
// with a debounced writer and a backup copy.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	fileName     = "LocalSettings.json"
	debounceTime = 300 * time.Millisecond
)

// Store is a key/value settings file. Values are stored as raw JSON so the
// caller decides the concrete type on read.
type Store struct {
	logger *zap.Logger
	path   string

	mu     sync.Mutex
	values map[string]json.RawMessage
	timer  *time.Timer
}

// Open loads (or creates) the settings file inside dir.
func Open(dir string, logger *zap.Logger) (*Store, error) {
	s := &Store{
		logger: logger.Named("Settings"),
		path:   filepath.Join(dir, fileName),
		values: map[string]json.RawMessage{},
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create settings directory: %w", err)
	}

	backup := s.path + ".bak"
	values, err := readFile(s.path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			s.logger.Warn("Primary settings file was corrupt", zap.Error(err))
		}
	}
	if values == nil {
		s.logger.Warn("Restoring default settings")
		values = map[string]json.RawMessage{}
		if err := s.writeLocked(values); err != nil {
			return nil, err
		}
	}
	s.values = values
	_ = copyFile(s.path, backup)
	return s, nil
}

func readFile(path string) (map[string]json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	if values == nil {
		values = map[string]json.RawMessage{}
	}
	return values, nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// Get unmarshals key into v. found is false when the key does not exist.
func (s *Store) Get(key string, v any) (found bool, err error) {
	s.mu.Lock()
	raw, ok := s.values[key]
	s.mu.Unlock()
	if !ok {
		return false, nil
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return true, fmt.Errorf("decode setting %q: %w", key, err)
	}
	return true, nil
}

// GetOr reads key into v and falls back to def when it is missing or
// unreadable.
func GetOr[T any](s *Store, key string, def T) T {
	var v T
	found, err := s.Get(key, &v)
	if err != nil {
		s.logger.Warn("Ignoring unreadable setting", zap.String("key", key), zap.Error(err))
		return def
	}
	if !found {
		return def
	}
	return v
}

// Set stores v under key and schedules a debounced write.
func (s *Store) Set(key string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("encode setting %q: %w", key, err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = raw
	if s.timer != nil {
		s.timer.Stop()
	}
	s.timer = time.AfterFunc(debounceTime, func() {
		if err := s.Flush(); err != nil {
			s.logger.Error("Failed to save settings", zap.Error(err))
		}
	})
	return nil
}

// Flush writes pending changes to disk immediately.
func (s *Store) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	return s.writeLocked(s.values)
}

func (s *Store) writeLocked(values map[string]json.RawMessage) error {
	data, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return os.Rename(tmp, s.path)
}
