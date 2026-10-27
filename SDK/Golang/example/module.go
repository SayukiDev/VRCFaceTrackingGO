package main

import (
	"context"
	"embed"
	"math"
	"time"

	"go.uber.org/zap"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

// files holds metadata.json and the icons it names.
//
//go:embed metadata.json
var files embed.FS

var metadata = adapter.MustLoadMetadata(files, "metadata.json")

const (
	blinkEvery    = 4 * time.Second
	blinkDuration = 150 * time.Millisecond
)

// Module animates the face from the clock.
type Module struct {
	logger  *zap.Logger
	granted adapter.Support
	start   time.Time
}

var _ adapter.Module = (*Module)(nil)

func New() *Module { return &Module{} }

// ID must never change: it is the settings key and the installed file name.
func (m *Module) ID() string { return "Demo" }

func (m *Module) Metadata() adapter.Metadata { return metadata }

func (m *Module) Supported() adapter.Support {
	return adapter.Support{Eye: true, Expression: true}
}

// Initialize would open the device. avail is what the host still needs;
// return what this module will provide.
func (m *Module) Initialize(_ context.Context, logger *zap.Logger, avail adapter.Support) (adapter.Support, error) {
	m.logger = logger.Named("Demo")
	m.granted = avail
	m.start = time.Now()
	m.logger.Info("Demo module started", zap.Bool("eye", avail.Eye), zap.Bool("expression", avail.Expression))
	return avail, nil
}

// Update writes the current state into data. A real module would copy the
// newest values its receiving goroutine stored, without waiting.
func (m *Module) Update(_ context.Context, data *tracking.TrackingData) error {
	t := time.Since(m.start).Seconds()

	if m.granted.Expression {
		data.Shapes[tracking.JawOpen].Weight = float32(0.4 + 0.4*math.Sin(t*2))
		smile := float32(0.5 + 0.5*math.Sin(t*0.5))
		data.Shapes[tracking.MouthCornerPullLeft].Weight = smile
		data.Shapes[tracking.MouthCornerPullRight].Weight = smile
	}

	if m.granted.Eye {
		openness := float32(1)
		if time.Duration(math.Mod(t, blinkEvery.Seconds())*float64(time.Second)) < blinkDuration {
			openness = 0
		}
		// Gaze is a tangent-space direction: X right, Y up.
		gaze := tracking.Vector2{X: float32(0.3 * math.Cos(t)), Y: float32(0.2 * math.Sin(t))}
		for _, eye := range []*tracking.SingleEyeData{&data.Eye.Left, &data.Eye.Right} {
			eye.Openness = openness
			eye.Gaze = gaze
			eye.PupilDiameterMM = 4
		}
		data.Eye.MinDilation, data.Eye.MaxDilation = 2, 7
	}
	return nil
}

// Teardown would release the device.
func (m *Module) Teardown() error {
	m.logger.Info("Demo module stopped")
	return nil
}
