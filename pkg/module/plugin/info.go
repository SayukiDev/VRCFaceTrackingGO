package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ReadInfo runs the module program at path with --info and returns its
// Description. Use a ctx with a deadline; the program is killed when it
// expires.
func ReadInfo(ctx context.Context, path string) (Description, error) {
	cmd := exec.CommandContext(ctx, path, "--"+InfoFlag)
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
	if d.ID == "" {
		return Description{}, errors.New("module info has no ID")
	}
	return d, nil
}
