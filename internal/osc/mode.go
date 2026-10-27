package osc

// Mode selects how VRChat's OSC endpoints and the loaded avatar are found.
type Mode string

const (
	// ModeOSCQuery discovers VRChat through OSCQuery, negotiates the ports
	// and reads the loaded avatar's parameters from it.
	ModeOSCQuery Mode = "oscquery"
	// ModeOSC uses the configured ports only. The avatar is not resolved, so
	// every parameter is sent.
	ModeOSC Mode = "osc"
)

// Valid reports whether m is a known mode.
func (m Mode) Valid() bool { return m == ModeOSCQuery || m == ModeOSC }
