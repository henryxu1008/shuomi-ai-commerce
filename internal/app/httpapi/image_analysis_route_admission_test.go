package httpapi

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	"testing"
	"time"
)

func TestImageAnalysisAdmissionKeepsItsExactBoundaryIndependentOfGeneration(t *testing.T) {
	core := []httproute.Descriptor{}
	for _, r := range currentWorkbenchApplicationRoutes {
		core = append(core, httproute.Descriptor{Method: r.Method, Path: r.Path})
	}
	analysis := imageAnalysisRoutes(nil)
	routes := append(append([]httproute.Descriptor{}, core...), analysis...)
	verify := func(rs []httproute.Descriptor, analysis, generation bool) error {
		return validateCurrentApplicationRoutesInternal(rs, false, false, false, false, false, false, false, currentApplicationOptionalRoutes{ImageAnalysis: analysis, FullImageSet: generation})
	}
	require.NoError(t, verify(routes, true, false), "readonly analysis must not require a generation worker or Flare")
	require.Error(t, verify(routes, false, false), "absent module cannot grant analysis")
	for i := range analysis {
		for _, field := range []string{"module", "permission", "auth", "organization", "timeout", "method"} {
			t.Run(analysis[i].Path+field, func(t *testing.T) {
				altered := append([]httproute.Descriptor{}, routes...)
				r := &altered[len(core)+i]
				switch field {
				case "module":
					r.Module = "product-image-set"
				case "permission":
					r.Permission = authz.PermissionImageAgentRead
				case "auth":
					r.AuthPolicy = httproute.AuthPolicyCurrentIdentity
				case "organization":
					r.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyNone
				case "timeout":
					r.RequestTimeout = 30 * time.Second
				case "method":
					r.Method = http.MethodDelete
				}
				require.Error(t, verify(altered, true, false))
			})
		}
	}
	withGeneration := append(append([]httproute.Descriptor{}, routes...), (fullImageModule{}).routes()...)
	require.Error(t, verify(withGeneration, true, false), "analysis admission cannot grant generation")
	require.NoError(t, verify(withGeneration, true, true))
}
