package goods

import (
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/marketplace/temu/model"
	"testing"
)

func TestLocalCreationRequiresExactSKUIdentityAndDistinctRemoteIDs(t *testing.T) {
	p := func(v int64) *int64 { return &v }
	in := model.LocalGoodsV2{SKUs: []model.LocalSKU{{ExternalSKUID: "sku-a", Specifications: []model.LocalSpecificationValue{{ParentSpecID: p(10), Name: "Blue"}}}, {ExternalSKUID: "sku-b", Specifications: []model.LocalSpecificationValue{{ParentSpecID: p(10), Name: "Red"}}}}}
	out := model.LocalGoodsCreated{GoodsID: p(9007199254740993), ProductType: p(1), SKUs: []model.LocalCreatedSKU{{SKUID: p(21), ExternalSKUSN: "sku-b", Specifications: []model.LocalSpecificationValue{{ParentSpecID: p(10), SpecID: p(102), Name: "Red"}}}, {SKUID: p(20), ExternalSKUSN: "sku-a", Specifications: []model.LocalSpecificationValue{{ParentSpecID: p(10), SpecID: p(101), Name: "Blue"}}}}}
	require.True(t, CorrelatesLocalCreation(in, out))
	out.SKUs[0].SKUID = p(20)
	require.False(t, CorrelatesLocalCreation(in, out))
	out.SKUs[0].SKUID = p(21)
	out.SKUs[0].ExternalSKUSN = "sku-other"
	require.False(t, CorrelatesLocalCreation(in, out))
	out.SKUs[0].ExternalSKUSN = "sku-b"
	out.SKUs[0].Specifications[0].Name = "Green"
	require.False(t, CorrelatesLocalCreation(in, out))
	out.SKUs[0].Specifications[0].Name = "Red"
	out.GoodsID = nil
	require.False(t, CorrelatesLocalCreation(in, out))
}

func TestSupplierCreationCorrelatesAllSKCGroupsByOriginalSKUCode(t *testing.T) {
	p := func(v int64) *int64 { return &v }
	in := model.SupplierGoods{SKCs: []model.SupplierSKC{{SKUs: []model.SupplierSKU{{ExternalCode: "a"}, {ExternalCode: "b"}}}, {SKUs: []model.SupplierSKU{{ExternalCode: "c"}}}}}
	out := model.SupplierGoodsCreated{ProductID: p(1), SKCs: []model.SupplierCreatedSKC{{ID: p(10)}, {ID: p(11)}}, SKUs: []model.SupplierCreatedSKU{{ID: p(100), SKCID: p(10), ExternalCode: "b"}, {ID: p(101), SKCID: p(11), ExternalCode: "c"}, {ID: p(102), SKCID: p(10), ExternalCode: "a"}}}
	require.True(t, CorrelatesSupplierCreation(in, out))
	in.SKCs[0].SKUs[1].Specs = []model.SupplierSpecification{{ID: p(51), ParentID: p(31)}}
	out.SKUs[0].Specs = []model.SupplierSpecification{{ID: p(52), ParentID: p(31)}}
	require.False(t, CorrelatesSupplierCreation(in, out), "a returned SKU cannot correlate a different preset specification")
	out.SKUs[0].Specs = []model.SupplierSpecification{{ID: p(51), ParentID: p(31)}}
	require.True(t, CorrelatesSupplierCreation(in, out))
	require.True(t, ValidCreatedGoods(CreatedGoods{Supplier: &out, ResponseHash: strings.Repeat("a", 64)}))
	out.SKUs[0].SKCID = p(11)
	require.False(t, CorrelatesSupplierCreation(in, out), "a partial group match cannot seal a product success")
	out.SKUs[0].SKCID = p(10)
	out.SKUs = out.SKUs[:2]
	require.False(t, CorrelatesSupplierCreation(in, out))
}

func TestLocalCreationCorrelatesPresetWithoutInventingMissingParent(t *testing.T) {
	p := func(v int64) *int64 { return &v }
	in := model.LocalGoodsV2{SKUs: []model.LocalSKU{{ExternalSKUID: "sku-a", Specifications: []model.LocalSpecificationValue{{SpecID: p(101)}}}}}
	out := model.LocalGoodsCreated{GoodsID: p(100), ProductType: p(1), SKUs: []model.LocalCreatedSKU{{SKUID: p(201), ExternalSKUSN: "sku-a", Specifications: []model.LocalSpecificationValue{{ParentSpecID: p(10), SpecID: p(101), Name: "Blue"}}}}}
	require.True(t, CorrelatesLocalCreation(in, out))
	require.Nil(t, in.SKUs[0].Specifications[0].ParentSpecID, "correlation must not rewrite approved input")
	out.SKUs[0].Specifications[0].SpecID = p(102)
	require.False(t, CorrelatesLocalCreation(in, out))
	out.SKUs[0].Specifications[0].SpecID = p(101)
	in.SKUs[0].Specifications[0].ParentSpecID = p(11)
	require.False(t, CorrelatesLocalCreation(in, out), "explicit parent remains exact")
	in.SKUs[0].Specifications[0].ParentSpecID = nil
	in.SKUs[0].Specifications = append(in.SKUs[0].Specifications, in.SKUs[0].Specifications[0])
	out.SKUs[0].Specifications = append(out.SKUs[0].Specifications, out.SKUs[0].Specifications[0])
	require.False(t, CorrelatesLocalCreation(in, out), "repeated preset cannot be correlated twice")
}

func TestLocalCreationCorrelatesUnspecifiedExternalCodeByUniqueOriginalSpecs(t *testing.T) {
	p := func(v int64) *int64 { return &v }
	in := model.LocalGoodsV2{SKUs: []model.LocalSKU{{Specifications: []model.LocalSpecificationValue{{SpecID: p(101)}}}}}
	out := model.LocalGoodsCreated{GoodsID: p(100), ProductType: p(1), SKUs: []model.LocalCreatedSKU{{SKUID: p(201), ExternalSKUSN: "returned-code", Specifications: []model.LocalSpecificationValue{{ParentSpecID: p(10), SpecID: p(101)}}}}}
	require.True(t, CorrelatesLocalCreation(in, out), "unspecified optional code is not an approved empty remote value")
	require.Empty(t, in.SKUs[0].ExternalSKUID)
	in.SKUs = append(in.SKUs, in.SKUs[0])
	out.SKUs = append(out.SKUs, model.LocalCreatedSKU{SKUID: p(202), ExternalSKUSN: "second-code", Specifications: out.SKUs[0].Specifications})
	require.False(t, CorrelatesLocalCreation(in, out), "ambiguous original specs cannot be assigned by remote row order")
}
