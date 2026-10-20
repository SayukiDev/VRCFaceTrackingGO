// Package manager owns the lifecycle of tracking modules: initialising
// them, pumping Update on a timer and exposing the shared tracking data.
package manager

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"VRCFaceTracking/module/adapter"
	"VRCFaceTracking/module/modules"
	"VRCFaceTracking/pkg/tracking"
)

const (
	// DefaultUpdateInterval is the default time between two Update calls
	// of one module.
	DefaultUpdateInterval = 10 * time.Millisecond
	// DefaultInitTimeout bounds a single module's Initialize so a
	// misbehaving device driver cannot stall the whole application.
	DefaultInitTimeout = 15 * time.Second
)

// Manager drives the registered modules.
type Manager struct {
	logger  *zap.Logger
	modules map[string]adapter.Module

	dataMu sync.Mutex
	data   *tracking.TrackingData

	mu      sync.Mutex
	config  []Config // priority order; drives Initialize
	entries []*entry

	initMu sync.Mutex // serialises Initialize/Teardown

	eyeStatus        atomic.Int32
	expressionStatus atomic.Int32

	// updateInterval and initTimeout are stored as nanoseconds so the UI
	// can change them while module loops are running.
	updateInterval atomic.Int64
	initTimeout    atomic.Int64

	// OnChange is called whenever the module list or a module state changes.
	OnChange func([]ModuleInfo)
}

// New creates a manager for the given modules. The argument order is the
// default priority.
func New(logger *zap.Logger) *Manager {
	m := &Manager{
		logger:  logger.Named("LibManager"),
		modules: make(map[string]adapter.Module, len(modules.Modules)),
		data:    tracking.NewTrackingData(),
	}
	m.SetUpdateInterval(DefaultUpdateInterval)
	m.SetInitTimeout(DefaultInitTimeout)
	for _, mod := range modules.Modules {
		m.modules[mod.ID()] = mod
		m.config = append(m.config, defaultConfig(mod))
	}
	m.eyeStatus.Store(int32(adapter.Uninitialized))
	m.expressionStatus.Store(int32(adapter.Uninitialized))
	return m
}

// UpdateInterval returns the time between two Update calls of one module.
func (m *Manager) UpdateInterval() time.Duration { return time.Duration(m.updateInterval.Load()) }

// SetUpdateInterval changes the update interval. Running module loops pick
// the new value up on their next tick. Non-positive values are ignored.
func (m *Manager) SetUpdateInterval(d time.Duration) {
	if d <= 0 {
		return
	}
	m.updateInterval.Store(int64(d))
}

// InitTimeout returns the maximum time one module may spend in Initialize.
func (m *Manager) InitTimeout() time.Duration {
	return time.Duration(m.initTimeout.Load())
}

// SetInitTimeout changes the Initialize timeout used for subsequent
// (re)initialisations. Non-positive values are ignored.
func (m *Manager) SetInitTimeout(d time.Duration) {
	if d <= 0 {
		return
	}
	m.initTimeout.Store(int64(d))
}

func (m *Manager) EyeStatus() adapter.State {
	return adapter.State(m.eyeStatus.Load())
}

func (m *Manager) ExpressionStatus() adapter.State {
	return adapter.State(m.expressionStatus.Load())
}

// Snapshot copies the latest module output into dst.
func (m *Manager) Snapshot(dst *tracking.TrackingData) {
	m.dataMu.Lock()
	defer m.dataMu.Unlock()
	dst.CopyFrom(m.data)
}

type candidate struct {
	mod adapter.Module
	cfg Config
}

// candidates returns the enabled modules in priority order together with
// the config that limits what they may claim.
func (m *Manager) candidates() []candidate {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []candidate
	for _, c := range m.config {
		if mod, ok := m.modules[c.ID]; ok && c.Enabled {
			out = append(out, candidate{mod: mod, cfg: c})
		}
	}
	return out
}

