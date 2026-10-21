// Package osc sends and receives OSC over UDP using go-osc.
package osc

import (
	"fmt"

	"github.com/hypebeast/go-osc/osc"

	"VRCFaceTrackingGo/internal/params"
	"VRCFaceTrackingGo/pkg/tracking"
)

// toOSC converts a parameter message into an OSC message.
func toOSC(m params.Message) (*osc.Message, error) {
	switch v := m.Value.(type) {
	case bool, float32, int32, string, []byte:
		return osc.NewMessage(m.Address, v), nil
	case tracking.Vector4:
		return osc.NewMessage(m.Address, v.X, v.Y, v.Z, v.W), nil
	case tracking.Vector2:
		return osc.NewMessage(m.Address, v.X, v.Y), nil
	}
	return nil, fmt.Errorf("osc: unsupported value type %T for %s", m.Value, m.Address)
}
