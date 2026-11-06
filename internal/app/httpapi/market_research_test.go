package httpapi

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/httproute"
	"task-processor/internal/product/dataacquisition"
	"task-processor/internal/product/research"
	rh "task-processor/internal/product/research/httpapi"
	"testing"
)

func TestResearchRouteAdmissionRequiresCapabilityAndExactNativeBoundary(t *testing.T) {
	base := []httproute.Descriptor{}
	for _, r := range currentWorkbenchApplicationRoutes {
		base = append(base, httproute.Descriptor{Method: r.Method, Path: r.Path})
	}
	routes := append(base, rh.Routes(nil)...)
	routes = append(routes, marketResearchAnalysisRoutes(nil, nil)...)
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

func TestResearchSourceRejectionIsDefiniteAndDoesNotMasqueradeAsUnknown(t *testing.T) {
	require.ErrorIs(t, researchProvenanceFailure(dataacquisition.ErrNotFound), research.ErrUnsupported)
	require.ErrorIs(t, researchProvenanceFailure(dataacquisition.ErrForbidden), research.ErrForbidden)
	require.ErrorIs(t, researchProvenanceFailure(dataacquisition.ErrUnavailable), research.ErrUnavailable)
}

func TestResearchAnalysisDescriptorsPreserveTheirExactBoundary(t *testing.T) {
	for _, d := range marketResearchAnalysisRoutes(nil, nil) {
		require.NoError(t, validateMarketResearchAnalysisDescriptor(d))
		changed := d
		changed.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyCachedRead
		require.Error(t, validateMarketResearchAnalysisDescriptor(changed))
		require.Contains(t, d.Path, ":project_id/analysis/")
	}
}
