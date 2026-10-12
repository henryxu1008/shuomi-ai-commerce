package goods

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/marketplace/temu/model"
	"testing"
)

func TestSupplierSizeUnknownIsNotOptionalAndReusableNeedsApprovedCreation(t *testing.T) {
	in := model.SupplierGoods{}
	snapshot := RuleSnapshot{Selection: model.RuleSelection{CatID: 34}}
	require.NotEmpty(t, checkSupplierSize(in, snapshot))
	snapshot.SupplierSizeRequirement = &model.SupplierSizeRequirement{CatID: 34, Observed: true}
	require.Empty(t, checkSupplierSize(in, snapshot), "explicitly observed no mapping must not keep a blanket gate")
	snapshot.SupplierSizeRequirement.Class = &model.SupplierSizeClass{CatID: draftPointer(int64(34)), ClassID: draftPointer(int64(7)), ClassType: draftPointer(int64(0))}
	require.NotEmpty(t, checkSupplierSize(in, snapshot))
	in.SizeTemplateIDs = []int64{100}
	in.SpecProperties = []model.SupplierSpecProperty{{SupplierBasicProperty: model.SupplierBasicProperty{RefPID: draftPointer(int64(2)), Value: "S"}}}
	snapshot.SupplierAttributes = &model.SupplierAttributeTemplate{Properties: []model.SupplierProperty{{RefPID: draftPointer(int64(2)), IsSale: draftPointer(true), ControlType: draftPointer(int64(10))}}}
	snapshot.SupplierSizeCharts = &model.SupplierSizePage{Charts: []model.SupplierSizeChart{{ID: draftPointer(int64(100)), ClassID: draftPointer(int64(7)), Reusable: draftPointer(false), Content: &model.SupplierSizeContent{Records: []model.SupplierSizeRecord{{Values: map[string]string{"1": "S", "22": "10"}}}}}}}
	snapshot.SupplierSizeSettings = []model.SupplierSizeSettingsObservation{{ClassID: 7, Settings: model.SupplierSizeSettings{Sizes: []string{"S", "M"}, Mapping: &model.SupplierSizeMapping{Meta: &model.SupplierSizeMeta{Groups: []model.SupplierSizeGroup{{ID: draftPointer(int64(1)), Name: "Current size group"}}}, Records: []model.SupplierSizeRecord{{Values: map[string]string{"1": "S"}}, {Values: map[string]string{"1": "M"}}}}}}}
	require.Empty(t, checkSupplierSize(in, snapshot))
	firstPage := *snapshot.SupplierSizeCharts
	snapshot.SupplierSizeChartPages = []model.SupplierSizePage{firstPage}
	snapshot.SupplierSizeCharts = &model.SupplierSizePage{Charts: []model.SupplierSizeChart{}}
	require.Empty(t, checkSupplierSize(in, snapshot), "a current reread referenced page proves the selected table after navigation")
	snapshot.SupplierSizeCharts = &firstPage
	snapshot.SupplierSizeChartPages = nil
	snapshot.SupplierSizeCharts.Charts[0].Reusable = draftPointer(true)
	require.NotEmpty(t, checkSupplierSize(in, snapshot), "a reusable table ID is not a silently created publish business ID")
	snapshot.SupplierSizeCharts.Charts[0].Reusable = draftPointer(false)
	snapshot.SupplierSizeCharts.Charts[0].Content.Records[0].Values["1"] = "M"
	require.NotEmpty(t, checkSupplierSize(in, snapshot), "wrong chart records cannot be inferred from count alone")
	snapshot.SupplierSizeCharts.Charts[0].Content.Records[0].Values["22"] = "S"
	require.NotEmpty(t, checkSupplierSize(in, snapshot), "a measurement equal to a size label cannot replace its current official size group")
}
