package main

import (
	"context"
	"math"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

const (
	blinkEvery    = 4 * time.Second
	blinkDuration = 150 * time.Millisecond
)

// Module animates the eyes and/or the face from the clock. The modules of
// the driver differ only in what they support.
type Module struct {
	id        string
	name      string
	supported adapter.Support

	logger  *zap.Logger
	granted adapter.Support
	start   time.Time
}

var _ adapter.Module = (*Module)(nil)

func (m *Module) ID() string { return m.id }

func (m *Module) Metadata() adapter.Metadata {
	return adapter.Metadata{
		Name:              m.name,
		Description:       "Demo module inside Demo Driver",
		Author:            "SayukiDev",
		Version:           "1.0.0",
		UsageInstructions: "Needs no hardware",
	}
}

func (m *Module) Supported() adapter.Support { return m.supported }

func (m *Module) Initialize(_ context.Context, logger *zap.Logger, avail adapter.Support) (adapter.Support, error) {
	m.logger = logger.Named(m.id)
	m.granted = adapter.Support{Eye: avail.Eye && m.supported.Eye, Expression: avail.Expression && m.supported.Expression}
	m.start = time.Now()
	m.logger.Info("Demo module started", zap.Int("pid", os.Getpid()),
		zap.Bool("eye", m.granted.Eye), zap.Bool("expression", m.granted.Expression))
	return m.granted, nil
}

func (m *Module) Update(_ context.Context, data *tracking.TrackingData) error {
	t := time.Since(m.start).Seconds()

	if m.granted.Expression {
		data.Shapes[tracking.JawOpen].Weight = float32(0.4 + 0.4*math.Sin(t*2))
		smile := float32(0.5 + 0.5*math.Sin(t*0.5))
		data.Shapes[tracking.MouthCornerPullLeft].Weight = smile
		data.Shapes[tracking.MouthCornerPullRight].Weight = smile
		data.Head.Yaw = float32(0.2 * math.Sin(t*0.7))
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

func (m *Module) Teardown() error {
	m.logger.Info("Demo module stopped", zap.Int("pid", os.Getpid()))
	return nil
}
