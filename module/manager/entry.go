package manager

import (
	"VRCFaceTracking/module/adapter"
	"VRCFaceTracking/pkg/tracking"
	"context"
	"sync/atomic"
)

// ModuleInfo is the UI facing view of a loaded module.
type ModuleInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Active is true when at least one tracking kind is switched on.
	Active bool `json:"active"`
	// UsingEye / UsingExpression say what the module was granted; the
	// *Active flags say whether that data is currently passed through.
	UsingEye         bool     `json:"usingEye"`
	UsingExpression  bool     `json:"usingExpression"`
	EyeActive        bool     `json:"eyeActive"`
	ExpressionActive bool     `json:"expressionActive"`
	Icons            []string `json:"icons"` // data: URIs
}

type entry struct {
	mod    adapter.Module
	info   ModuleInfo
	cancel context.CancelFunc
	done   chan struct{}
	// scratch is the module's private output buffer; see Manager.update.
	scratch *tracking.TrackingData

	// Read by the update goroutine without taking m.mu.
	eyeActive        atomic.Bool
	expressionActive atomic.Bool
}

// running reports whether Update should still be called.
func (e *entry) running() bool { return e.eyeActive.Load() || e.expressionActive.Load() }

// setTracking updates the per-kind switches; kinds the module was not
// granted stay off.
func (e *entry) setTracking(eye, expression bool) {
	eye = eye && e.info.UsingEye
	expression = expression && e.info.UsingExpression
	e.eyeActive.Store(eye)
	e.expressionActive.Store(expression)
	e.info.EyeActive = eye
	e.info.ExpressionActive = expression
	e.info.Active = eye || expression
}
