// Command VirtualDesktop is the tracking module program that reads face and
// eye data published by the Virtual Desktop Streamer through a Windows
// shared memory section. It follows the official VirtualDesktop.FaceTracking
// module v1.5.
package main

import (
	"context"
	"embed"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

const (
	bodyStateMapName   = "VirtualDesktop.BodyState"
	bodyStateEventName = "VirtualDesktop.BodyStateEvent"

	// frameTimeout bounds one wait for a frame. The Streamer signals at the
	// headset's tracking rate, so a timeout only happens when nothing is
	// being forwarded.
	frameTimeout = 50 * time.Millisecond
	// silenceWarnAfter is how long tracking may be silent before the
	// "not active" hint is logged.
	silenceWarnAfter = 5 * time.Second
)

// files holds metadata.json and the icons it names.
//
//go:embed metadata.json logo256.png
var files embed.FS

var metadata = adapter.MustLoadMetadata(files, "metadata.json")

// Module reads tracking data from Virtual Desktop.
type Module struct {
	logger  *zap.Logger
	shm     *sharedState
	granted adapter.Support
	stop    chan struct{}
	done    chan struct{}

	// Owned by the receive goroutine.
	tracking     bool
	lastFrameAt  time.Time
	silenceAlert bool

	mu     sync.Mutex
	latest tracking.TrackingData
}

var _ adapter.Module = (*Module)(nil)

// New returns an uninitialised Virtual Desktop module.
func New() *Module { return &Module{} }

func (m *Module) ID() string { return "VirtualDesktop" }

func (m *Module) Metadata() adapter.Metadata { return metadata }

func (m *Module) Supported() adapter.Support {
	return adapter.Support{Eye: true, Expression: true}
}

func (m *Module) Initialize(_ context.Context, logger *zap.Logger, avail adapter.Support) (adapter.Support, error) {
	m.logger = logger.Named("VirtualDesktop")

	shm, err := openSharedState()
	if err != nil {
		m.logger.Error("Failed to open shared memory. Make sure the Virtual Desktop Streamer (v1.30 or later) is running.", zap.Error(err))
		return adapter.Support{}, err
	}

	m.shm = shm
	m.granted = avail
	m.tracking = false
	m.lastFrameAt = time.Now()
	m.silenceAlert = false
	m.latest = *tracking.NewTrackingData()
	m.stop, m.done = make(chan struct{}), make(chan struct{})
	go m.receive()
	return m.granted, nil
}

// receive converts every frame into the latest values until stop is
// closed. Without a frame it only refreshes the tracking status.
func (m *Module) receive() {
	defer close(m.done)
	for {
		select {
		case <-m.stop:
			return
		default:
		}

		gotFrame, err := m.shm.waitFrame(frameTimeout)
		if err != nil {
			m.logger.Warn("Waiting for a frame failed", zap.Error(err))
			select {
			case <-m.stop:
				return
			case <-time.After(frameTimeout):
			}
			continue
		}

		state := m.shm.snapshot()
		if !gotFrame {
			m.setTracking(state.anyValid())
			continue
		}
		m.lastFrameAt = time.Now()
		m.mu.Lock()
		tracked := m.convert(&state, &m.latest)
		m.mu.Unlock()
		m.setTracking(tracked)
	}
}

// Update writes the latest converted frame into data without waiting.
func (m *Module) Update(_ context.Context, data *tracking.TrackingData) error {
	if m.shm == nil {
		return errors.New("VirtualDesktop: module not initialised")
	}
	m.mu.Lock()
	data.CopyFrom(&m.latest)
	m.mu.Unlock()
	return nil
}

// convert maps one frame into data and reports whether any part of it was
// valid. Credit to https://github.com/regzo2/VRCFaceTracking-QuestProOpenXR
// for the OpenXR weight to Unified Expressions conversion.
func (m *Module) convert(state *faceState, data *tracking.TrackingData) bool {
	e := (*weights)(&state.ExpressionWeights)
	tracked := false
	if m.granted.Eye && (state.LeftEyeIsValid || state.RightEyeIsValid) {
		updateEyeData(&data.Eye, state.LeftEyePose.Orientation, state.RightEyePose.Orientation)
		tracked = true
	}
	if m.granted.Eye && state.IsEyeFollowingBlendshapesValid {
		updateEyeExpressions(&data.Eye, &data.Shapes, e)
		tracked = true
	}
	if m.granted.Expression && state.faceValid() {
		updateMouthExpressions(&data.Shapes, e, state.gxr())
		tracked = true
	}
	return tracked
}

// setTracking logs transitions between tracking and not tracking. The
// "not active" hint is delayed so a momentary drop-out does not spam the log.
func (m *Module) setTracking(active bool) {
	if active {
		if !m.tracking || m.silenceAlert {
			m.logger.Info("Tracking is now active!")
		}
		m.tracking, m.silenceAlert = true, false
		return
	}
	if m.tracking {
		m.tracking = false
		m.lastFrameAt = time.Now()
	}
	if !m.silenceAlert && time.Since(m.lastFrameAt) >= silenceWarnAfter {
		m.silenceAlert = true
		m.logger.Warn("Tracking is not active. Make sure you are connected to your computer, 'Forward tracking data to PC' is checked in the Streaming tab and that a VR game or SteamVR is launched.")
	}
}

func (m *Module) Teardown() error {
	if m.shm == nil {
		return nil
	}
	close(m.stop)
	<-m.done
	err := m.shm.close()
	m.shm = nil
	m.granted = adapter.Support{}
	return err
}
