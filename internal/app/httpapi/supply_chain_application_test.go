package httpapi

import (
	"github.com/stretchr/testify/require"
	supplyhttp "task-processor/internal/app/supplychain/httpapi"
	"task-processor/internal/httproute"
	collectionhttp "task-processor/internal/product/collection/httpapi"
	"testing"
)

func TestSupplyRouteAdmissionRetainsExactPrivatePermissions(t *testing.T) {
	var routes []httproute.Descriptor
	for _, r := range currentWorkbenchApplicationRoutes {
		routes = append(routes, httproute.Descriptor{Method: r.Method, Path: r.Path})
	}
	routes = append(routes, supplyhttp.SupplyRoutes(nil, nil)...)
	check := func(r []httproute.Descriptor, enabled bool) error {
		return validateCurrentApplicationRoutesInternal(r, false, false, false, false, false, false, false, currentApplicationOptionalRoutes{SupplyChain: enabled})
	}
	require.NoError(t, check(routes, true))
	require.Error(t, check(routes, false))
	for i := len(currentWorkbenchApplicationRoutes); i < len(routes); i++ {
		for _, mutate := range []func(*httproute.Descriptor){func(r *httproute.Descriptor) { r.AuthPolicy = httproute.AuthPolicyPublic }, func(r *httproute.Descriptor) {
			r.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyCachedRead
		}, func(r *httproute.Descriptor) { r.Permission = "workbench.collection.read" }, func(r *httproute.Descriptor) { r.RequestTimeout = 0 }} {
			changed := append([]httproute.Descriptor(nil), routes...)
			mutate(&changed[i])
			require.Error(t, check(changed, true))
		}
	}
}

func TestSourceMaterialsAdmissionCoexistsWithFullSupplyAndRetainsPrivateBoundaries(t *testing.T) {
	for _, full := range []bool{false, true} {
		var routes []httproute.Descriptor
		for _, r := range currentWorkbenchApplicationRoutes {
			routes = append(routes, httproute.Descriptor{Method: r.Method, Path: r.Path})
		}
		if full {
			routes = append(routes, supplyhttp.SupplyRoutes(nil, nil)...)
		} else {
			routes = append(routes, supplyhttp.PreparationRoutes(nil, nil)...)
		}
		for _, r := range supplyhttp.SourceMaterialRoutes(nil, nil) {
			if !full || r.Method == "GET" {
				routes = append(routes, r)
			}
		}
		routes = append(routes, acquisitionPlanRoutes(nil, nil)...)
		routes = append(routes, collectionhttp.Routes(nil, nil)...)
		flags := currentApplicationOptionalRoutes{SupplyChain: full, SourceMaterials: true, AcquisitionPlans: true, Collections: true}
		check := func(r []httproute.Descriptor) error {
			return validateCurrentApplicationRoutesInternal(r, false, false, false, false, false, false, false, flags)
		}
		require.NoError(t, check(routes))
		for i, r := range routes {
			if !supplyhttp.IsSourceMaterialRoute(r.Method, r.Path) {
				continue
			}
			for _, mutate := range []func(*httproute.Descriptor){func(d *httproute.Descriptor) { d.Permission = "workbench.supply.read" }, func(d *httproute.Descriptor) { d.AuthPolicy = httproute.AuthPolicyPublic }, func(d *httproute.Descriptor) {
				d.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyCachedRead
			}} {
				changed := append([]httproute.Descriptor(nil), routes...)
				mutate(&changed[i])
				require.Error(t, check(changed))
			}
		}
		flags.SourceMaterials = false
		require.Error(t, check(routes), "the new ordinary diagnostic route requires its explicit capability")
	}
}
