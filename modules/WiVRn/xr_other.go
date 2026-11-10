//go:build !linux

package main

import "errors"

// WiVRn only runs on Linux. This stub keeps the package building elsewhere;
// metadata.json limits the module to Linux.
func openXR() (xrAPI, error) {
	return nil, errors.New("WiVRn: the WiVRn OpenXR runtime is only available on Linux")
}
