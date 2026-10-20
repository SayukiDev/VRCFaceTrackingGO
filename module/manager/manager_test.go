package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"VRCFaceTracking/module/adapter"
	"VRCFaceTracking/pkg/tracking"
)

// fakeModule lets tests script Initialize/Update behaviour.
type fakeModule struct {
	name string
	init func(ctx context.Context) (adapter.Support, error)
	upd  func(data *tracking.TrackingData) error
}

func (f *fakeModule) ID() string                 { return f.name }
func (f *fakeModule) Metadata() adapter.Metadata { return adapter.Metadata{Name: f.name} }
func (f *fakeModule) Supported() adapter.Support { return adapter.Support{Eye: true, Expression: true} }
func (f *fakeModule) Initialize(ctx context.Context, _ *zap.Logger, _ adapter.Support) (adapter.Support, error) {
	return f.init(ctx)
}
func (f *fakeModule) Update(_ context.Context, d *tracking.TrackingData) error { return f.upd(d) }
func (f *fakeModule) Teardown() error                                          { return nil }

func TestInitializeSurvivesHangingAndPanickingModules(t *testing.T) {
	hang := &fakeModule{name: "hang", init: func(ctx context.Context) (adapter.Support, error) {
		<-ctx.Done() // behaves: gives up when asked
		return adapter.Support{}, ctx.Err()
	}}
	panicky := &fakeModule{name: "panic", init: func(context.Context) (adapter.Support, error) { panic("boom") }}
	good := &fakeModule{
		name: "good",
		init: func(context.Context) (adapter.Support, error) {
			return adapter.Support{Eye: true, Expression: true}, nil
		},
		upd: func(d *tracking.TrackingData) error { d.Shapes[tracking.JawOpen].Weight = 1; return nil },
	}

	m := New(zap.NewNop(), hang, panicky, good)
	m.SetInitTimeout(200 * time.Millisecond)
	start := time.Now()
	m.Initialize(context.Background(), zap.NewNop())
	defer m.Teardown()
	if time.Since(start) > 2*time.Second {
		t.Fatalf("Initialize took %s, hanging module was not cut off", time.Since(start))
	}

	mods := m.Modules()
	if len(mods) != 1 || mods[0].Name != "good" {
		t.Fatalf("modules = %+v", mods)
	}

	data := tracking.NewTrackingData()
	deadline := time.Now().Add(time.Second)
	for data.Shapes[tracking.JawOpen].Weight != 1 {
		if time.Now().After(deadline) {
			t.Fatal("good module never updated data")
		}
		time.Sleep(DefaultUpdateInterval)
		m.Snapshot(data)
	}
}

func TestUpdatePanicIsContained(t *testing.T) {
	calls := 0
	mod := &fakeModule{
		name: "flaky",
		init: func(context.Context) (adapter.Support, error) { return adapter.Support{Eye: true}, nil },
		upd: func(*tracking.TrackingData) error {
			calls++
			if calls == 1 {
				panic("first call explodes")
			}
			return errors.New("later calls just error")
		},
	}
	m := New(zap.NewNop(), mod)
	m.Initialize(context.Background(), zap.NewNop())
	defer m.Teardown()

	time.Sleep(10 * DefaultUpdateInterval)
	if calls < 2 {
		t.Fatalf("update loop stopped after the panic (calls=%d)", calls)
	}
}
