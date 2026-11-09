package httpapi

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/httproute"
	kernel "task-processor/internal/kernel/module"
	"task-processor/internal/product/dataacquisition"
	"task-processor/internal/product/research"
	rh "task-processor/internal/product/research/httpapi"
)

// Only capability checks run in this assembly test. No store/source methods,
// HTTP listeners, databases or external transports are invoked.
type assessmentRouteCapabilityStore struct {
	research.Repository
	research.AssessmentStore
	ready bool
}

func (s assessmentRouteCapabilityStore) AssessmentsReady() bool { return s.ready }

func TestResearchAssessmentRoutesAdmittedByCurrentApplication(t *testing.T) {
	for _, ready := range []bool{false, true} {
		t.Run(map[bool]string{false: "schema_missing", true: "schema_ready"}[ready], func(t *testing.T) {
			service := &research.Service{
				Store:            assessmentRouteCapabilityStore{ready: ready},
				SupplierSources:  struct{ research.SupplierSources }{},
				AssessmentExpiry: func(context.Context) time.Time { return time.Now().Add(time.Second) },
			}
			require.Equal(t, ready, service.AssessmentsEnabled())
			registry := kernel.NewRegistry()
			require.NoError(t, (researchModule{handler: &rh.Handler{Service: service}, service: service}).Register(registry))
			registered := registry.Routes()
			want := 3
			if ready {
				want += 3
			}
			require.Len(t, registered, want)
			routes := []httproute.Descriptor{}
			for _, r := range currentWorkbenchApplicationRoutes {
				routes = append(routes, httproute.Descriptor{Method: r.Method, Path: r.Path})
			}
			routes = append(routes, registered...)
			routes = append(routes, marketResearchAnalysisRoutes(nil, nil)...)
			optional := currentApplicationOptionalRoutes{MarketResearch: true, ResearchAssessments: service.AssessmentsEnabled()}
			validate := func(rs []httproute.Descriptor, flags currentApplicationOptionalRoutes) error {
				return validateCurrentApplicationRoutesInternal(rs, false, false, false, false, false, false, false, flags)
			}
			require.NoError(t, validate(routes, optional))
			if !ready {
				optional.ResearchAssessments = true
				require.Error(t, validate(routes, optional), "admission must require all three assessment routes")
				return
			}
			require.Error(t, validate(routes, currentApplicationOptionalRoutes{MarketResearch: true}), "an absent capability must not admit assessment routes")
			require.Error(t, validate(routes, currentApplicationOptionalRoutes{ResearchAssessments: true}), "assessments require Research")
			for i, route := range routes {
				if !strings.Contains(route.Path, "/assessment") {
					continue
				}
				t.Run(route.Method+route.Path, func(t *testing.T) {
					for name, mutate := range map[string]func(*httproute.Descriptor){
						"permission": func(d *httproute.Descriptor) {
							d.Permission = "workbench.project.manage"
							if d.Method == http.MethodPost {
								d.Permission = "workbench.project.read"
							}
						},
						"identity": func(d *httproute.Descriptor) { d.AuthPolicy = httproute.AuthPolicyPublic },
						"organization": func(d *httproute.Descriptor) {
							d.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyCachedRead
						},
						"target": func(d *httproute.Descriptor) {
							d.OrganizationTargetResolver = func(*http.Request) (string, error) { return "other", nil }
						},
						"deadline":    func(d *httproute.Descriptor) { d.RequestTimeout = 0 },
						"body_policy": func(d *httproute.Descriptor) { d.RejectUnreadRequestBody = !d.RejectUnreadRequestBody },
						"module":      func(d *httproute.Descriptor) { d.Module = "other" },
						"handler":     func(d *httproute.Descriptor) { d.Handler = nil },
						"method":      func(d *httproute.Descriptor) { d.Method = http.MethodDelete },
						"path":        func(d *httproute.Descriptor) { d.Path += "/unadmitted" },
					} {
						t.Run(name, func(t *testing.T) {
							changed := append([]httproute.Descriptor(nil), routes...)
							mutate(&changed[i])
							require.Error(t, validate(changed, optional))
						})
					}
					missing := append([]httproute.Descriptor(nil), routes[:i]...)
					missing = append(missing, routes[i+1:]...)
					require.Error(t, validate(missing, optional))
				})
			}
		})
	}
}

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
	supplier := rh.Routes(nil)[0]
	supplier.Path = rh.SupplierReviewPath
	withSupplier := append(append([]httproute.Descriptor(nil), routes...), supplier)
	require.Error(t, validate(withSupplier, true), "supplier review must remain closed without its own capability")
	enabled := currentApplicationOptionalRoutes{MarketResearch: true, SupplierReview: true}
	require.NoError(t, validateCurrentApplicationRoutesInternal(withSupplier, false, false, false, false, false, false, false, enabled))
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
