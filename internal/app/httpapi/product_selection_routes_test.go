package httpapi

import (
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	collectionhttp "github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection/httpapi"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSelectionRoutesAlwaysPreserveLiveBoundaryWithoutStoreModule(t *testing.T) {
	routes := productSelectionRoutes(nil, nil)
	require.Len(t, routes, 5)
	var base []httproute.Descriptor
	for _, r := range currentWorkbenchApplicationRoutes {
		base = append(base, httproute.Descriptor{Method: r.Method, Path: r.Path})
	}
	base = append(base, collectionhttp.Routes(nil, nil)...)
	all := append(append([]httproute.Descriptor(nil), base...), routes...)
	require.NoError(t, validateCurrentApplicationRoutesInternal(all, false, false, false, false, false, false, false, currentApplicationOptionalRoutes{Collections: true, ProductSelection: true}))
	for n, route := range routes {
		require.NoError(t, validateProductSelectionDescriptor(route))
		weakened := route
		weakened.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyNone
		mutated := append([]httproute.Descriptor(nil), all...)
		mutated[len(base)+n] = weakened
		require.Error(t, validateCurrentApplicationRoutesInternal(mutated, false, false, false, false, false, false, false, currentApplicationOptionalRoutes{Collections: true, ProductSelection: true}))
		require.Error(t, validateCurrentApplicationRoutesInternal(all, false, false, false, false, false, false, false, currentApplicationOptionalRoutes{}))
	}
}
