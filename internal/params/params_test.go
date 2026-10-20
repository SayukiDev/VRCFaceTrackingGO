package params

import (
	"testing"

	"VRCFaceTracking/module/adapter"
	"VRCFaceTracking/pkg/tracking"
)

type fakeStatus struct{ eye, expr adapter.State }

func (f fakeStatus) EyeStatus() adapter.State        { return f.eye }
func (f fakeStatus) ExpressionStatus() adapter.State { return f.expr }

func TestNameMatcherRejectsVersionedPrefix(t *testing.T) {
	m := nameMatcher("JawOpen")
	cases := map[string]bool{
		"/avatar/parameters/JawOpen":    true,
		"JawOpen":                       true,
		"/avatar/parameters/v2/JawOpen": false,
		"/avatar/parameters/MyJawOpen":  false,
	}
	for addr, want := range cases {
		got, err := m.MatchString(addr)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%s: got %v want %v", addr, got, want)
		}
	}
}

func TestBaseResetAndProcess(t *testing.T) {
	p := NewBase("v2/JawOpen", func(d *tracking.TrackingData) float32 { return d.Shapes[tracking.JawOpen].Weight }, false)
	defs := []Definition{{Address: "/avatar/parameters/v2/JawOpen", Name: "v2/JawOpen", Type: TypeFloat}}
	outs := p.Reset(defs, false)
	if len(outs) != 1 {
		t.Fatalf("expected relevant, got %d outputs", len(outs))
	}
	data := tracking.NewTrackingData()
	data.Shapes[tracking.JawOpen].Weight = 0.3
	msg, ok := outs[0].Process(data)
	if !ok || msg.Value.(float32) != 0.3 || msg.Address != defs[0].Address {
		t.Fatalf("unexpected message %+v ok=%v", msg, ok)
	}
	if _, ok := outs[0].Process(data); ok {
		t.Fatal("unchanged value must not re-send")
	}
	if outs := p.Reset([]Definition{{Address: "/avatar/parameters/v2/JawOpen", Type: TypeBool}}, false); len(outs) != 0 {
		t.Fatal("type mismatch must not be relevant")
	}
}

func TestBinaryReset(t *testing.T) {
	b := NewBinary("v2/JawOpen", func(d *tracking.TrackingData) float32 { return d.Shapes[tracking.JawOpen].Weight })
	defs := []Definition{
		{Address: "/avatar/parameters/v2/JawOpen1", Name: "v2/JawOpen1", Type: TypeBool},
		{Address: "/avatar/parameters/v2/JawOpen2", Name: "v2/JawOpen2", Type: TypeBool},
		{Address: "/avatar/parameters/v2/JawOpen4", Name: "v2/JawOpen4", Type: TypeBool},
		{Address: "/avatar/parameters/v2/JawOpen3", Name: "v2/JawOpen3", Type: TypeBool},
	}
	outs := b.Reset(defs, false)
	if len(outs) != 3 {
		t.Fatalf("expected 3 bit outputs, got %d", len(outs))
	}
	data := tracking.NewTrackingData()
	data.Shapes[tracking.JawOpen].Weight = 5.0 / 8.0 // 0b101
	got := map[string]bool{}
	for _, o := range outs {
		msg, ok := o.Process(data)
		if !ok {
			t.Fatalf("%s produced nothing", o.Name())
		}
		got[o.Name()] = msg.Value.(bool)
	}
	if !got["v2/JawOpen1"] || got["v2/JawOpen2"] || !got["v2/JawOpen4"] {
		t.Fatalf("bits = %v", got)
	}
}

func TestAllParametersForceRelevant(t *testing.T) {
	all := AllParameters(fakeStatus{eye: adapter.Active, expr: adapter.Active})
	outs := ResetAll(all, nil, true)
	if len(outs) == 0 {
		t.Fatal("no outputs")
	}
	data := tracking.NewTrackingData()
	for _, o := range outs {
		o.Process(data)
	}
}
