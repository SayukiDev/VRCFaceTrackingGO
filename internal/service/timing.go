package service

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	"VRCFaceTracking/internal/engine"
	"VRCFaceTracking/internal/settings"
	"VRCFaceTracking/module/manager"
)

const moduleTimingKey = "ModuleTiming"

const (
	minUpdateIntervalMs = 1
	maxUpdateIntervalMs = 1000
	minInitTimeoutSec   = 1
	maxInitTimeoutSec   = 300
	minSendIntervalMs   = 1
	maxSendIntervalMs   = 1000
)

// ModuleTiming holds the user adjustable module scheduling values.
type ModuleTiming struct {
	// UpdateIntervalMs is the time between two Update calls of one module.
	UpdateIntervalMs int `json:"updateIntervalMs"`
	// InitTimeoutSec is the maximum time one module may spend in Initialize.
	InitTimeoutSec int `json:"initTimeoutSec"`
	// SendIntervalMs is the time between two OSC send passes.
	SendIntervalMs int `json:"sendIntervalMs"`
}

// DefaultModuleTiming mirrors the manager defaults.
func DefaultModuleTiming() ModuleTiming {
	return ModuleTiming{
		UpdateIntervalMs: int(manager.DefaultUpdateInterval / time.Millisecond),
		InitTimeoutSec:   int(manager.DefaultInitTimeout / time.Second),
		SendIntervalMs:   int(engine.DefaultSendInterval / time.Millisecond),
	}
}

// Validate checks the values are within a sane range.
func (t ModuleTiming) Validate() error {
	if t.UpdateIntervalMs < minUpdateIntervalMs || t.UpdateIntervalMs > maxUpdateIntervalMs {
		return fmt.Errorf("update interval must be between %d and %d ms", minUpdateIntervalMs, maxUpdateIntervalMs)
	}
	if t.InitTimeoutSec < minInitTimeoutSec || t.InitTimeoutSec > maxInitTimeoutSec {
		return fmt.Errorf("init timeout must be between %d and %d s", minInitTimeoutSec, maxInitTimeoutSec)
	}
	if t.SendIntervalMs < minSendIntervalMs || t.SendIntervalMs > maxSendIntervalMs {
		return fmt.Errorf("send interval must be between %d and %d ms", minSendIntervalMs, maxSendIntervalMs)
	}
	return nil
}

func (s *Service) applyTiming(t ModuleTiming) {
	s.manager.SetUpdateInterval(time.Duration(t.UpdateIntervalMs) * time.Millisecond)
	s.manager.SetInitTimeout(time.Duration(t.InitTimeoutSec) * time.Second)
	s.engine.SetSendInterval(time.Duration(t.SendIntervalMs) * time.Millisecond)
}

// loadTiming applies the persisted timing or the defaults.
func (s *Service) loadTiming() {
	t := settings.GetOr(s.store, moduleTimingKey, DefaultModuleTiming())
	if err := t.Validate(); err != nil {
		s.logger.Warn("Invalid module timing in settings, reverting to default", zap.Error(err))
		t = DefaultModuleTiming()
	}
	s.applyTiming(t)
}

// ModuleTiming returns the current module scheduling values.
func (s *Service) ModuleTiming() ModuleTiming {
	return ModuleTiming{
		UpdateIntervalMs: int(s.manager.UpdateInterval() / time.Millisecond),
		InitTimeoutSec:   int(s.manager.InitTimeout() / time.Second),
		SendIntervalMs:   int(s.engine.SendInterval() / time.Millisecond),
	}
}

// SetModuleTiming validates, persists and applies new scheduling values.
// The update and send intervals take effect immediately; the init timeout
// on the next (re)initialisation.
func (s *Service) SetModuleTiming(t ModuleTiming) error {
	if err := t.Validate(); err != nil {
		return err
	}
	s.applyTiming(t)
	return s.store.Set(moduleTimingKey, t)
}
