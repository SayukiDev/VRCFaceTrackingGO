package mutation

import (
	"testing"

	"go.uber.org/zap"

	"VRCFaceTracking/internal/settings"
	"VRCFaceTracking/pkg/tracking"
)

func TestMutatorAppliesCorrectors(t *testing.T) {
	store, err := settings.Open(t.TempDir(), zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	m := NewMutator(zap.NewNop(), store)
	m.Load(tracking.NewTrackingData())
	m.SetEnabled(true)

	in := tracking.NewTrackingData()
	in.Shapes[tracking.MouthClosed].Weight = 1
	in.Shapes[tracking.JawOpen].Weight = 0.2

	out := m.Mutate(in)
	if out == in {
		t.Fatal("mutator must not return its input when enabled")
	}
	if got := out.Shapes[tracking.MouthClosed].Weight; got != 0.2 {
		t.Fatalf("MouthClosed clamp not applied: %v", got)
	}
	if in.Shapes[tracking.MouthClosed].Weight != 1 {
		t.Fatal("input must not be modified")
	}
}

func TestMutatorPersistsMutationState(t *testing.T) {
	dir := t.TempDir()
	store, err := settings.Open(dir, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	m := NewMutator(zap.NewNop(), store)
	m.Load(tracking.NewTrackingData())

	f, _ := m.Find("Data Filter")
	f.SetActive(true)
	for _, c := range f.Components() {
		if c.Name == "Minimum Cutoff" {
			if err := c.Set(float64(1.5)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}
	if err := store.Flush(); err != nil {
		t.Fatal(err)
	}

	store2, _ := settings.Open(dir, zap.NewNop())
	m2 := NewMutator(zap.NewNop(), store2)
	m2.Load(tracking.NewTrackingData())
	f2, _ := m2.Find("Data Filter")
	if !f2.Active() || f2.(*Filter).MinCutoff != 1.5 {
		t.Fatalf("filter state not restored: active=%v cutoff=%v", f2.Active(), f2.(*Filter).MinCutoff)
	}
}

func TestComponentsCoverEveryKind(t *testing.T) {
	for _, mut := range NewMutator(zap.NewNop(), mustStore(t)).Mutations() {
		for _, c := range mut.Components() {
			switch c.Kind {
			case KindButton:
				if c.Invoke == nil {
					t.Errorf("%s/%s: button without Invoke", mut.Name(), c.Name)
				}
			default:
				if c.Get == nil || c.Set == nil {
					t.Errorf("%s/%s: missing Get/Set", mut.Name(), c.Name)
				}
			}
		}
	}
}

func mustStore(t *testing.T) *settings.Store {
	t.Helper()
	s, err := settings.Open(t.TempDir(), zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	return s
}
