package dataservicesapp

import (
	"context"
	"errors"
	"time"

	"task-processor/internal/product/collection"
	acq "task-processor/internal/product/dataacquisition"
)

type AmazonFirstPageTrial struct {
	module           *Module
	approval         acq.AmazonFirstPageApproval
	authorizeProject func(context.Context, collection.Scope, string, bool) error
	send             acq.AmazonFirstPageSender
}

func (m *Module) AmazonFirstPageTrial(approval acq.AmazonFirstPageApproval, authorizeProject func(context.Context, collection.Scope, string, bool) error, send acq.AmazonFirstPageSender) (*AmazonFirstPageTrial, error) {
	if m == nil || m.acquisition == nil || m.repo == nil || m.prices == nil || m.funding == nil ||
		approval.Validate() != nil || authorizeProject == nil || send == nil {
		return nil, acq.ErrUnavailable
	}
	return &AmazonFirstPageTrial{module: m, approval: approval, authorizeProject: authorizeProject, send: send}, nil
}

func (t *AmazonFirstPageTrial) authorize(ctx context.Context, p acq.Principal, project, id string, write bool) error {
	a := t.approval
	if p.Scope != a.Scope || p.CredentialID != "" || p.CredentialRevision != 0 || project != a.ProjectID || id != a.ID {
		return acq.ErrForbidden
	}
	if err := t.authorizeProject(ctx, p.Scope, project, write); err != nil {
		return acq.ErrForbidden
	}
	return t.module.live.CheckRead(ctx, p)
}

func (t *AmazonFirstPageTrial) checkBeforeSend(ctx context.Context, p acq.Principal) error {
	a := t.approval
	if !time.Now().Before(a.SendBefore()) {
		return acq.ErrForbidden
	}
	return t.checkPreparation(ctx, p)
}

func (t *AmazonFirstPageTrial) checkPreparation(ctx context.Context, p acq.Principal) error {
	a := t.approval
	if err := t.authorize(ctx, p, a.ProjectID, a.ID, true); err != nil {
		return err
	}
	funding, err := t.module.funding.Funding(ctx, p.Scope)
	if err != nil || funding != a.Funding {
		return acq.ErrForbidden
	}
	if err := t.module.live.CheckExecution(ctx, p, funding); err != nil {
		return err
	}
	price, err := t.module.prices.CurrentDataRowPrice(ctx)
	if err != nil {
		return acq.ErrUnavailable
	}
	maximum, err := price.Cost(acq.AmazonFirstPageLimit)
	if err != nil || price.Version != a.Price.Version || maximum != a.Price.MaximumCostFen {
		return acq.ErrConflict
	}
	return nil
}

func (t *AmazonFirstPageTrial) Read(ctx context.Context, p acq.Principal, project, id string) (acq.AmazonFirstPageReceipt, error) {
	if err := t.authorize(ctx, p, project, id, false); err != nil {
		return acq.AmazonFirstPageReceipt{}, err
	}
	out, err := t.module.repo.ReadAmazonFirstPage(ctx, t.approval)
	if err == nil {
		err = t.authorize(ctx, p, project, id, false)
	}
	if err != nil {
		return acq.AmazonFirstPageReceipt{}, err
	}
	return out, nil
}

func (t *AmazonFirstPageTrial) Submit(ctx context.Context, p acq.Principal, project, id string) (acq.AmazonFirstPageReceipt, error) {
	a := t.approval
	if err := t.authorize(ctx, p, project, id, true); err != nil {
		return acq.AmazonFirstPageReceipt{}, err
	}
	out, err := t.module.repo.ReadAmazonFirstPage(ctx, a)
	if !errors.Is(err, acq.ErrNotFound) {
		if err != nil {
			return acq.AmazonFirstPageReceipt{}, err
		}
		return t.resume(ctx, p, out)
	}
	if err = t.checkBeforeSend(ctx, p); err != nil {
		return acq.AmazonFirstPageReceipt{}, err
	}
	won, err := t.module.repo.ClaimAmazonFirstPage(ctx, a)
	if err != nil {
		return acq.AmazonFirstPageReceipt{}, err
	}
	if !won {
		out, err = t.Read(ctx, p, project, id)
		if err != nil {
			return acq.AmazonFirstPageReceipt{}, err
		}
		return t.resume(ctx, p, out)
	}
	if err = t.checkBeforeSend(ctx, p); err != nil {
		return acq.AmazonFirstPageReceipt{}, err
	}

	submission := t.send(ctx, a)
	out = submission.Receipt
	if out.Validate() != nil || out.ApprovalID != a.ID || out.State == acq.AmazonFirstPagePrepared {
		return acq.AmazonFirstPageReceipt{}, acq.ErrUnknown
	}
	if len(submission.Evidence) >= 2 && out.State == acq.AmazonFirstPageUnknown && out.RequestID != "" {
		err = t.checkPreparation(ctx, p)
		if err == nil {
			job, err := t.module.repo.PrepareAmazonFirstPage(ctx, a, submission)
			if err != nil {
				return acq.AmazonFirstPageReceipt{}, acq.ErrUnknown
			}
			out.State, out.JobID = acq.AmazonFirstPagePrepared, job.ID
			return t.resume(ctx, p, out)
		}
		out.State = acq.AmazonFirstPageRejected
	}
	// Accounting of the original dispatch may complete after request cancel.
	// This branch never prepares or publishes product data.
	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if recordErr := t.module.repo.RecordAmazonFirstPage(recordCtx, a, out); recordErr != nil {
		return acq.AmazonFirstPageReceipt{}, acq.ErrUnknown
	}
	if err != nil {
		return acq.AmazonFirstPageReceipt{}, err
	}
	return t.Read(ctx, p, project, id)
}

func (t *AmazonFirstPageTrial) resume(ctx context.Context, p acq.Principal, out acq.AmazonFirstPageReceipt) (acq.AmazonFirstPageReceipt, error) {
	if err := t.authorize(ctx, p, t.approval.ProjectID, t.approval.ID, true); err != nil {
		return acq.AmazonFirstPageReceipt{}, err
	}
	if out.State == acq.AmazonFirstPagePrepared {
		if _, err := t.module.acquisition.Read(ctx, p, out.JobID); err != nil {
			if errors.Is(err, acq.ErrUnavailable) {
				return acq.AmazonFirstPageReceipt{}, acq.ErrUnknown
			}
			return acq.AmazonFirstPageReceipt{}, err
		}
	}
	return out, nil
}
