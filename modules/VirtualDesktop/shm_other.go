//go:build !windows

package main

import (
	"errors"
	"time"
)

// The Virtual Desktop Streamer only exists on Windows. This stub keeps the
// package building elsewhere; metadata.json limits the module to Windows.
type sharedState struct{}

func openSharedState() (*sharedState, error) {
	return nil, errors.New("VirtualDesktop: the Virtual Desktop Streamer is only available on Windows")
}

func (*sharedState) waitFrame(time.Duration) (bool, error) { return false, nil }

func (*sharedState) snapshot() faceState { return faceState{} }

func (*sharedState) close() error { return nil }
