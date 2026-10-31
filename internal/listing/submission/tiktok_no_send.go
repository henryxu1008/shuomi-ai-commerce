package submission

import (
	"context"
	"strconv"
	"sync"
	"time"
)

const (
	ExecutionNotSent           ExecutionStatus = "not_sent"
	EvidenceLocalNoSend        EvidenceKind    = "local_no_send"
	NoSendPreflightUnavailable                 = "preflight_unavailable"
	NoSendFactsChanged                         = "facts_changed"
	NoSendAccessRecheckFailed                  = "access_recheck_failed"
	NoSendImageContentChanged                  = "image_content_changed"
	NoSendPayloadInvalid                       = "payload_invalid"
)

// The repository receives proof only after the original live gate has closed.
// A bare ExecutionClaim or caller-supplied evidence cannot construct it.
type TikTokNoSendRepository interface {
	StopTikTokBeforeSend(context.Context, TikTokNoSendProof) (ExecutionAttempt, error)
}

type TikTokPreparedAcquisition struct {
	Attempt     ExecutionAttempt
	Preparation *TikTokPreparation
	Replayed    bool
}

// Copies share the gate; serializing or replaying an attempt cannot restore it.
type TikTokPreparation struct{ gate *tikTokPreparationGate }
type tikTokPreparationGate struct {
	mu      sync.Mutex
	kernel  *ExecutionKernel
	attempt ExecutionAttempt
	permit  SendPermit
	started bool
	proof   *tikTokNoSendSeal
}
type TikTokNoSendProof struct{ seal *tikTokNoSendSeal }
type tikTokNoSendSeal struct {
	attempt ExecutionAttempt
	claim   ExecutionClaim
	reason  string
}

func (k *ExecutionKernel) PrepareTikTokExecution(ctx context.Context, command AcquireExecutionCommand) (TikTokPreparedAcquisition, error) {
	if k == nil || command.Target.Platform != "tiktokshop" || !tikTokNoSendAction(command.Action) {
		return TikTokPreparedAcquisition{}, ErrExecutionInvalid
	}
	if _, ok := k.repository.(TikTokNoSendRepository); !ok {
		return TikTokPreparedAcquisition{}, ErrExecutionUnavailable
	}
	acquired, err := k.Acquire(ctx, command)
	if err != nil {
		return TikTokPreparedAcquisition{}, err
	}
	result := TikTokPreparedAcquisition{Attempt: acquired.Attempt, Replayed: acquired.Replayed}
	if acquired.Permit != nil {
		result.Preparation = &TikTokPreparation{gate: &tikTokPreparationGate{kernel: k, attempt: acquired.Attempt, permit: *acquired.Permit}}
	}
	return result, nil
}

