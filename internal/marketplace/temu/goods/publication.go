package goods

import (
	"task-processor/internal/marketplace/temu/model"
)

// CreatedGoods is only the correlated creation response. Review/on-sale facts
// belong to the separate read-only observation owner.
type CreatedGoods struct {
	Local        *model.LocalGoodsCreated    `json:"local,omitempty"`
	Supplier     *model.SupplierGoodsCreated `json:"supplier,omitempty"`
	ResponseHash string                      `json:"responseHash"`
}

func positiveID(v *int64) bool { return v != nil && *v > 0 }

func CorrelatesLocalCreation(input model.LocalGoodsV2, result model.LocalGoodsCreated) bool {
	if !positiveID(result.GoodsID) || !positiveID(result.ProductType) || len(input.SKUs) < 1 || len(input.SKUs) > 500 || len(input.SKUs) != len(result.SKUs) {
		return false
	}
	typeID := int64(1)
	if input.Basic.ProductType != nil {
		typeID = *input.Basic.ProductType
	}
	if *result.ProductType != typeID {
		return false
	}
	seenIDs := map[int64]bool{}
	matched := make([]bool, len(input.SKUs))
	for _, remote := range result.SKUs {
		if !positiveID(remote.SKUID) || seenIDs[*remote.SKUID] {
			return false
		}
		seenIDs[*remote.SKUID] = true
		found := -1
		for i, local := range input.SKUs {
			if matched[i] || local.ExternalSKUID != remote.ExternalSKUSN || !matchesLocalSpecs(local.Specifications, remote.Specifications) {
				continue
			}
			if found != -1 {
				return false
			}
			found = i
		}
		if found < 0 {
			return false
		}
		matched[found] = true
	}
	return true
}

func matchesLocalSpecs(input, result []model.LocalSpecificationValue) bool {
	if len(input) < 1 || len(input) != len(result) {
		return false
	}
	seen := map[int64]bool{}
	for _, local := range input {
		if !positiveID(local.ParentSpecID) || seen[*local.ParentSpecID] {
			return false
		}
		seen[*local.ParentSpecID] = true
		found := false
		for _, remote := range result {
			if !positiveID(remote.ParentSpecID) || *remote.ParentSpecID != *local.ParentSpecID {
				continue
			}
			if found || !positiveID(remote.SpecID) || local.SpecID != nil && *local.SpecID != *remote.SpecID || local.Name != "" && local.Name != remote.Name {
				return false
			}
			found = true
		}
		if !found {
			return false
		}
	}
	return true
}

func CorrelatesSupplierCreation(input model.SupplierGoods, result model.SupplierGoodsCreated) bool {
	if !positiveID(result.ProductID) || len(input.SKCs) < 1 || len(input.SKCs) > 40 || len(input.SKCs) != len(result.SKCs) {
		return false
	}
	remoteGroups := map[int64]bool{}
	for _, g := range result.SKCs {
		if !positiveID(g.ID) || remoteGroups[*g.ID] {
			return false
		}
		remoteGroups[*g.ID] = true
	}
	expected := map[string]int{}
	count := 0
	for i, g := range input.SKCs {
		if len(g.SKUs) < 1 {
			return false
		}
		for _, s := range g.SKUs {
			if s.ExternalCode == "" {
				return false
			}
			if _, dup := expected[s.ExternalCode]; dup {
				return false
			}
			expected[s.ExternalCode] = i
			count++
		}
	}
	if count < 1 || count > 1000 || count != len(result.SKUs) {
		return false
	}
	ids := map[int64]bool{}
	codes := map[string]bool{}
	assigned := map[int]int64{}
	groupOwners := map[int64]int{}
	for _, s := range result.SKUs {
		group, ok := expected[s.ExternalCode]
		if !ok || codes[s.ExternalCode] || !positiveID(s.ID) || ids[*s.ID] || !positiveID(s.SKCID) || !remoteGroups[*s.SKCID] {
			return false
		}
		if prior, ok := assigned[group]; ok && prior != *s.SKCID {
			return false
		}
		if prior, ok := groupOwners[*s.SKCID]; ok && prior != group {
			return false
		}
		assigned[group] = *s.SKCID
		groupOwners[*s.SKCID] = group
		ids[*s.ID] = true
		codes[s.ExternalCode] = true
	}
	return len(assigned) == len(input.SKCs)
}

func ValidCreatedGoods(value CreatedGoods) bool {
	if value.Local != nil && value.Supplier != nil {
		return false
	}
	if value.Local != nil {
		local := model.LocalGoodsV2{Basic: model.LocalGoodsBasic{ProductType: value.Local.ProductType}}
		for _, s := range value.Local.SKUs {
			local.SKUs = append(local.SKUs, model.LocalSKU{ExternalSKUID: s.ExternalSKUSN, Specifications: s.Specifications})
		}
		return CorrelatesLocalCreation(local, *value.Local)
	}
	if value.Supplier != nil {
		local := model.SupplierGoods{}
		for _, group := range value.Supplier.SKCs {
			if !positiveID(group.ID) {
				return false
			}
			g := model.SupplierSKC{}
			for _, sku := range value.Supplier.SKUs {
				if positiveID(sku.SKCID) && *sku.SKCID == *group.ID {
					g.SKUs = append(g.SKUs, model.SupplierSKU{ExternalCode: sku.ExternalCode})
				}
			}
			local.SKCs = append(local.SKCs, g)
		}
		return CorrelatesSupplierCreation(local, *value.Supplier)
	}
	return false
}
