package manager

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/plugin"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

const (
	// teardownTimeout bounds a single module's Teardown call.
	teardownTimeout = 3 * time.Second
	// errorLogInterval is how often the same Update error is logged again.
	errorLogInterval = 10 * time.Second
)

// Restart policy for crashed or hung module programs: at most maxRestarts
// failures within restartWindow are retried, waiting restartDelay, then
// twice as long, and so on. More failures give up for good.
const (
	maxRestarts   = 3
	restartWindow = time.Minute
)

// Instance is a running module program. It polls the module for data in
// the background, keeps the latest result, and restarts the program when
// it crashes or stops responding.
type Instance struct {
	manager *Manager
	logger  *zap.Logger
	info    Info
	granted adapter.Support
	// conn is only replaced by the loop goroutine, and only read by Stop
	// after the loop ended.
	conn plugin.Conn

	cancel context.CancelFunc
	done   chan struct{}

	polling atomic.Bool
	crashed atomic.Bool
	// lastChange is when the module's output last changed (unix nanos).
	lastChange atomic.Int64

	mu     sync.Mutex
	latest tracking.TrackingData

	stopOnce sync.Once
}

func newInstance(m *Manager, info Info, conn plugin.Conn, granted adapter.Support) *Instance {
	logger := m.logger.With(zap.String("module", info.Metadata.Name))
	if info.Driver != "" {
		logger = logger.With(zap.String("driver", info.Driver))
	}
	inst := &Instance{
		manager: m,
		logger:  logger,
		info:    info,
		granted: granted,
		conn:    conn,
		done:    make(chan struct{}),
		latest:  *tracking.NewTrackingData(),
	}
	inst.polling.Store(true)
	inst.lastChange.Store(time.Now().UnixNano())
	return inst
}

func (i *Instance) Info() Info { return i.info }

// Granted reports what the module agreed to provide.
func (i *Instance) Granted() adapter.Support { return i.granted }

// Latest returns the module's most recent data.
func (i *Instance) Latest() tracking.TrackingData {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.latest
}

// LastChange returns when the module's output last changed. Live tracking
// data always jitters, so output that stays identical means no data is
// arriving even if Update succeeds.
func (i *Instance) LastChange() time.Time { return time.Unix(0, i.lastChange.Load()) }

func (i *Instance) setLatest(d tracking.TrackingData) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if d != i.latest {
		i.latest = d
		i.lastChange.Store(time.Now().UnixNano())
	}
}

// SetPolling pauses or resumes Update calls. The process keeps running so
// resuming is instant.
func (i *Instance) SetPolling(v bool) {
	if v && !i.polling.Load() {
		// The pause is not a data gap.
		i.lastChange.Store(time.Now().UnixNano())
	}
	i.polling.Store(v)
}

// Done is closed when the instance stopped, either by Stop or because the
// program kept crashing and was given up.
func (i *Instance) Done() <-chan struct{} { return i.done }

// Crashed reports whether the instance was given up after crashing.
func (i *Instance) Crashed() bool { return i.crashed.Load() }

func (i *Instance) start(ctx context.Context) {
	ctx, i.cancel = context.WithCancel(ctx)
	go i.loop(ctx)
}

func (i *Instance) loop(ctx context.Context) {
	defer close(i.done)
	i.logger.Debug("Starting thread for module")
	var failures []time.Time
	for {
		err := i.poll(ctx)
		if ctx.Err() != nil {
			i.logger.Debug("Thread for module ended")
			return
		}
		_ = i.conn.Close()
		// Do not freeze the avatar on the last frame while restarting.
		i.mu.Lock()
		i.latest = *tracking.NewTrackingData()
		i.mu.Unlock()

		for {
			now := time.Now()
			failures = append(failures, now)
			failures = recent(failures, now.Add(-restartWindow))
			if len(failures) > maxRestarts {
				i.logger.Error("Module keeps failing, giving up", zap.Error(err), zap.Int("failures", len(failures)))
				i.crashed.Store(true)
				i.manager.release(i)
				return
			}
			delay := i.manager.restartDelay << (len(failures) - 1)
			i.logger.Error("Module failed, restarting", zap.Error(err), zap.Duration("in", delay))
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
			if err = i.restart(ctx); err == nil {
				i.logger.Info("Module restarted")
				break
			}
			if ctx.Err() != nil {
				return
			}
		}
	}
}

// recent drops the times before since.
func recent(times []time.Time, since time.Time) []time.Time {
	out := times[:0]
	for _, t := range times {
		if t.After(since) {
			out = append(out, t)
		}
	}
	return out
}

// poll calls Update every update interval until ctx is done (nil) or the
// program died or hung (the reason).
func (i *Instance) poll(ctx context.Context) error {
	interval := i.manager.UpdateInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastErr string
	var lastLog time.Time
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
		if d := i.manager.UpdateInterval(); d != interval {
			interval = d
			ticker.Reset(interval)
		}
		if !i.polling.Load() {
			continue
		}

		timeout := i.manager.updateTimeout
		uctx, cancel := context.WithTimeout(ctx, timeout)
		d, err := i.conn.Update(uctx)
		cancel()
		switch {
		case err == nil:
			i.setLatest(d)
			lastErr = ""
		case ctx.Err() != nil:
			return nil
		case errors.Is(err, plugin.ErrExited):
			return err
		case errors.Is(err, context.DeadlineExceeded):
			return fmt.Errorf("module did not respond within %s", timeout)
		default:
			// Modules may fail every tick (e.g. no device); do not flood the log.
			if msg := err.Error(); msg != lastErr || time.Since(lastLog) >= errorLogInterval {
				i.logger.Warn("Module update failed", zap.Error(err))
				lastErr, lastLog = msg, time.Now()
			}
		}
	}
}

// restart starts the program again and asks for what it had before.
func (i *Instance) restart(ctx context.Context) error {
	conn, err := i.manager.launch(i.info.Path, i.info.source())
	if err != nil {
		return err
	}
	ictx, cancel := context.WithTimeout(ctx, i.manager.InitTimeout())
	defer cancel()
	granted, err := conn.Initialize(ictx, i.granted)
	if err == nil && (granted.Eye != i.granted.Eye || granted.Expression != i.granted.Expression) {
		err = fmt.Errorf("module now provides %+v instead of %+v", granted, i.granted)
	}
	if err != nil {
		_ = conn.Close()
		return err
	}
	i.conn = conn
	return nil
}

// Stop tears the module down and ends its process. It is safe to call more
// than once and after the instance was given up.
func (i *Instance) Stop() {
	i.stopOnce.Do(func() {
		i.cancel()
		<-i.done
		if !i.Crashed() {
			ctx, cancel := context.WithTimeout(context.Background(), teardownTimeout)
			defer cancel()
			if err := i.conn.Teardown(ctx); err != nil && !errors.Is(err, plugin.ErrExited) {
				i.logger.Warn("Module teardown failed", zap.Error(err))
			}
		}
		if err := i.conn.Close(); err != nil {
			i.logger.Warn("Failed to stop module", zap.Error(err))
		}
		i.manager.release(i)
	})
}
