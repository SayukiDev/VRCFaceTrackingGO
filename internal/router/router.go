// Package router decides which installed modules run and what they are
// used for, merges their output into one set of tracking data and
// periodically passes it through the mixer to the sender. Running the
// module programs themselves is the module manager's job.
package router

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"VRCFaceTracking/internal/mixer"
	"VRCFaceTracking/internal/module/adapter"
	"VRCFaceTracking/internal/module/manager"
	"VRCFaceTracking/pkg/tracking"
)

// DefaultSendInterval is the default time between two send passes.
const DefaultSendInterval = 10 * time.Millisecond

const (
	// staleAfter is how long a module's output may stay unchanged before
	// it is reported as not receiving data.
	staleAfter = 10 * time.Second
	// staleCheckInterval is how often that is checked.
	staleCheckInterval = time.Second
)

// Host is what the router needs from the module manager.
type Host interface {
	Installed() []manager.Info
	Start(ctx context.Context, id string, avail adapter.Support) (Module, error)
}

// Module is a started module program.
type Module interface {
	Granted() adapter.Support
	Latest() tracking.TrackingData
	LastChange() time.Time
	SetPolling(bool)
	Done() <-chan struct{}
	Crashed() bool
	Stop()
}

// managerHost adapts *manager.Manager to Host.
type managerHost struct{ *manager.Manager }

func (h managerHost) Start(ctx context.Context, id string, avail adapter.Support) (Module, error) {
	inst, err := h.Manager.Start(ctx, id, avail)
	if err != nil {
		return nil, err
	}
	return inst, nil
}

// Sink receives the mixed tracking data every send interval.
type Sink interface {
	Send(data *tracking.TrackingData)
}

// Router chooses the running modules and owns the output loop.
type Router struct {
	logger *zap.Logger
	mixer  *mixer.Mixer
	host   Host

	mu         sync.Mutex
	registry   []manager.Info // installed modules, nil until the first Refresh
	config     []Config       // priority order; drives Initialize
	assignment Assignment     // manual override of config, not persisted
	entries    []*entry

	initMu sync.Mutex // serialises Initialize/Teardown

	eyeStatus        atomic.Int32
	expressionStatus atomic.Int32

	// sendInterval is stored as nanoseconds so the UI can change it while
	// the loop is running.
	sendInterval atomic.Int64

	// OnChange is called whenever the module list or a module state changes.
	OnChange func([]ModuleInfo)
}

// New creates a router that runs the modules installed in mgr.
func New(logger *zap.Logger, mix *mixer.Mixer, mgr *manager.Manager) *Router {
	return newRouter(logger, mix, managerHost{mgr})
}

func newRouter(logger *zap.Logger, mix *mixer.Mixer, host Host) *Router {
	r := &Router{
		logger: logger.Named("Router"),
		mixer:  mix,
		host:   host,
	}
	r.SetSendInterval(DefaultSendInterval)
	r.eyeStatus.Store(int32(adapter.Uninitialized))
	r.expressionStatus.Store(int32(adapter.Uninitialized))
	return r
}

// SendInterval returns the time between two send passes.
func (r *Router) SendInterval() time.Duration { return time.Duration(r.sendInterval.Load()) }

// SetSendInterval changes the send interval. The running loop picks the
// new value up on its next tick. Non-positive values are ignored.
func (r *Router) SetSendInterval(d time.Duration) {
	if d > 0 {
		r.sendInterval.Store(int64(d))
	}
}

func (r *Router) EyeStatus() adapter.State { return adapter.State(r.eyeStatus.Load()) }

func (r *Router) ExpressionStatus() adapter.State {
	return adapter.State(r.expressionStatus.Load())
}

// Snapshot merges the latest output of the running modules into dst.
//
// Every module contributes only the kinds it was granted, which lets two
// modules provide eye and expression independently. Kinds that are
// switched off are neutral instead, so the module keeps running (instant
// resume) but its data does not reach the output.
func (r *Router) Snapshot(dst *tracking.TrackingData) {
	*dst = *tracking.NewTrackingData()
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.entries {
		if !e.info.UsingEye && !e.info.UsingExpression {
			continue
		}
		latest := e.mod.Latest()
		if e.info.UsingEye {
			if e.info.EyeActive {
				dst.Eye = latest.Eye
			} else {
				dst.Eye = neutralEye()
			}
		}
		if e.info.UsingExpression && e.info.ExpressionActive {
			dst.Shapes = latest.Shapes
			dst.Head = latest.Head
		}
	}
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

// Run sends the mixed module output to sink until ctx is cancelled.
func (r *Router) Run(ctx context.Context, sink Sink) {
	data := tracking.NewTrackingData()
	interval := r.SendInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	staleTicker := time.NewTicker(staleCheckInterval)
	defer staleTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-staleTicker.C:
			r.checkStale(now)
			continue
		case <-ticker.C:
		}
		if d := r.SendInterval(); d != interval {
			interval = d
			ticker.Reset(interval)
		}
		r.Snapshot(data)
		sink.Send(r.mixer.Mix(data))
	}
}
