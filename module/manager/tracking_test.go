package manager

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"VRCFaceTracking/module/adapter"
	"VRCFaceTracking/pkg/tracking"
)

func TestSetTrackingMasksOnlyTheSwitchedOffKind(t *testing.T) {
	mod := &fakeModule{
		name: "both",
		init: func(context.Context) (adapter.Support, error) {
			return adapter.Support{Eye: true, Expression: true}, nil
		},
		upd: func(d *tracking.TrackingData) error {
			d.Eye.Left.Openness = 0.2
			d.Shapes[tracking.JawOpen].Weight = 0.8
			return nil
		},
	}
	m := New(zap.NewNop(), mod)
	m.Initialize(context.Background(), zap.NewNop())
	defer m.Teardown()

	waitFor := func(cond func(*tracking.TrackingData) bool) *tracking.TrackingData {
		t.Helper()
		data := tracking.NewTrackingData()
		deadline := time.Now().Add(time.Second)
		for {
			m.Snapshot(data)
			if cond(data) {
				return data
			}
			if time.Now().After(deadline) {
				t.Fatalf("condition never met: %+v", data)
			}
			time.Sleep(DefaultUpdateInterval)
		}
	}

	waitFor(func(d *tracking.TrackingData) bool {
		return d.Shapes[tracking.JawOpen].Weight == 0.8 && d.Eye.Left.Openness == 0.2
	})

	m.SetTracking("both", false, true)
	d := waitFor(func(d *tracking.TrackingData) bool { return d.Eye.Left.Openness == 1 })
	if d.Shapes[tracking.JawOpen].Weight != 0.8 {
		t.Fatalf("expression must keep flowing, got %v", d.Shapes[tracking.JawOpen].Weight)
	}
	if m.EyeStatus() != adapter.Idle || m.ExpressionStatus() != adapter.Active {
		t.Fatalf("status eye=%v expr=%v", m.EyeStatus(), m.ExpressionStatus())
	}
	info := m.Modules()[0]
	if info.EyeActive || !info.ExpressionActive || !info.Active {
		t.Fatalf("info = %+v", info)
	}

	m.SetTracking("both", false, false)
	waitFor(func(d *tracking.TrackingData) bool { return d.Shapes[tracking.JawOpen].Weight == 0 })
	if m.Modules()[0].Active {
		t.Fatal("module should report inactive when both kinds are off")
	}

	m.SetTracking("both", true, true)
	waitFor(func(d *tracking.TrackingData) bool {
		return d.Shapes[tracking.JawOpen].Weight == 0.8 && d.Eye.Left.Openness == 0.2
	})
	if m.EyeStatus() != adapter.Active {
		t.Fatalf("eye status = %v", m.EyeStatus())
	}
}
