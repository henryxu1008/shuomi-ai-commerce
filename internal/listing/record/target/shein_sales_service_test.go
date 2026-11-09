package target

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/marketplace/shein/goods"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"testing"
)

type salesProducts struct{ product catalog.PublishedSnapshot }

func (p *salesProducts) ReadEffectiveTargetProduct(context.Context, preparation.AuthorizedSource, uint64, string) (catalog.PublishedSnapshot, error) {
	return p.product, nil
}

type salesAssets struct{ inventory asset.ApprovedAssetInventory }

func (a *salesAssets) GetApprovedInventory(_ context.Context, scope asset.InventoryScope) (asset.ApprovedAssetInventory, error) {
	v := a.inventory
	v.Scope = scope
	return v, nil
}
func salesServiceFixture(t *testing.T) (context.Context, *TargetService, *importRecords, *importVersions, *importRules, *salesAssets, SheinSalesRequest) {
	ctx, s, repo, c, r, old := importServiceFixture(t)
	b, source, candidate, sel, rules, inv := salesFixture()
	base := repo.head
	base.Input.Draft = b
	base.ProductHash = collection.Digest(source)
	repo.head = base
	repo.byID[base.ID] = base
	c.saved.Content = candidate
	r.rules = rules
	s.dependencies.Products = &salesProducts{catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: "org-a", ProductKey: base.Source.Source.ProductKey}, Version: base.EffectiveVersion, PublicationID: base.Source.Source.PublicationID, Snapshot: source}}
	assets := &salesAssets{inv}
	s.dependencies.Assets = assets
	return ctx, s, repo, c, r, assets, SheinSalesRequest{SourceID: old.SourceID, BaseRecordID: base.ID, CompletionDraft: old.CompletionDraft, Selection: sel}
}
func TestSalesPreviewSaveAndExactOriginalKeyReplay(t *testing.T) {
	ctx, s, repo, c, r, _, in := salesServiceFixture(t)
	before := collection.Digest(c.saved)
	preview, e := s.PreviewSheinSalesMapping(ctx, in)
	require.NoError(t, e)
	require.Empty(t, preview.Issues)
	require.NotNil(t, preview.Input)
	require.NotNil(t, preview.Attributes)
	require.Equal(t, r.rules.Attributes, *preview.Attributes, "confirmation uses the actual current template, not previously loaded labels")
	require.Equal(t, 1, repo.writes)
	key := uuid.NewString()
	saved, e := s.Create(ctx, key, *preview.Input)
	require.NoError(t, e)
	require.Equal(t, int64(2), saved.Record.Revision)
	require.Equal(t, repo.head.TargetID, saved.Record.TargetID)
	require.Len(t, saved.Record.Input.Draft.Product.SKCs, 2)
	require.Equal(t, before, collection.Digest(c.saved))
	require.False(t, saved.Record.Result.ReadyForUpload)
	require.Empty(t, saved.Record.Result.SubmissionPayload)
	r.merchant.ConnectionRevision++
	c.saved.Revision++
	replay, e := s.Create(ctx, key, *preview.Input)
	require.NoError(t, e)
	require.True(t, replay.Replayed)
	require.Equal(t, saved.Record, replay.Record)
	require.Equal(t, 2, repo.writes)
	_, e = s.Create(ctx, uuid.NewString(), *preview.Input)
	require.Error(t, e)
}
func TestSalesSaveRejectsCompletePayloadAndScopeDrift(t *testing.T) {
	for _, name := range []string{"draft", "reference", "rules", "merchant", "inventory", "foreign source", "head", "commit", "mixed branch"} {
		t.Run(name, func(t *testing.T) {
			ctx, s, repo, c, r, a, in := salesServiceFixture(t)
			p, e := s.PreviewSheinSalesMapping(ctx, in)
			require.NoError(t, e)
			require.NotNil(t, p.Input)
			raw, _ := json.Marshal(p.Input)
			var input TargetInput
			require.NoError(t, json.Unmarshal(raw, &input))
			switch name {
			case "draft":
				input.Draft.Product.Names[0].Name = "INJECTED"
			case "reference":
				input.SheinSalesMapping.CompletionDraft.Revision++
			case "rules":
				r.rules.Attributes.Attributes[1].Options[0].Name = "DRIFT"
			case "merchant":
				r.merchant.ConnectionRevision++
			case "inventory":
				a.inventory.Assets[0].URL = "changed"
			case "foreign source":
				c.saved.Source.CollectionItemID = uuid.NewString()
			case "head":
				repo.head.Revision++
			case "commit":
				repo.beforeCommit = func() { c.saved.Revision++ }
			case "mixed branch":
				input.SheinImport = &SheinImport{}
			}
			_, e = s.Create(ctx, uuid.NewString(), input)
			require.Error(t, e)
			require.Equal(t, 1, repo.writes)
		})
	}
}
func TestSalesPreviewReturnsScopedFieldIssueWithoutWriting(t *testing.T) {
	ctx, s, repo, _, _, _, in := salesServiceFixture(t)
	in.Selection.Values = in.Selection.Values[:1]
	p, e := s.PreviewSheinSalesMapping(ctx, in)
	require.NoError(t, e)
	require.Nil(t, p.Input)
	require.NotEmpty(t, p.Issues)
	require.NotEmpty(t, p.Issues[0].Field)
	require.Equal(t, 1, repo.writes)
	_, e = s.PreviewSheinSalesMapping(context.Background(), in)
	require.Error(t, e)
}

func TestSalesOriginalReceiptPreservesHistoricalDiagnostic(t *testing.T) {
	ctx, s, repo, _, _, _, in := salesServiceFixture(t)
	preview, e := s.PreviewSheinSalesMapping(ctx, in)
	require.NoError(t, e)
	key := uuid.NewString()
	saved, e := s.Create(ctx, key, *preview.Input)
	require.NoError(t, e)
	saved.Record.Result.Issues = []goods.OfficialIssue{{Field: "skc_list.1.sku_list", Code: "invalid", Message: "historical diagnostic text"}}
	repo.values[key] = saved
	replay, e := s.Create(ctx, key, *preview.Input)
	require.NoError(t, e)
	require.True(t, replay.Replayed)
	require.Equal(t, saved.Record.Result, replay.Record.Result, "read/replay returns immutable stored diagnostics; never recompute with current builder")
}
