package goods

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/marketplace/temu/model"
	"testing"
)

func TestSupplierSpecificationsRequireCartesianCoverageAndCurrentParentIDs(t *testing.T) {
	p := draftPointer[int64]
	template := model.SupplierAttributeTemplate{InputMaxSpecNum: p(0), Properties: []model.SupplierProperty{
		{RefPID: p(1), PID: p(11), TemplatePID: p(21), ParentSpecID: p(31), Name: "Color", Required: draftPointer(true), IsSale: draftPointer(true), MainSale: draftPointer(true), Values: []model.SupplierPropertyValue{{VID: p(41), SpecID: p(51), Value: "Blue"}, {VID: p(42), SpecID: p(52), Value: "Red"}}},
		{RefPID: p(2), PID: p(12), TemplatePID: p(22), ParentSpecID: p(32), Name: "Size", Required: draftPointer(true), IsSale: draftPointer(true), MainSale: draftPointer(false), Values: []model.SupplierPropertyValue{{VID: p(43), SpecID: p(53), Value: "S"}, {VID: p(44), SpecID: p(54), Value: "M"}}},
	}}
	spec := func(parent, id int64, name, value string) model.SupplierSpecification {
		return model.SupplierSpecification{ID: p(id), ParentID: p(parent), ParentName: name, Name: value}
	}
	properties := []model.SupplierSpecProperty{}
	for _, r := range template.Properties {
		for _, v := range r.Values {
			properties = append(properties, model.SupplierSpecProperty{SupplierBasicProperty: model.SupplierBasicProperty{VID: v.VID, PID: r.PID, TemplatePID: r.TemplatePID, RefPID: r.RefPID, Name: r.Name, Value: v.Value}, SpecID: v.SpecID, ParentSpecID: r.ParentSpecID})
		}
	}
	in := model.SupplierGoods{SpecProperties: properties, SKCs: []model.SupplierSKC{
		{MainSpecs: []model.SupplierSpecification{spec(31, 51, "Color", "Blue")}, SKUs: []model.SupplierSKU{{ExternalCode: "blue-s", Specs: []model.SupplierSpecification{spec(31, 51, "Color", "Blue"), spec(32, 53, "Size", "S")}}, {ExternalCode: "blue-m", Specs: []model.SupplierSpecification{spec(31, 51, "Color", "Blue"), spec(32, 54, "Size", "M")}}}},
		{MainSpecs: []model.SupplierSpecification{spec(31, 52, "Color", "Red")}, SKUs: []model.SupplierSKU{{ExternalCode: "red-s", Specs: []model.SupplierSpecification{spec(31, 52, "Color", "Red"), spec(32, 53, "Size", "S")}}, {ExternalCode: "red-m", Specs: []model.SupplierSpecification{spec(31, 52, "Color", "Red"), spec(32, 54, "Size", "M")}}}},
	}}
	require.Empty(t, checkSupplierSpecifications(in, template))
	in.SKCs[1].SKUs = in.SKCs[1].SKUs[:1]
	require.NotEmpty(t, checkSupplierSpecifications(in, template), "three SKUs cannot cover two colors by two sizes")
	in.SKCs[1].SKUs = append(in.SKCs[1].SKUs, in.SKCs[0].SKUs[1])
	require.NotEmpty(t, checkSupplierSpecifications(in, template), "duplicate combinations cannot replace the missing SKU")
	in.SKCs[0].SKUs[0].Specs[0].ParentID = p(999)
	require.NotEmpty(t, checkSupplierSpecifications(in, template), "preset child IDs cannot move to another parent")
}
