package submission

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestShelfIdentityAndConservativeCooldown(t *testing.T) {
	a, e := ShelfSubjectID("skc-1", "shein-us")
	require.NoError(t, e)
	b, e := ShelfSubjectID("skc-1", "shein-us")
	require.NoError(t, e)
	require.Equal(t, a, b)
	_, e = ShelfSubjectID("skc-1", "shein-ca")
	require.Error(t, e)
	claim := time.Now().UTC()
	finish := claim.Add(8 * time.Second)
	previous := ExecutionAttempt{Status: ExecutionFailedDefinitive, CreatedAt: claim, FinishedAt: &finish, LeaseExpiresAt: claim.Add(time.Minute)}
	require.False(t, ShelfCooldownReady(previous, finish.Add(20*time.Minute)))
	require.True(t, ShelfCooldownReady(previous, finish.Add(20*time.Minute+time.Nanosecond)))
	previous.Status = ExecutionOutcomeUnknown
	require.False(t, ShelfCooldownReady(previous, claim.Add(24*time.Hour)))
	previous.Status = ExecutionSucceeded
	require.False(t, ShelfCooldownReady(previous, claim.Add(24*time.Hour)))
	previous.Status = ExecutionCancelled
	previous.FinishedAt = nil
	require.False(t, ShelfCooldownReady(previous, claim.Add(20*time.Minute+30*time.Second)))
	require.True(t, ShelfCooldownReady(previous, claim.Add(21*time.Minute+time.Nanosecond)))
}