// Initialize tears down any running modules and initialises every enabled
// module, in priority order, that can still contribute eye or expression
// tracking.
func (m *Manager) Initialize(ctx context.Context, moduleLogger *zap.Logger) {
	m.initMu.Lock()
	defer m.initMu.Unlock()

	m.Teardown()
	m.logger.Info("Starting initialization tracking")

	// Modules initialise one at a time, but without holding m.mu so the UI
	// can still query the manager while a slow device is probed.
	for _, c := range m.candidates() {
		if ctx.Err() != nil {
			return
		}
		mod := c.mod
		name := mod.Metadata().Name
		avail := adapter.Support{
			Eye:        c.cfg.Eye && m.EyeStatus() == adapter.Uninitialized,
			Expression: c.cfg.Expression && m.ExpressionStatus() == adapter.Uninitialized,
		}
		if !avail.Eye && !avail.Expression {
			m.logger.Debug("Skipping module, nothing left to provide", zap.String("module", name))
			continue
		}

		m.logger.Info("Initializing module", zap.String("module", name), zap.Bool("eye", avail.Eye), zap.Bool("expression", avail.Expression))
		granted, err := m.initializeModule(ctx, mod, moduleLogger, avail)
		if err != nil {
			m.logger.Warn("Module failed to initialize", zap.String("module", name), zap.Error(err))
			continue
		}
		if !granted.Eye && !granted.Expression {
			continue
		}

		if granted.Eye {
			m.eyeStatus.Store(int32(adapter.Active))
		}
		if granted.Expression {
			m.expressionStatus.Store(int32(adapter.Active))
		}

		meta := mod.Metadata()
		e := &entry{mod: mod, scratch: tracking.NewTrackingData(), info: ModuleInfo{
			ID:              mod.ID(),
			Name:            meta.Name,
			UsingEye:        granted.Eye,
			UsingExpression: granted.Expression,
			Icons:           encodeIcons(meta.Icons),
		}}
		e.setTracking(true, true)

		m.mu.Lock()
		m.startLocked(ctx, e)
		m.entries = append(m.entries, e)
		m.mu.Unlock()
		m.logger.Info("Tracking initialized via module", zap.String("module", meta.Name))
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.entries) == 0 {
		m.logger.Warn("No modules loaded.")
	}
	m.notifyLocked()
}

// initializeModule calls mod.Initialize with a timeout and converts a panic
// into an error. If the module ignores the timeout the call is abandoned
// (its goroutine leaks, but the manager keeps going).
func (m *Manager) initializeModule(ctx context.Context, mod adapter.Module, logger *zap.Logger, avail adapter.Support) (adapter.Support, error) {
	timeout := m.InitTimeout()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type result struct {
		granted adapter.Support
		err     error
	}
	done := make(chan result, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- result{err: fmt.Errorf("module panicked during Initialize: %v", r)}
			}
		}()
		granted, err := mod.Initialize(ctx, logger, avail)
		done <- result{granted: granted, err: err}
	}()

	select {
	case r := <-done:
		return r.granted, r.err
	case <-ctx.Done():
		return adapter.Support{}, fmt.Errorf("initialize did not finish within %s: %w", timeout, ctx.Err())
	}
}

func encodeIcons(icons [][]byte) []string {
	out := make([]string, 0, len(icons))
	for _, icon := range icons {
		out = append(out, "data:image/png;base64,"+base64.StdEncoding.EncodeToString(icon))
	}
	return out
}

func (m *Manager) startLocked(ctx context.Context, e *entry) {
	ctx, cancel := context.WithCancel(ctx)
	e.cancel = cancel
	e.done = make(chan struct{})
	go m.loop(ctx, e)
}

func (m *Manager) loop(ctx context.Context, e *entry) {
	defer close(e.done)
	m.logger.Debug("Starting thread for module", zap.String("module", e.info.Name))
	interval := m.UpdateInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.logger.Debug("Thread for module ended", zap.String("module", e.info.Name))
			return
		case <-ticker.C:
		}
		if d := m.UpdateInterval(); d != interval {
			interval = d
			ticker.Reset(interval)
		}
		if err := m.update(ctx, e); err != nil && ctx.Err() == nil {
			m.logger.Warn("Module update failed", zap.String("module", e.info.Name), zap.Error(err))
		}
	}
}

