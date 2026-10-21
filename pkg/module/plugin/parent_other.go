//go:build !unix

package plugin

import "go.uber.org/zap"

// watchParent is not needed on Windows: the host's job object ends the
// module together with the host.
func watchParent(*zap.Logger) {}
