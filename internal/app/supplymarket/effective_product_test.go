package supplymarketapp

import (
	"context"
	"encoding/json"
	"fmt"
	"task-processor/internal/authidentity"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/review"
	reviewproposal "task-processor/internal/product/review/proposal"
	"task-processor/internal/product/supplymarket"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestChoiceReturnsNormalIneligibilityWithoutWeakeningPublication(t *testing.T) {
	for _, optimized := range []bool{false, true} {
		for _, complete := range []bool{false, true} {
			t.Run(fmt.Sprintf("optimized=%t/complete=%t", optimized, complete), func(t *testing.T) {
				scope := collection.Scope{"org-a", "actor-a", "member-a"}
				id := uuid.NewString()
				original := catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: "own"}, PublicationID: "original", Version: 1, Snapshot: catalog.ProductSnapshot{Title: "原商品"}}
				if complete {
					original.Snapshot.Images = []catalog.Image{{URL: "https://images.example.org/source.png"}}
				}
				current := original
				if optimized {
					current.Version = 2
					current.PublicationID = "review:" + id
					current.Snapshot.Title = "优化商品"
				}
				record := review.Record{ID: id, Org: scope.OrganizationID, Owner: scope.ActorID, Input: review.CreateInput{ProductKey: "own", BaseVersion: 1}, BasePublicationID: original.PublicationID, Policy: "title-review-v1", Before: original.Snapshot.Title, Title: current.Snapshot.Title, State: "applied", Revision: 2, Original: reviewproposal.Proposal{Changes: []reviewproposal.FieldChange{{Field: "title"}}}, Receipt: &review.Receipt{ProposalID: id, Revision: 2, ProductVersion: 2, PublicationID: current.PublicationID, Actor: scope.ActorID, At: time.Now()}}
				reader := EffectiveProductReader{Snapshots: snapshotFixture{original, current}, Applied: reviewFixture{record}}
				item := collection.ItemDetail{Item: collection.Item{ID: uuid.NewString(), Revision: 1, Source: collection.Source{Kind: "own", ProductKey: "own", PublicationID: "original", Version: 1}}, Product: original.Snapshot}
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				choice, err := reader.ReadChoice(ctx, scope, item)
				require.NoError(t, err, "a saved own product is a normal candidate even when unqualified")
				require.Equal(t, optimized, choice.Optimized)
				raw, err := json.Marshal(choice)
				require.NoError(t, err)
				var body map[string]any
				require.NoError(t, json.Unmarshal(raw, &body))
				if !optimized {
					require.Equal(t, "not_optimized", body["ineligibleReason"])
				} else if !complete {
					require.Equal(t, "incomplete_product", body["ineligibleReason"])
				} else {
					require.NotContains(t, body, "ineligibleReason")
				}
				if complete {
					require.Contains(t, body, "product")
				} else {
					require.NotContains(t, body, "product")
					_, err = supplymarket.PublicProjection(current.Snapshot)
					require.ErrorIs(t, err, supplymarket.ErrInvalid, "publication requirements must remain strict")
				}
			})
		}
	}
}

type snapshotFixture struct{ original, applied catalog.PublishedSnapshot }

func (r snapshotFixture) GetCurrentSnapshot(_ context.Context, id catalog.SnapshotIdentity) (catalog.PublishedSnapshot, error) {
	if id != r.original.Identity {
		return catalog.PublishedSnapshot{}, catalog.ErrSnapshotNotReady
	}
	return r.applied, nil
}

func (r snapshotFixture) GetSnapshot(_ context.Context, id catalog.SnapshotIdentity, v uint64) (catalog.PublishedSnapshot, error) {
	if id != r.original.Identity {
		return catalog.PublishedSnapshot{}, catalog.ErrSnapshotNotReady
	}
	if v == 1 {
		return r.original, nil
	}
	if v == 2 {
		return r.applied, nil
	}
	return catalog.PublishedSnapshot{}, catalog.ErrSnapshotNotReady
}

type reviewFixture struct{ record review.Record }

