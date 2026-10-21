package main

import (
	"errors"
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// OpenFileMappingW is not wrapped by golang.org/x/sys/windows.
var procOpenFileMappingW = windows.NewLazySystemDLL("kernel32.dll").NewProc("OpenFileMappingW")

func openFileMapping(desiredAccess uint32, name *uint16) (windows.Handle, error) {
	r, _, e := procOpenFileMappingW.Call(uintptr(desiredAccess), 0, uintptr(unsafe.Pointer(name)))
	if r == 0 {
		return 0, e
	}
	return windows.Handle(r), nil
}

// sharedState is the Virtual Desktop Streamer's body state section plus the
// event it signals whenever a new frame was written.
type sharedState struct {
	mapping windows.Handle
	view    uintptr
	event   windows.Handle
	state   *faceState
}

var errStreamerNotRunning = errors.New("virtualdesktop: shared memory not found; is the Virtual Desktop Streamer (v1.30+) running?")

// openSharedState attaches to the Streamer's section. It fails fast when the
// Streamer is not running.
func openSharedState() (*sharedState, error) {
	mapName, err := windows.UTF16PtrFromString(bodyStateMapName)
	if err != nil {
		return nil, err
	}
	eventName, err := windows.UTF16PtrFromString(bodyStateEventName)
	if err != nil {
		return nil, err
	}

	s := &sharedState{}
	s.mapping, err = openFileMapping(windows.FILE_MAP_READ|windows.FILE_MAP_WRITE, mapName)
	if err != nil {
		if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
			return nil, errStreamerNotRunning
		}
		return nil, fmt.Errorf("open file mapping %q: %w", bodyStateMapName, err)
	}

	s.view, err = windows.MapViewOfFile(s.mapping, windows.FILE_MAP_READ|windows.FILE_MAP_WRITE, 0, 0, unsafe.Sizeof(faceState{}))
	if err != nil {
		s.close()
		return nil, fmt.Errorf("map view of file: %w", err)
	}
	// view is a uintptr returned by the OS and stays valid until unmapped;
	// converting through &view keeps go vet's unsafeptr check quiet.
	s.state = (*faceState)(*(*unsafe.Pointer)(unsafe.Pointer(&s.view)))

	s.event, err = windows.OpenEvent(windows.SYNCHRONIZE, false, eventName)
	if err != nil {
		s.close()
		if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
			return nil, errStreamerNotRunning
		}
		return nil, fmt.Errorf("open event %q: %w", bodyStateEventName, err)
	}
	return s, nil
}

// waitFrame blocks until the Streamer signals a new frame or timeout
// elapses. It reports whether a frame arrived.
func (s *sharedState) waitFrame(timeout time.Duration) (bool, error) {
	rc, err := windows.WaitForSingleObject(s.event, uint32(timeout.Milliseconds()))
	switch {
	case err != nil:
		return false, fmt.Errorf("wait for body state event: %w", err)
	case rc == windows.WAIT_OBJECT_0:
		return true, nil
	case rc == uint32(windows.WAIT_TIMEOUT):
		return false, nil
	}
	return false, fmt.Errorf("wait for body state event: unexpected result %#x", rc)
}

// snapshot copies the whole state out of shared memory so a frame is read
// consistently even if the Streamer writes meanwhile.
func (s *sharedState) snapshot() faceState { return *s.state }

func (s *sharedState) close() error {
	var errs []error
	if s.view != 0 {
		errs = append(errs, windows.UnmapViewOfFile(s.view))
		s.view, s.state = 0, nil
	}
	if s.mapping != 0 {
		errs = append(errs, windows.CloseHandle(s.mapping))
		s.mapping = 0
	}
	if s.event != 0 {
		errs = append(errs, windows.CloseHandle(s.event))
		s.event = 0
	}
	return errors.Join(errs...)
}
