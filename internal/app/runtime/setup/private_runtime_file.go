package setup

import (
	"context"
	"task-processor/internal/app/setupcontrol"
)

// Windows ACLs protect the host file; Docker Desktop additionally exposes a
// Linux inode mode. Protect this newly-created candidate at both consumers.
// The sole writable mount is this file, never a retained directory or volume.
func (d *Driver) protectRuntimeFile(ctx context.Context, file string) error {
	before, e := readPrivate(file, 65536)
	if e != nil {
		return e
	}
	_, e = d.run(ctx, "run", "--rm", "--network", "none", "--read-only", "--cap-drop", "ALL", "--cap-add", "FOWNER", "--security-opt", "no-new-privileges", "--mount", "type=bind,src="+file+",dst=/runtime-manifest", "alpine:3.22", "chmod", "600", "/runtime-manifest")
	if e != nil {
		return e
	}
	after, e := readPrivate(file, 65536)
	if e != nil || hash(before) != hash(after) {
		return setupcontrol.ErrDrift
	}
	return nil
}
