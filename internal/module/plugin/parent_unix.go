//go:build unix

package plugin

import (
	"os"
	"time"

	"go.uber.org/zap"
)

// parentPollInterval is how often a module checks that its host is alive.
const parentPollInterval = 500 * time.Millisecond

// watchParent exits the module once the host is gone. A dead parent's
// children are re-parented, so the parent PID changes. This also ends a
// module that is stuck and never sees stdin close.
func watchParent(logger *zap.Logger) {
	ppid := os.Getppid()
	go func() {
		for range time.Tick(parentPollInterval) {
			if os.Getppid() != ppid {
				logger.Warn("Host is gone, exiting")
				_ = logger.Sync()
				os.Exit(1)
			}
		}
	}()
}
