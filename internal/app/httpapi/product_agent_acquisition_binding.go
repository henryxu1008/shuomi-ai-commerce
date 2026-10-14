package httpapi

import (
	"context"
	"reflect"
	"strconv"
	"task-processor/internal/agent"
	"task-processor/internal/authidentity"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/review"
)

type productAgentSourceBindingKey struct{}

func acquisitionBinding(operationID, target string, published catalog.PublishedSnapshot) agent.Binding {
	return agent.Binding{ContextKind: "acquisition", ContextID: operationID, ProductKey: published.Identity.ProductKey,
		CatalogVersion: strconv.FormatUint(published.Version, 10), PublicationID: published.PublicationID, TargetPlatform: target}
}

func (a *productAgentApplication) currentAcquisitionBinding(ctx context.Context, identity authidentity.AuthenticatedIdentity, operationID, target string) (agent.Binding, error) {
	if !a.validTargetPlatform(target) || a.snapshots == nil {
		return agent.Binding{}, agent.ErrUnavailable
	}
	original, err := a.acquisitionSnapshot(ctx, identity, operationID)
	if err != nil {
		return agent.Binding{}, err
	}
	current, err := a.snapshots.GetCurrentSnapshot(ctx, original.Identity)
	if err != nil {
		return agent.Binding{}, err
	}
	if err = a.checkAcquisitionLineage(ctx, identity, original, current); err != nil {
		return agent.Binding{}, err
	}
	return acquisitionBinding(operationID, target, current), nil
}

// Both original and legitimately adopted historical versions remain readable.
// Only a new operation selects the current head; history never rewrites its SRC
// receipt or original execution binding.
func (a *productAgentApplication) authorizeAcquisitionBinding(ctx context.Context, identity authidentity.AuthenticatedIdentity, binding agent.Binding) error {
	if !binding.Valid() || binding.ContextKind != "acquisition" || !a.validTargetPlatform(binding.TargetPlatform) {
		return agent.ErrInvalid
	}
	original, err := a.acquisitionSnapshot(ctx, identity, binding.ContextID)
	if err != nil {
		return err
	}
	if binding == acquisitionBinding(binding.ContextID, binding.TargetPlatform, original) {
		return nil
	}
	if a.snapshots == nil {
		return agent.ErrUnavailable
	}
	version, err := strconv.ParseUint(binding.CatalogVersion, 10, 63)
	if err != nil || version <= original.Version || binding.ProductKey != original.Identity.ProductKey {
		return agent.ErrConflict
	}
	published, err := a.snapshots.GetSnapshot(ctx, original.Identity, version)
	if err != nil {
		return err
	}
	if binding != acquisitionBinding(binding.ContextID, binding.TargetPlatform, published) {
		return agent.ErrConflict
	}
	return a.checkAcquisitionLineage(ctx, identity, original, published)
}

func (a *productAgentApplication) checkAcquisitionLineage(ctx context.Context, identity authidentity.AuthenticatedIdentity, original, published catalog.PublishedSnapshot) error {
	if published.Identity != original.Identity || published.Version < original.Version {
		return agent.ErrConflict
	}
	if published.Version == original.Version {
		if !reflect.DeepEqual(published, original) {
			return agent.ErrConflict
		}
		return nil
	}
	lineage, err := review.ResolveAppliedSnapshot(ctx, review.Scope{Org: identity.TenantID, Actor: identity.UserID}, published, a.snapshots, a.reviewLookup)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(lineage.Original, original) {
		return agent.ErrConflict
	}
	return nil
}
