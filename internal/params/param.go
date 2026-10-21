// Package params defines the OSC output parameters derived from tracking
// data and the logic that decides which of them are relevant for the
// currently loaded avatar.
package params

import (
	"strings"

	"VRCFaceTrackingGo/pkg/tracking"
)

// ValueType is the OSC argument type of a parameter.
type ValueType int

const (
	TypeUnknown ValueType = iota
	TypeBool
	TypeFloat
	TypeInt
	TypeString
)

// ValueTypeFromConfig maps the VRChat avatar config "type" string.
func ValueTypeFromConfig(s string) ValueType {
	switch s {
	case "Bool":
		return TypeBool
	case "Float":
		return TypeFloat
	case "Int":
		return TypeInt
	case "String":
		return TypeString
	}
	return TypeUnknown
}

// ValueTypeFromOSCTag maps an OSC type tag character.
func ValueTypeFromOSCTag(tag byte) ValueType {
	switch tag {
	case 'T', 'F':
		return TypeBool
	case 'f':
		return TypeFloat
	case 'i':
		return TypeInt
	case 's':
		return TypeString
	}
	return TypeUnknown
}

// Definition is a parameter exposed by the avatar (from the OSC config file
// or OSCQuery).
type Definition struct {
	Address string
	Name    string
	Type    ValueType
}

// Message is a single OSC value ready to be sent. Value is one of bool,
// float32, int32 or string.
type Message struct {
	Address string
	Value   any
}

const (
	defaultPrefix        = "/avatar/parameters/"
	currentVersionPrefix = "v2/"
)

// Output is a leaf parameter that produces OSC messages.
type Output interface {
	Name() string
	Address() string
	Deprecated() bool
	// Process evaluates the parameter against data and returns a message when
	// the value changed since the last call.
	Process(data *tracking.TrackingData) (Message, bool)
}

// Parameter is a (possibly composite) parameter definition. Reset is called
// whenever a new avatar loads; it returns the outputs that are relevant for
// the avatar. When forceRelevant is set every output is relevant regardless
// of the avatar definition.
type Parameter interface {
	Reset(defs []Definition, forceRelevant bool) []Output
	Names() []string
}

// ResetAll resets every parameter and returns the flattened list of relevant
// outputs.
func ResetAll(all []Parameter, defs []Definition, forceRelevant bool) []Output {
	var outputs []Output
	for _, p := range all {
		outputs = append(outputs, p.Reset(defs, forceRelevant)...)
	}
	return outputs
}

func isDeprecatedName(name string) bool { return !strings.HasPrefix(name, currentVersionPrefix) }
