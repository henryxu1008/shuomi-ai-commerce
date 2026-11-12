//go:build integration

package workbenchschedules

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"os"
	"task-processor/internal/aiworkbench"
	"testing"
	"time"
)

type receivedSlot struct {
	id   string
	slot time.Time
}
type captureDispatch struct{ received chan receivedSlot }

func (d captureDispatch) DispatchScheduled(_ context.Context, id string, slot time.Time) error {
	d.received <- receivedSlot{id, slot}
	return nil
}
func (captureDispatch) ReconcileScheduled(context.Context, string, time.Time) (bool, error) {
	return true, nil
}

func TestTemporalScheduleRegistrationUsesVerifiedNominalSlot(t *testing.T) {
	if os.Getenv("ISSUE1244_TEMPORAL_TEST") != "1" {
		t.Skip("requires task-owned Temporal dev server via SDK")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	server, err := testsuite.StartDevServer(ctx, testsuite.DevServerOptions{CachedDownload: testsuite.CachedDownload{Version: "default"}, LogLevel: "error"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, server.Stop()) })
	b := &Backend{Client: server.Client()}
	d := captureDispatch{received: make(chan receivedSlot, 1)}
	stop, err := b.Start(d)
	require.NoError(t, err)
	t.Cleanup(stop)
	now := time.Now().UTC().Truncate(time.Second)
	plan := aiworkbench.ScheduledPlan{ScheduledPlanDraft: aiworkbench.ScheduledPlanDraft{ID: uuid.NewString(), CreatedAt: now, Rule: aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleOnce, TimeZone: "UTC", AtUTC: now.Add(10 * time.Second)}}, Revision: 1, State: "PENDING_REGISTRATION"}
	require.NoError(t, b.Synchronize(ctx, plan))
	plan.State, plan.Revision = "PAUSED", 2
	require.NoError(t, b.Synchronize(ctx, plan))
	plan.State, plan.Revision = "ACTIVE", 3
	require.NoError(t, b.Synchronize(ctx, plan))
	stale := plan
	stale.Revision = 2
	stale.State = "PAUSED"
	require.ErrorIs(t, b.Synchronize(ctx, stale), aiworkbench.ErrRevisionMismatch, "a late older projection must not overwrite a newer schedule")
	select {
	case got := <-d.received:
		require.Equal(t, plan.ID, got.id)
		require.True(t, got.slot.Equal(plan.Rule.AtUTC))
	case <-time.After(25 * time.Second):
		t.Fatal("scheduled workflow did not reach the local dispatch port")
	}
}