// update runs one Update call and publishes the result.
//
// The module writes into its own scratch buffer without any lock held, so
// a module that blocks while waiting for its device (VD waits on an event,
// PICO on a socket) never stalls the sender loop. Afterwards only the kinds
// the module was granted are copied into the shared data, which also lets
// two modules contribute eye and expression independently. Kinds that are
// switched off are published as neutral instead, so the module keeps running
// (instant resume) but its data does not reach the output.
func (m *Manager) update(ctx context.Context, e *entry) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("module panicked during Update: %v", r)
		}
	}()
	// With both kinds off the module is not polled at all, but the publish
	// step still runs so stale data is cleared right after switching off.
	if e.running() {
		err = e.mod.Update(ctx, e.scratch)
	}

	m.dataMu.Lock()
	defer m.dataMu.Unlock()
	if e.info.UsingEye {
		if e.eyeActive.Load() {
			m.data.Eye = e.scratch.Eye
		} else {
			m.data.Eye = neutralEye()
		}
	}
	if e.info.UsingExpression {
		if e.expressionActive.Load() {
			m.data.Shapes = e.scratch.Shapes
			m.data.Head = e.scratch.Head
		} else {
			m.data.Shapes = [tracking.ExpressionCount]tracking.Shape{}
			m.data.Head = tracking.HeadData{}
		}
	}
	return err
}

// neutralEye is what the output sees while eye tracking is switched off:
// eyes open, looking straight ahead.
func neutralEye() tracking.EyeData {
	d := tracking.NewTrackingData()
	d.Eye.Left.PupilDiameterMM = 5
	d.Eye.Right.PupilDiameterMM = 5
	d.Eye.MinDilation, d.Eye.MaxDilation = 0, 10
	return d.Eye
}

// Modules returns the UI view of every loaded module.
func (m *Manager) Modules() []ModuleInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.modulesLocked()
}

func (m *Manager) modulesLocked() []ModuleInfo {
	out := make([]ModuleInfo, 0, len(m.entries))
	for _, e := range m.entries {
		out = append(out, e.info)
	}
	return out
}

func (m *Manager) notifyLocked() {
	if m.OnChange != nil {
		m.OnChange(m.modulesLocked())
	}
}

// SetActive pauses or resumes every tracking kind of a loaded module.
func (m *Manager) SetActive(id string, active bool) { m.SetTracking(id, active, active) }

// SetTracking switches eye and expression output of a loaded module
// independently. The module itself keeps running either way.
func (m *Manager) SetTracking(id string, eye, expression bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.entries {
		if e.info.ID != id {
			continue
		}
		e.setTracking(eye, expression)
		m.logger.Info("Module tracking changed", zap.String("module", e.info.Name),
			zap.Bool("eye", e.info.EyeActive), zap.Bool("expression", e.info.ExpressionActive))
	}
	m.recomputeStatusLocked()
	m.notifyLocked()
}

// recomputeStatusLocked derives the global eye/expression status from the
// loaded modules: Active if any module passes that kind through, Idle if a
// module owns it but has it switched off, Uninitialized otherwise.
func (m *Manager) recomputeStatusLocked() {
	eye, expression := adapter.Uninitialized, adapter.Uninitialized
	for _, e := range m.entries {
		if e.info.UsingEye {
			eye = max(eye, stateOf(e.info.EyeActive))
		}
		if e.info.UsingExpression {
			expression = max(expression, stateOf(e.info.ExpressionActive))
		}
	}
	m.eyeStatus.Store(int32(eye))
	m.expressionStatus.Store(int32(expression))
}

func stateOf(active bool) adapter.State {
	if active {
		return adapter.Active
	}
	return adapter.Idle
}

// Teardown stops every loaded module.
func (m *Manager) Teardown() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.entries) > 0 {
		m.logger.Info("Tearing down all modules...")
	}
	for _, e := range m.entries {
		e.cancel()
		<-e.done
		if err := e.mod.Teardown(); err != nil {
			m.logger.Warn("Module teardown failed", zap.String("module", e.info.Name), zap.Error(err))
		}
	}
	m.entries = nil
	m.eyeStatus.Store(int32(adapter.Uninitialized))
	m.expressionStatus.Store(int32(adapter.Uninitialized))
	m.notifyLocked()
}
