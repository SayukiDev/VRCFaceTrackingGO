// Package manager owns the installed module programs: it keeps the list of
// installed modules (read with --info), installs and uninstalls them, and
// starts and supervises the running ones.
package manager

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/plugin"
)

const (
	// DefaultUpdateInterval is the default time between two Update calls
	// of one module.
	DefaultUpdateInterval = 10 * time.Millisecond
	// DefaultInitTimeout bounds a single module's Initialize so a
	// misbehaving device driver cannot stall the whole application.
	DefaultInitTimeout = 15 * time.Second

	// infoTimeout bounds how long a module program may take for --info.
	infoTimeout = 5 * time.Second
	// defaultUpdateTimeout is how long one Update may take before the
	// module is considered hung and restarted.
	defaultUpdateTimeout = 5 * time.Second
	// defaultRestartDelay is the wait before the first restart of a failed
	// module; it doubles with every further failure.
	defaultRestartDelay = time.Second
)

// ErrRunning is returned when a running module would be replaced or
// removed.
var ErrRunning = errors.New("module is running")

// validID keeps module IDs usable as file names.
var validID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Info is an installed module: a module program or one of the modules
// inside a driver program.
type Info struct {
	plugin.Description
	// Path is the module program, or the driver program containing it.
	Path string `json:"path"`
	// Driver is the ID of the driver containing the module; empty for a
	// module program.
	Driver string `json:"driver,omitempty"`
}

// source returns the --source to start the module with: its ID inside a
// driver, nothing for a module program.
func (i Info) source() string {
	if i.Driver == "" {
		return ""
	}
	return i.ID
}

// DriverInfo is an installed driver program. Drivers are never started
// themselves; the modules they contain are listed by Installed.
type DriverInfo struct {
	plugin.Description
	Path string `json:"path"`
}

// programFile is a scanned module or driver program together with what
// identified its version.
type programFile struct {
	desc    plugin.Description
	path    string
	modTime time.Time
	size    int64
}

// modules returns the loadable modules of p.
func (p programFile) modules() []Info {
	if !p.desc.IsDriver() {
		return []Info{{Description: p.desc, Path: p.path}}
	}
	out := make([]Info, 0, len(p.desc.Modules))
	for _, d := range p.desc.Modules {
		out = append(out, Info{Description: d, Path: p.path, Driver: p.desc.ID})
	}
	return out
}

// ids returns every ID p claims: its own and those of its modules.
func (p programFile) ids() []string {
	out := []string{p.desc.ID}
	for _, d := range p.desc.Modules {
		out = append(out, d.ID)
	}
	return out
}

// Manager manages the module programs in one directory.
type Manager struct {
	logger *zap.Logger
	dir    string
	// launch starts a module program; replaced in tests.
	launch func(path, source string) (plugin.Conn, error)
	// updateTimeout and restartDelay drive crash recovery; shortened in
	// tests.
	updateTimeout time.Duration
	restartDelay  time.Duration

	mu       sync.Mutex
	programs []programFile // file name order
	// running maps the IDs of the running modules to their instance.
	running map[string]*Instance

	// updateInterval and initTimeout are stored as nanoseconds so the UI
	// can change them while modules are running.
	updateInterval atomic.Int64
	initTimeout    atomic.Int64
}

// DefaultDir returns the Modules folder inside the application data
// folder, next to the settings file.
func DefaultDir(dataDir string) string { return filepath.Join(dataDir, "Modules") }

// New manages the module programs in dir. Module logs go to logger.
func New(logger *zap.Logger, dir string) *Manager {
	m := &Manager{
		logger:        logger.Named("ModuleManager"),
		dir:           dir,
		running:       map[string]*Instance{},
		updateTimeout: defaultUpdateTimeout,
		restartDelay:  defaultRestartDelay,
	}
	m.launch = func(path, source string) (plugin.Conn, error) { return plugin.Start(path, source, logger) }
	m.SetUpdateInterval(DefaultUpdateInterval)
	m.SetInitTimeout(DefaultInitTimeout)
	return m
}

// UpdateInterval returns the time between two Update calls of one module.
func (m *Manager) UpdateInterval() time.Duration { return time.Duration(m.updateInterval.Load()) }

// SetUpdateInterval changes the update interval. Running modules pick the
// new value up on their next tick. Non-positive values are ignored.
func (m *Manager) SetUpdateInterval(d time.Duration) {
	if d > 0 {
		m.updateInterval.Store(int64(d))
	}
}

// InitTimeout returns the maximum time one module may spend in Initialize.
func (m *Manager) InitTimeout() time.Duration { return time.Duration(m.initTimeout.Load()) }

