package plugin

import (
	"os"
	"os/exec"
	"syscall"
)

// prepare asks the kernel to kill the module as soon as the host dies,
// even when the module is stuck.
func prepare(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
}

func adopt(*os.Process) error { return nil }
