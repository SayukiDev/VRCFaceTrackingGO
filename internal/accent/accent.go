// Package accent reads the system accent colour. Only Windows has one;
// elsewhere Color returns ErrUnsupported.
package accent

import "errors"

// ErrUnsupported is returned where the OS has no accent colour to read.
var ErrUnsupported = errors.New("the system accent colour is only available on Windows")
