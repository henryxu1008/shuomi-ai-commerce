package shein

import "encoding/json"

// ClonePackageForPersistence copies a SHEIN package using its persisted JSON
// contract while preserving owner-private resolution state without mutating the source.
func ClonePackageForPersistence(pkg *Package) (*Package, error) {
	if pkg == nil {
		return nil, nil
	}

	encoded, err := json.Marshal(pkg)
	if err != nil {
		return nil, err
	}
	var cloned Package
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		return nil, err
	}
	if pkg.SaleAttributeResolution != nil {
		if cloned.SaleAttributeResolution == nil {
			cloned.SaleAttributeResolution = &SaleAttributeResolution{}
		}
		cloned.SaleAttributeResolution.skcAssignments = cloneResolvedSaleAttributeMap(pkg.SaleAttributeResolution.skcAssignments)
		cloned.SaleAttributeResolution.skuAssignments = cloneResolvedSaleAttributeSliceMap(pkg.SaleAttributeResolution.skuAssignments)
		cloned.SaleAttributeResolution.skcValueAssignments = cloneResolvedSaleAttributeMap(pkg.SaleAttributeResolution.skcValueAssignments)
		cloned.SaleAttributeResolution.skuValueAssignments = cloneResolvedSaleAttributeMap(pkg.SaleAttributeResolution.skuValueAssignments)
	}
	return &cloned, nil
}
