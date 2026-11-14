package orgresourceadapter

import (
	"context"
	"encoding/json"
	"time"

	"task-processor/internal/aicapability"
	"task-processor/internal/ledger/orgresource"
)

type modelPointFinalResult struct {
	State, Proof string
	Points       int64
}

// ConfirmModelInvocationNoDispatchRelease verifies the existing owner's
// immutable zero-consumption finalization without releasing anything.
func (r *GormModelInvocationRepository) ConfirmModelInvocationNoDispatchRelease(ctx context.Context, org, invocationID string) error {
	fact, err := r.readFact(ctx, org, invocationID)
	if err != nil {
		return err
	}
	if fact.Operation != aicapability.OperationAIWorkbenchStoreRoute || fact.AgentRunID != "" || fact.BusinessTaskID != "" {
		return orgresource.ErrInvalidOwnerProof
	}
	if err = r.authorizer.AuthorizeModelInvocation(ctx, fact); err != nil {
		return err
	}
	points, noDispatch, err := modelPointTerminal(fact)
	if err != nil || !noDispatch || points != 0 {
		return orgresource.ErrInvalidOwnerProof
	}
	fact.StartedAt = fact.StartedAt.UTC().Truncate(time.Microsecond)
	fact.FinishedAt = fact.FinishedAt.UTC().Truncate(time.Microsecond)
	proof := modelPointFactHash(fact)
	var final organizationResourceOperationRow
	if err = r.db.WithContext(ctx).Where("organization_id=? AND operation_id=?", org, modelPointKey("finalize", invocationID)).Take(&final).Error; err != nil {
		return err
	}
	var result modelPointFinalResult
	if final.State != "succeeded" || final.OperationType != "model_point_finalize" || final.RequestFingerprint != proof || final.CompletedAt == nil || json.Unmarshal([]byte(final.ImmutableResult), &result) != nil || result != (modelPointFinalResult{State: "released", Proof: proof, Points: 0}) {
		return orgresource.ErrInvalidOwnerProof
	}
	var reserve organizationResourceOperationRow
	if err = r.db.WithContext(ctx).Where("organization_id=? AND operation_id=?", org, modelPointKey("reserve", invocationID)).Take(&reserve).Error; err != nil {
		return err
	}
	if reserve.OperationType != "model_point_reserve" || reserve.RequestFingerprint != modelPointFingerprint(fact) {
		return orgresource.ErrInvalidOwnerProof
	}
	if reserve.State == "failed" && reserve.FailureCode == "model_not_dispatched" {
		return nil
	}
	if reserve.State != "succeeded" {
		return orgresource.ErrInvalidOwnerProof
	}
	view, err := r.InspectModelInvocationReservation(ctx, org, invocationID)
	if err != nil || view.MemberID != fact.MemberID || view.State != "released" || view.SettledAt == nil || view.SettledAt.IsZero() {
		return orgresource.ErrInvalidOwnerProof
	}
	return nil
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
