// Package engine runs the main output loop: snapshot module data, mutate
// it, evaluate the relevant parameters and send the changed values as OSC.
package engine

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"VRCFaceTracking/internal/mutation"
	"VRCFaceTracking/internal/osc"
	"VRCFaceTracking/internal/params"
	"VRCFaceTracking/module/manager"
	"VRCFaceTracking/pkg/tracking"
)

// DefaultSendInterval is the default time between two OSC send passes.
const DefaultSendInterval = 10 * time.Millisecond

// ParameterStats summarises the relevant parameter set for the UI.
type ParameterStats struct {
	Count      int  `json:"count"`
	Deprecated int  `json:"deprecated"`
	HasNative  bool `json:"hasNative"`
}

// Engine is the parameter sender loop.
type Engine struct {
	logger  *zap.Logger
	manager *manager.Manager
	mutator *mutation.Mutator
	sender  *osc.Sender

	all  []params.Parameter
	data *tracking.TrackingData

	// sendInterval is stored as nanoseconds so the UI can change it while
	// the loop is running.
	sendInterval atomic.Int64

	mu          sync.Mutex
	defs        []params.Definition
	outputs     []params.Output
	allRelevant bool
	stats       ParameterStats

	// OnParametersChanged is called after every Reset with the new stats.
	OnParametersChanged func(ParameterStats)
}

func New(logger *zap.Logger, mgr *manager.Manager, mutator *mutation.Mutator, sender *osc.Sender) *Engine {
	e := &Engine{
		logger:  logger.Named("ParameterSender"),
		manager: mgr,
		mutator: mutator,
		sender:  sender,
		all:     params.AllParameters(mgr),
		data:    tracking.NewTrackingData(),
	}
	e.SetSendInterval(DefaultSendInterval)
	return e
}

// SendInterval returns the time between two send passes.
func (e *Engine) SendInterval() time.Duration { return time.Duration(e.sendInterval.Load()) }

// SetSendInterval changes the send interval. The running loop picks the new
// value up on its next tick. Non-positive values are ignored.
func (e *Engine) SetSendInterval(d time.Duration) {
	if d <= 0 {
		return
	}
	e.sendInterval.Store(int64(d))
}

// Run blocks until ctx is cancelled.
func (e *Engine) Run(ctx context.Context) {
	interval := e.SendInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if d := e.SendInterval(); d != interval {
			interval = d
			ticker.Reset(interval)
		}
		e.tick()
	}
}

func (e *Engine) tick() {
	e.manager.Snapshot(e.data)
	mutated := e.mutator.Mutate(e.data)

	e.mu.Lock()
	var messages []params.Message
	for _, out := range e.outputs {
		if msg, ok := out.Process(mutated); ok {
			messages = append(messages, msg)
		}
	}
	e.mu.Unlock()

	if err := e.sender.Send(messages); err != nil {
		e.logger.Warn("Failed to send OSC bundle", zap.Error(err))
	}
}

// SetDefinitions resets every parameter against a new avatar definition.
func (e *Engine) SetDefinitions(defs []params.Definition) {
	e.mu.Lock()
	e.defs = defs
	e.resetLocked()
	stats := e.stats
	e.mu.Unlock()
	if e.OnParametersChanged != nil {
		e.OnParametersChanged(stats)
	}
}

// AllRelevant reports whether every parameter is forced to be sent.
func (e *Engine) AllRelevant() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.allRelevant
}

// SetAllRelevant forces (or stops forcing) every parameter to be sent.
func (e *Engine) SetAllRelevant(v bool) {
	e.mu.Lock()
	if e.allRelevant == v {
		e.mu.Unlock()
		return
	}
	e.allRelevant = v
	e.resetLocked()
	stats := e.stats
	e.mu.Unlock()
	if e.OnParametersChanged != nil {
		e.OnParametersChanged(stats)
	}
}

// Stats returns the current parameter statistics.
func (e *Engine) Stats() ParameterStats {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.stats
}

func (e *Engine) resetLocked() {
	e.outputs = params.ResetAll(e.all, e.defs, e.allRelevant)

	stats := ParameterStats{Count: len(e.outputs)}
	var deprecated []string
	for _, o := range e.outputs {
		if strings.HasPrefix(o.Address(), "/tracking/") {
			stats.HasNative = true
		}
		if o.Deprecated() {
			deprecated = append(deprecated, o.Name())
		}
	}
	stats.Deprecated = len(deprecated)
	e.stats = stats

	e.logger.Debug("Configuration loaded. Checking for native tracking parameters...")
	if stats.HasNative {
		e.logger.Warn("Native tracking parameters detected.")
	}
	e.logger.Info("Parameters loaded", zap.Int("count", stats.Count))
	if len(deprecated) > 0 {
		e.logger.Warn("Legacy parameters detected. Please consider updating the avatar to use the latest documented parameters.", zap.Int("count", len(deprecated)))
		e.logger.Debug("Loaded deprecated parameters", zap.Strings("parameters", deprecated))
	}
}
