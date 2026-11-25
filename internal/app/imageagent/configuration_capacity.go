package imageagentapp

import (
	"context"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent"
	"golang.org/x/sync/semaphore"
)

// NewConfigurationCapacity shares an admission budget across the picture
// HTTP and activity consumers of one already-configured SQL pool.
func NewConfigurationCapacity(maxConnections int) (imageagent.ImageConfigurationCapacity, int) {
	budget := semaphore.NewWeighted(int64(max(0, maxConnections-2)))
	activities := 1
	if maxConnections >= 8 {
		activities = 2
	}
	return func(ctx context.Context, connections int64) (func(), error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !budget.TryAcquire(connections) {
			return nil, imageagent.ErrImageConfigurationBusy
		}
		return func() { budget.Release(connections) }, nil
	}, activities
}
