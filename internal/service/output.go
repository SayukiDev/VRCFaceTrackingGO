package service

import (
	"fmt"

	"go.uber.org/zap"

	"VRCFaceTracking/internal/sender"
	"VRCFaceTracking/internal/settings"
)

// OutputMode returns where tracking data is sent.
func (s *Service) OutputMode() sender.OutputMode { return s.sender.OutputMode() }

// SetOutputMode validates, persists and applies a new output mode.
func (s *Service) SetOutputMode(m sender.OutputMode) error {
	if !m.Valid() {
		return fmt.Errorf("unknown output mode %q", m)
	}
	s.sender.SetOutputMode(m)
	s.logger.Info("Output mode changed", zap.String("mode", string(m)))
	s.store.Update(func(c *settings.Settings) { c.Output = m })
	return nil
}

func (s *Service) applyTiming(t settings.Timing) {
	s.modules.SetUpdateInterval(t.UpdateInterval())
	s.modules.SetInitTimeout(t.InitTimeout())
	s.router.SetSendInterval(t.SendInterval())
}

// Timing returns the current scheduling values.
func (s *Service) Timing() settings.Timing { return s.store.Get().Timing }

// SetTiming validates, persists and applies new scheduling values. The
// update and send intervals take effect immediately; the init timeout on
// the next (re)initialisation.
func (s *Service) SetTiming(t settings.Timing) error {
	if err := t.Validate(); err != nil {
		return err
	}
	s.applyTiming(t)
	s.store.Update(func(c *settings.Settings) { c.Timing = t })
	return nil
}
