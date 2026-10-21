package params

import (
	"github.com/dlclark/regexp2"

	"VRCFaceTrackingGo/pkg/tracking"
)

// Value is the set of Go types an output can carry. Besides the scalar OSC
// types, small structs of float32 fields (e.g. tracking.Vector4) are
// accepted and expanded into multiple OSC arguments by the sender.
type Value interface {
	comparable
}

func valueTypeOf[T Value]() ValueType {
	var zero T
	switch any(zero).(type) {
	case bool:
		return TypeBool
	case float32:
		return TypeFloat
	case int32:
		return TypeInt
	case string:
		return TypeString
	}
	return TypeUnknown
}

// nameMatcher matches an avatar parameter address against a parameter name.
// The negative lookbehind stops a legacy "JawOpen" from matching "v2/JawOpen".
func nameMatcher(name string) *regexp2.Regexp {
	n := regexp2.Escape(name)
	return regexp2.MustCompile(`(?<!(v\d+))(/`+n+`)$|^(`+n+`)$`, regexp2.None)
}

// Base is the fundamental typed output parameter.
type Base[T Value] struct {
	name       string
	typ        ValueType
	get        func(*tracking.TrackingData) T
	matcher    *regexp2.Regexp
	sendOnLoad bool

	address  string
	relevant bool
	last     *T
}

// NewBase creates a parameter named name (without the /avatar/parameters/
// prefix). sendOnLoad forces the current value to be sent as soon as the
// parameter becomes relevant.
func NewBase[T Value](name string, get func(*tracking.TrackingData) T, sendOnLoad bool) *Base[T] {
	return &Base[T]{
		name:       name,
		typ:        valueTypeOf[T](),
		get:        get,
		matcher:    nameMatcher(name),
		sendOnLoad: sendOnLoad,
		address:    defaultPrefix + name,
	}
}

func (b *Base[T]) Name() string     { return b.name }
func (b *Base[T]) Address() string  { return b.address }
func (b *Base[T]) Relevant() bool   { return b.relevant }
func (b *Base[T]) Deprecated() bool { return isDeprecatedName(b.name) }
func (b *Base[T]) Names() []string  { return []string{b.name} }

func (b *Base[T]) setRelevant(relevant bool, address string) []Output {
	b.relevant = relevant
	b.address = address
	// Force the next Process call to emit the value for a freshly loaded
	// avatar.
	b.last = nil
	if !relevant {
		return nil
	}
	return []Output{b}
}

func (b *Base[T]) Reset(defs []Definition, forceRelevant bool) []Output {
	if forceRelevant {
		return b.setRelevant(true, defaultPrefix+b.name)
	}
	for _, def := range defs {
		if def.Type != b.typ {
			continue
		}
		if ok, _ := b.matcher.MatchString(def.Address); ok {
			return b.setRelevant(true, def.Address)
		}
	}
	return b.setRelevant(false, defaultPrefix+b.name)
}

func (b *Base[T]) Process(data *tracking.TrackingData) (Message, bool) {
	v := b.get(data)
	if b.last != nil && *b.last == v {
		return Message{}, false
	}
	b.last = &v
	return Message{Address: b.address, Value: v}, true
}

// AlwaysRelevant is a Base with a fixed address that is relevant for every
// avatar (used for VRChat native tracking endpoints).
type AlwaysRelevant[T Value] struct {
	*Base[T]
}

func NewAlwaysRelevant[T Value](address string, get func(*tracking.TrackingData) T) *AlwaysRelevant[T] {
	b := NewBase(currentVersionPrefix, get, false)
	b.address = address
	b.relevant = true
	return &AlwaysRelevant[T]{Base: b}
}

func (a *AlwaysRelevant[T]) Reset([]Definition, bool) []Output {
	return a.setRelevant(true, a.address)
}

// Native is an AlwaysRelevant that is only enabled when condition holds for
// the avatar's definitions (e.g. the avatar has no eye parameters of its own).
type Native[T Value] struct {
	*AlwaysRelevant[T]
	condition func([]Definition) bool
}

func NewNative[T Value](address string, get func(*tracking.TrackingData) T, condition func([]Definition) bool) *Native[T] {
	return &Native[T]{AlwaysRelevant: NewAlwaysRelevant(address, get), condition: condition}
}

func (n *Native[T]) Reset(defs []Definition, forceRelevant bool) []Output {
	if n.condition(defs) {
		return n.AlwaysRelevant.Reset(defs, forceRelevant)
	}
	n.relevant = false
	return nil
}

// ConditionalBool only updates its value while the second return of get is
// true. It is never considered deprecated.
type ConditionalBool struct {
	*Base[bool]
	getCond func(*tracking.TrackingData) (value, enabled bool)
}

func NewConditionalBool(name string, get func(*tracking.TrackingData) (value, enabled bool)) *ConditionalBool {
	c := &ConditionalBool{getCond: get}
	c.Base = NewBase(name, func(d *tracking.TrackingData) bool { v, _ := get(d); return v }, false)
	return c
}

func (c *ConditionalBool) Deprecated() bool { return false }

func (c *ConditionalBool) Process(data *tracking.TrackingData) (Message, bool) {
	if _, enabled := c.getCond(data); !enabled {
		return Message{}, false
	}
	return c.Base.Process(data)
}
