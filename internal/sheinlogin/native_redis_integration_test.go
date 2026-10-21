//go:build integration

package sheinlogin

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestCurrentRedisNativeAtomicQueueAndCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	c, e := tc.GenericContainer(ctx, tc.GenericContainerRequest{ContainerRequest: tc.ContainerRequest{Image: "redis:7-alpine", ExposedPorts: []string{"6379/tcp"}, WaitingFor: wait.ForListeningPort("6379/tcp")}, Started: true})
	if c != nil {
		t.Cleanup(func() { _ = c.Terminate(context.Background()) })
	}
	require.NoError(t, e)
	host, e := c.Host(ctx)
	require.NoError(t, e)
	port, e := c.MappedPort(ctx, "6379/tcp")
	require.NoError(t, e)
	client := redis.NewClient(&redis.Options{Addr: host + ":" + port.Port()})
	t.Cleanup(func() { client.Close() })
	s, e := NewRedisStoreFromClient(client, uuid.NewString())
	require.NoError(t, e)
	require.NoError(t, s.Initialize(ctx))
	require.NoError(t, s.EnsureGroup(ctx))
	attempt := fixtureAttempt()
	attempt.Scope.InstallationID = s.installation
	first, e := s.Enqueue(ctx, attempt)
	require.NoError(t, e)
	again, e := s.Enqueue(ctx, attempt)
	require.NoError(t, e)
	require.Equal(t, first.ID, again.ID)
	other := fixtureAttempt()
	other.ID = uuid.NewString()
	other.Scope.InstallationID = s.installation
	other.Scope.StoreID = uuid.NewString()
	_, e = s.Enqueue(ctx, other)
	require.NoError(t, e)
	jobs, e := s.ReadJobs(ctx, "synthetic-consumer")
	require.NoError(t, e)
	require.Len(t, jobs, 1)
	pending, e := client.XPending(ctx, s.stream(), attemptGroup).Result()
	require.NoError(t, e)
	require.EqualValues(t, 1, pending.Count, "delivery must not prefetch a second execution slot")
	launching := *first
	launching.Status = LoginAttemptLaunching
	require.NoError(t, s.CompareAndSwap(ctx, *first, launching, nil))
	waiting := launching
	waiting.Status = LoginAttemptWaitingVerifyCode
	require.NoError(t, s.CompareAndSwap(ctx, launching, waiting, nil))
	launching = waiting
	require.NoError(t, s.SubmitCode(ctx, launching, "123456"))
	code, e := s.ConsumeCode(ctx, launching.ID)
	require.NoError(t, e)
	require.Equal(t, "123456", code)
	code, e = s.ConsumeCode(ctx, launching.ID)
	require.NoError(t, e)
	require.Empty(t, code)
	require.NoError(t, s.Cancel(ctx, launching))
	late := launching
	late.Status = LoginAttemptSucceeded
	late.SyncStatus = "pending"
	require.Error(t, s.CompareAndSwap(ctx, launching, late, syntheticState()))
	_, exists, e := s.Cookie(ctx, attempt.Scope)
	require.NoError(t, e)
	require.False(t, exists)
	require.NoError(t, s.Ack(ctx, jobs[0].ID))
	next, e := s.ReadJobs(ctx, "second-consumer")
	require.NoError(t, e)
	require.Len(t, next, 1)
	require.Equal(t, other.ID, next[0].Values["attempt_id"])
	require.NoError(t, s.Ack(ctx, next[0].ID))
	legacy, e := client.Keys(ctx, "shein:*").Result()
	require.NoError(t, e)
	require.Empty(t, legacy)
}
