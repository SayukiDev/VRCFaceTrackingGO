package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Launcher is the content of a launcher file (LauncherSuffix): instead of
// being a program itself it names the command that runs the module, e.g.
// an interpreter and its script:
//
//	{"Command": "python", "Args": ["main.py"]}
//
// The host runs Command with Args followed by its own flags (--info,
// --vfgmod=true, --source=...), in the folder of the launcher file. A
// relative Command that exists in that folder is run from there; otherwise
// it is looked up in PATH.
type Launcher struct {
	Command string   `json:"Command"`
	Args    []string `json:"Args"`
}

// IsLauncher reports whether path names a launcher file. Case is ignored.
func IsLauncher(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), LauncherSuffix)
}

// command builds the command that runs the module program or launcher
// file at path with the host's args, in the folder of path.
func command(ctx context.Context, path string, args ...string) (*exec.Cmd, error) {
	dir := filepath.Dir(path)
	name := path
	if IsLauncher(path) {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var l Launcher
		if err := json.Unmarshal(data, &l); err != nil {
			return nil, fmt.Errorf("decode launcher %s: %w", filepath.Base(path), err)
		}
		name = l.Command
		if p := filepath.Join(dir, name); !filepath.IsAbs(name) && isFile(p) {
			name = p
		}
		args = append(l.Args[:len(l.Args):len(l.Args)], args...)
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	return cmd, nil
}

func isFile(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}
