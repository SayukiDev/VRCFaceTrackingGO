package router

import (
	"context"
	"encoding/base64"
	"fmt"
	"slices"
	"time"

	"go.uber.org/zap"

	"github.com/SayukiDev/VRCFaceTrackingGO/internal/module/manager"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
)

// ModuleInfo is the UI facing view of a loaded module.
type ModuleInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Active is true when at least one tracking kind is switched on.
	Active bool `json:"active"`
	// UsingEye / UsingExpression say what the module was granted; the
	// *Active flags say whether that data is currently passed through.
	UsingEye         bool `json:"usingEye"`
	UsingExpression  bool `json:"usingExpression"`
	EyeActive        bool `json:"eyeActive"`
	ExpressionActive bool `json:"expressionActive"`
	// Stale is set while an active module has not delivered new data for
	// a while (device off, stream stopped, ...).
	Stale bool     `json:"stale"`
	Icons []string `json:"icons"` // data: URIs
}

// entry is one running module. info is guarded by Router.mu.
type entry struct {
	mod  Module
	info ModuleInfo
}

// setTracking updates the per-kind switches; kinds the module was not
// granted stay off. Polling stops while both kinds are off.
func (e *entry) setTracking(eye, expression bool) {
	e.info.EyeActive = eye && e.info.UsingEye
	e.info.ExpressionActive = expression && e.info.UsingExpression
	e.info.Active = e.info.EyeActive || e.info.ExpressionActive
	if !e.info.Active {
		e.info.Stale = false
	}
	e.mod.SetPolling(e.info.Active)
}

// checkStale updates the Stale flags and notifies on changes.
func (r *Router) checkStale(now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	changed := false
	for _, e := range r.entries {
		stale := e.info.Active && now.Sub(e.mod.LastChange()) > staleAfter
		if stale != e.info.Stale {
			e.info.Stale = stale
			changed = true
			if stale {
				r.logger.Warn("Module is not receiving data", zap.String("module", e.info.Name))
			} else {
				r.logger.Info("Module is receiving data again", zap.String("module", e.info.Name))
			}
		}
	}
	if changed {
		r.notifyLocked()
	}
}

// Assignment manually picks the module for each tracking kind. An empty ID
// leaves that kind to the priority list. It lives only as long as the
// router.
type Assignment struct {
	Eye        string `json:"eye"`
	Expression string `json:"expression"`
}

type candidate struct {
	info manager.Info
	// wants is what the module may be asked for, before checking what
	// higher modules already took.
	wants adapter.Support
}

// candidates returns the modules worth starting in priority order. A
// manually assigned kind goes only to its module, which is started even
// when it is disabled in the list.
func (r *Router) candidates() []candidate {
	r.mu.Lock()
	defer r.mu.Unlock()
	a := r.assignment
	var out []candidate
	for _, c := range r.config {
		info, ok := r.infoLocked(c.ID)
		if !ok {
			continue
		}
		wants := adapter.Support{Eye: c.Enabled && c.Eye, Expression: c.Enabled && c.Expression}
		if a.Eye != "" {
			wants.Eye = a.Eye == c.ID
		}
		if a.Expression != "" {
			wants.Expression = a.Expression == c.ID
		}
		if wants.Eye || wants.Expression {
			out = append(out, candidate{info: info, wants: wants})
		}
	}
	return out
}

// Assignment returns the manual tracking assignment.
func (r *Router) Assignment() Assignment {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.assignment
}

// SetAssignment changes the manual tracking assignment. Each ID must name
// an installed module that supports the kind, or be empty. It does not
// re-initialise anything; call Initialize afterwards.
func (r *Router) SetAssignment(a Assignment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if a.Eye != "" {
		if info, ok := r.infoLocked(a.Eye); !ok || !info.Supported.Eye {
			return fmt.Errorf("module %q cannot provide eye tracking", a.Eye)
		}
	}
	if a.Expression != "" {
		if info, ok := r.infoLocked(a.Expression); !ok || !info.Supported.Expression {
			return fmt.Errorf("module %q cannot provide expression tracking", a.Expression)
		}
	}
	r.assignment = a
	return nil
}

// Refresh picks up the module manager's current list of installed modules
// and normalises the config and assignment against it.
func (r *Router) Refresh() {
	installed := r.host.Installed()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.registry = installed
	r.config = r.normalizeLocked(r.config)
	if _, ok := r.infoLocked(r.assignment.Eye); !ok {
		r.assignment.Eye = ""
	}
	if _, ok := r.infoLocked(r.assignment.Expression); !ok {
		r.assignment.Expression = ""
	}
}

