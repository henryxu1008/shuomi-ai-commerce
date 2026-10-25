package contentagent

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/review"
	reviewproposal "task-processor/internal/product/review/proposal"
)

type ownSnapshots struct {
	head     uint64
	versions map[uint64]catalog.PublishedSnapshot
}

func (s ownSnapshots) GetSnapshot(_ context.Context, id catalog.SnapshotIdentity, v uint64) (catalog.PublishedSnapshot, error) {
	p, ok := s.versions[v]
	if !ok || p.Identity != id {
		return catalog.PublishedSnapshot{}, catalog.ErrSnapshotNotReady
	}
	return p, nil
}
func (s ownSnapshots) GetCurrentSnapshot(ctx context.Context, id catalog.SnapshotIdentity) (catalog.PublishedSnapshot, error) {
	return s.GetSnapshot(ctx, id, s.head)
}

type ownApplies map[string]review.Record

func (s ownApplies) ReadAppliedPublication(_ context.Context, scope review.Scope, key string, v uint64, pub string) (review.Record, error) {
	r, ok := s[pub]
	if !ok || r.Org != scope.Org || r.Owner != scope.Actor || r.Input.ProductKey != key || r.Receipt.ProductVersion != v {
		return review.Record{}, review.ErrNotFound
	}
	return r, nil
}

func TestOwnSourceRequiresExactOriginalAndGenuineApplyLineage(t *testing.T) {
	scope := collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	root := catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: "own-product"}, PublicationID: uuid.NewString(), Version: 1, Snapshot: catalog.ProductSnapshot{Title: "原商品"}}
	item := collection.ItemDetail{Item: collection.Item{ID: uuid.NewString(), Revision: 1, Source: collection.Source{Kind: "own", ProductKey: root.Identity.ProductKey, PublicationID: root.PublicationID, Version: 1}}, Product: root.Snapshot}
	snapshots := ownSnapshots{1, map[uint64]catalog.PublishedSnapshot{1: root}}
	applies := ownApplies{}
	got, err := ResolveOwnSource(context.Background(), scope, item, snapshots, applies, "", "")
	require.NoError(t, err, "text optimization must not require images or a shop")
	require.Equal(t, agent.Binding{ContextKind: "own_product", ContextID: item.Item.ID, ProductKey: root.Identity.ProductKey, CatalogVersion: "1", PublicationID: root.PublicationID, TargetPlatform: "product"}, got.Binding)
	for v := uint64(2); v <= 3; v++ {
		base := snapshots.versions[v-1]
		next := base
		next.Version = v
		id := uuid.NewString()
		next.PublicationID = "review:" + id
		next.Snapshot.Title = base.Snapshot.Title + "优化"
		applies[next.PublicationID] = review.Record{ID: id, Org: scope.OrganizationID, Owner: scope.ActorID, Input: review.CreateInput{ProductKey: root.Identity.ProductKey, BaseVersion: base.Version}, BasePublicationID: base.PublicationID, Policy: "title-review-v1", Before: base.Snapshot.Title, Title: next.Snapshot.Title, State: "applied", Revision: 2, Original: reviewproposal.Proposal{Changes: []reviewproposal.FieldChange{{Field: "title"}}}, Receipt: &review.Receipt{ProposalID: id, Revision: 2, ProductVersion: v, PublicationID: next.PublicationID, Actor: scope.ActorID, At: time.Now()}}
		snapshots.versions[v] = next
		snapshots.head = v
	}
	got, err = ResolveOwnSource(context.Background(), scope, item, snapshots, applies, "", "")
	require.NoError(t, err)
	require.Equal(t, "3", got.Binding.CatalogVersion)
	require.Equal(t, root, got.Original)
	old, err := ResolveOwnSource(context.Background(), scope, item, snapshots, applies, "2", snapshots.versions[2].PublicationID)
	require.NoError(t, err)
	require.Equal(t, "2", old.Binding.CatalogVersion, "an existing run keeps its frozen source after a later Apply")
	for _, mutate := range []func(*collection.ItemDetail){
		func(i *collection.ItemDetail) { i.Item.Source.Kind = "acquisition" },
		func(i *collection.ItemDetail) { now := time.Now(); i.Item.ArchivedAt = &now },
		func(i *collection.ItemDetail) { i.Item.Source.PublicationID = uuid.NewString() },
		func(i *collection.ItemDetail) { i.Product.Brand = "unpublished" },
		func(i *collection.ItemDetail) { i.Item.ID = "not-an-item" },
	} {
		bad := item
		mutate(&bad)
		_, err = ResolveOwnSource(context.Background(), scope, bad, snapshots, applies, "", "")
		require.ErrorIs(t, err, agent.ErrConflict)
	}
	_, err = ResolveOwnSource(context.Background(), collection.Scope{OrganizationID: "other", ActorID: "actor", MemberID: "member"}, item, snapshots, applies, "", "")
	require.ErrorIs(t, err, agent.ErrConflict)
	_, err = ResolveOwnSource(context.Background(), scope, item, snapshots, applies, "2", uuid.NewString())
	require.ErrorIs(t, err, agent.ErrConflict)
	forged := snapshots.versions[3]
	forged.Snapshot.Brand = "not applied"
	snapshots.versions[3] = forged
	_, err = ResolveOwnSource(context.Background(), scope, item, snapshots, applies, "", "")
	require.ErrorIs(t, err, agent.ErrConflict)
	delete(applies, forged.PublicationID)
	_, err = ResolveOwnSource(context.Background(), scope, item, snapshots, applies, "", "")
	require.ErrorIs(t, err, agent.ErrConflict)
}
