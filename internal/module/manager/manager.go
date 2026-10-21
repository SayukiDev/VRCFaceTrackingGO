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

	"VRCFaceTracking/internal/module/adapter"
	"VRCFaceTracking/internal/module/plugin"
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

// Info is an installed module.
type Info struct {
	plugin.Description
	// Path is the module program.
	Path string `json:"path"`
}

// cached is a scanned program together with what identified its version.
type cached struct {
	info    Info
	modTime time.Time
	size    int64
}

// Manager manages the module programs in one directory.
type Manager struct {
	logger *zap.Logger
	dir    string
	// launch starts a module program; replaced in tests.
	launch func(path string) (plugin.Conn, error)
	// updateTimeout and restartDelay drive crash recovery; shortened in
	// tests.
	updateTimeout time.Duration
	restartDelay  time.Duration

	mu        sync.Mutex
	installed []cached // file name order
	running   map[string]*Instance

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
	m.launch = func(path string) (plugin.Conn, error) { return plugin.Start(path, logger) }
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

// Installed returns the modules found by the last Scan, in file name order.
func (m *Manager) Installed() []Info {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Info, 0, len(m.installed))
	for _, c := range m.installed {
		out = append(out, c.info)
	}
	return out
}

// Info returns one installed module.
func (m *Manager) Info(id string) (Info, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if i := m.indexLocked(id); i >= 0 {
		return m.installed[i].info, true
	}
	return Info{}, false
}

func (m *Manager) indexLocked(id string) int {
	return slices.IndexFunc(m.installed, func(c cached) bool { return c.info.ID == id })
}

// Scan reads the module directory again. Programs that did not change
// since the last scan are not run again. Programs that fail --info, and
// duplicate IDs, are skipped.
func (m *Manager) Scan(ctx context.Context) []Info {
	entries, err := os.ReadDir(m.dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		m.logger.Warn("Cannot read module directory", zap.String("path", m.dir), zap.Error(err))
	}

	m.mu.Lock()
	previous := make(map[string]cached, len(m.installed))
	for _, c := range m.installed {
		previous[c.info.Path] = c
	}
	m.mu.Unlock()

	var scanned []*cached
	var wg sync.WaitGroup
	for _, e := range entries {
		if !e.Type().IsRegular() || !strings.EqualFold(filepath.Ext(e.Name()), plugin.Suffix) {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(m.dir, e.Name())
		c := &cached{info: Info{Path: path}, modTime: fi.ModTime(), size: fi.Size()}
		scanned = append(scanned, c)
		if p, ok := previous[path]; ok && p.modTime.Equal(c.modTime) && p.size == c.size {
			*c = p
			continue
		}
		wg.Go(func() {
			desc, err := m.inspect(ctx, path)
			if err != nil {
				m.logger.Warn("Skipping module program", zap.String("path", path), zap.Error(err))
				c.info.ID = ""
				return
			}
			c.info.Description = desc
		})
	}
	wg.Wait()

	installed := make([]cached, 0, len(scanned))
	for _, c := range scanned {
		if c.info.ID == "" {
			continue
		}
		if slices.ContainsFunc(installed, func(o cached) bool { return o.info.ID == c.info.ID }) {
			m.logger.Warn("Skipping module with duplicate ID", zap.String("id", c.info.ID), zap.String("path", c.info.Path))
			continue
		}
		installed = append(installed, *c)
	}
	m.mu.Lock()
	m.installed = installed
	m.mu.Unlock()
	m.logger.Info("Modules scanned", zap.Int("count", len(installed)))
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
	if !validID.MatchString(desc.ID) {
		return plugin.Description{}, fmt.Errorf("invalid module ID %q", desc.ID)
	}
	return desc, nil
}

// Inspect reads the description of a module program that is not
// installed (yet).
func (m *Manager) Inspect(ctx context.Context, path string) (plugin.Description, error) {
	return m.inspect(ctx, path)
}

// Install copies the module program at src into the module directory as
// <id>.vfgmod, replacing an installed module with the same ID. A running
// module cannot be replaced.
func (m *Manager) Install(ctx context.Context, src string) (Info, error) {
	desc, err := m.inspect(ctx, src)
	if err != nil {
		return Info{}, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running[desc.ID] != nil {
		return Info{}, fmt.Errorf("install %s: %w", desc.ID, ErrRunning)
	}
	dst := filepath.Join(m.dir, desc.ID+plugin.Suffix)
	if err := os.MkdirAll(m.dir, 0o755); err != nil {
		return Info{}, fmt.Errorf("create module directory: %w", err)
	}
	if !sameFile(src, dst) {
		if err := copyFile(src, dst); err != nil {
			return Info{}, fmt.Errorf("install %s: %w", desc.ID, err)
		}
	}
	fi, err := os.Stat(dst)
	if err != nil {
		return Info{}, err
	}

	info := Info{Description: desc, Path: dst}
	c := cached{info: info, modTime: fi.ModTime(), size: fi.Size()}
	if i := m.indexLocked(desc.ID); i >= 0 {
		// The same module may have been installed under another file name.
		if old := m.installed[i].info.Path; !sameFile(old, dst) {
			if err := os.Remove(old); err != nil {
				m.logger.Warn("Failed to remove the previous module program", zap.String("path", old), zap.Error(err))
			}
		}
		m.installed[i] = c
	} else {
		m.installed = append(m.installed, c)
		slices.SortFunc(m.installed, func(a, b cached) int { return strings.Compare(a.info.Path, b.info.Path) })
	}
	m.logger.Info("Module installed", zap.String("id", desc.ID), zap.String("name", desc.Metadata.Name))
	return info, nil
}

// Uninstall deletes an installed module. A running module cannot be
// removed.
func (m *Manager) Uninstall(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running[id] != nil {
		return fmt.Errorf("uninstall %s: %w", id, ErrRunning)
	}
	i := m.indexLocked(id)
	if i < 0 {
		return fmt.Errorf("unknown module %q", id)
	}
	if err := os.Remove(m.installed[i].info.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("uninstall %s: %w", id, err)
	}
	m.installed = slices.Delete(m.installed, i, i+1)
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
	out, err := os.Create(tmp)
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
// module that fails or does not answer within InitTimeout is killed. The
// returned instance may have been granted nothing; the caller decides
// whether to keep it.
func (m *Manager) Start(ctx context.Context, id string, avail adapter.Support) (*Instance, error) {
	m.mu.Lock()
	i := m.indexLocked(id)
	running := m.running[id] != nil
	m.mu.Unlock()
	if i < 0 {
		return nil, fmt.Errorf("unknown module %q", id)
	}
	if running {
		return nil, fmt.Errorf("start %s: %w", id, ErrRunning)
	}
	info, _ := m.Info(id)

	conn, err := m.launch(info.Path)
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
