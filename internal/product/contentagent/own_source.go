package contentagent

import (
	"context"
	"reflect"
	"strconv"

	"task-processor/internal/agent"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/review"
)

const OwnProductContext = "own_product"

// OwnSource contains owner-read facts, never a browser-supplied product snapshot.
// The caller must read ItemDetail through the actor-private Collection service.
type OwnSource struct {
	Binding             agent.Binding
	Original, Effective catalog.PublishedSnapshot
}

// ResolveOwnSource selects the current legitimate head, or verifies an existing
// run's exact immutable binding. A later Apply never silently rebinds that run.
func ResolveOwnSource(ctx context.Context, scope collection.Scope, item collection.ItemDetail, snapshots catalog.CompleteSnapshotReader, applied review.AppliedPublicationLookup, version, publication string) (OwnSource, error) {
	if ctx == nil || ctx.Err() != nil || scope.Validate() != nil || snapshots == nil || item.Item.Source.Kind != "own" || item.Item.ArchivedAt != nil || !collection.ValidID(item.Item.ID) || item.Item.Revision < 1 || item.Item.Source.Version == 0 || (version == "") != (publication == "") {
		return OwnSource{}, agent.ErrConflict
	}
	identity := catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: item.Item.Source.ProductKey}
	original, err := snapshots.GetSnapshot(ctx, identity, item.Item.Source.Version)
	if err != nil || original.Identity != identity || original.Version != item.Item.Source.Version || original.PublicationID != item.Item.Source.PublicationID || !reflect.DeepEqual(original.Snapshot, item.Product) {
		return OwnSource{}, agent.ErrConflict
	}
	var effective catalog.PublishedSnapshot
	if version == "" {
		effective, err = snapshots.GetCurrentSnapshot(ctx, identity)
	} else {
		v, parseErr := strconv.ParseUint(version, 10, 63)
		if parseErr != nil || v == 0 || strconv.FormatUint(v, 10) != version {
			return OwnSource{}, agent.ErrConflict
		}
		effective, err = snapshots.GetSnapshot(ctx, identity, v)
		if effective.Version != v || effective.PublicationID != publication {
			return OwnSource{}, agent.ErrConflict
		}
	}
	if err != nil || effective.Identity != identity || effective.Version < original.Version {
		return OwnSource{}, agent.ErrConflict
	}
	if effective.Version == original.Version {
		if !reflect.DeepEqual(effective, original) {
			return OwnSource{}, agent.ErrConflict
		}
	} else {
		lineage, lineageErr := review.ResolveAppliedSnapshot(ctx, review.Scope{Org: scope.OrganizationID, Actor: scope.ActorID}, effective, snapshots, applied)
		if lineageErr != nil || len(lineage.Applied) == 0 || !reflect.DeepEqual(lineage.Original, original) {
			return OwnSource{}, agent.ErrConflict
		}
	}
	if err := ctx.Err(); err != nil {
		return OwnSource{}, err
	}
	binding := agent.Binding{ContextKind: OwnProductContext, ContextID: item.Item.ID, ProductKey: identity.ProductKey, CatalogVersion: strconv.FormatUint(effective.Version, 10), PublicationID: effective.PublicationID, TargetPlatform: "product"}
	if !binding.Valid() {
		return OwnSource{}, agent.ErrConflict
	}
	return OwnSource{Binding: binding, Original: original, Effective: effective}, nil
}
