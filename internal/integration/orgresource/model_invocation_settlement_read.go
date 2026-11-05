package orgresourceadapter

import (
	"context"
	"encoding/json"
	"time"

	"task-processor/internal/ledger/orgresource"
)

type modelPointFinalResult struct {
	State, Proof string
	Points       int64
}

// ConfirmModelInvocationSettlement verifies the original resource transaction's
// immutable receipt. An invocation result alone does not prove settlement.
func (r *GormModelInvocationRepository) ConfirmModelInvocationSettlement(ctx context.Context, org, invocationID, runID string) error {
	fact, err := r.readFact(ctx, org, invocationID)
	if err != nil {
		return err
	}
	if runID == "" || fact.AgentRunID != runID {
		return orgresource.ErrInvalidOwnerProof
	}
	if err := r.authorizer.AuthorizeModelInvocation(ctx, fact); err != nil {
		return err
	}
	points, noDispatch, err := modelPointTerminal(fact)
	if err != nil {
		return err
	}
	fact.StartedAt = fact.StartedAt.UTC().Truncate(time.Microsecond)
	fact.FinishedAt = fact.FinishedAt.UTC().Truncate(time.Microsecond)
	proof := modelPointFactHash(fact)
	state := string(orgresource.ReservationCommitted)
	if noDispatch {
		state = string(orgresource.ReservationReleased)
	}
	var final organizationResourceOperationRow
	if err := r.db.WithContext(ctx).Where("organization_id = ? AND operation_id = ?", org, modelPointKey("finalize", invocationID)).Take(&final).Error; err != nil {
		return err
	}
	var result modelPointFinalResult
	if final.State != "succeeded" || final.OperationType != "model_point_finalize" || final.RequestFingerprint != proof ||
		final.CompletedAt == nil || json.Unmarshal([]byte(final.ImmutableResult), &result) != nil ||
		result != (modelPointFinalResult{State: state, Proof: proof, Points: points}) {
		return orgresource.ErrInvalidOwnerProof
	}
	return nil
}
