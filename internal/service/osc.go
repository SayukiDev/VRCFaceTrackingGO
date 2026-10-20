package service

import (
	"go.uber.org/zap"

	"VRCFaceTracking/internal/osc"
)

// OscStatus is the state of the OSC endpoints plus per-second counters.
type OscStatus struct {
	ListenPort  int  `json:"listenPort"`
	SendPort    int  `json:"sendPort"`
	Connected   bool `json:"connected"`
	OscQuery    bool `json:"oscQuery"`
	MessagesIn  int  `json:"messagesIn"`
	MessagesOut int  `json:"messagesOut"`
	AllRelevant bool `json:"allRelevant"`
}

// OscStatus returns the current status and resets the message counters.
func (s *Service) OscStatus() OscStatus {
	return OscStatus{
		ListenPort:  s.query.ListenPort(),
		SendPort:    s.Target().OutPort,
		Connected:   s.sender.Connected(),
		OscQuery:    s.query.OSCQueryActive(),
		MessagesIn:  int(s.messagesIn.Swap(0)),
		MessagesOut: int(s.messagesOut.Swap(0)),
		AllRelevant: s.engine.AllRelevant(),
	}
}

// Target returns the configured OSC endpoint.
func (s *Service) Target() osc.Target {
	s.targetMu.RLock()
	defer s.targetMu.RUnlock()
	return s.target
}

func (s *Service) setTarget(t osc.Target) {
	s.targetMu.Lock()
	defer s.targetMu.Unlock()
	s.target = t
}

// SetTarget validates, persists and applies a new OSC endpoint.
func (s *Service) SetTarget(t osc.Target) error {
	if err := t.Validate(); err != nil {
		return err
	}
	s.setTarget(t)
	if err := s.store.Set(oscTargetKey, t); err != nil {
		return err
	}
	if err := s.sender.SetTarget(t.Address, t.OutPort); err != nil {
		s.logger.Warn("Failed to bind to sender endpoint", zap.Error(err))
	}
	s.query.SetTarget(t)
	return nil
}

// AllParametersRelevant reports whether every parameter is forced to send.
func (s *Service) AllParametersRelevant() bool { return s.engine.AllRelevant() }

// SetAllParametersRelevant forces every parameter to be sent every frame.
func (s *Service) SetAllParametersRelevant(v bool) { s.engine.SetAllRelevant(v) }
