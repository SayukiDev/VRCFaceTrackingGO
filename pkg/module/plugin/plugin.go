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
//
// A driver program bundles several modules and calls ServeDriver instead.
// Its --info lists the modules, and the host starts it with --source
// naming the one module to serve.
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
	// SourceFlag tells a driver program which of its modules to serve.
	SourceFlag = "source"

	// KindDriver is the Description.Kind of a driver program.
	KindDriver = "driver"
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
	// Kind is KindDriver for a driver program and empty for a module.
	Kind      string           `json:"kind,omitempty"`
	ID        string           `json:"id"`
	Metadata  adapter.Metadata `json:"metadata"`
	Supported adapter.Support  `json:"supported,omitzero"`
	// Modules are the modules a driver program contains. A driver is
	// never loaded itself; its modules are.
	Modules []Description `json:"modules,omitempty"`
}

// IsDriver reports whether d describes a driver program.
func (d Description) IsDriver() bool { return d.Kind == KindDriver }

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
