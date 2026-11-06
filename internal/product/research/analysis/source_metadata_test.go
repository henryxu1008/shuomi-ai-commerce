package analysis

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
	"task-processor/internal/product/selection"
	"task-processor/internal/product/sourcing"
	"testing"
)

type pipelineItems struct {
	collection.Repository
	items map[string]collection.Item
}

func (p pipelineItems) ReadItem(_ context.Context, _ collection.Scope, id string) (collection.Item, error) {
	return p.items[id], nil
}

type pipelineAuthority struct{}

func (pipelineAuthority) Authorize(context.Context, string) (collection.Scope, error) {
	return collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}, nil
}

type pipelineSources struct{}

func (pipelineSources) ReadPublished(context.Context, string) (sourcing.PublishedAcquisition, error) {
	return sourcing.PublishedAcquisition{}, collection.ErrNotFound
}

type pipelineSnapshots struct {
	snapshots map[string]catalog.PublishedSnapshot
}

func (p pipelineSnapshots) GetSnapshot(_ context.Context, id catalog.SnapshotIdentity, _ uint64) (catalog.PublishedSnapshot, error) {
	return p.snapshots[id.ProductKey], nil
}
func TestResearchFiltersRealAmazonEnvelopeSystemAttributesAndSiblingFacts(t *testing.T) {
	i, ctx, project, choices, _ := fixture(t)
	evidence := dataacquisition.Evidence{Site: "us", ASIN: "B000123456", MainImage: "https://m.media-amazon.com/images/a.jpg", Title: "Saved item", Attributes: map[string]string{"material": "steel"}, Availability: "unavailable", CapturedAt: "2026-11-02T00:00:00Z", ParserVersion: "amazon-v1"}
	envelope, e := evidence.Envelope(choices[0].Selection.PublicationID)
	require.NoError(t, e)
	snapshot, e := sourcing.ToSnapshot(envelope)
	require.NoError(t, e)
	items := pipelineItems{items: map[string]collection.Item{}}
	snapshots := pipelineSnapshots{snapshots: map[string]catalog.PublishedSnapshot{}}
	inputs := []collection.SelectionInput{}
	for _, c := range choices {
		key := c.Selection.ItemID
		source := collection.Source{Kind: "own", ProductKey: key, PublicationID: c.Selection.PublicationID, Version: 1}
		items.items[key] = collection.Item{ID: key, Revision: 1, Source: source}
		snapshots.snapshots[key] = catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: "org", ProductKey: key}, PublicationID: source.PublicationID, Version: 1, Snapshot: snapshot}
		inputs = append(inputs, collection.SelectionInput{ItemID: key, ExpectedRevision: 1, OriginalPublicationID: source.PublicationID, OriginalVersion: 1})
	}
	collections, e := collection.NewService(items, pipelineAuthority{}, pipelineSources{})
	require.NoError(t, e)
	collections.WithSnapshots(snapshots)
	actual := selection.Inspector{Collections: collections}
	_, original, e := actual.Prepare(ctx, project, inputs)
	require.NoError(t, e)
	wire, e := json.Marshal(original)
	require.NoError(t, e)
	require.Contains(t, string(wire), "www.amazon.com/dp/")
	i.Products = actual
	_, out, e := i.Prepare(ctx, project, choices)
	require.NoError(t, e)
	wire, e = json.Marshal(out)
	require.NoError(t, e)
	require.NotContains(t, string(wire), "https://")
	require.NotContains(t, string(wire), `"Name":"sourceUrl"`)
	require.NotContains(t, string(wire), `"Name":"capturedAt"`)
	require.Contains(t, string(wire), "steel")
	require.Contains(t, string(wire), "source_metadata_omitted")
	// The same exclusion applies to whole variant attribute pairs and their
	// uncertainty sibling; unrelated real product attributes remain unchanged.
	c := out.Candidates[0]
	value := json.RawMessage(`[["sourceUrl","https://www.amazon.com/dp/B000123456"],["color","red"]]`)
	p := selection.Inspection{Products: []selection.Product{{Ref: c.Ref, Facts: []selection.Fact{{Path: "variants.0.attributes", Value: value}, {Path: "variants.0.uncertainty", Value: json.RawMessage("true")}}}}}
	probe := Inspection{Candidates: []Candidate{{Research: c.Research}}}
	attachProducts(&probe, p)
	for _, f := range probe.Candidates[0].Facts {
		require.NotEqual(t, "variants.0.attributes", f.Path)
		require.NotEqual(t, "variants.0.uncertainty", f.Path)
	}
}
