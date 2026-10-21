//go:build !windows && !linux

package plugin

import (
	"os"
	"os/exec"
)

// There is no system facility tying a child's life to its parent here;
// the module watches its parent itself (see watchParent).

func prepare(*exec.Cmd) {}

func adopt(*os.Process) error { return nil }
