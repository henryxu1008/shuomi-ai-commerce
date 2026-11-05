package dataacquisition

import (
	"context"
	"time"

	"github.com/google/uuid"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/product/collection"
)

// SHEINUSTrialAccessCheck reads current trusted approval, source and account
// authority for this action. It must not infer permission from the frozen plan.
type SHEINUSTrialAccessCheck func(context.Context, Principal, SHEINUSTrialApproval, SHEINUSTrialAction) error

type SHEINUSTrialBudgetEvidence struct {
	SHEINUSTrialBudget
	ValidUntil time.Time
}

type SHEINUSTrialBudgetReader func(context.Context, SHEINUSTrialApproval) (SHEINUSTrialBudgetEvidence, error)

type fixedSHEINUSTrialAuthority struct {
	approval   SHEINUSTrialApproval
	reconciler collection.Scope
	access     SHEINUSTrialAccessCheck
	budget     SHEINUSTrialBudgetReader
}

// NewFixedSHEINUSTrialAuthority admits one immutable, privately supplied
// approval. Product still owns the durable send claim and usage record.
func NewFixedSHEINUSTrialAuthority(approval SHEINUSTrialApproval, reconciler collection.Scope, access SHEINUSTrialAccessCheck, budget SHEINUSTrialBudgetReader) (SHEINUSTrialAuthority, error) {
	id, idErr := uuid.Parse(approval.ID)
	project, projectErr := uuid.Parse(approval.ProjectID)
	if idErr != nil || id == uuid.Nil || id.String() != approval.ID ||
		projectErr != nil || project == uuid.Nil || project.String() != approval.ProjectID ||
		approval.Scope.Validate() != nil || reconciler.Validate() != nil || reconciler.OrganizationID != approval.Scope.OrganizationID ||
		!collection.ValidID(approval.AccountRef) || !collection.ValidID(approval.KeyRef) || approval.ExpiresAt.IsZero() ||
		(approval.Funding != orgresource.FundingEnterprise && approval.Funding != orgresource.FundingMember) || access == nil || budget == nil {
		return nil, ErrUnavailable
	}
	return &fixedSHEINUSTrialAuthority{approval, reconciler, access, budget}, nil
}

func (a *fixedSHEINUSTrialAuthority) AuthorizeSHEINUSTrial(ctx context.Context, p Principal, id string, action SHEINUSTrialAction) (SHEINUSTrialApproval, error) {
	approved := a.approval
	if ctx == nil || ctx.Err() != nil || id != approved.ID || p.CredentialID != "" || p.CredentialRevision != 0 {
		return SHEINUSTrialApproval{}, ErrForbidden
	}
	switch action {
	case SHEINUSTrialView, SHEINUSTrialSend, SHEINUSTrialPublish:
		if p.Scope != approved.Scope {
			return SHEINUSTrialApproval{}, ErrForbidden
		}
	case SHEINUSTrialReconcile:
		if p.Scope != a.reconciler {
			return SHEINUSTrialApproval{}, ErrForbidden
		}
	default:
		return SHEINUSTrialApproval{}, ErrForbidden
	}
	if (action == SHEINUSTrialSend && !time.Now().Before(approved.ExpiresAt)) ||
		a.access(ctx, p, approved, action) != nil || ctx.Err() != nil {
		return SHEINUSTrialApproval{}, ErrForbidden
	}
	return approved, nil
}

func (a *fixedSHEINUSTrialAuthority) CheckSHEINUSTrialBudget(ctx context.Context, approved SHEINUSTrialApproval) (SHEINUSTrialBudget, error) {
	if ctx == nil || ctx.Err() != nil || approved.Binding() != a.approval.Binding() || !time.Now().Before(a.approval.ExpiresAt) {
		return SHEINUSTrialBudget{}, ErrUnavailable
	}
	deadline, bounded := ctx.Deadline()
	if !bounded || deadline.After(a.approval.ExpiresAt) {
		return SHEINUSTrialBudget{}, ErrUnavailable
	}
	evidence, err := a.budget(ctx, a.approval)
	if err != nil || ctx.Err() != nil || evidence.Binding != a.approval.Binding() || evidence.ValidUntil.Before(deadline) {
		return SHEINUSTrialBudget{}, ErrUnavailable
	}
	return evidence.SHEINUSTrialBudget, nil
}
