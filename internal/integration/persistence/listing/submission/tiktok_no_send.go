package submissionpersistence

import (
	"context"
	"crypto/subtle"

	"gorm.io/gorm"
	"task-processor/internal/listing/submission"
)

func (r *Repository) StopTikTokBeforeSend(ctx context.Context, proof submission.TikTokNoSendProof) (submission.ExecutionAttempt, error) {
	if r == nil || r.transactionFinalizer {
		return submission.ExecutionAttempt{}, submission.ErrExecutionUnavailable
	}
	claim, err := proof.Claim()
	if err != nil {
		return submission.ExecutionAttempt{}, err
	}
	return r.write(ctx, func(tx *gorm.DB) (mutation, error) {
		attempt, tokenHash, err := r.lockAttempt(ctx, tx, claim.Scope.OrganizationID, claim.AttemptID)
		if err != nil {
			return mutation{}, err
		}
		if attempt.FenceEpoch != claim.FenceEpoch || attempt.ClaimOwnerID != claim.OwnerID || subtle.ConstantTimeCompare([]byte(tokenHash), []byte(submission.ExecutionClaimTokenHash(claim.Token))) != 1 {
			return mutation{}, submission.ErrExecutionClaimRejected
		}
		// Idempotent old proof reads its own terminal fact, never a new fence.
		if attempt.Status == submission.ExecutionNotSent {
			previous, err := proof.Terminate(attempt, attempt.UpdatedAt)
			return mutation{attempt: previous}, err
		}
		if _, err = r.lockCurrentFence(ctx, tx, attempt); err != nil {
			return mutation{}, err
		}
		now, err := r.mutationTime(attempt)
		if err != nil {
			return mutation{}, err
		}
		updated, err := proof.Terminate(attempt, now)
		if err != nil {
			return mutation{}, err
		}
		if err = r.persistAttemptAndFence(ctx, tx, updated); err != nil {
			return mutation{}, err
		}
		return mutation{attempt: updated}, nil
	})
}

var _ submission.TikTokNoSendRepository = (*Repository)(nil)
