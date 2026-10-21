// Command example is a demo tracking module built with the Go SDK. It needs
// no hardware: it opens and closes the jaw, blinks and moves the eyes in a
// loop so the whole pipeline can be tried out.
//
// Build it into a module program and install it from the Modules page:
//
//	go build -o Demo.vfgmod ./SDK/Golang/example
package main

import "github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/plugin"

func main() { plugin.Serve(New()) }
