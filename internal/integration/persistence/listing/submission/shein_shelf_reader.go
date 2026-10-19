package submissionpersistence

import (
	"context"
	"task-processor/internal/listing/submission"
)

func (r *Repository) ReadShelfAttempt(ctx context.Context, scope submission.ExecutionScope, target submission.ExecutionTarget) (submission.ExecutionAttempt, error) {
	if r == nil || ctx == nil || scope.OrganizationID == "" || !isShelfTarget(submission.ExecutionAttempt{Target: target}) {
		return submission.ExecutionAttempt{}, submission.ErrExecutionInvalid
	}
	fence, found, e := r.findTarget(ctx, r.db, scope.OrganizationID, target, false)
	if e != nil {
		return submission.ExecutionAttempt{}, e
	}
	if !found {
		return submission.ExecutionAttempt{}, submission.ErrExecutionNotFound
	}
	attempt, found, e := r.findByID(ctx, r.db, scope.OrganizationID, fence.CurrentAttemptID, false)
	if e != nil || !found || attempt.Target != target || attempt.FenceEpoch != fence.Epoch {
		return submission.ExecutionAttempt{}, submission.ErrExecutionUnavailable
	}
	return attempt, nil
}
