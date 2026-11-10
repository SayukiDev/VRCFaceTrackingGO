// Command WiVRn is the tracking module program for WiVRn
// (https://github.com/WiVRn/WiVRn). WiVRn only offers face and eye tracking
// to OpenXR applications, so this module opens a headless OpenXR session
// next to the game and reads XR_FB_face_tracking2, XR_HTC_facial_tracking,
// XR_ANDROID_face_tracking and XR_EXT_eye_gaze_interaction from it.
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

// files holds metadata.json and the icons it names.
//
//go:embed metadata.json wivrn.png
var files embed.FS

var metadata = adapter.MustLoadMetadata(files, "metadata.json")

// teardownTimeout is how long Teardown waits for the session loop.
const teardownTimeout = 2 * time.Second

// Module reads face tracking from the WiVRn OpenXR runtime.
type Module struct {
	logger *zap.Logger

	// open loads the OpenXR loader; tests replace it.
	open         func() (xrAPI, error)
	pollInterval time.Duration
	retryDelay   time.Duration

	xr   xrAPI
	stop chan struct{}
	done chan struct{}

	mu     sync.Mutex
	latest tracking.TrackingData
}

var _ adapter.Module = (*Module)(nil)

// New returns an uninitialised WiVRn module.
func New() *Module {
	return &Module{open: openXR, pollInterval: pollInterval, retryDelay: retryDelay}
}

func (m *Module) ID() string { return "WiVRn" }

func (m *Module) Metadata() adapter.Metadata { return metadata }

func (m *Module) Supported() adapter.Support {
	return adapter.Support{Eye: true, Expression: true}
}

// Initialize loads the OpenXR loader and starts connecting in the
// background. WiVRn, the headset and the host can start in any order, so it
// does not wait for a session.
func (m *Module) Initialize(_ context.Context, logger *zap.Logger, avail adapter.Support) (adapter.Support, error) {
	m.logger = logger.Named("WiVRn")

	xr, err := m.open()
	if err != nil {
		return adapter.Support{}, err
	}
	m.xr = xr
	m.latest = *tracking.NewTrackingData()
	m.stop = make(chan struct{})
	m.done = make(chan struct{})
	go m.run(xr)
	return avail, nil
}

// Update writes the latest values into data without waiting.
func (m *Module) Update(_ context.Context, data *tracking.TrackingData) error {
	if m.xr == nil {
		return errors.New("WiVRn: module not initialised")
	}
	m.mu.Lock()
	data.CopyFrom(&m.latest)
	m.mu.Unlock()
	return nil
}

func (m *Module) Teardown() error {
	if m.xr == nil {
		return nil
	}
	close(m.stop)
	m.xr = nil
	// WiVRn blocks xrCreateInstance until a headset connects, so the loop
	// may be stuck inside the runtime. The process exits after Teardown
	// either way.
	select {
	case <-m.done:
		return nil
	case <-time.After(teardownTimeout):
		return errors.New("WiVRn: the OpenXR runtime did not return in time")
	}
}
