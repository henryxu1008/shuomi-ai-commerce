package imageagentapp

import (
	"context"
	"testing"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent"
	"github.com/stretchr/testify/require"
)

func TestImageConfigurationCapacityReservesSecondConnectionAtEntry(t *testing.T) {
	ctx := context.Background()
	for _, poolSize := range []int{5, 8} {
		t.Run(map[int]string{5: "five", 8: "eight"}[poolSize], func(t *testing.T) {
			admit, activityLimit := NewConfigurationCapacity(poolSize)
			require.Equal(t, map[int]int{5: 1, 8: 2}[poolSize], activityLimit)
			for range activityLimit {
				release, err := admit.Acquire(ctx, 2)
				require.NoError(t, err)
				defer release()
			}
			if poolSize == 8 {
				release, err := admit.Acquire(ctx, 2)
				require.NoError(t, err, "the first confirmation fits beside two activities")
				release()
				release()
				release, err = admit.Acquire(ctx, 2)
				require.NoError(t, err)
				defer release()
			}
			_, err := admit.Acquire(ctx, 2)
			require.ErrorIs(t, err, imageagent.ErrImageConfigurationBusy)
			if poolSize == 5 {
				release, err := admit.Acquire(ctx, 1)
				require.NoError(t, err, "the refused confirmation did not hold a partial reservation")
				release()
			}
		})
	}
}
