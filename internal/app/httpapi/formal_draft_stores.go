package httpapi

import (
	"context"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"time"
)

// This decoration sits at the original Store port, before rule, merchant,
// uploader and worker consumers are constructed. It does not block the Source
// shared by the two purposes, or change any other Store's UNKNOWN owner.
type formalDraftStores struct {
	base                    storecenter.ProductExecutionReader
	organizationID, storeID string
}

func (s formalDraftStores) ReadProductExecution(ctx context.Context, subject storecenter.ProductExecutionSubject, id string, auth storecenter.ProductExecutionAuthorizer, now time.Time) (storecenter.ProductExecutionMaterial, error) {
	if subject.OrganizationID == s.organizationID && id == s.storeID {
		return storecenter.ProductExecutionMaterial{}, storecenter.ErrNotFound
	}
	return s.base.ReadProductExecution(ctx, subject, id, auth, now)
}

func formalDraftRead(_ context.Context, _ collection.Scope, v record.TargetRecord) error {
	if v.LocalTrial != nil || v.Input.LocalTrial != nil {
		return record.ErrForbidden
	}
	return nil
}
