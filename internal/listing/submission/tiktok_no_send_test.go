package submission

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type noSendMemoryRepository struct {
	ExecutionRepository
	mu       sync.Mutex
	attempts map[string]ExecutionAttempt
}

func (r *noSendMemoryRepository) Acquire(_ context.Context, reservation ExecutionReservation) (ExecutionAttempt, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if a, ok := r.attempts[reservation.Attempt.IntentKey]; ok {
		return a, false, nil
	}
	a := reservation.Attempt
	a.FenceEpoch = 1
	r.attempts[a.IntentKey] = a
	return a, true, nil
}
func (r *noSendMemoryRepository) Get(_ context.Context, scope ExecutionScope, id string) (ExecutionAttempt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, a := range r.attempts {
		if a.OrganizationID == scope.OrganizationID && a.AttemptID == id {
			return a, nil
		}
	}
	return ExecutionAttempt{}, ErrExecutionNotFound
}
func (r *noSendMemoryRepository) StopTikTokBeforeSend(_ context.Context, proof TikTokNoSendProof) (ExecutionAttempt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	claim, err := proof.Claim()
	if err != nil {
		return ExecutionAttempt{}, err
	}
	for key, a := range r.attempts {
		if a.AttemptID == claim.AttemptID {
			next, e := proof.Terminate(a, a.CreatedAt.Add(time.Millisecond))
			if e == nil {
				r.attempts[key] = next
			}
			return next, e
		}
	}
	return ExecutionAttempt{}, ErrExecutionNotFound
}

func noSendKernelFixture(t *testing.T) (*ExecutionKernel, AcquireExecutionCommand) {
	t.Helper()
	r := &noSendMemoryRepository{attempts: map[string]ExecutionAttempt{}}
	now := time.Date(2026, 10, 31, 1, 0, 0, 0, time.UTC)
	k, err := NewExecutionKernel(r, WithExecutionClock(func() time.Time { return now }))
	require.NoError(t, err)
	return k, AcquireExecutionCommand{Scope: ExecutionScope{OrganizationID: "org"}, IntentKey: "intent", Target: ExecutionTarget{Platform: "tiktokshop", StoreID: "store", SubjectID: "product"}, Action: TikTokPublishAction, Payload: []byte(`{"approved":true}`), ClaimOwnerID: "worker", Lease: time.Minute}
}

func TestTikTokPreparationClosesAllCopiesAndOriginalKeyCannotSend(t *testing.T) {
	k, command := noSendKernelFixture(t)
	prepared, err := k.PrepareTikTokExecution(context.Background(), command)
	require.NoError(t, err)
	require.NotNil(t, prepared.Preparation)
	copied := *prepared.Preparation
	a, err := k.StopTikTokBeforeSend(context.Background(), &copied, NoSendFactsChanged)
	require.NoError(t, err)
	require.Equal(t, ExecutionNotSent, a.Status)
	require.NoError(t, ValidatePersistedExecutionAttempt(a))
	_, err = k.BeginTikTokSend(context.Background(), prepared.Preparation)
	require.Error(t, err)
	same, err := k.StopTikTokBeforeSend(context.Background(), prepared.Preparation, NoSendFactsChanged)
	require.NoError(t, err)
	require.Equal(t, a, same)
	_, err = k.StopTikTokBeforeSend(context.Background(), prepared.Preparation, NoSendPayloadInvalid)
	require.ErrorIs(t, err, ErrExecutionIntentConflict)
	replayed, err := k.PrepareTikTokExecution(context.Background(), command)
	require.NoError(t, err)
	require.Nil(t, replayed.Preparation)
	require.Equal(t, a, replayed.Attempt)
	_, err = CompleteClaimedExecution(a, *a.Evidence, a.UpdatedAt)
	require.ErrorIs(t, err, ErrExecutionEvidenceRequired)
	_, err = ResolveUnknownExecution(a, *a.Evidence, a.UpdatedAt)
	require.ErrorIs(t, err, ErrExecutionEvidenceRequired)
	var restored TikTokPreparation
	raw, err := json.Marshal(prepared.Preparation)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &restored))
	_, err = k.BeginTikTokSend(context.Background(), &restored)
	require.Error(t, err)
	_, err = k.StopTikTokBeforeSend(context.Background(), &restored, NoSendFactsChanged)
	require.Error(t, err)
}

func TestTikTokBeginAndNoSendAreMutuallyExclusive(t *testing.T) {
	for n := 0; n < 80; n++ {
		k, command := noSendKernelFixture(t)
		p, err := k.PrepareTikTokExecution(context.Background(), command)
		require.NoError(t, err)
		copyHandle := *p.Preparation
		start := make(chan struct{})
		send := make(chan error, 1)
		stop := make(chan error, 1)
		go func() { <-start; _, e := k.BeginTikTokSend(context.Background(), p.Preparation); send <- e }()
		go func() {
			<-start
			_, e := k.StopTikTokBeforeSend(context.Background(), &copyHandle, NoSendFactsChanged)
			stop <- e
		}()
		close(start)
		sendErr, stopErr := <-send, <-stop
		require.NotEqual(t, sendErr == nil, stopErr == nil, "exactly one branch may win")
		_, err = k.BeginTikTokSend(context.Background(), p.Preparation)
		require.Error(t, err)
	}
}

func TestTikTokPreparationCannotAdoptBarePermitOrOtherKernel(t *testing.T) {
	k, command := noSendKernelFixture(t)
	raw, err := k.Acquire(context.Background(), command)
	require.NoError(t, err)
	require.NotNil(t, raw.Permit)
	p, err := k.PrepareTikTokExecution(context.Background(), command)
	require.NoError(t, err)
	require.Nil(t, p.Preparation)
	command.IntentKey = "prepared"
	p, err = k.PrepareTikTokExecution(context.Background(), command)
	require.NoError(t, err)
	other, _ := noSendKernelFixture(t)
	_, err = other.StopTikTokBeforeSend(context.Background(), p.Preparation, NoSendFactsChanged)
	require.Error(t, err)
	permit, err := k.BeginTikTokSend(context.Background(), p.Preparation)
	require.NoError(t, err)
	require.NotNil(t, permit)
	_, err = k.StopTikTokBeforeSend(context.Background(), p.Preparation, NoSendFactsChanged)
	require.Error(t, err)
	command.Target.Platform = "shein"
	_, err = k.PrepareTikTokExecution(context.Background(), command)
	require.Error(t, err)
}