// SetInitTimeout changes the Initialize timeout used for subsequent starts.
// Non-positive values are ignored.
func (m *Manager) SetInitTimeout(d time.Duration) {
	if d > 0 {
		m.initTimeout.Store(int64(d))
	}
}

// Installed returns the loadable modules found by the last Scan, in file
// name order; the modules of a driver follow each other.
func (m *Manager) Installed() []Info {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Info
	for _, p := range m.programs {
		out = append(out, p.modules()...)
	}
	return out
}

// Drivers returns the driver programs found by the last Scan.
func (m *Manager) Drivers() []DriverInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []DriverInfo
	for _, p := range m.programs {
		if p.desc.IsDriver() {
			out = append(out, DriverInfo{Description: p.desc, Path: p.path})
		}
	}
	return out
}

// Info returns one installed module.
func (m *Manager) Info(id string) (Info, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.infoLocked(id)
}

func (m *Manager) infoLocked(id string) (Info, bool) {
	for _, p := range m.programs {
		for _, info := range p.modules() {
			if info.ID == id {
				return info, true
			}
		}
	}
	return Info{}, false
}

// indexLocked returns the program whose own ID (module or driver) is id.
func (m *Manager) indexLocked(id string) int {
	return slices.IndexFunc(m.programs, func(p programFile) bool { return p.desc.ID == id })
}

// runningLocked reports whether a module of p is running.
func (m *Manager) runningLocked(p programFile) bool {
	return slices.ContainsFunc(p.modules(), func(info Info) bool { return m.running[info.ID] != nil })
}

