// Package plugin is the protocol between the host and a module program.
//
// A module program is a main package that calls Serve:
//
//	func main() { plugin.Serve(mymodule.New()) }
//
// Started with --info it prints its Description as JSON to stdout and
// exits, which is how the host inspects a module that is not running.
// Started by the host with --vfgmod=true it answers the requests of the
// host's protocol (see protocol.go) on its stdin/stdout, one at a time;
// its stderr carries its zap log as JSON lines, which the host re-logs. Without either flag it refuses to run, so a
// module program opened by accident does not touch any device.
package plugin

import (
	"context"
	"errors"
	"runtime"
	"strings"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

const (
	// Suffix is the file extension of module programs. They are ordinary
	// executables under a distinct name so they are not mistaken for
	// stand-alone applications. The file name also states the platform
	// the program was built for: <id>.<GOOS>.<GOARCH>.vfgmod (see
	// PlatformSuffix).
	Suffix = ".vfgmod"
	// InfoFlag makes a module program print its Description and exit.
	InfoFlag = "info"
	// HostFlag must be set to true for a module program to serve the host.
	HostFlag = "vfgmod"
)

// HostSuffix is the file name suffix of module programs for the platform
// the host runs on. The host only recognises module programs ending in it.
var HostSuffix = PlatformSuffix(runtime.GOOS, runtime.GOARCH)

// PlatformSuffix returns the file name suffix of module programs built for
// goos and goarch (GOOS and GOARCH values), e.g. ".windows.amd64.vfgmod".
func PlatformSuffix(goos, goarch string) string {
	return "." + goos + "." + goarch + Suffix
}

// HasHostSuffix reports whether name is a module program file for the
// platform the host runs on. Case is ignored.
func HasHostSuffix(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), HostSuffix)
}

// ErrExited is returned once the module process is gone.
var ErrExited = errors.New("module process exited")

// Description is the static information a module program reports.
type Description struct {
	ID        string           `json:"id"`
	Metadata  adapter.Metadata `json:"metadata"`
	Supported adapter.Support  `json:"supported"`
}

// Conn is a running module program as seen by the host.
type Conn interface {
	// Initialize asks the module to start tracking what avail allows and
	// returns what it will provide.
	Initialize(ctx context.Context, avail adapter.Support) (adapter.Support, error)
	// Update returns the module's latest tracking data.
	Update(ctx context.Context) (tracking.TrackingData, error)
	Teardown(ctx context.Context) error
	// Close stops the module process.
	Close() error
}
