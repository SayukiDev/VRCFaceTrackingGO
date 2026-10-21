// Package vrchat knows where VRChat keeps its OSC data on this machine and
// how to make sure OSC is enabled.
//
// Only the Windows build of VRChat is detected. Elsewhere (e.g. VRChat
// under Proton) Supported is false: there is nothing to detect, and the
// host sends every parameter instead of the ones of the current avatar.
package vrchat

import (
	"errors"
	"runtime"
)

// Supported reports whether VRChat, its OSC settings and its avatar
// configs can be detected on this OS.
const Supported = runtime.GOOS == "windows"

// ErrUnsupported is returned by operations that need a detected VRChat.
var ErrUnsupported = errors.New("VRChat detection is only supported on Windows")
