//go:build !windows

package accent

// Color is not available on this OS.
func Color() (string, error) { return "", ErrUnsupported }
