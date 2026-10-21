package service

import (
	"go.uber.org/zap"

	"VRCFaceTrackingGo/internal/osc"
	"VRCFaceTrackingGo/internal/settings"
	"VRCFaceTrackingGo/internal/vrchat"
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
	st := OscStatus{
		SendPort:    s.Target().OutPort,
		Connected:   s.transport.Connected(),
		MessagesIn:  int(s.messagesIn.Swap(0)),
		MessagesOut: int(s.messagesOut.Swap(0)),
		AllRelevant: s.sender.AllRelevant(),
	}
	if s.query != nil {
		st.ListenPort = s.query.ListenPort()
		st.OscQuery = s.query.OSCQueryActive()
	}
	return st
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
	if s.query != nil {
		s.query.SetTarget(t)
	}
	return nil
}

// AllParametersRelevant reports whether every parameter is forced to send.
func (s *Service) AllParametersRelevant() bool { return s.sender.AllRelevant() }

// SetAllParametersRelevant forces every parameter to be sent every frame.
// Without VRChat detection every parameter is always sent, so turning it
// off is ignored there.
func (s *Service) SetAllParametersRelevant(v bool) {
	if !v && !vrchat.Supported {
		return
	}
	s.sender.SetAllRelevant(v)
}
