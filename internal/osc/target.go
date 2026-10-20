package osc

import (
	"fmt"
	"net"
)

// Target is the user configured OSC endpoint. It is persisted in settings.
type Target struct {
	InPort  int    `json:"inPort"`
	OutPort int    `json:"outPort"`
	Address string `json:"address"`
}

// DefaultTarget matches VRChat's defaults.
func DefaultTarget() Target {
	return Target{InPort: 9001, OutPort: 9000, Address: "127.0.0.1"}
}

// Validate checks the ports and address; invalid targets fall back to the
// defaults for the offending field.
func (t Target) Validate() error {
	if t.InPort < 1 || t.InPort > 65535 {
		return fmt.Errorf("invalid in port %d", t.InPort)
	}
	if t.OutPort < 1 || t.OutPort > 65535 {
		return fmt.Errorf("invalid out port %d", t.OutPort)
	}
	if net.ParseIP(t.Address) == nil {
		return fmt.Errorf("invalid address %q", t.Address)
	}
	return nil
}
