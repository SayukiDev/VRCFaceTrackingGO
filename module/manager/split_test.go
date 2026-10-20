package manager

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"VRCFaceTracking/module/adapter"
	"VRCFaceTracking/pkg/tracking"
)

// grantingModule records what it was asked for and grants exactly that.
func grantingModule(name string, asked *adapter.Support) *fakeModule {
	return &fakeModule{
		name: name,
		init: func(context.Context) (adapter.Support, error) { return *asked, nil },
		upd:  func(*tracking.TrackingData) error { return nil },
	}
}

func TestEyeAndExpressionCanComeFromDifferentModules(t *testing.T) {
	var askedA, askedB adapter.Support
	a := grantingModule("a", &askedA)
	a.init = func(ctx context.Context) (adapter.Support, error) { return askedA, nil }
	b := grantingModule("b", &askedB)

	// Capture the avail argument through the interface.
	a.init = func(context.Context) (adapter.Support, error) { return askedA, nil }
	m := New(zap.NewNop(), &capture{fakeModule: a, into: &askedA}, &capture{fakeModule: b, into: &askedB})

	// "a" first but restricted to expression; "b" gets eye.
	m.SetConfig([]Config{
		{ID: "a", Enabled: true, Eye: false, Expression: true},
		{ID: "b", Enabled: true, Eye: true, Expression: true},
	})
	m.Initialize(context.Background(), zap.NewNop())
	defer m.Teardown()

	if askedA != (adapter.Support{Expression: true}) {
		t.Fatalf("a was asked for %+v", askedA)
	}
	if askedB != (adapter.Support{Eye: true}) {
		t.Fatalf("b was asked for %+v", askedB)
	}
	mods := m.Modules()
	if len(mods) != 2 || mods[0].UsingEye || !mods[0].UsingExpression || !mods[1].UsingEye || mods[1].UsingExpression {
		t.Fatalf("loaded = %+v", mods)
	}
	if m.EyeStatus() != adapter.Active || m.ExpressionStatus() != adapter.Active {
		t.Fatalf("status eye=%v expr=%v", m.EyeStatus(), m.ExpressionStatus())
	}
}

// capture wraps fakeModule to record the avail passed to Initialize.
type capture struct {
	*fakeModule
	into *adapter.Support
}

func (c *capture) Initialize(ctx context.Context, l *zap.Logger, avail adapter.Support) (adapter.Support, error) {
	*c.into = avail
	return c.fakeModule.Initialize(ctx, l, avail)
}
