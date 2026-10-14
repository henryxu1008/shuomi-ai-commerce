package goods

import "task-processor/internal/marketplace/temu/model"

func checkSupplierCategory(input model.SupplierGoods, snapshot RuleSnapshot) []Issue {
	fields := []*int64{input.Cat1ID, input.Cat2ID, input.Cat3ID, input.Cat4ID, input.Cat5ID, input.Cat6ID, input.Cat7ID, input.Cat8ID, input.Cat9ID, input.Cat10ID}
	invalid := func() []Issue {
		return []Issue{{"category", "supplier_category_path_unverified", "重新选择本店从一级到末级的完整官方类目"}}
	}
	if len(snapshot.SupplierCategoryPath) == 0 || len(snapshot.SupplierCategoryPath) > 10 {
		return invalid()
	}
	parent := int64(0)
	for index, value := range fields {
		if value == nil {
			return invalid()
		}
		if index >= len(snapshot.SupplierCategoryPath) {
			if *value != 0 {
				return invalid()
			}
			continue
		}
		c := snapshot.SupplierCategoryPath[index]
		if *value != c.CatID || snapshot.Selection.SupplierCategoryPath[index] != c.CatID || c.ParentCatID == nil || *c.ParentCatID != parent || c.Level == nil || *c.Level != int64(index+1) || c.Hidden == nil || *c.Hidden || c.HiddenType == nil || *c.HiddenType != 0 || c.Leaf == nil || *c.Leaf != (index == len(snapshot.SupplierCategoryPath)-1) {
			return invalid()
		}
		parent = c.CatID
	}
	if parent != snapshot.Selection.CatID || !snapshot.SupplierCategoryPath[len(snapshot.SupplierCategoryPath)-1].SelectableLeaf() {
		return invalid()
	}
	return nil
}
