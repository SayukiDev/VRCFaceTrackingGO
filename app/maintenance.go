package app

import (
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"VRCFaceTracking/internal/logging"
	"VRCFaceTracking/internal/settings"
)

func (a *App) GetLogs() []logging.Line { return a.svc.Logs() }

// CopyLogs puts the full log into the clipboard.
func (a *App) CopyLogs() error { return runtime.ClipboardSetText(a.ctx, a.svc.LogText()) }

// SaveLogs asks for a destination and writes the full log there. It returns
// the chosen file name, or "" when cancelled.
func (a *App) SaveLogs() (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: "VRCFaceTrackingGO-" + time.Now().Format("2006-01-02-15-04") + ".txt",
		Filters:         []runtime.FileFilter{{DisplayName: "Plain Text", Pattern: "*.txt"}},
	})
	if err != nil || path == "" {
		return "", err
	}
	if err := a.svc.SaveLogs(path); err != nil {
		return "", err
	}
	return filepath.Base(path), nil
}

func (a *App) GetTheme() string { return a.svc.Store().Get().Theme }

func (a *App) SetTheme(theme string) {
	a.svc.Store().Update(func(s *settings.Settings) { s.Theme = theme })
}

// GetLocale returns the saved UI language, or "" to follow the system.
func (a *App) GetLocale() string { return a.svc.Store().Get().Locale }

func (a *App) SetLocale(locale string) {
	a.svc.Store().Update(func(s *settings.Settings) { s.Locale = locale })
}

func (a *App) ResetVRCFT() error { return a.svc.ScheduleReset() }

func (a *App) ResetAvatarConfigs() error { return a.svc.ResetAvatarConfigs() }
