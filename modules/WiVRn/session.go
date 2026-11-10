package main

import (
	"errors"
	"runtime"
	"time"

	"go.uber.org/zap"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

const (
	// pollInterval is how often the runtime is asked for new values. Asking
	// is also what makes the headset keep its face tracker running.
	pollInterval = 10 * time.Millisecond

	// retryDelay is the wait before connecting again after a failure. The
	// runtime only has a system to offer while a headset is connected.
	retryDelay = 5 * time.Second

	// maxReadFailures is how many reads in a row may fail before the session
	// is torn down and opened again.
	maxReadFailures = 50
)

// errLost reports that the runtime dropped the instance or session; the
// only way forward is to connect again.
var errLost = errors.New("WiVRn: OpenXR session lost")

// features is what a connected runtime offers.
type features struct {
	runtime string
	system  string

	eyeGaze bool
	fb2     bool
	htcEye  bool
	htcLip  bool
	android bool
}

func (f features) any() bool {
	return f.eyeGaze || f.fb2 || f.htcEye || f.htcLip || f.android
}

// frame is the raw data of one poll. A part is only meaningful while its
// valid flag is set.
type frame struct {
	fb2      [fbFace2Count]float32
	fb2Valid bool

	android      [androidCount]float32
	androidValid bool

	htcEye      [htcEyesCount]float32
	htcEyeValid bool
	htcLip      [htcLipCount]float32
	htcLipValid bool

	// gaze is the combined gaze orientation (x, y, z, w) in view space.
	gaze      [4]float32
	gazeValid bool
}

// apply converts f into Unified Expressions and eye data.
func (f *frame) apply(data *tracking.TrackingData) {
	if f.fb2Valid {
		setFace2Fb(f.fb2[:], data)
	}
	if f.androidValid {
		setFaceAndroid(f.android[:], data)
	}
	if f.htcEyeValid {
		setEyesHtc(f.htcEye[:], data)
	}
	if f.htcLipValid {
		setLipHtc(f.htcLip[:], data)
	}
	// The gaze pose is more precise than the HTC look weights, so it wins.
	if f.gazeValid {
		setCombEyesQuat(f.gaze[:], data)
	}
}

type sessionEvent int

const (
	// eventNone means the event queue is empty.
	eventNone sessionEvent = iota
	// eventOther is an event this module does not act on.
	eventOther
	// eventReady asks for the session to begin.
	eventReady
	// eventStopping asks for the session to end.
	eventStopping
	// eventEnded means the session is gone and must be opened again.
	eventEnded
)

// xrAPI is the part of OpenXR the session loop needs. Every method is
// called from the one goroutine running Module.run.
type xrAPI interface {
	// connect creates the instance, a headless session and the trackers.
	connect() (features, error)
	pollEvent() (sessionEvent, error)
	begin() error
	end() error
	// read fills f with the newest values.
	read(f *frame) error
	// disconnect releases whatever connect created. It is safe to call
	// after a failed or missing connect.
	disconnect()
}

// run connects, polls and reconnects until stop is closed. It owns xr;
// Teardown may give up waiting for it while it is blocked in the runtime.
func (m *Module) run(xr xrAPI) {
	defer close(m.done)
	// OpenXR handles are used from a single thread, and the runtime may
	// keep per-thread state.
	runtime.LockOSThread()

	var lastErr string
	for {
		err := m.serve(xr, &lastErr)
		xr.disconnect()
		if err == nil {
			return
		}
		m.mu.Lock()
		m.latest = *tracking.NewTrackingData()
		m.mu.Unlock()

		// The same failure repeats every retry while no headset is
		// connected; say it once.
		if msg := err.Error(); msg != lastErr {
			lastErr = msg
			m.logger.Warn("Not connected to WiVRn, retrying", zap.Error(err), zap.Duration("every", m.retryDelay))
		}
		select {
		case <-m.stop:
			return
		case <-time.After(m.retryDelay):
		}
	}
}

// serve runs one connection. It returns nil once stop is closed and the
// reason otherwise.
func (m *Module) serve(xr xrAPI, lastErr *string) error {
	feat, err := xr.connect()
	if err != nil {
		return err
	}
	if !feat.any() {
		return errors.New("WiVRn: the headset reports no face or eye tracking; enable them in the WiVRn headset app")
	}
	*lastErr = ""
	m.logger.Info("Connected",
		zap.String("runtime", feat.runtime), zap.String("system", feat.system),
		zap.Bool("eyeGaze", feat.eyeGaze), zap.Bool("fb2", feat.fb2),
		zap.Bool("htcEye", feat.htcEye), zap.Bool("htcLip", feat.htcLip),
		zap.Bool("android", feat.android))

	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	var (
		f        frame
		running  bool
		failures int
	)
	for {
		select {
		case <-m.stop:
			return nil
		case <-ticker.C:
		}

		for {
			ev, err := xr.pollEvent()
			if err != nil {
				return err
			}
			if ev == eventNone {
				break
			}
			switch ev {
			case eventReady:
				if err := xr.begin(); err != nil {
					return err
				}
				running = true
			case eventStopping:
				running = false
				if err := xr.end(); err != nil {
					return err
				}
			case eventEnded:
				return errLost
			}
		}
		if !running {
			continue
		}

		if err := xr.read(&f); err != nil {
			failures++
			if errors.Is(err, errLost) || failures >= maxReadFailures {
				return err
			}
			continue
		}
		failures = 0
		m.mu.Lock()
		f.apply(&m.latest)
		m.mu.Unlock()
	}
}
