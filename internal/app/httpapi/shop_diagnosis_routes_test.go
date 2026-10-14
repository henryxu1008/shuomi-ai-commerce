package httpapi

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/httproute"
	"testing"
)

func TestDiagnosisRoutesKeepExactLiveBoundary(t *testing.T) {
	routes := shopDiagnosisRoutes(nil, nil)
	require.Len(t, routes, 5)
	for _, route := range routes {
		require.NoError(t, validateShopDiagnosisDescriptor(route))
		for _, mutate := range []func(*httproute.Descriptor){func(d *httproute.Descriptor) {
			d.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyCachedRead
		}, func(d *httproute.Descriptor) { d.Permission = "unregistered" }, func(d *httproute.Descriptor) { d.RequestTimeout = 0 }} {
			changed := route
			mutate(&changed)
			require.Error(t, validateShopDiagnosisDescriptor(changed))
		}
	}
}
