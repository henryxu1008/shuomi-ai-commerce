package httpapi

import (
	"github.com/stretchr/testify/require"
	supplyhttp "task-processor/internal/app/supplychain/httpapi"
	"task-processor/internal/httproute"
	collectionhttp "task-processor/internal/product/collection/httpapi"
	"task-processor/internal/product/sourcing"
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
	_, _, err = collectionAcquisitionOptions([]bool{true, false, false, false, false, false, true, true})
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
			routes = append(routes, supplyhttp.SupplyRoutes(nil, nil)...)
		} else {
			routes = append(routes, supplyhttp.PreparationRoutes(nil, nil)...)
		}
		if combination.plans {
			routes = append(routes, acquisitionPlanRoutes(nil, nil)...)
		}
		drafts := supplyhttp.CompletionDraftRoutes(nil, nil)
		routes = append(routes, drafts...)
		features := currentApplicationOptionalRoutes{Collections: true, CompletionDrafts: true, AcquisitionPlans: combination.plans, SupplyChain: combination.full}
		require.NoError(t, validateCurrentApplicationRoutesInternal(routes, false, false, false, false, false, false, false, features))
		features.CompletionDrafts = false
		require.Error(t, validateCurrentApplicationRoutesInternal(routes, false, false, false, false, false, false, false, features))
		for _, r := range drafts {
			r.Permission = "workbench.collection.read"
			require.Error(t, supplyhttp.ValidateCompletionDraftDescriptor(r))
		}
	}
}