func (k *ExecutionKernel) BeginTikTokSend(ctx context.Context, preparation *TikTokPreparation) (*SendPermit, error) {
	if k == nil || ctx == nil || preparation == nil || preparation.gate == nil {
		return nil, ErrExecutionInvalid
	}
	g := preparation.gate
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.kernel != k {
		return nil, ErrExecutionInvalid
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if g.started || g.proof != nil || !canonicalExecutionTime(k.now()).Before(g.permit.LeaseExpiresAt) {
		return nil, ErrExecutionClaimRejected
	}
	g.started = true
	permit := g.permit
	return &permit, nil
}

func (k *ExecutionKernel) StopTikTokBeforeSend(ctx context.Context, preparation *TikTokPreparation, reason string) (ExecutionAttempt, error) {
	if k == nil || preparation == nil || preparation.gate == nil || !validNoSendReason(reason) {
		return ExecutionAttempt{}, ErrExecutionInvalid
	}
	g := preparation.gate
	g.mu.Lock()
	if g.kernel != k || g.started {
		g.mu.Unlock()
		return ExecutionAttempt{}, ErrExecutionClaimRejected
	}
	if g.proof != nil && g.proof.reason != reason {
		g.mu.Unlock()
		return ExecutionAttempt{}, ErrExecutionIntentConflict
	}
	if g.proof == nil {
		g.proof = &tikTokNoSendSeal{attempt: g.attempt, reason: reason, claim: ExecutionClaim{Scope: ExecutionScope{OrganizationID: g.attempt.OrganizationID}, AttemptID: g.permit.AttemptID, FenceEpoch: g.permit.FenceEpoch, OwnerID: g.permit.ClaimOwnerID, Token: g.permit.ClaimToken}}
	}
	proof := TikTokNoSendProof{seal: g.proof}
	g.mu.Unlock() // No database or provider I/O holds the process-local lock.
	ctx, cancel, err := executionContext(ctx)
	if err != nil {
		return ExecutionAttempt{}, err
	}
	defer cancel()
	r, ok := k.repository.(TikTokNoSendRepository)
	if !ok {
		return ExecutionAttempt{}, ErrExecutionUnavailable
	}
	stopped, err := r.StopTikTokBeforeSend(ctx, proof)
	if err != nil {
		// A lost commit acknowledgement never reopens the gate or resends.
		if prior, e := k.repository.Get(ctx, proof.seal.claim.Scope, proof.seal.claim.AttemptID); e == nil && prior.Status == ExecutionNotSent {
			if validated, e := proof.Terminate(prior, prior.UpdatedAt); e == nil {
				return validated, nil
			}
		}
		return ExecutionAttempt{}, err
	}
	if stopped.Status != ExecutionNotSent {
		return ExecutionAttempt{}, ErrExecutionUnavailable
	}
	return proof.Terminate(stopped, stopped.UpdatedAt)
}

func (p TikTokNoSendProof) Claim() (ExecutionClaim, error) {
	if p.seal == nil || !validExecutionClaim(p.seal.claim) {
		return ExecutionClaim{}, ErrExecutionEvidenceRequired
	}
	return p.seal.claim, nil
}

// Terminate is shared with the adapter so persisted and in-memory facts agree.
// Only a proof issued by the closed live gate can invoke this transition.
func (p TikTokNoSendProof) Terminate(attempt ExecutionAttempt, now time.Time) (ExecutionAttempt, error) {
	if p.seal == nil || ValidatePersistedExecutionAttempt(attempt) != nil || !validNoSendReason(p.seal.reason) {
		return ExecutionAttempt{}, ErrExecutionEvidenceRequired
	}
	original := p.seal.attempt
	if ValidateExecutionReplay(attempt, original) != nil || attempt.AttemptID != original.AttemptID || attempt.ClaimOwnerID != original.ClaimOwnerID || attempt.FenceEpoch != original.FenceEpoch || !attempt.CreatedAt.Equal(original.CreatedAt) || !attempt.LeaseExpiresAt.Equal(original.LeaseExpiresAt) {
		return ExecutionAttempt{}, ErrExecutionClaimRejected
	}
	if attempt.Status == ExecutionNotSent {
		if attempt.Evidence.Reason != p.seal.reason {
			return ExecutionAttempt{}, ErrExecutionIntentConflict
		}
		return attempt, nil
	}
	if attempt.Status != ExecutionClaimed {
		return ExecutionAttempt{}, ErrExecutionInvalidTransition
	}
	now = canonicalExecutionTime(now)
	if now.IsZero() || now.Before(attempt.UpdatedAt) || !now.Before(attempt.LeaseExpiresAt) {
		return ExecutionAttempt{}, ErrExecutionClaimRejected
	}
	evidence := ExecutionEvidence{Kind: EvidenceLocalNoSend, Outcome: ExecutionNotSent, Reference: attempt.AttemptID, Fingerprint: noSendFingerprint(attempt, p.seal.reason), Reason: p.seal.reason, ObservedAt: now}
	return finishExecution(attempt, evidence, now), nil
}

func tikTokNoSendAction(action string) bool {
	return action == TikTokImageAction || action == TikTokGlobalCreateAction || action == TikTokPublishAction
}
func validNoSendReason(reason string) bool {
	switch reason {
	case NoSendPreflightUnavailable, NoSendFactsChanged, NoSendAccessRecheckFailed, NoSendImageContentChanged, NoSendPayloadInvalid:
		return true
	}
	return false
}
func noSendFingerprint(a ExecutionAttempt, reason string) string {
	return digestExecutionParts("task-processor/submission/local-no-send/v1", a.OrganizationID, a.AttemptID, a.IntentKey, a.Target.Platform, a.Target.StoreID, a.Target.SubjectID, a.Action, a.PayloadFingerprint, a.ProviderExecutionKey, strconv.FormatInt(a.FenceEpoch, 10), a.ClaimOwnerID, reason)
}
func validNoSendAttempt(a ExecutionAttempt) bool {
	return a.Target.Platform == "tiktokshop" && tikTokNoSendAction(a.Action) && a.Evidence != nil && a.Evidence.Kind == EvidenceLocalNoSend && a.Evidence.Reference == a.AttemptID && validNoSendReason(a.Evidence.Reason) && a.Evidence.Fingerprint == noSendFingerprint(a, a.Evidence.Reason) && a.FinishedAt != nil && a.FinishedAt.Before(a.LeaseExpiresAt)
}
