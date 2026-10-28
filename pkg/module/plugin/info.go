package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// ReadInfo runs the module program at path with --info and returns its
// Description. Use a ctx with a deadline; the program is killed when it
// expires.
func ReadInfo(ctx context.Context, path string) (Description, error) {
	cmd := exec.CommandContext(ctx, path, "--"+InfoFlag)
	cmd.Dir = filepath.Dir(path)
	prepare(cmd)
	out, err := cmd.Output()
	if err != nil {
		if exit := (*exec.ExitError)(nil); errors.As(err, &exit) && len(exit.Stderr) > 0 {
			err = fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exit.Stderr)))
		}
		return Description{}, fmt.Errorf("read module info: %w", err)
	}
	var d Description
	if err := json.Unmarshal(out, &d); err != nil {
		return Description{}, fmt.Errorf("decode module info: %w", err)
	}
	if err := d.validate(); err != nil {
		return Description{}, fmt.Errorf("module info: %w", err)
	}
	return d, nil
}

// validate checks the structure of a description read from a program.
func (d Description) validate() error {
	if d.ID == "" {
		return errors.New("no ID")
	}
	switch d.Kind {
	case "":
		if len(d.Modules) > 0 {
			return fmt.Errorf("module %s lists modules but is not a driver", d.ID)
		}
		return nil
	case KindDriver:
	default:
		return fmt.Errorf("unknown kind %q", d.Kind)
	}
	if len(d.Modules) == 0 {
		return fmt.Errorf("driver %s contains no modules", d.ID)
	}
	seen := make(map[string]bool, len(d.Modules))
	for _, m := range d.Modules {
		if m.Kind != "" || len(m.Modules) > 0 {
			return fmt.Errorf("driver %s: module %s must be a plain module", d.ID, m.ID)
		}
		if m.ID == "" {
			return fmt.Errorf("driver %s: a module has no ID", d.ID)
		}
		if m.ID == d.ID || seen[m.ID] {
			return fmt.Errorf("driver %s: duplicate ID %s", d.ID, m.ID)
		}
		seen[m.ID] = true
	}
	return nil
}
