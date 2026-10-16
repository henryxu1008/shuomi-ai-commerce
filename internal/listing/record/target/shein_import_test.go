package target

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/authidentity"
	"task-processor/internal/listing/completiondraft"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/marketplace/shein/goods"
	model "task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type importVersions struct{ saved completiondraft.Record }

func (v *importVersions) ReadVersion(_ context.Context, _ collection.Scope, id string) (completiondraft.Record, error) {
	if id != v.saved.ID {
		return completiondraft.Record{}, ErrNotFound
	}
	return v.saved, nil
}

type importRecords struct {
	targetRecords
	head         TargetRecord
	byID         map[string]TargetRecord
	beforeCommit func()
}

func (r *importRecords) ReadTargetRecord(_ context.Context, _ collection.Scope, id string) (TargetRecord, error) {
	v, ok := r.byID[id]
	if !ok {
		return v, ErrNotFound
	}
	return v, nil
}
func (r *importRecords) ReadTargetHead(context.Context, collection.Scope, string) (TargetRecord, error) {
	return r.head, nil
}
func (r *importRecords) SaveTarget(ctx context.Context, p TargetPrepared) (TargetReceipt, error) {
	_, _, _, value, err := p.Read(ctx)
	if err != nil {
		return TargetReceipt{}, err
	}
	if value.Input.ExpectedRevision != r.head.Revision {
		return TargetReceipt{}, ErrConflict
	}
	if r.beforeCommit != nil {
		r.beforeCommit()
	}
	saved, err := r.targetRecords.SaveTarget(ctx, p)
	if err == nil {
		r.head = saved.Record
		r.byID[saved.Record.ID] = saved.Record
	}
	return saved, err
}

type importRules struct {
	merchant storecenter.ProductMerchantBinding
	rules    goods.OfficialRuleSnapshot
}

func (r *importRules) ReadTargetRules(context.Context, collection.Scope, string, goods.OfficialDraftInput) (storecenter.ProductMerchantBinding, goods.OfficialRuleSnapshot, error) {
	return r.merchant, r.rules, nil
}
func importServiceFixture(t *testing.T) (context.Context, *TargetService, *importRecords, *importVersions, *importRules, SheinImportRequest) {
	t.Helper()
	scope := collection.Scope{"org-a", "actor-a", "member-a"}
	authority := targetAuth{scope}
	collections, err := collection.NewService(targetCollectionRepo{}, authority, targetUnusedSources{})
	require.NoError(t, err)
	source := preparation.SourceItem{ID: uuid.NewString(), PreparationID: uuid.NewString(), CollectionItemID: uuid.NewString(), CollectionRevision: 1, Source: collection.Source{ProductKey: "fixed", PublicationID: uuid.NewString(), Version: 1, Kind: "own"}}
	prep := targetPrepRepo{source: source}
	preparations, err := preparation.NewService(prep, collections, authority)
	require.NoError(t, err)
	snapshot := catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: "fixed"}, PublicationID: source.Source.PublicationID, Version: 1, Snapshot: catalog.ProductSnapshot{Title: "Original"}}
	selector, err := preparation.NewSourceSelector(preparations, collections, prep, targetCatalog{snapshot})
	require.NoError(t, err)
	base, candidate, selection, rules := importFixture()
	records := &importRecords{targetRecords: targetRecords{values: map[string]TargetReceipt{}, hashes: map[string]string{}}, byID: map[string]TargetRecord{}}
	merchant, _, err := (targetRules{}).ReadTargetRules(context.Background(), scope, "", base)
	require.NoError(t, err)
	merchant.StoreID = uuid.NewString()
	current := &importRules{merchant: merchant, rules: rules}
	versions := &importVersions{saved: completiondraft.Record{ID: uuid.NewString(), Source: source, Revision: 3, Content: candidate}}
	service, err := NewTargetService(TargetDependencies{Sources: selector, Products: OriginalTargetProduct{}, Assets: targetAssets{}, Rules: current, Records: records, Images: targetTestImages{}, Authorizer: authority, CompletionVersions: versions})
	require.NoError(t, err)
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute)})
	saved, err := service.Create(ctx, uuid.NewString(), TargetInput{SourceID: source.ID, StoreID: merchant.StoreID, EffectiveVersion: 1, Draft: base})
	require.NoError(t, err)
	return ctx, service, records, versions, current, SheinImportRequest{SourceID: source.ID, BaseRecordID: saved.Record.ID, CompletionDraft: CompletionDraftSelection{RecordID: versions.saved.ID, Revision: 3}, Selection: selection}
}
func TestSheinImportPreviewAndSaveBindExactCandidateAndBaseline(t *testing.T) {
	ctx, service, repo, versions, rules, request := importServiceFixture(t)
	preview, err := service.PreviewSheinImport(ctx, request)
	require.NoError(t, err)
	require.Equal(t, 1, repo.writes, "preview has no write")
	// The Console normalizes nullable array responses to empty arrays. This
	// representation must not be treated as an unselected business edit.
	preview.Input.Draft.Images = []goods.OfficialImageSlot{}
	for _, name := range []string{"draft", "reference", "effective version", "rule drift", "merchant drift"} {
		t.Run(name, func(t *testing.T) {
			raw, _ := json.Marshal(preview.Input)
			var changed TargetInput
			require.NoError(t, json.Unmarshal(raw, &changed))
			original := *rules
			switch name {
			case "draft":
				changed.Draft.Product.SKCs[0].SKUs[1].SupplierSKU = "injected"
			case "reference":
				changed.SheinImport.CompletionDraft.Revision++
			case "effective version":
				changed.ExpectedRevision = 0
			case "rule drift":
				rules.rules.Fill.Currency = importPtr("CNY")
			case "merchant drift":
				rules.merchant.ConnectionRevision++
			}
			_, err := service.Create(ctx, uuid.NewString(), changed)
			require.Error(t, err)
			*rules = original
			require.Equal(t, 1, repo.writes)
		})
	}
	key := uuid.NewString()
	saved, err := service.Create(ctx, key, preview.Input)
	require.NoError(t, err)
	require.Equal(t, int64(2), saved.Record.Revision)
	require.Equal(t, versions.saved.ID, saved.Record.Input.SheinImport.CompletionDraft.RecordID)
	rules.merchant.ConnectionRevision++
	versions.saved.Revision = 99
	replay, err := service.Create(ctx, key, preview.Input)
	require.NoError(t, err)
	require.True(t, replay.Replayed)
	require.Equal(t, saved.Record, replay.Record)
	require.Equal(t, 2, repo.writes)
	_, err = service.Create(ctx, uuid.NewString(), preview.Input)
	require.Error(t, err)
}
func TestSheinImportRejectsForeignSourceStaleHeadAndCommitReferenceDrift(t *testing.T) {
	for _, name := range []string{"foreign source", "stale head", "no identity", "commit drift"} {
		t.Run(name, func(t *testing.T) {
			ctx, s, r, v, _, in := importServiceFixture(t)
			switch name {
			case "foreign source":
				v.saved.Source.CollectionItemID = uuid.NewString()
			case "stale head":
				r.head.ID = uuid.NewString()
			case "no identity":
				ctx = context.Background()
			}
			p, err := s.PreviewSheinImport(ctx, in)
			if name != "commit drift" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			r.beforeCommit = func() { v.saved.Source.CollectionRevision++ }
			_, err = s.Create(ctx, uuid.NewString(), p.Input)
			require.ErrorIs(t, err, ErrConflict)
			require.Equal(t, 1, r.writes)
		})
	}
}

