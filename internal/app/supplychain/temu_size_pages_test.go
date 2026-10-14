package supplychainapp

import (
	"context"
	"github.com/stretchr/testify/require"
	record "task-processor/internal/listing/record/target"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"testing"
)

type temuSizePagesFixture struct {
	TemuSupplierSizeMerchant
	pages map[int64]model.SupplierSizePage
	reads []int64
}

type temuPagedRulesFixture struct {
	*temuRulesFixture
	pages  *temuSizePagesFixture
	revoke bool
}

func (f *temuPagedRulesFixture) TemuMerchant(ctx context.Context, scope collection.Scope, store string, expected *storecenter.ProductMerchantBinding) (TemuRulesMerchant, error) {
	if _, err := f.temuRulesFixture.TemuMerchant(ctx, scope, store, expected); err != nil {
		return nil, err
	}
	return f, nil
}
func (f *temuPagedRulesFixture) SupplierSizeClass(_ context.Context, cat int64) (model.SupplierSizeRequirement, error) {
	p := func(v int64) *int64 { return &v }
	return model.SupplierSizeRequirement{CatID: cat, Observed: true, Class: &model.SupplierSizeClass{CatID: p(cat), ClassID: p(7), ClassType: p(1), RelatedClassIDs: []int64{7, 8}}}, nil
}
func (f *temuPagedRulesFixture) SupplierSizeCharts(ctx context.Context, cat, offset int64) (model.SupplierSizePage, error) {
	page, err := f.pages.SupplierSizeCharts(ctx, cat, offset)
	if f.revoke {
		f.changed = true
	}
	return page, err
}
func (f *temuPagedRulesFixture) SupplierSizeSettings(_ context.Context, cat, class int64) (model.SupplierSizeSettings, error) {
	p := func(v int64) *int64 { return &v }
	return model.SupplierSizeSettings{Sizes: []string{"S"}, Mapping: &model.SupplierSizeMapping{Meta: &model.SupplierSizeMeta{Groups: []model.SupplierSizeGroup{{ID: p(1)}}}, Records: []model.SupplierSizeRecord{{Values: map[string]string{"1": "S"}}}}}, nil
}

func (f *temuPagedRulesFixture) SupplierAttributes(_ context.Context, cat int64) (model.SupplierAttributeTemplate, error) {
	p := func(v int64) *int64 { return &v }
	sale := true
	return model.SupplierAttributeTemplate{Properties: []model.SupplierProperty{{RefPID: p(2), IsSale: &sale, ControlType: p(10)}}}, nil
}

