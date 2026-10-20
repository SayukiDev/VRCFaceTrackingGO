// Package adapter defines the contract a tracking module implements. Modules
// are compiled into the host binary and registered in main.
package adapter

import (
	"context"

	"go.uber.org/zap"

	"VRCFaceTracking/pkg/tracking"
)

// State is the host-side lifecycle state of a module.
type State int

const (
	// Uninitialized means the module has not (successfully) initialised.
	Uninitialized State = iota - 1
	// Idle means the module is initialised but tracking is paused.
	Idle
	// Active means the module is receiving Update calls.
	Active
)

func (s State) String() string {
	switch s {
	case Uninitialized:
		return "Uninitialized"
	case Idle:
		return "Idle"
	case Active:
		return "Active"
	}
	return "Unknown"
}

// Support describes which tracking kinds a module offers or was granted.
type Support struct {
	Eye        bool
	Expression bool
}

// Metadata is static, user facing information about a module.
type Metadata struct {
	Name string
	// Icons are PNG encoded images shown next to the module.
	Icons [][]byte
}

// Module is implemented by every tracking backend.
//
// The host drives the module: Initialize once, Update repeatedly (only while
// the module is Active) and Teardown on shutdown. Update writes the latest
// tracking values into the supplied data.
type Module interface {
	// ID is a stable, machine friendly identifier used in settings. It must
	// not change between versions or with the module's runtime state.
	ID() string
	Metadata() Metadata
	// Supported reports what the module can provide before initialisation.
	Supported() Support
	// Initialize opens the underlying device. avail tells the module which
	// kinds of tracking the host still needs; the returned Support states what
	// the module will actually provide.
	Initialize(ctx context.Context, logger *zap.Logger, avail Support) (Support, error)
	// Update blocks until fresh data is available (or a short timeout) and
	// writes it into data.
	Update(ctx context.Context, data *tracking.TrackingData) error
	Teardown() error
}