func TestSheinImportRequiresExplicitMappingPositions(t *testing.T) {
	for _, raw := range []string{`{"title":true,"description":false,"attributes":[{"attributeId":9}],"skus":[]}`, `{"title":true,"description":false,"attributes":[],"skus":[{"sourceIndex":0,"skuIndex":0,"price":true,"stock":false,"warehouseId":""}]}`, `{"title":true,"description":false,"attributes":[],"skus":[{"sourceIndex":null,"skcIndex":0,"skuIndex":0,"price":true,"stock":false,"warehouseId":""}]}`} {
		var selection SheinImportSelection
		require.Error(t, json.Unmarshal([]byte(raw), &selection))
	}
}

func importPtr[T any](v T) *T { return &v }
func importFixture() (goods.OfficialDraftInput, completiondraft.Content, SheinImportSelection, goods.OfficialRuleSnapshot) {
	base := goods.OfficialDraftInput{Product: model.PublishProduct{CategoryID: 123, ProductTypeID: 456, Names: []model.LanguageContent{{Language: "en", Name: "old"}, {Language: "fr", Name: "conserver"}}, Attributes: []model.AttributeValue{{AttributeID: 8, ExtraValue: "keep"}, {AttributeID: 9, AttributeValueID: importPtr(int64(10))}}, SKCs: []model.ProductSKC{{SKUs: []model.ProductSKU{{SupplierSKU: "", Length: "12", Prices: []model.ProductPrice{{BasePrice: 8, Currency: "USD", SubSite: "shein-us"}}, Stock: []model.ProductStock{{Quantity: 7, WarehouseID: "a", WarehouseName: "A"}, {Quantity: 4, WarehouseID: "b", WarehouseName: "B"}}}, {SupplierSKU: "", Stock: []model.ProductStock{{Quantity: 99}}}}}}}}
	candidate := completiondraft.Content{Title: "new", Description: "", Structured: &completiondraft.StructuredContent{Attributes: []completiondraft.CandidateAttribute{{Name: "material", Value: "cotton"}}, Variants: []completiondraft.CandidateVariant{{SourceIndex: importPtr(0), Price: &completiondraft.CandidatePrice{Currency: "USD", Amount: importPtr(0.0)}, Stock: importPtr(int64(0))}}}}
	selection := SheinImportSelection{Title: true, Attributes: []SheinAttributeMapping{{SourceIndex: 0, AttributeID: 9, OptionID: importPtr(int64(11))}}, SKUs: []SheinSKUMapping{{SourceIndex: 0, SKCIndex: 0, SKUIndex: 0, Price: true, Stock: true, WarehouseID: "a"}}}
	rules := goods.OfficialRuleSnapshot{ApplicationMode: model.ModeSelfOperated, Attributes: model.AttributeTemplate{Attributes: []model.Attribute{{ID: 9, Show: importPtr(1), Type: importPtr(4), Mode: importPtr(2), Status: importPtr(2), Dimension: importPtr(1), Options: []model.AttributeOption{{ID: 11, Show: importPtr(1)}}}}}, Warehouses: []model.Warehouse{{Code: "a", Name: "A", Type: 1, SaleCountries: []string{"US"}}, {Code: "b", Name: "B", Type: 1, SaleCountries: []string{"US"}}}}
	return base, candidate, selection, rules
}
func TestSheinImportPreservesEveryUnselectedFieldAndExplicitZero(t *testing.T) {
	base, candidate, selection, rules := importFixture()
	result, err := deriveSheinDraft(base, candidate, selection, rules)
	require.NoError(t, err)
	expected, _, _, _ := importFixture()
	expected.Product.Names[0].Name = "new"
	expected.Product.Attributes[1].AttributeValueID = importPtr(int64(11))
	expected.Product.SKCs[0].SKUs[0].Prices[0].BasePrice = 0
	expected.Product.SKCs[0].SKUs[0].Stock[0].Quantity = 0
	require.Equal(t, expected, result)
	untouched, _, _, _ := importFixture()
	require.Equal(t, untouched, base, "must not mutate baseline slices")
}
func TestSheinImportRejectsAmbiguousOrUnavailableMappings(t *testing.T) {
	for _, name := range []string{"negative source", "bad target", "duplicate source", "duplicate target", "missing price", "missing stock", "wrong currency", "unknown warehouse", "ambiguous stock", "stock range", "wrong attribute layer", "hidden option", "duplicate attr", "empty selection"} {
		t.Run(name, func(t *testing.T) {
			base, c, s, r := importFixture()
			switch name {
			case "negative source":
				s.SKUs[0].SourceIndex = -1
			case "bad target":
				s.SKUs[0].SKUIndex = 9
			case "duplicate source":
				s.SKUs = append(s.SKUs, s.SKUs[0])
				s.SKUs[1].SKUIndex = 1
			case "duplicate target":
				c.Structured.Variants = append(c.Structured.Variants, c.Structured.Variants[0])
				c.Structured.Variants[1].SourceIndex = importPtr(1)
				s.SKUs = append(s.SKUs, s.SKUs[0])
				s.SKUs[1].SourceIndex = 1
			case "missing price":
				c.Structured.Variants[0].Price = nil
			case "missing stock":
				c.Structured.Variants[0].Stock = nil
			case "wrong currency":
				c.Structured.Variants[0].Price.Currency = "CNY"
			case "unknown warehouse":
				s.SKUs[0].WarehouseID = "unknown"
			case "ambiguous stock":
				base.Product.SKCs[0].SKUs[0].Stock[0].WarehouseID = ""
			case "stock range":
				c.Structured.Variants[0].Stock = importPtr(int64(100000))
			case "wrong attribute layer":
				r.Attributes.Attributes[0].Dimension = importPtr(3)
			case "hidden option":
				r.Attributes.Attributes[0].Options[0].Show = importPtr(0)
			case "duplicate attr":
				s.Attributes = append(s.Attributes, s.Attributes[0])
			case "empty selection":
				s = SheinImportSelection{}
			}
			_, err := deriveSheinDraft(base, c, s, r)
			require.Error(t, err)
		})
	}
}
func TestSheinImportManagedPriceAndNoWarehouseStock(t *testing.T) {
	base, c, s, r := importFixture()
	s.Attributes = nil
	r.ApplicationMode = model.ModeFullyManaged
	r.Fill.Currency = importPtr("CNY")
	c.Structured.Variants[0].Price.Currency = "CNY"
	c.Structured.Variants[0].Price.Amount = importPtr(12.3)
	s.SKUs[0].WarehouseID = ""
	base.Product.SKCs[0].SKUs[0].Stock = []model.ProductStock{{Quantity: 7}}
	result, err := deriveSheinDraft(base, c, s, r)
	require.NoError(t, err)
	require.Equal(t, &model.CostPrice{Price: "12.3", Currency: "CNY"}, result.Product.SKCs[0].SKUs[0].Cost)
	require.Equal(t, base.Product.SKCs[0].SKUs[0].Prices, result.Product.SKCs[0].SKUs[0].Prices, "unselected retail field remains")
	require.Equal(t, 0, result.Product.SKCs[0].SKUs[0].Stock[0].Quantity)
	c.Structured = nil
	s = SheinImportSelection{Description: true}
	result, err = deriveSheinDraft(base, c, s, r)
	require.NoError(t, err)
	require.Equal(t, []model.LanguageContent{{Language: "en", Name: ""}}, result.Product.Descriptions)
}
