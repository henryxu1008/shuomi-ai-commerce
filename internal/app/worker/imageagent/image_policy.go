package imageagentworker

import (
	policyassembly "github.com/henryxu1008/shuomi-ai-commerce/internal/app/imageagentpolicy"
	policycatalog "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/policy/productimage"
	imagepolicy "github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/imagepolicy"
)

func loadEmbeddedImagePolicyResolver() (*imagepolicy.Resolver, error) {
	return policyassembly.LoadEmbeddedResolver()
}

func newImagePolicyResolver(catalog policycatalog.Catalog) (*imagepolicy.Resolver, error) {
	return policyassembly.NewResolver(catalog)
}
