// Package vrchat knows where VRChat keeps its OSC data on this machine and
// how to make sure OSC is enabled.
package vrchat

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/v4/process"
	"golang.org/x/sys/windows/registry"
)

// OSCDirectory returns %LOCALAPPDATA%Low\VRChat\VRChat\OSC.
func OSCDirectory() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA")+"Low", "VRChat", "VRChat", "OSC")
}

// ForceEnableOSC sets every OSC related VRChat registry value to 1. It
// reports whether anything had to be changed.
func ForceEnableOSC() (changed bool, err error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\VRChat\VRChat`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			// VRChat has never run; assume OSC will be enabled by default.
			return false, nil
		}
		return false, fmt.Errorf("open VRChat registry key: %w", err)
	}
	defer key.Close()

	names, err := key.ReadValueNames(-1)
	if err != nil {
		return false, fmt.Errorf("read VRChat registry values: %w", err)
	}
	for _, name := range names {
		if !strings.HasPrefix(name, "VRC_INPUT_OSC") && !strings.HasPrefix(name, "UI.Settings.Osc") {
			continue
		}
		value, _, err := key.GetIntegerValue(name)
		if err != nil || value != 0 {
			continue
		}
		if err := key.SetDWordValue(name, 1); err != nil {
			return changed, fmt.Errorf("set %s: %w", name, err)
		}
		changed = true
	}
	return changed, nil
}

// IsRunning reports whether a VRChat process exists.
func IsRunning() bool {
	procs, err := process.Processes()
	if err != nil {
		return false
	}
	for _, p := range procs {
		name, err := p.Name()
		if err != nil {
			continue
		}
		if name == "VRChat.exe" || name == "VRChat" {
			return true
		}
	}
	return false
}

// ResetAvatarConfigs deletes every generated Avatars folder so VRChat
// regenerates the OSC config on next avatar load.
func ResetAvatarConfigs() error {
	entries, err := os.ReadDir(OSCDirectory())
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(OSCDirectory(), e.Name(), "Avatars")
		if _, err := os.Stat(dir); err == nil {
			if err := os.RemoveAll(dir); err != nil {
				return err
			}
		}
	}
	return nil
}
