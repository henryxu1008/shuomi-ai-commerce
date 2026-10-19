package shein

import (
	"task-processor/internal/product/catalog/canonical"
)

type CategoryResolver interface {
	Resolve(req *BuildRequest, canonical *canonical.Product, pkg *Package) *CategoryResolution
}

type AttributeResolver interface {
	Resolve(req *BuildRequest, canonical *canonical.Product, pkg *Package) *AttributeResolution
}

type SaleAttributeResolver interface {
	Resolve(req *BuildRequest, canonical *canonical.Product, pkg *Package) *SaleAttributeResolution
}
