// Package sender turns tracking data into OSC messages: it evaluates the
// parameters relevant to the current avatar and/or encodes VRChat's native
// tracking inputs, then hands the result to the OSC transport.
package sender

import (
	"bytes"
	"strings"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"

	"github.com/SayukiDev/VRCFaceTrackingGO/internal/osc"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/params"
	"github.com/SayukiDev/VRCFaceTrackingGO/internal/vrchat"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

const nativeEyePrefix = "/tracking/eye/"

// OutputMode selects how tracking data is sent to VRChat.
type OutputMode string

const (
	// OutputParameters drives avatar parameters (/avatar/parameters/v2/*).
	OutputParameters OutputMode = "parameters"
	// OutputNative feeds VRChat's built-in face tracking (/tracking/face/v1).
	OutputNative OutputMode = "native"
	// OutputBoth sends both at once.
	OutputBoth OutputMode = "both"
)

// Valid reports whether m is a known mode.
func (m OutputMode) Valid() bool {
	switch m {
	case OutputParameters, OutputNative, OutputBoth:
		return true
	}
	return false
}

func (m OutputMode) parameters() bool { return m == OutputParameters || m == OutputBoth }
func (m OutputMode) native() bool     { return m == OutputNative || m == OutputBoth }

// ParameterStats summarises the relevant parameter set for the UI.
type ParameterStats struct {
	Count      int  `json:"count"`
	Deprecated int  `json:"deprecated"`
	HasNative  bool `json:"hasNative"`
}

// Sender evaluates parameters and sends them over OSC.
type Sender struct {
	logger    *zap.Logger
	status    params.StatusProvider
	transport *osc.Sender

	all        []params.Parameter
	outputMode atomic.Value // OutputMode

	mu          sync.Mutex
	defs        []params.Definition
	outputs     []params.Output
	allRelevant bool
	stats       ParameterStats

	// last holds the value most recently sent by each output, indexed like
	// outputs; nil means nothing was sent yet. Keying by index rather than
	// address keeps outputs that share an address (e.g. the bool and float
	// variant of an EParam under forceRelevant) from defeating each other.
	last []any
	// lastNative holds the last value sent to each native endpoint.
	lastNative map[string]any

	// OnParametersChanged is called after every reset with the new stats.
	OnParametersChanged func(ParameterStats)
}

func New(logger *zap.Logger, status params.StatusProvider, transport *osc.Sender) *Sender {
	s := &Sender{
		logger:     logger.Named("ParameterSender"),
		status:     status,
		transport:  transport,
		all:        params.AllParameters(status),
		lastNative: map[string]any{},
	}
	s.SetOutputMode(OutputParameters)
	return s
}

// OutputMode returns where tracking data is sent.
func (s *Sender) OutputMode() OutputMode { return s.outputMode.Load().(OutputMode) }

// SetOutputMode changes where tracking data is sent. Invalid modes are
// ignored.
func (s *Sender) SetOutputMode(m OutputMode) {
	if !m.Valid() {
		return
	}
	s.outputMode.Store(m)
	// Resend everything for the new mode.
	s.mu.Lock()
	clear(s.last)
	clear(s.lastNative)
	s.mu.Unlock()
}

// Send evaluates every relevant output against d and sends the changed
// values.
func (s *Sender) Send(d *tracking.TrackingData) {
	mode := s.OutputMode()

	var messages []params.Message
	s.mu.Lock()
	if mode.parameters() {
		for i, out := range s.outputs {
			// Native mode sends the eye endpoints itself below.
			if mode.native() && strings.HasPrefix(out.Address(), nativeEyePrefix) {
				continue
			}
			msg, ok := out.Process(d)
			if !ok {
				continue
			}
			if s.last[i] != nil && valueEqual(s.last[i], msg.Value) {
				continue
			}
			s.last[i] = msg.Value
			messages = append(messages, msg)
		}
	}
	// Only feed native tracking while a module is tracking, so an idle VRCFT
	// does not override VRChat's own sources (e.g. Selfie Expression) with
	// zeros.
	if mode.native() {
		eye := s.status.EyeStatus() == adapter.Active
		if eye || s.status.ExpressionStatus() == adapter.Active {
			messages = s.appendNative(messages, vrchat.FaceAddress, vrchat.EncodeFace(d))
		} else {
			delete(s.lastNative, vrchat.FaceAddress)
		}
		// Bone-rotation gaze and the default eyelids read VRChat's eye
		// tracking input, not the face tracking EyeLook*/EyeClosed* keys.
		if eye {
			messages = s.appendNative(messages, params.NativeGazeAddress, params.NativeGaze(d))
			messages = s.appendNative(messages, params.NativeEyesClosedAddress, params.NativeEyesClosed(d))
		} else {
			// Resend as soon as eye tracking comes back.
			delete(s.lastNative, params.NativeGazeAddress)
			delete(s.lastNative, params.NativeEyesClosedAddress)
		}
	}
	s.mu.Unlock()

	if err := s.transport.Send(messages); err != nil {
		s.logger.Warn("Failed to send OSC bundle", zap.Error(err))
	}
}

// appendNative appends a native endpoint message when value differs from
// the one last sent there. Callers hold s.mu.
func (s *Sender) appendNative(messages []params.Message, address string, value any) []params.Message {
	if prev, ok := s.lastNative[address]; ok && valueEqual(prev, value) {
		return messages
	}
	s.lastNative[address] = value
	return append(messages, params.Message{Address: address, Value: value})
}

// valueEqual compares two message values. []byte is not comparable with ==
// (it would panic through any), so it is handled explicitly. Stored []byte
// values must not be reused by their producer.
func valueEqual(a, b any) bool {
	if x, ok := a.([]byte); ok {
		y, ok := b.([]byte)
		return ok && bytes.Equal(x, y)
	}
	return a == b
}

// SetDefinitions resets every parameter against a new avatar definition.
func (s *Sender) SetDefinitions(defs []params.Definition) {
	s.mu.Lock()
	s.defs = defs
	s.resetLocked()
	stats := s.stats
	s.mu.Unlock()
	if s.OnParametersChanged != nil {
		s.OnParametersChanged(stats)
	}
}

// AllRelevant reports whether every parameter is forced to be sent.
func (s *Sender) AllRelevant() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.allRelevant
}

