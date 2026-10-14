package selection

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/authidentity"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
)

type inspectionStore struct {
	collection.Repository
	items map[string]collection.Item
	deny  string
}

func (s *inspectionStore) ReadItem(_ context.Context, scope collection.Scope, id string) (collection.Item, error) {
	if id == s.deny || scope != (collection.Scope{"org", "actor", "member"}) {
		return collection.Item{}, collection.ErrNotFound
	}
	item, ok := s.items[id]
	if !ok {
		return item, collection.ErrNotFound
	}
	return item, nil
}

type inspectionAuth struct{}

func (inspectionAuth) Authorize(context.Context, string) (collection.Scope, error) {
	return collection.Scope{"org", "actor", "member"}, nil
}

type inspectionSnapshots struct {
	products map[string]catalog.ProductSnapshot
}

func (s inspectionSnapshots) GetSnapshot(_ context.Context, id catalog.SnapshotIdentity, v uint64) (catalog.PublishedSnapshot, error) {
	return catalog.PublishedSnapshot{Identity: id, Version: v, PublicationID: "publication-" + id.ProductKey, Snapshot: s.products[id.ProductKey]}, nil
}

type inspectionSources struct{}

func (inspectionSources) ReadPublished(context.Context, string) (sourcing.PublishedAcquisition, error) {
	return sourcing.PublishedAcquisition{}, collection.ErrNotFound
}

func inspectionFixture(t *testing.T) (Inspector, context.Context, []collection.SelectionInput, *inspectionStore) {
	t.Helper()
	store := &inspectionStore{items: map[string]collection.Item{}}
	products := map[string]catalog.ProductSnapshot{}
	var choices []collection.SelectionInput
	for i := 0; i < 2; i++ {
		key := "product-" + strconv.Itoa(i)
		id := uuid.NewString()
		item := collection.Item{ID: id, Revision: 1, Source: collection.Source{ProductKey: key, Version: 1, PublicationID: "publication-" + key, Kind: "own"}}
		store.items[id] = item
		products[key] = catalog.ProductSnapshot{Title: key, Variants: []catalog.Variant{{SKU: "sku", Price: &catalog.Price{Amount: 12, Currency: "CNY"}, Stock: 0}}}
		choices = append(choices, collection.SelectionInput{ItemID: id, ExpectedRevision: 1, OriginalPublicationID: item.Source.PublicationID, OriginalVersion: 1})
	}
	service, err := collection.NewService(store, inspectionAuth{}, inspectionSources{})
	require.NoError(t, err)
	service.WithSnapshots(inspectionSnapshots{products})
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", UserID: "actor", EffectiveMemberID: "member", TokenExpiresAt: time.Now().Add(time.Minute)})
	return Inspector{Collections: service}, ctx, choices, store
}

func TestInspectionBindsAllPrivateCandidatesAndMakesUnknownFactsExplicit(t *testing.T) {
	inspector, ctx, choices, store := inspectionFixture(t)
	binding, out, err := inspector.Prepare(ctx, uuid.NewString(), choices)
	require.NoError(t, err)
	require.True(t, binding.Valid())
	require.Len(t, out.Products, 2)
	for _, p := range out.Products {
		require.Contains(t, p.Missing, "sales")
		require.Contains(t, p.Missing, "landed_cost")
		require.Contains(t, p.Missing, "variants.0.stock")
		for _, f := range p.Facts {
			require.NotEqual(t, "variants.0.stock", f.Path)
			require.NotEmpty(t, f.ID)
		}
	}
	raw, err := json.Marshal(out)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "TenantID")
	require.NotContains(t, string(raw), "member")
	_, err = inspector.Inspect(ctx, binding)
	require.NoError(t, err)
	store.deny = choices[1].ItemID
	_, err = inspector.Inspect(ctx, binding)
	require.ErrorIs(t, err, collection.ErrNotFound)
	_, partial, err := inspector.Prepare(ctx, uuid.NewString(), choices)
	require.Error(t, err)
	require.Empty(t, partial.Products, "no partial disclosure when one candidate is unauthorized")
}

func TestInspectionRejectsStaleSelectionAndProposalCannotBorrowEvidence(t *testing.T) {
	inspector, ctx, choices, store := inspectionFixture(t)
	binding, out, err := inspector.Prepare(ctx, uuid.NewString(), choices)
	require.NoError(t, err)
	proposal := Proposal{Overview: "依据已保存资料，进一步核对成本", RecommendedItemID: out.Products[0].Ref.ItemID, Assessments: []Assessment{{ItemID: out.Products[0].Ref.ItemID, Advice: "核实当前供货", EvidenceIDs: []string{out.Products[0].Facts[0].ID}}}}
	require.NoError(t, ValidateProposal(out, proposal))
	proposal.Assessments[0].EvidenceIDs[0] = out.Products[1].Facts[0].ID
	require.Error(t, ValidateProposal(out, proposal))
	proposal.Assessments[0].EvidenceIDs[0] = "invented-evidence"
	require.Error(t, ValidateProposal(out, proposal))
	item := store.items[choices[1].ItemID]
	item.Revision++
	store.items[item.ID] = item
	_, err = inspector.Inspect(ctx, binding)
	require.ErrorIs(t, err, collection.ErrConflict)
	scope, err := inspector.AuthorizeHistory(ctx, binding)
	require.NoError(t, err, "moving a still-owned original product must not erase saved comparison history")
	require.Equal(t, collection.Scope{"org", "actor", "member"}, scope)
	store.deny = choices[0].ItemID
	_, err = inspector.AuthorizeHistory(ctx, binding)
	require.Error(t, err)
}

func TestInferredAttributeIsDisclosedAsUncertain(t *testing.T) {
	product := project(agent.SelectionProductRef{}, collection.ItemDetail{Product: catalog.ProductSnapshot{Attributes: []catalog.Attribute{{Name: "材质", Value: "棉", Trace: catalog.Trace{IsInferred: true}}}}})
	var found bool
	for _, fact := range product.Facts {
		if fact.Path == "attributes.0.uncertainty" {
			require.JSONEq(t, "true", string(fact.Value))
			found = true
		}
	}
	require.True(t, found, "an inferred source attribute cannot be presented as a confirmed fact")
}
