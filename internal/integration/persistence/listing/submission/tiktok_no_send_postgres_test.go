package submissionpersistence

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/listing/submission"
)

func TestPostgresTikTokNoSendAtomicallyTerminatesAndPreservesOriginalKey(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, _ := openExecutionPostgres(t, ctx)
	require.NoError(t, InstallSchema(db))
	r, err := NewRepository(db)
	require.NoError(t, err)
	k, err := submission.NewExecutionKernel(r)
	require.NoError(t, err)
	command := executionCommand("org-a", "not-sent", "no-send-target", `{"approved":true}`)
	command.Target.Platform = "tiktokshop"
	command.Action = submission.TikTokPublishAction
	p, err := k.PrepareTikTokExecution(ctx, command)
	require.NoError(t, err)
	require.NotNil(t, p.Preparation)
	a, err := k.StopTikTokBeforeSend(ctx, p.Preparation, submission.NoSendFactsChanged)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionNotSent, a.Status)
	require.NoError(t, submission.ValidatePersistedExecutionAttempt(a))
	var fence targetFenceRow
	require.NoError(t, db.Table(targetTable).Take(&fence).Error)
	require.Equal(t, string(a.Status), fence.CurrentStatus)
	require.Equal(t, a.AttemptID, fence.CurrentAttemptID)
	replayed, err := k.PrepareTikTokExecution(ctx, command)
	require.NoError(t, err)
	require.Nil(t, replayed.Preparation)
	require.Equal(t, a, replayed.Attempt)
	changed := command
	changed.Payload = []byte(`{"changed":true}`)
	_, err = k.PrepareTikTokExecution(ctx, changed)
	require.ErrorIs(t, err, submission.ErrExecutionIntentConflict)
	_, err = k.Get(ctx, submission.ExecutionScope{OrganizationID: "other-org"}, a.AttemptID)
	require.ErrorIs(t, err, submission.ErrExecutionNotFound)
	command.IntentKey = "new-confirmation"
	next, err := k.PrepareTikTokExecution(ctx, command)
	require.NoError(t, err)
	require.NotNil(t, next.Preparation)
	require.Equal(t, a.FenceEpoch+1, next.Attempt.FenceEpoch)
	old, err := k.StopTikTokBeforeSend(ctx, p.Preparation, submission.NoSendFactsChanged)
	require.NoError(t, err)
	require.Equal(t, a, old)
	fence = targetFenceRow{}
	require.NoError(t, db.Table(targetTable).Take(&fence).Error)
	require.Equal(t, next.Attempt.AttemptID, fence.CurrentAttemptID)
	require.Equal(t, "claimed", fence.CurrentStatus)
	_, err = k.StopTikTokBeforeSend(ctx, next.Preparation, submission.NoSendPreflightUnavailable)
	require.NoError(t, err)
	// Competing explicit confirmations cannot both obtain the same target.
	var wg sync.WaitGroup
	results := make(chan submission.TikTokPreparedAcquisition, 2)
	failures := make(chan error, 2)
	for _, key := range []string{"concurrent-a", "concurrent-b"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			c := command
			c.IntentKey = key
			v, e := k.PrepareTikTokExecution(ctx, c)
			results <- v
			failures <- e
		}(key)
	}
	wg.Wait()
	close(results)
	close(failures)
	winners := 0
	var winner submission.TikTokPreparedAcquisition
	for result := range results {
		if result.Preparation != nil {
			winners++
			winner = result
		}
	}
	require.Equal(t, 1, winners)
	for e := range failures {
		if e != nil {
			require.ErrorIs(t, e, submission.ErrExecutionTargetClaimed)
		}
	}
	permit, err := k.BeginTikTokSend(ctx, winner.Preparation)
	require.NoError(t, err)
	_, err = k.StopTikTokBeforeSend(ctx, winner.Preparation, submission.NoSendFactsChanged)
	require.Error(t, err)
	_, err = k.MarkUnknown(ctx, permitClaim("org-a", permit), submission.UnknownResponseLost)
	require.NoError(t, err)
	command.IntentKey = "cannot-retry-unknown"
	_, err = k.PrepareTikTokExecution(ctx, command)
	require.ErrorIs(t, err, submission.ErrExecutionTargetClaimed)
	for _, mutation := range []string{"platform='shein'", "evidence_reason='untrusted_reason'", "finished_at=lease_expires_at,updated_at=lease_expires_at,evidence_observed_at=lease_expires_at"} {
		require.Error(t, db.Exec("UPDATE "+attemptTable+" SET "+mutation+" WHERE attempt_id=?", a.AttemptID).Error)
	}
	require.NoError(t, db.Exec("ALTER TABLE "+attemptTable+" DROP CONSTRAINT listing_submission_execution_attempts_no_send_check").Error)
	require.Error(t, VerifySchema(ctx, db), "serving must reject a weakened no-send contract")
}