// SetAllRelevant forces (or stops forcing) every parameter to be sent.
func (s *Sender) SetAllRelevant(v bool) {
	s.mu.Lock()
	if s.allRelevant == v {
		s.mu.Unlock()
		return
	}
	s.allRelevant = v
	s.resetLocked()
	stats := s.stats
	s.mu.Unlock()
	if s.OnParametersChanged != nil {
		s.OnParametersChanged(stats)
	}
}

// Stats returns the current parameter statistics.
func (s *Sender) Stats() ParameterStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stats
}

func (s *Sender) resetLocked() {
	s.outputs = params.ResetAll(s.all, s.defs, s.allRelevant)
	// A freshly loaded avatar gets every value once.
	s.last = make([]any, len(s.outputs))
	clear(s.lastNative)

	stats := ParameterStats{Count: len(s.outputs)}
	var deprecated []string
	for _, o := range s.outputs {
		if strings.HasPrefix(o.Address(), "/tracking/") {
			stats.HasNative = true
		}
		if o.Deprecated() {
			deprecated = append(deprecated, o.Name())
		}
	}
	stats.Deprecated = len(deprecated)
	s.stats = stats

	s.logger.Debug("Configuration loaded. Checking for native tracking parameters...")
	if stats.HasNative {
		s.logger.Warn("Native tracking parameters detected.")
	}
	s.logger.Info("Parameters loaded", zap.Int("count", stats.Count))
	if len(deprecated) > 0 {
		s.logger.Warn("Legacy parameters detected. Please consider updating the avatar to use the latest documented parameters.", zap.Int("count", len(deprecated)))
		s.logger.Debug("Loaded deprecated parameters", zap.Strings("parameters", deprecated))
	}
}
