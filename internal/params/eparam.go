package params

import "github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"

// EParam ("everything param") bundles the bool, float and binary variants of
// one logical parameter so an avatar can pick whichever encoding it uses.
type EParam struct {
	children []Parameter
}

// NewEParam creates the float/bool/binary trio. The bool variant is true when
// the value is below minBoolThreshold.
func NewEParam(name string, get func(*tracking.TrackingData) float32, minBoolThreshold float32) *EParam {
	return &EParam{children: []Parameter{
		NewBase(name, func(d *tracking.TrackingData) bool { return get(d) < minBoolThreshold }),
		NewBase(name, get),
		NewBinary(name, get),
	}}
}

// NewEParamNoBinary is NewEParam without the binary encoding.
func NewEParamNoBinary(name string, get func(*tracking.TrackingData) float32, minBoolThreshold float32) *EParam {
	return &EParam{children: []Parameter{
		NewBase(name, func(d *tracking.TrackingData) bool { return get(d) < minBoolThreshold }),
		NewBase(name, get),
	}}
}

// NewEParamVector2 creates <name>X and <name>Y trios from a 2D getter.
func NewEParamVector2(name string, get func(*tracking.TrackingData) tracking.Vector2, minBoolThreshold float32) *EParam {
	x := func(d *tracking.TrackingData) float32 { return get(d).X }
	y := func(d *tracking.TrackingData) float32 { return get(d).Y }
	return &EParam{children: []Parameter{
		NewBase(name+"X", func(d *tracking.TrackingData) bool { return x(d) < minBoolThreshold }),
		NewBase(name+"X", x),
		NewBinary(name+"X", x),
		NewBase(name+"Y", func(d *tracking.TrackingData) bool { return y(d) < minBoolThreshold }),
		NewBase(name+"Y", y),
		NewBinary(name+"Y", y),
	}}
}

func (e *EParam) Reset(defs []Definition, forceRelevant bool) []Output {
	var outputs []Output
	for _, c := range e.children {
		outputs = append(outputs, c.Reset(defs, forceRelevant)...)
	}
	return outputs
}

func (e *EParam) Names() []string {
	var names []string
	for _, c := range e.children {
		names = append(names, c.Names()...)
	}
	return names
}
