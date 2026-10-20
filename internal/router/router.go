// Package router runs the main output loop: it snapshots the module data,
// passes it through the mixer and hands the result to the sender.
package router

import (
	"context"
	"sync/atomic"
	"time"

	"VRCFaceTracking/internal/mixer"
	"VRCFaceTracking/internal/sender"
	"VRCFaceTracking/module/manager"
	"VRCFaceTracking/pkg/tracking"
)

// DefaultInterval is the default time between two routing passes.
const DefaultInterval = 10 * time.Millisecond

// Router connects the module manager, the mixer and the sender.
type Router struct {
	manager *manager.Manager
	mixer   *mixer.Mixer
	sender  *sender.Sender

	data *tracking.TrackingData

	// interval is stored as nanoseconds so the UI can change it while the
	// loop is running.
	interval atomic.Int64
}

func New(mgr *manager.Manager, mix *mixer.Mixer, snd *sender.Sender) *Router {
	r := &Router{
		manager: mgr,
		mixer:   mix,
		sender:  snd,
		data:    tracking.NewTrackingData(),
	}
	r.SetInterval(DefaultInterval)
	return r
}

// Interval returns the time between two routing passes.
func (r *Router) Interval() time.Duration { return time.Duration(r.interval.Load()) }

// SetInterval changes the routing interval. The running loop picks the new
// value up on its next tick. Non-positive values are ignored.
func (r *Router) SetInterval(d time.Duration) {
	if d <= 0 {
		return
	}
	r.interval.Store(int64(d))
}

// Run blocks until ctx is cancelled.
func (r *Router) Run(ctx context.Context) {
	interval := r.Interval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if d := r.Interval(); d != interval {
			interval = d
			ticker.Reset(interval)
		}
		r.tick()
	}
}

func (r *Router) tick() {
	r.manager.Snapshot(r.data)
	r.sender.Send(r.mixer.Mix(r.data))
}
