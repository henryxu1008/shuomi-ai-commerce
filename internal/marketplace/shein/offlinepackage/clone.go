package offlinepackage

import (
	"encoding/json"
	"fmt"
	"task-processor/internal/marketplace/shein/offlinepackage/previewdata"
)

// ClonePackageForPersistence copies a SHEIN package using its persisted JSON
// contract without mutating the source.
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
	return &cloned, nil
}

func CloneProductForSubmit(product *previewdata.Product) (*previewdata.Product, error) {
	if product == nil {
		return nil, nil
	}
	data, err := json.Marshal(product)
	if err != nil {
		return nil, fmt.Errorf("clone shein product: %w", err)
	}
	var cloned previewdata.Product
	if err := json.Unmarshal(data, &cloned); err != nil {
		return nil, fmt.Errorf("clone shein product: %w", err)
	}
	return &cloned, nil
}
