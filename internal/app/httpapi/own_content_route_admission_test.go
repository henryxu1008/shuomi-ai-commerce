package httpapi

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/httproute"
	collectionhttp "task-processor/internal/product/collection/httpapi"
)

func TestOwnContentAgentGlobalAdmissionPreservesTrustedProfiles(t *testing.T) {
	for _, titleOnly := range []bool{false, true} {
		name := "full"
		if titleOnly {
			name = "title-only"
		}
		t.Run(name, func(t *testing.T) {
			routes := append(agentAndPointRoutes(false, false), collectionhttp.Routes(nil, nil)...)
			routes = append(routes, productAgentProfileRoutes(nil, titleOnly)...)
			routes = append(routes, ownContentAgentRoutes(nil)...)
			for _, route := range productReviewRoutes(nil, nil) {
				if route.Method != http.MethodPost || route.Path != "/api/product/text-proposals" {
					routes = append(routes, route)
				}
			}
			validate := func(bundle []httproute.Descriptor, profile bool) error {
				return validateCurrentApplicationRoutesInternal(bundle, false, false, false, false, false, false, false, currentApplicationOptionalRoutes{ProductAgent: true, ContentTitleOnly: profile, Collections: true})
			}
			require.NoError(t, validate(routes, titleOnly))
			require.Error(t, validate(routes, !titleOnly), "own routes cannot blur the trusted execution profile")
			for i, route := range routes {
				if !strings.HasPrefix(route.Path, ownContentAgentBase) {
					continue
				}
				missing := append([]httproute.Descriptor(nil), routes[:i]...)
				missing = append(missing, routes[i+1:]...)
				require.Error(t, validate(missing, titleOnly), route.Method+" "+route.Path)
				changed := append([]httproute.Descriptor(nil), routes...)
				changed[i].AuthPolicy = httproute.AuthPolicyPublic
				require.Error(t, validate(changed, titleOnly), "own route must retain verified identity: "+route.Path)
				changed[i] = route
				changed[i].OrganizationAccessPolicy = httproute.OrganizationAccessPolicyCachedRead
				require.Error(t, validate(changed, titleOnly), "own route must retain live organization authorization: "+route.Path)
			}
		})
	}
}
