//go:build !windows

package setup

import (
	"context"
	"task-processor/internal/app/setupcontrol"
)

func (d *Driver) desktopClientsIdle(context.Context) (bool, error) {
	return false, setupcontrol.ErrUnavailable
}
