package manager

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"VRCFaceTracking/module/adapter"
	"VRCFaceTracking/pkg/tracking"
)

func okModule(name string) *fakeModule {
	return &fakeModule{
		name: name,
		init: func(context.Context) (adapter.Support, error) {
			return adapter.Support{Eye: true, Expression: true}, nil
		},
		upd: func(*tracking.TrackingData) error { return nil },
	}
}

func TestSetConfigOrderAndEnabled(t *testing.T) {
	m := New(zap.NewNop(), okModule("a"), okModule("b"), okModule("c"))

	// Reverse the order, disable "a", forget "c" and mention a stranger.
	both := func(id string, enabled bool) Config {
		return Config{ID: id, Enabled: enabled, Eye: true, Expression: true}
	}
	m.SetConfig([]Config{both("b", true), both("stranger", true), both("a", false)})

	got := m.Config()
	want := []Config{both("b", true), both("a", false), both("c", true)}
	if len(got) != len(want) {
		t.Fatalf("config = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("config[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}

	m.Initialize(context.Background(), zap.NewNop())
	defer m.Teardown()

	// "b" is first and claims everything; "a" is disabled; "c" has nothing left.
	mods := m.Modules()
	if len(mods) != 1 || mods[0].ID != "b" {
		t.Fatalf("loaded = %+v", mods)
	}

	cat := m.Catalog()
	if len(cat) != 3 || !cat[0].Loaded || cat[1].Enabled || cat[2].Loaded {
		t.Fatalf("catalog = %+v", cat)
	}
}
