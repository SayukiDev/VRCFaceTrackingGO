//go:build !windows

package vrchat

// OSCDirectory is unknown without a detected VRChat.
func OSCDirectory() string { return "" }

// ForceEnableOSC does nothing without a detected VRChat.
func ForceEnableOSC() (changed bool, err error) { return false, nil }

// IsRunning always reports false without a detected VRChat.
func IsRunning() bool { return false }

// ResetAvatarConfigs needs a detected VRChat.
func ResetAvatarConfigs() error { return ErrUnsupported }
