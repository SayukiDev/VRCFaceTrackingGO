// Package oscquery discovers VRChat through mDNS, serves the OSCQuery
// endpoint VRChat expects and resolves the current avatar's parameters.
package oscquery

import "VRCFaceTrackingGo/internal/params"

// AvatarInfo describes the currently loaded avatar.
type AvatarInfo struct {
	Name       string              `json:"name"`
	ID         string              `json:"id"`
	Parameters []params.Definition `json:"-"`
}

// NullAvatar is used before any avatar has been resolved.
func NullAvatar(name, id string) AvatarInfo {
	return AvatarInfo{Name: name, ID: id}
}
