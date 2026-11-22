package httpapi

import (
	listinghttp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/listing/httpapi"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	collectionhttp "github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection/httpapi"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/sourcing"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCompletionDraftProductCapabilityIsExplicitAndIndependentOfStoreAndCollector(t *testing.T) {
	c, _, err := collectionAcquisitionOptions([]bool{true, false, false, false, false, false, true})
	require.NoError(t, err)
	require.True(t, c.CompletionDrafts)
	require.True(t, c.Collections)
	require.False(t, c.SupplyChain)
	require.False(t, c.AcquisitionPlans)
	_, _, err = collectionAcquisitionOptions([]bool{false, false, false, false, false, false, true})
	require.ErrorIs(t, err, sourcing.ErrAcquisitionUnavailable)
	_, _, err = collectionAcquisitionOptions([]bool{true, false, false, false, false, false, true, true, true})
	require.ErrorIs(t, err, sourcing.ErrAcquisitionUnavailable)
}

func TestCompletionDraftRoutesAdmitOnlyNativeReadsAndSaveWithOriginalPreparationRoutes(t *testing.T) {
	for _, combination := range []struct{ plans, full bool }{{false, false}, {true, false}, {false, true}, {true, true}} {
		var routes []httproute.Descriptor
		for _, r := range currentWorkbenchApplicationRoutes {
			routes = append(routes, httproute.Descriptor{Method: r.Method, Path: r.Path})
		}
		routes = append(routes, collectionhttp.Routes(nil, nil)...)
		if combination.full {
			routes = append(routes, listinghttp.ListingRoutes(nil, nil)...)
		} else {
			routes = append(routes, listinghttp.PreparationRoutes(nil, nil)...)
		}
		if combination.plans {
			routes = append(routes, acquisitionPlanRoutes(nil, nil)...)
		}
		drafts := listinghttp.CompletionDraftRoutes(nil, nil)
		routes = append(routes, drafts...)
		features := currentApplicationOptionalRoutes{Collections: true, CompletionDrafts: true, AcquisitionPlans: combination.plans, SupplyChain: combination.full}
		require.NoError(t, validateCurrentApplicationRoutesInternal(routes, false, false, false, false, false, false, false, features))
		features.CompletionDrafts = false
		require.Error(t, validateCurrentApplicationRoutesInternal(routes, false, false, false, false, false, false, false, features))
		for _, r := range drafts {
			r.Permission = "workbench.collection.read"
			require.Error(t, listinghttp.ValidateCompletionDraftDescriptor(r))
		}
	}
}
