package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SayukiDev/VRCFaceTrackingGO/internal/logging"
)

// resetMarker is created by ScheduleReset and consumed by
// ApplyResetIfRequested on the next launch.
const resetMarker = "reset"

// ApplyResetIfRequested wipes dataDir when a reset marker exists. Call it
// before anything opens files in dataDir.
func ApplyResetIfRequested(dataDir string) error {
	if _, err := os.Stat(filepath.Join(dataDir, resetMarker)); err != nil {
		return nil
	}
	if err := os.RemoveAll(dataDir); err != nil {
		return fmt.Errorf("reset data directory: %w", err)
	}
	return nil
}

// ScheduleReset marks the persistent data for deletion on next launch.
func (s *Service) ScheduleReset() error {
	s.logger.Info("Resetting VRCFT...")
	return os.WriteFile(filepath.Join(s.dataDir, resetMarker), nil, 0o644)
}

// Logs returns every buffered log line.
func (s *Service) Logs() []logging.Line { return s.logBuf.Lines() }

// LogText renders the whole log as plain text.
func (s *Service) LogText() string { return s.logBuf.Text() }

// SaveLogs writes the whole log to path.
func (s *Service) SaveLogs(path string) error {
	return os.WriteFile(path, []byte(s.logBuf.Text()), 0o644)
}
