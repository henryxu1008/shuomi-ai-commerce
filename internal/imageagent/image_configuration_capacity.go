package imageagent

import (
	"context"
	"errors"
	"sync"
)

var ErrImageConfigurationBusy = errors.New("image configuration capacity unavailable")

// ImageConfigurationCapacity reserves the operation's full peak use of the
// shared configuration pool before an outer consumer starts borrowing from it.
type ImageConfigurationCapacity func(context.Context, int64) (func(), error)

func (c ImageConfigurationCapacity) Acquire(ctx context.Context, connections int64) (func(), error) {
	if c == nil {
		return func() {}, nil
	}
	release, err := c(ctx, connections)
	if err != nil {
		return nil, err
	}
	return sync.OnceFunc(release), nil
}