func (r reviewFixture) ReadAppliedPublication(_ context.Context, scope review.Scope, _ string, _ uint64, _ string) (review.Record, error) {
	if scope.Org != r.record.Org || scope.Actor != r.record.Owner {
		return review.Record{}, review.ErrNotFound
	}
	return r.record, nil
}
func TestEffectiveProductRequiresExactOwnApplyLineage(t *testing.T) {
	scope := collection.Scope{"org-a", "actor-a", "member-a"}
	id := uuid.NewString()
	identity := catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: "own-product"}
	original := catalog.PublishedSnapshot{Identity: identity, PublicationID: "original", Version: 1, Snapshot: catalog.ProductSnapshot{Title: "原商品", Images: []catalog.Image{{URL: "https://images.example.org/source.png"}}}}
	applied := original
	applied.PublicationID = "review:" + id
	applied.Version = 2
	applied.Snapshot.Title = "优化商品"
	record := review.Record{ID: id, Org: scope.OrganizationID, Owner: scope.ActorID, Input: review.CreateInput{ProductKey: identity.ProductKey, BaseVersion: 1}, BasePublicationID: original.PublicationID, Policy: "title-review-v1", Before: original.Snapshot.Title, Title: applied.Snapshot.Title, State: "applied", Revision: 2, Original: reviewproposal.Proposal{Changes: []reviewproposal.FieldChange{{Field: "title"}}}, Receipt: &review.Receipt{ProposalID: id, Revision: 2, ProductVersion: 2, PublicationID: applied.PublicationID, Actor: scope.ActorID, At: time.Now()}}
	reader := EffectiveProductReader{Snapshots: snapshotFixture{original, applied}, Applied: reviewFixture{record}}
	item := collection.ItemDetail{Item: collection.Item{ID: uuid.NewString(), Source: collection.Source{Kind: "own", ProductKey: identity.ProductKey, PublicationID: "original", Version: 1}}, Product: original.Snapshot}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	got, err := reader.ReadEffective(ctx, scope, item, 2, id)
	require.NoError(t, err)
	require.Equal(t, applied, got)
	choice, err := reader.ReadChoice(ctx, scope, item)
	require.NoError(t, err)
	require.Equal(t, id, choice.Selection.ApplyID)
	require.Equal(t, uint64(2), choice.Selection.EffectiveVersion)
	require.Equal(t, "优化商品", choice.Product.Title)
	for _, bad := range []struct {
		version uint64
		apply   string
	}{{2, ""}, {2, uuid.NewString()}, {1, id}, {3, id}} {
		_, err := reader.ReadEffective(ctx, scope, item, bad.version, bad.apply)
		require.ErrorIs(t, err, supplymarket.ErrConflict)
	}
	record.Owner = "other"
	reader.Applied = reviewFixture{record}
	_, err = reader.ReadEffective(ctx, scope, item, 2, id)
	require.ErrorIs(t, err, supplymarket.ErrConflict)
	record.Owner = scope.ActorID
	reader.Applied = reviewFixture{record}
	applied.Snapshot.Brand = "unapproved change"
	reader.Snapshots = snapshotFixture{original, applied}
	_, err = reader.ReadEffective(ctx, scope, item, 2, id)
	require.ErrorIs(t, err, supplymarket.ErrConflict)
	_, err = reader.ReadChoice(ctx, scope, item)
	require.ErrorIs(t, err, supplymarket.ErrConflict, "a changed head cannot masquerade as an eligible own-product choice")
}
func TestOriginalOwnerContextDropsPlatformIdentityAndPreservesCancellation(t *testing.T) {
	input := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{UserID: "platform-a"})
	input, cancel := context.WithTimeout(input, time.Second)
	defer cancel()
	owner, stop, err := originalOwnerContext(input)
	require.NoError(t, err)
	defer stop()
	_, ok := authidentity.AuthenticatedIdentityFromContext(owner)
	require.False(t, ok, "this is tokenless execution, not a fabricated member")
	cancel()
	select {
	case <-owner.Done():
	case <-time.After(time.Second):
		t.Fatal("parent cancellation must stop the owner check")
	}
	_, _, err = originalOwnerContext(context.Background())
	require.ErrorIs(t, err, supplymarket.ErrForbidden)
}
