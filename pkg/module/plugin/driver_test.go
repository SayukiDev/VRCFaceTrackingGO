package plugin

import (
	"testing"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
)

func TestPick(t *testing.T) {
	a, b := &testModule{}, &namedModule{id: "b"}
	mods := []adapter.Module{a, b}
	for _, source := range []string{"", "missing"} {
		if _, err := pick(mods, source); err == nil {
			t.Fatalf("%q accepted", source)
		}
	}
	if m, _ := pick(mods, "b"); m != b {
		t.Fatalf("picked %v", m)
	}
}

// namedModule is a testModule with another ID.
type namedModule struct {
	testModule
	id string
}

func (m *namedModule) ID() string { return m.id }

func TestValidateDriver(t *testing.T) {
	mod := func(id string) Description { return Description{ID: id} }
	for name, d := range map[string]Description{
		"empty driver":        {Kind: KindDriver, ID: "d"},
		"duplicate":           {Kind: KindDriver, ID: "d", Modules: []Description{mod("a"), mod("a")}},
		"driver ID reuse":     {Kind: KindDriver, ID: "d", Modules: []Description{mod("d")}},
		"nested":              {Kind: KindDriver, ID: "d", Modules: []Description{{Kind: KindDriver, ID: "e", Modules: []Description{mod("a")}}}},
		"unknown kind":        {Kind: "plugin", ID: "d"},
		"module with modules": {ID: "d", Modules: []Description{mod("a")}},
	} {
		if err := d.validate(); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	if err := (Description{Kind: KindDriver, ID: "d", Modules: []Description{mod("a"), mod("b")}}).validate(); err != nil {
		t.Fatal(err)
	}
}