func TestPostgresTikTokNoSendRejectsChangedClaimAndCommitAmbiguity(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, _ := openExecutionPostgres(t, ctx)
	require.NoError(t, InstallSchema(db))
	r, err := NewRepository(db)
	require.NoError(t, err)
	k, err := submission.NewExecutionKernel(r)
	require.NoError(t, err)
	for _, field := range []string{"claim_token_hash", "claim_owner_id", "payload_fingerprint"} {
		t.Run(field, func(t *testing.T) {
			c := executionCommand("org-a", "tamper-"+field, "target-"+field, `{"approved":true}`)
			c.Target.Platform = "tiktokshop"
			c.Action = submission.TikTokPublishAction
			p, e := k.PrepareTikTokExecution(ctx, c)
			require.NoError(t, e)
			var original string
			require.NoError(t, db.Table(attemptTable).Select(field).Where("attempt_id=?", p.Attempt.AttemptID).Scan(&original).Error)
			changed := "different-owner"
			if field != "claim_owner_id" {
				changed = strings.Repeat("a", 64)
			}
			require.NoError(t, db.Table(attemptTable).Where("attempt_id=?", p.Attempt.AttemptID).Update(field, changed).Error)
			_, e = k.StopTikTokBeforeSend(ctx, p.Preparation, submission.NoSendFactsChanged)
			require.Error(t, e)
			_, e = k.BeginTikTokSend(ctx, p.Preparation)
			require.Error(t, e, "failed closeout cannot reopen the gate")
			require.NoError(t, db.Table(attemptTable).Where("attempt_id=?", p.Attempt.AttemptID).Update(field, original).Error)
			_, e = k.StopTikTokBeforeSend(ctx, p.Preparation, submission.NoSendFactsChanged)
			require.NoError(t, e)
		})
	}
	for _, stage := range []string{"before_commit", "after_commit"} {
		t.Run(stage, func(t *testing.T) {
			c := executionCommand("org-a", "commit-"+stage, "target-"+stage, `{"approved":true}`)
			c.Target.Platform = "tiktokshop"
			c.Action = submission.TikTokImageAction
			p, e := k.PrepareTikTokExecution(ctx, c)
			require.NoError(t, e)
			r.fault = func(point string) error {
				if point == stage {
					return errors.New("isolated commit interruption")
				}
				return nil
			}
			result, e := k.StopTikTokBeforeSend(ctx, p.Preparation, submission.NoSendPreflightUnavailable)
			r.fault = nil
			if stage == "before_commit" {
				require.Error(t, e)
				persisted, e := k.Get(ctx, c.Scope, p.Attempt.AttemptID)
				require.NoError(t, e)
				require.Equal(t, submission.ExecutionClaimed, persisted.Status)
			} else {
				require.NoError(t, e)
				require.Equal(t, submission.ExecutionNotSent, result.Status, "readback recovers a committed no-send receipt")
			}
			_, e = k.BeginTikTokSend(ctx, p.Preparation)
			require.Error(t, e)
			_, e = k.StopTikTokBeforeSend(ctx, p.Preparation, submission.NoSendPreflightUnavailable)
			require.NoError(t, e)
		})
	}
	base := time.Now().UTC().Truncate(time.Microsecond)
	now := base
	r.now = func() time.Time { return now }
	k, err = submission.NewExecutionKernel(r, submission.WithExecutionClock(func() time.Time { return now }))
	require.NoError(t, err)
	c := executionCommand("org-a", "expired", "expired-target", `{"approved":true}`)
	c.Target.Platform = "tiktokshop"
	c.Action = submission.TikTokPublishAction
	c.Lease = time.Second
	p, err := k.PrepareTikTokExecution(ctx, c)
	require.NoError(t, err)
	now = base.Add(2 * time.Second)
	_, err = k.StopTikTokBeforeSend(ctx, p.Preparation, submission.NoSendFactsChanged)
	require.Error(t, err)
	a, err := k.Expire(ctx, c.Scope, p.Attempt.AttemptID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionOutcomeUnknown, a.Status)
	_, err = k.StopTikTokBeforeSend(ctx, p.Preparation, submission.NoSendFactsChanged)
	require.Error(t, err)
}
