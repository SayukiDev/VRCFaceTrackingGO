package mixer

import (
	"testing"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

func TestMixerAppliesCorrectors(t *testing.T) {
	m := New(DefaultConfig())

	in := tracking.NewTrackingData()
	in.Shapes[tracking.MouthClosed].Weight = 1
	in.Shapes[tracking.JawOpen].Weight = 0.2

	out := m.Mix(in)
	if out == in {
		t.Fatal("mixer must not return its input when enabled")
	}
	if got := out.Shapes[tracking.MouthClosed].Weight; got != 0.2 {
		t.Fatalf("MouthClosed clamp not applied: %v", got)
	}
	if in.Shapes[tracking.MouthClosed].Weight != 1 {
		t.Fatal("input must not be modified")
	}
}

func TestMixerConfigRoundTrip(t *testing.T) {
	m := New(DefaultConfig())
	if err := m.SetActive("Data Filter", true); err != nil {
		t.Fatal(err)
	}
	if err := m.SetValue("Data Filter", "Minimum Cutoff", float64(1.5)); err != nil {
		t.Fatal(err)
	}

	m2 := New(m.Config())
	if f := m2.Config().Filter; !f.IsActive || f.MinCutoff != 1.5 {
		t.Fatalf("filter state not restored: %+v", f)
	}
}

func TestMixerResetButton(t *testing.T) {
	m := New(DefaultConfig())
	if err := m.SetValue("Parameter Adjustment", "Jaw", Range{Floor: 0.2, Ceil: 0.8}); err != nil {
		t.Fatal(err)
	}
	if err := m.Invoke("Parameter Adjustment", "Reset Values"); err != nil {
		t.Fatal(err)
	}
	if got := m.Config().Adjustment.JawOpen; got != (Range{Floor: 0, Ceil: 1}) {
		t.Fatalf("range not reset: %+v", got)
	}
}

func TestComponentsCoverEveryKind(t *testing.T) {
	m := New(DefaultConfig())
	for _, s := range m.stages {
		for _, c := range s.Components() {
			switch c.Kind {
			case KindButton:
				if c.Invoke == nil {
					t.Errorf("%s/%s: button without Invoke", s.Name(), c.Name)
				}
			default:
				if c.Get == nil || c.Set == nil {
					t.Errorf("%s/%s: missing Get/Set", s.Name(), c.Name)
				}
			}
		}
	}
}
