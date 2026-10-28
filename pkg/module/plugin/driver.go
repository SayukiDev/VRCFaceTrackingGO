package plugin

import (
	"fmt"
	"slices"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
)

// Driver is the static information of a driver program: one program that
// bundles several modules. The host lists the driver separately and never
// loads it; it loads the modules inside it instead.
type Driver struct {
	ID       string
	Metadata adapter.Metadata
}

// ServeDriver runs a driver program for mods. With --info it prints the
// driver and all of mods; serving the host it runs the module named by
// --source. It must be called from main and nothing else may write to
// stdout.
func ServeDriver(d Driver, mods ...adapter.Module) {
	desc := Description{Kind: KindDriver, ID: d.ID, Metadata: d.Metadata}
	for _, mod := range mods {
		desc.Modules = append(desc.Modules, describe(mod))
	}
	serveMain(desc, func(source string) (adapter.Module, error) { return pick(mods, source) })
}

// pick returns the module of mods with the given ID.
func pick(mods []adapter.Module, source string) (adapter.Module, error) {
	if source == "" {
		return nil, fmt.Errorf("no module selected: use --%s", SourceFlag)
	}
	if i := slices.IndexFunc(mods, func(m adapter.Module) bool { return m.ID() == source }); i >= 0 {
		return mods[i], nil
	}
	return nil, fmt.Errorf("this driver contains no module %q", source)
}
