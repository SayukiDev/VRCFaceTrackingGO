package modules

import (
	"VRCFaceTracking/module/adapter"
	"VRCFaceTracking/module/modules/alvr"
	"VRCFaceTracking/module/modules/pico"
	"VRCFaceTracking/module/modules/virtualdesktop"
)

var Modules []adapter.Module

func init() {
	Modules = []adapter.Module{
		alvr.New(),
		pico.New(),
		virtualdesktop.New(),
	}
}
