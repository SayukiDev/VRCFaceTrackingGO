package service

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/SayukiDev/VRCFaceTrackingGO/internal/osc"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/settings"
)

// OscStatus is the state of the OSC endpoints plus per-second counters.
type OscStatus struct {
	Mode        osc.Mode `json:"mode"`
	ListenPort  int      `json:"listenPort"`
	SendPort    int      `json:"sendPort"`
	Connected   bool     `json:"connected"`
	OscQuery    bool     `json:"oscQuery"`
	MessagesIn  int      `json:"messagesIn"`
	MessagesOut int      `json:"messagesOut"`
	AllRelevant bool     `json:"allRelevant"`
}

// OscStatus returns the current status and resets the message counters.
func (s *Service) OscStatus() OscStatus {
	return OscStatus{
		Mode:        s.OSCMode(),
		ListenPort:  s.query.ListenPort(),
		SendPort:    s.Target().OutPort,
		Connected:   s.transport.Connected(),
		OscQuery:    s.query.OSCQueryActive(),
		MessagesIn:  int(s.messagesIn.Swap(0)),
		MessagesOut: int(s.messagesOut.Swap(0)),
		AllRelevant: s.sender.AllRelevant(),
	}
}

// Target returns the configured OSC endpoint.
func (s *Service) Target() osc.Target { return s.store.Get().OSC }

// SetTarget validates, persists and applies a new OSC endpoint.
func (s *Service) SetTarget(t osc.Target) error {
	if err := t.Validate(); err != nil {
		return err
	}
	s.store.Update(func(c *settings.Settings) { c.OSC = t })
	if err := s.transport.SetTarget(t.Address, t.OutPort); err != nil {
		s.logger.Warn("Failed to bind to sender endpoint", zap.Error(err))
	}
	s.query.SetTarget(t)
	return nil
}

// OSCMode returns how VRChat and the loaded avatar are found.
func (s *Service) OSCMode() osc.Mode {
	if m := s.store.Get().OSCMode; m.Valid() {
		return m
	}
	return osc.ModeOSCQuery
}

// SetOSCMode persists the mode and restarts the OSC endpoints with it.
func (s *Service) SetOSCMode(m osc.Mode) error {
	if !m.Valid() {
		return fmt.Errorf("unknown OSC mode %q", m)
	}
	if m == s.OSCMode() {
		return nil
	}
	s.store.Update(func(c *settings.Settings) { c.OSCMode = m })
	s.logger.Info("OSC mode changed", zap.String("mode", string(m)))
	_ = s.query.Close()
	s.startQuery()
	return nil
}

// startQuery starts receiving OSC in the configured mode. Only the OSCQuery
// mode discovers VRChat and resolves the avatar; the OSC mode sends every
// parameter.
func (s *Service) startQuery() {
	s.applyRelevance()
	if err := s.query.Start(s.ctx, s.Target(), s.OSCMode() == osc.ModeOSCQuery); err != nil {
		s.logger.Error("OSCQuery service failed to start", zap.Error(err))
	}
}

// applyRelevance sends every parameter when the user asked for it or in the
// OSC mode, which does not know the avatar's parameters.
func (s *Service) applyRelevance() {
	s.relevanceMu.Lock()
	all := s.forceRelevant || s.OSCMode() == osc.ModeOSC
	s.relevanceMu.Unlock()
	s.sender.SetAllRelevant(all)
}

// AllParametersRelevant reports whether the user forces every parameter to
// be sent.
func (s *Service) AllParametersRelevant() bool {
	s.relevanceMu.Lock()
	defer s.relevanceMu.Unlock()
	return s.forceRelevant
}

// SetAllParametersRelevant forces every parameter to be sent every frame.
func (s *Service) SetAllParametersRelevant(v bool) {
	s.relevanceMu.Lock()
	s.forceRelevant = v
	s.relevanceMu.Unlock()
	s.applyRelevance()
}
