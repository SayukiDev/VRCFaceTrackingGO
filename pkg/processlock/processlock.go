package processlock

import (
	"path/filepath"

	"github.com/shirou/gopsutil/v4/process"
)

func IsLocked(executable string, localPid int) bool {
	procs, err := process.Processes()
	if err != nil {
		return false
	}
	for _, p := range procs {
		if p.Pid == int32(localPid) {
			continue
		}
		e, err := p.Exe()
		if err != nil {
			continue
		}
		if filepath.ToSlash(e) == filepath.ToSlash(executable) {
			return true
		}
	}
	return false
}