// Scan reads the module directory again. Programs that did not change
// since the last scan are not run again. Programs that fail --info, and
// programs claiming an ID already taken, are skipped.
func (m *Manager) Scan(ctx context.Context) []Info {
	entries, err := os.ReadDir(m.dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		m.logger.Warn("Cannot read module directory", zap.String("path", m.dir), zap.Error(err))
	}

	m.mu.Lock()
	previous := make(map[string]programFile, len(m.programs))
	for _, p := range m.programs {
		previous[p.path] = p
	}
	m.mu.Unlock()

	var scanned []*programFile
	var wg sync.WaitGroup
	for _, e := range entries {
		if !e.Type().IsRegular() || !plugin.HasHostSuffix(e.Name()) {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(m.dir, e.Name())
		p := &programFile{path: path, modTime: fi.ModTime(), size: fi.Size()}
		scanned = append(scanned, p)
		if old, ok := previous[path]; ok && old.modTime.Equal(p.modTime) && old.size == p.size {
			*p = old
			continue
		}
		wg.Go(func() {
			desc, err := m.inspect(ctx, path)
			if err != nil {
				m.logger.Warn("Skipping module program", zap.String("path", path), zap.Error(err))
				return
			}
			p.desc = desc
		})
	}
	wg.Wait()

	programs := make([]programFile, 0, len(scanned))
	taken := map[string]bool{}
	for _, p := range scanned {
		if p.desc.ID == "" {
			continue
		}
		ids := p.ids()
		if i := slices.IndexFunc(ids, func(id string) bool { return taken[id] }); i >= 0 {
			m.logger.Warn("Skipping module program with duplicate ID", zap.String("id", ids[i]), zap.String("path", p.path))
			continue
		}
		for _, id := range ids {
			taken[id] = true
		}
		programs = append(programs, *p)
	}
	m.mu.Lock()
	m.programs = programs
	m.mu.Unlock()
	m.logger.Info("Modules scanned", zap.Int("programs", len(programs)))
	return m.Installed()
}

// inspect runs the program with --info and validates the result.
func (m *Manager) inspect(ctx context.Context, path string) (plugin.Description, error) {
	ctx, cancel := context.WithTimeout(ctx, infoTimeout)
	defer cancel()
	desc, err := plugin.ReadInfo(ctx, path)
	if err != nil {
		return plugin.Description{}, err
	}
	for _, id := range (programFile{desc: desc}).ids() {
		if !validID.MatchString(id) {
			return plugin.Description{}, fmt.Errorf("invalid module ID %q", id)
		}
	}
	return desc, nil
}

// Inspect reads the description of a module program that is not
// installed (yet).
func (m *Manager) Inspect(ctx context.Context, path string) (plugin.Description, error) {
	return m.inspect(ctx, path)
}

// Install copies the module or driver program at src into the module
// directory as <id>.<GOOS>.<GOARCH>.vfgmod, replacing an installed program
// with the same ID. Only programs named for the host's platform are
// accepted. A program whose modules are running cannot be replaced, and
// none of its IDs may belong to another installed program.
func (m *Manager) Install(ctx context.Context, src string) (plugin.Description, error) {
	if !plugin.HasHostSuffix(src) {
		return plugin.Description{}, fmt.Errorf("%s is not a module program for this platform (*%s)", filepath.Base(src), plugin.HostSuffix)
	}
	desc, err := m.inspect(ctx, src)
	if err != nil {
		return plugin.Description{}, err
	}
	p := programFile{desc: desc}

	m.mu.Lock()
	defer m.mu.Unlock()
	i := m.indexLocked(desc.ID)
	if m.runningLocked(p) || i >= 0 && m.runningLocked(m.programs[i]) {
		return plugin.Description{}, fmt.Errorf("install %s: %w", desc.ID, ErrRunning)
	}
	for j, other := range m.programs {
		if j == i {
			continue
		}
		if k := slices.IndexFunc(p.ids(), func(id string) bool { return slices.Contains(other.ids(), id) }); k >= 0 {
			return plugin.Description{}, fmt.Errorf("install %s: ID %s is already used by %s", desc.ID, p.ids()[k], other.desc.ID)
		}
	}
	p.path = filepath.Join(m.dir, desc.ID+plugin.HostSuffix)
	if err := os.MkdirAll(m.dir, 0o755); err != nil {
		return plugin.Description{}, fmt.Errorf("create module directory: %w", err)
	}
	if !sameFile(src, p.path) {
		if err := copyFile(src, p.path); err != nil {
			return plugin.Description{}, fmt.Errorf("install %s: %w", desc.ID, err)
		}
	}
	fi, err := os.Stat(p.path)
	if err != nil {
		return plugin.Description{}, err
	}
	p.modTime, p.size = fi.ModTime(), fi.Size()

	if i >= 0 {
		// The same program may have been installed under another file name.
		if old := m.programs[i].path; !sameFile(old, p.path) {
			if err := os.Remove(old); err != nil {
				m.logger.Warn("Failed to remove the previous module program", zap.String("path", old), zap.Error(err))
			}
		}
		m.programs[i] = p
	} else {
		m.programs = append(m.programs, p)
		slices.SortFunc(m.programs, func(a, b programFile) int { return strings.Compare(a.path, b.path) })
	}
	m.logger.Info("Module installed", zap.String("id", desc.ID), zap.String("name", desc.Metadata.Name), zap.Bool("driver", desc.IsDriver()))
	return desc, nil
}

// Uninstall deletes an installed module or driver program. The modules
// inside a driver can only be removed with the driver. A program whose
// modules are running cannot be removed.
func (m *Manager) Uninstall(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	i := m.indexLocked(id)
	if i < 0 {
		if info, ok := m.infoLocked(id); ok {
			return fmt.Errorf("module %s is part of driver %s and can only be uninstalled with it", id, info.Driver)
		}
		return fmt.Errorf("unknown module %q", id)
	}
	if m.runningLocked(m.programs[i]) {
		return fmt.Errorf("uninstall %s: %w", id, ErrRunning)
	}
	if err := os.Remove(m.programs[i].path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("uninstall %s: %w", id, err)
	}
	m.programs = slices.Delete(m.programs, i, i+1)
	m.logger.Info("Module uninstalled", zap.String("id", id))
	return nil
}

func sameFile(a, b string) bool {
	fa, errA := os.Stat(a)
	fb, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(fa, fb)
}

// copyFile writes src to dst through a temporary file so a failed copy
// never leaves a broken program behind.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".tmp"
	// Module programs are executed, which needs the execute bit outside
	// Windows.
	out, err := os.OpenFile(tmp, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

// Start launches an installed module and initialises it with avail. A
// module inside a driver runs in its own process of the driver program. A
// module that fails or does not answer within InitTimeout is killed. The
// returned instance may have been granted nothing; the caller decides
// whether to keep it.
func (m *Manager) Start(ctx context.Context, id string, avail adapter.Support) (*Instance, error) {
	m.mu.Lock()
	info, ok := m.infoLocked(id)
	running := m.running[id] != nil
	m.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("unknown module %q", id)
	}
	if running {
		return nil, fmt.Errorf("start %s: %w", id, ErrRunning)
	}

	conn, err := m.launch(info.Path, info.source())
	if err != nil {
		return nil, err
	}
	timeout := m.InitTimeout()
	ictx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	granted, err := conn.Initialize(ictx, avail)
	if err != nil {
		_ = conn.Close()
		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("initialize did not finish within %s: %w", timeout, err)
		}
		return nil, err
	}

	inst := newInstance(m, info, conn, granted)
	m.mu.Lock()
	m.running[id] = inst
	m.mu.Unlock()
	inst.start(ctx)
	return inst, nil
}

// release forgets a stopped or crashed instance.
func (m *Manager) release(inst *Instance) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running[inst.info.ID] == inst {
		delete(m.running, inst.info.ID)
	}
}