// Initialize stops every running module and starts the enabled ones, in
// priority order, as long as they can still contribute eye or expression
// tracking that no higher module took.
func (r *Router) Initialize(ctx context.Context) {
	r.initMu.Lock()
	defer r.initMu.Unlock()

	r.Teardown()
	r.Refresh()
	r.logger.Info("Starting initialization tracking")
	// Modules start one at a time, but without holding r.mu so the UI can
	// still query the router while a slow device is probed.
	for _, c := range r.candidates() {
		if ctx.Err() != nil {
			return
		}
		name := c.info.Metadata.Name
		avail := adapter.Support{
			Eye:        c.wants.Eye && r.EyeStatus() == adapter.Uninitialized,
			Expression: c.wants.Expression && r.ExpressionStatus() == adapter.Uninitialized,
		}
		if !avail.Eye && !avail.Expression {
			r.logger.Debug("Skipping module, nothing left to provide", zap.String("module", name))
			continue
		}

		r.logger.Info("Initializing module", zap.String("module", name), zap.Bool("eye", avail.Eye), zap.Bool("expression", avail.Expression))
		mod, err := r.host.Start(ctx, c.info.ID, avail)
		if err != nil {
			r.logger.Warn("Module failed to initialize", zap.String("module", name), zap.Error(err))
			continue
		}
		granted := mod.Granted()
		if !granted.Eye && !granted.Expression {
			mod.Stop()
			continue
		}
		if granted.Eye {
			r.eyeStatus.Store(int32(adapter.Active))
		}
		if granted.Expression {
			r.expressionStatus.Store(int32(adapter.Active))
		}

		e := &entry{mod: mod, info: ModuleInfo{
			ID:              c.info.ID,
			Name:            name,
			UsingEye:        granted.Eye,
			UsingExpression: granted.Expression,
			Icons:           encodeIcons(c.info.Metadata.Icons),
		}}
		e.setTracking(true, true)

		r.mu.Lock()
		r.entries = append(r.entries, e)
		r.mu.Unlock()
		go r.watch(e)
		r.logger.Info("Tracking initialized via module", zap.String("module", name))
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.entries) == 0 {
		r.logger.Warn("No modules loaded.")
	}
	r.notifyLocked()
}

func encodeIcons(icons [][]byte) []string {
	out := make([]string, 0, len(icons))
	for _, icon := range icons {
		out = append(out, "data:image/png;base64,"+base64.StdEncoding.EncodeToString(icon))
	}
	return out
}

// watch drops a module whose process died.
func (r *Router) watch(e *entry) {
	<-e.mod.Done()
	if e.mod.Crashed() && r.remove(e) {
		e.mod.Stop()
	}
}

// remove takes e out of the running set. It reports false if e was not
// running any more.
func (r *Router) remove(e *entry) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	i := slices.Index(r.entries, e)
	if i < 0 {
		return false
	}
	r.entries = slices.Delete(r.entries, i, i+1)
	r.recomputeStatusLocked()
	r.notifyLocked()
	return true
}

// Unload stops one running module, e.g. before it is replaced or removed.
func (r *Router) Unload(id string) {
	r.mu.Lock()
	var e *entry
	if i := slices.IndexFunc(r.entries, func(e *entry) bool { return e.info.ID == id }); i >= 0 {
		e = r.entries[i]
	}
	r.mu.Unlock()
	if e != nil && r.remove(e) {
		r.logger.Info("Module unloaded", zap.String("module", e.info.Name))
		e.mod.Stop()
	}
}

// Modules returns the UI view of every loaded module.
func (r *Router) Modules() []ModuleInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.modulesLocked()
}

func (r *Router) modulesLocked() []ModuleInfo {
	out := make([]ModuleInfo, 0, len(r.entries))
	for _, e := range r.entries {
		out = append(out, e.info)
	}
	return out
}

func (r *Router) notifyLocked() {
	if r.OnChange != nil {
		r.OnChange(r.modulesLocked())
	}
}

// SetActive pauses or resumes every tracking kind of a loaded module.
func (r *Router) SetActive(id string, active bool) { r.SetTracking(id, active, active) }

// SetTracking switches eye and expression output of a loaded module
// independently. The module itself keeps running either way.
func (r *Router) SetTracking(id string, eye, expression bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.entries {
		if e.info.ID != id {
			continue
		}
		e.setTracking(eye, expression)
		r.logger.Info("Module tracking changed", zap.String("module", e.info.Name),
			zap.Bool("eye", e.info.EyeActive), zap.Bool("expression", e.info.ExpressionActive))
	}
	r.recomputeStatusLocked()
	r.notifyLocked()
}

// recomputeStatusLocked derives the global eye/expression status from the
// loaded modules: Active if any module passes that kind through, Idle if a
// module owns it but has it switched off, Uninitialized otherwise.
func (r *Router) recomputeStatusLocked() {
	eye, expression := adapter.Uninitialized, adapter.Uninitialized
	for _, e := range r.entries {
		if e.info.UsingEye {
			eye = max(eye, stateOf(e.info.EyeActive))
		}
		if e.info.UsingExpression {
			expression = max(expression, stateOf(e.info.ExpressionActive))
		}
	}
	r.eyeStatus.Store(int32(eye))
	r.expressionStatus.Store(int32(expression))
}

func stateOf(active bool) adapter.State {
	if active {
		return adapter.Active
	}
	return adapter.Idle
}

// Teardown stops every running module.
func (r *Router) Teardown() {
	r.mu.Lock()
	entries := r.entries
	r.entries = nil
	r.eyeStatus.Store(int32(adapter.Uninitialized))
	r.expressionStatus.Store(int32(adapter.Uninitialized))
	r.notifyLocked()
	r.mu.Unlock()

	if len(entries) > 0 {
		r.logger.Info("Tearing down all modules...")
	}
	// Stopping can take a while; the output loop keeps running meanwhile.
	for _, e := range entries {
		e.mod.Stop()
	}
}
