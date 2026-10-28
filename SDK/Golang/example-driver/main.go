// Command example-driver is a demo driver built with the Go SDK. It bundles
// three hardware free modules: one for eyes only, one for the face only and
// one for both. The host starts each loaded module as its own process of
// this program (--source=<id>); every module logs its process ID when it
// starts.
//
// Build it into a driver program and install it from the Modules page:
//
//	go build -o DemoDriver.windows.amd64.vfgmod ./SDK/Golang/example-driver
package main

import (
	"embed"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/plugin"
)

// files holds metadata.json and the icons it names.
//
//go:embed metadata.json
var files embed.FS

func main() {
	plugin.ServeDriver(plugin.Driver{
		ID:       "DemoDriver",
		Metadata: adapter.MustLoadMetadata(files, "metadata.json"),
	},
		&Module{id: "DemoDriverEye", name: "Demo Eye", supported: adapter.Support{Eye: true}},
		&Module{id: "DemoDriverFace", name: "Demo Face", supported: adapter.Support{Expression: true}},
		&Module{id: "DemoDriverBoth", name: "Demo Eye & Face", supported: adapter.Support{Eye: true, Expression: true}},
	)
}
