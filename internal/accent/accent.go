// Package accent reads the Windows accent colour.
package accent

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

// Color returns the user's Windows accent colour as "#RRGGBB".
func Color() (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\DWM`, registry.QUERY_VALUE)
	if err != nil {
		return "", fmt.Errorf("open DWM registry key: %w", err)
	}
	defer key.Close()

	// AccentColor is stored as 0xAABBGGRR.
	v, _, err := key.GetIntegerValue("AccentColor")
	if err != nil {
		return "", fmt.Errorf("read AccentColor: %w", err)
	}
	return fmt.Sprintf("#%02X%02X%02X", v&0xff, v>>8&0xff, v>>16&0xff), nil
}
