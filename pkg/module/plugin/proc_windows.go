package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// prepare stops the console module programs from opening a window next to
// the GUI host.
func prepare(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
}

// job is a kill-on-close job object holding every module process. Its only
// handle stays open for the host's lifetime and is not inheritable, so the
// system ends all modules as soon as the host process ends for any reason,
// even when a module is stuck.
var job struct {
	once   sync.Once
	handle windows.Handle
	err    error
}

func killOnCloseJob() (windows.Handle, error) {
	job.once.Do(func() {
		h, err := windows.CreateJobObject(nil, nil)
		if err != nil {
			job.err = fmt.Errorf("create job object: %w", err)
			return
		}
		info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
			BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
				LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
			},
		}
		if _, err := windows.SetInformationJobObject(h, windows.JobObjectExtendedLimitInformation,
			uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
			_ = windows.CloseHandle(h)
			job.err = fmt.Errorf("configure job object: %w", err)
			return
		}
		job.handle = h
	})
	return job.handle, job.err
}

// adopt puts a started module process into the kill-on-close job.
func adopt(p *os.Process) error {
	j, err := killOnCloseJob()
	if err != nil {
		return err
	}
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(p.Pid))
	if err != nil {
		return fmt.Errorf("open module process: %w", err)
	}
	defer windows.CloseHandle(h)
	if err := windows.AssignProcessToJobObject(j, h); err != nil {
		return fmt.Errorf("assign module to job object: %w", err)
	}
	return nil
}
