// Package format knows the forms a module takes on disk and turns what the
// user picks into an installed module.
//
// A module is distributed either as a single program file
// <id>.<GOOS>.<GOARCH>.vfgmod or as a module archive
// <name>.<GOOS>.<GOARCH>.vfgmodf: a zip file with index.vfgmod and the
// files it needs at its top level. In the module directory a program file
// is kept as <id>.<GOOS>.<GOARCH>.vfgmod and an archive is extracted into
// the folder <id>.<GOOS>.<GOARCH>.vfgmodf.
package format

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/plugin"
)

// InfoTimeout bounds how long a program may take to answer --info.
const InfoTimeout = 5 * time.Second

// Validate checks that the program at path is a working module or driver
// program: it runs it with --info (killing it after InfoTimeout) and
// checks the description it prints, including every ID it claims.
func Validate(ctx context.Context, path string) (plugin.Description, error) {
	ctx, cancel := context.WithTimeout(ctx, InfoTimeout)
	defer cancel()
	desc, err := plugin.ReadInfo(ctx, path)
	if err != nil {
		return plugin.Description{}, err
	}
	if err := ValidateIDs(desc); err != nil {
		return plugin.Description{}, err
	}
	return desc, nil
}

// Validate runs Validate on the staged program.
func (s *Staged) Validate(ctx context.Context) (plugin.Description, error) {
	return Validate(ctx, s.Program)
}

// validID keeps module IDs usable as file names.
var validID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// ValidateIDs checks that every ID a program claims (its own and those of
// a driver's modules) is usable as a file name.
func ValidateIDs(desc plugin.Description) error {
	ids := []string{desc.ID}
	for _, m := range desc.Modules {
		ids = append(ids, m.ID)
	}
	for _, id := range ids {
		if !validID.MatchString(id) {
			return fmt.Errorf("invalid module ID %q", id)
		}
	}
	return nil
}

// Location is where a module lies in the module directory.
type Location struct {
	// Root is what was installed: the program file or the module folder.
	Root string
	// Program is the program that is run.
	Program string
	// Folder reports whether Root is a module folder.
	Folder bool
}

// Locate recognises the entry e of the module directory dir as a module
// for the host's platform: a program file or a module folder. Its program
// is not checked.
func Locate(dir string, e os.DirEntry) (Location, bool) {
	root := filepath.Join(dir, e.Name())
	switch {
	case e.Type().IsRegular() && plugin.HasHostSuffix(e.Name()):
		return Location{Root: root, Program: root}, true
	case e.IsDir() && plugin.HasHostFolderSuffix(e.Name()):
		return Location{Root: root, Program: filepath.Join(root, plugin.FolderEntry), Folder: true}, true
	}
	return Location{}, false
}

// Staged is a module picked for installation, ready to be inspected
// (Program) and installed. Close it when done.
type Staged struct {
	Location
	// tmp is the folder an archive was extracted into, if any.
	tmp string
}

// Stage prepares src for installation: a program file is used where it
// is; a module archive is extracted into a new folder inside dir, the
// module directory, so Install can move it into place. Folders named
// .install-* are not modules, so a scan meanwhile ignores it.
func Stage(dir, src string) (*Staged, error) {
	switch {
	case plugin.HasHostSuffix(src):
		return &Staged{Location: Location{Root: src, Program: src}}, nil
	case !plugin.HasHostFolderSuffix(src):
		return nil, fmt.Errorf("%s is not a module for this platform (*%s or *%s)",
			filepath.Base(src), plugin.HostSuffix, plugin.HostFolderSuffix)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create module directory: %w", err)
	}
	tmp, err := os.MkdirTemp(dir, ".install-")
	if err != nil {
		return nil, err
	}
	if err := Extract(src, tmp); err != nil {
		_ = os.RemoveAll(tmp)
		return nil, fmt.Errorf("extract %s: %w", filepath.Base(src), err)
	}
	return &Staged{
		Location: Location{Root: tmp, Program: filepath.Join(tmp, plugin.FolderEntry), Folder: true},
		tmp:      tmp,
	}, nil
}

// Close removes what is left of an extracted archive.
func (s *Staged) Close() error {
	if s.tmp == "" {
		return nil
	}
	return os.RemoveAll(s.tmp)
}

// Install puts the staged module into dir under id, replacing a module of
// the same form and ID, and returns where it now lies.
func (s *Staged) Install(dir, id string) (Location, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Location{}, fmt.Errorf("create module directory: %w", err)
	}
	if s.Folder {
		root := filepath.Join(dir, id+plugin.HostFolderSuffix)
		if err := replaceDir(s.Root, root); err != nil {
			return Location{}, err
		}
		return Location{Root: root, Program: filepath.Join(root, plugin.FolderEntry), Folder: true}, nil
	}
	root := filepath.Join(dir, id+plugin.HostSuffix)
	if !SameFile(s.Root, root) {
		if err := copyFile(s.Root, root); err != nil {
			return Location{}, err
		}
	}
	return Location{Root: root, Program: root}, nil
}

// Extract unpacks the module archive src into the empty folder dst. The
// archive must hold the entry program at its top level.
func Extract(src, dst string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	// As an fs.FS the archive only yields valid, local paths.
	if err := os.CopyFS(dst, r); err != nil {
		return err
	}
	// The entry program is executed, which needs the execute bit outside
	// Windows.
	if err := os.Chmod(filepath.Join(dst, plugin.FolderEntry), 0o755); err != nil {
		return fmt.Errorf("no %s at the top of the archive: %w", plugin.FolderEntry, err)
	}
	return nil
}

// SameFile reports whether a and b exist and are the same file or folder.
func SameFile(a, b string) bool {
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

// replaceDir moves the folder src to dst, replacing what is there. The old
// folder is moved aside first and restored if the move fails.
func replaceDir(src, dst string) error {
	old := dst + ".old"
	if err := os.RemoveAll(old); err != nil {
		return err
	}
	if err := os.Rename(dst, old); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(src, dst); err != nil {
		_ = os.Rename(old, dst)
		return err
	}
	return os.RemoveAll(old)
}