func TestRuleReaderCarriesBothSelectedSizePagesAndRevalidatesMerchant(t *testing.T) {
	scope, base, reader := newTemuRuleFixture(t, storecenter.MerchantFullyManaged)
	p := func(v int64) *int64 { return &v }
	f := &temuPagedRulesFixture{temuRulesFixture: base, pages: &temuSizePagesFixture{pages: map[int64]model.SupplierSizePage{}}}
	for _, offset := range []int64{0, 50} {
		f.pages.pages[offset] = model.SupplierSizePage{Offset: p(offset), PageSize: p(50), Total: p(100), Charts: []model.SupplierSizeChart{{ID: p(offset + 1), ClassID: p(7 + offset/50), UpdatedAt: p(1), Content: &model.SupplierSizeContent{Records: []model.SupplierSizeRecord{{Values: map[string]string{"1": "S"}}}}, Reusable: func() *bool { v := false; return &v }()}}}
	}
	reader.Stores = f
	selection := base.selection
	selection.Language = ""
	selection.CostTemplateID = ""
	selection.SupplierSizeOffset = 50
	selection.SupplierSizeOffsets[0] = 50
	_, snapshot, err := reader.ReadTemuTargetRules(context.Background(), scope, base.binding.StoreID, selection)
	require.NoError(t, err)
	require.Empty(t, snapshot.Issues)
	require.EqualValues(t, 50, *snapshot.SupplierSizeCharts.Offset)
	require.Len(t, snapshot.SupplierSizeChartPages, 1)
	require.EqualValues(t, 0, *snapshot.SupplierSizeChartPages[0].Offset)
	require.Len(t, snapshot.SupplierSizeSettings, 2)
	input := temugoods.DraftInput{Mode: "fully_managed", Selection: selection, Supplier: &model.SupplierGoods{SizeTemplateIDs: []int64{1, 51}, SpecProperties: []model.SupplierSpecProperty{{SupplierBasicProperty: model.SupplierBasicProperty{RefPID: p(2), Value: "S"}}}}}
	result := temugoods.Build(input, snapshot, asset.ApprovedAssetInventory{}, nil)
	for _, issue := range result.Issues {
		require.NotContains(t, []string{"supplier_size_chart_missing", "supplier_size_chart_unverified", "supplier_size_specification_unverified", "supplier_size_settings_unverified"}, issue.Code)
	}
	require.False(t, result.ReadyForUpload, "this bounded case only proves the actual size consumer, not complete product readiness")
	originalHash := collection.Digest(snapshot)
	page := f.pages.pages[0]
	page.Charts[0].UpdatedAt = p(2)
	f.pages.pages[0] = page
	_, snapshot, err = reader.ReadTemuTargetRules(context.Background(), scope, base.binding.StoreID, selection)
	require.NoError(t, err)
	require.NotEqual(t, originalHash, collection.Digest(snapshot), "current read must observe the selected prior page again")
	page.Charts[0].ID = p(51)
	f.pages.pages[0] = page
	_, snapshot, err = reader.ReadTemuTargetRules(context.Background(), scope, base.binding.StoreID, selection)
	require.NoError(t, err)
	require.Contains(t, snapshot.Issues, "supplier_size_charts_unverified")
	require.Nil(t, snapshot.SupplierSizeCharts)
	page.Charts[0].ID = p(1)
	f.pages.pages[0] = page
	delete(f.pages.pages, 50)
	_, snapshot, err = reader.ReadTemuTargetRules(context.Background(), scope, base.binding.StoreID, selection)
	require.NoError(t, err)
	require.Contains(t, snapshot.Issues, "supplier_size_charts_unverified")
	require.Nil(t, snapshot.SupplierSizeCharts)
	f.pages.pages[50] = model.SupplierSizePage{Offset: p(50), PageSize: p(50), Total: p(100), Charts: []model.SupplierSizeChart{{ID: p(51)}}}
	f.revoke = true
	_, snapshot, err = reader.ReadTemuTargetRules(context.Background(), scope, base.binding.StoreID, selection)
	require.ErrorIs(t, err, record.ErrNotReady)
	require.Empty(t, snapshot.Observations, "page facts cannot survive original merchant revocation")
}

func (f *temuSizePagesFixture) SupplierSizeCharts(_ context.Context, cat, offset int64) (model.SupplierSizePage, error) {
	f.reads = append(f.reads, offset)
	return f.pages[offset], nil
}

func TestSupplierSizePagesRereadSelectedPagesUnderOneCurrentMerchant(t *testing.T) {
	p := func(v int64) *int64 { return &v }
	f := &temuSizePagesFixture{pages: map[int64]model.SupplierSizePage{}}
	for _, offset := range []int64{0, 50, 100} {
		f.pages[offset] = model.SupplierSizePage{Offset: p(offset), PageSize: p(50), Total: p(150), Charts: []model.SupplierSizeChart{{ID: p(offset + 1)}}}
	}
	selection := model.RuleSelection{CatID: 34, SupplierSizeOffset: 100, SupplierSizeOffsets: [16]int64{50}}
	observed := []int64{}
	current, other, err := readTemuSupplierSizePages(context.Background(), f, selection, func(_, _ string, input, decoded any) {
		observed = append(observed, *decoded.(model.SupplierSizePage).Offset)
	})
	require.NoError(t, err)
	require.EqualValues(t, 100, *current.Offset)
	require.Len(t, other, 2)
	require.ElementsMatch(t, []int64{0, 50, 100}, f.reads)
	require.ElementsMatch(t, f.reads, observed)
	f.pages[50] = model.SupplierSizePage{Offset: p(50), PageSize: p(50), Total: p(150), Charts: []model.SupplierSizeChart{{ID: p(1)}}}
	_, _, err = readTemuSupplierSizePages(context.Background(), f, selection, func(string, string, any, any) {})
	require.Error(t, err, "a moving/overlapping page cannot prove two current references")
}
