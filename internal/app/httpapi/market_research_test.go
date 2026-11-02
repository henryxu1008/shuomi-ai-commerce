package httpapi

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/httproute"
	rh "task-processor/internal/product/research/httpapi"
	"testing"
)

func TestResearchRouteAdmissionRequiresCapabilityAndExactNativeBoundary(t *testing.T) {
	base := []httproute.Descriptor{}
	for _, r := range currentWorkbenchApplicationRoutes {
		base = append(base, httproute.Descriptor{Method: r.Method, Path: r.Path})
	}
	routes := append(base, rh.Routes(nil)...)
	validate := func(rs []httproute.Descriptor, enabled bool) error {
		return validateCurrentApplicationRoutesInternal(rs, false, false, false, false, false, false, false, currentApplicationOptionalRoutes{MarketResearch: enabled})
	}
	require.Error(t, validate(routes, false))
	require.Error(t, validate(base, true))
	require.NoError(t, validate(routes, true))
	for _, mutate := range []func(*httproute.Descriptor){func(d *httproute.Descriptor) { d.Permission = "workbench.project.read" }, func(d *httproute.Descriptor) { d.AuthPolicy = httproute.AuthPolicyPublic }, func(d *httproute.Descriptor) {
		d.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyCachedRead
	}} {
		changed := append([]httproute.Descriptor(nil), routes...)
		mutate(&changed[len(base)+1])
		require.Error(t, validate(changed, true))
	}
}
