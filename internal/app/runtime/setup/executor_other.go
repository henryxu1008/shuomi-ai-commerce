//go:build !windows

package setup

import (
	"context"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/app/setupcontrol"
)

func (d *Driver) desktopClientsIdle(context.Context) (bool, error) {
	return false, setupcontrol.ErrUnavailable
}
