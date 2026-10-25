package httpapi

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/httproute"
	collectionhttp "task-processor/internal/product/collection/httpapi"
	"testing"
)

func TestChatContentRoutesGlobalAdmission(t *testing.T) {
	routes := append(agentAndPointRoutes(false, false), collectionhttp.Routes(nil, nil)...)
	routes = append(routes, productAgentProfileRoutes(nil, true)...)
	routes = append(routes, ownContentAgentRoutes(nil)...)
	routes = append(routes, productReviewRoutes(nil, nil)...)
	routes = append(routes, aiWorkbenchRoutes(nil)...)
	require.NoError(t, validateCurrentApplicationRoutesInternal(routes, false, false, false, false, false, false, false, currentApplicationOptionalRoutes{ProductAgent: true, ContentOnly: true, Collections: true, AIWorkbench: true}))
	for i, r := range routes {
		if r.Path != "/api/v1/workbench/chat/subjects/:context_kind/:context_id" {
			continue
		}
		for _, mutate := range []func(*httproute.Descriptor){func(d *httproute.Descriptor) { d.AuthPolicy = httproute.AuthPolicyPublic }, func(d *httproute.Descriptor) {
			d.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyCachedRead
		}, func(d *httproute.Descriptor) { d.Permission = "wrong" }, func(d *httproute.Descriptor) { d.RequestTimeout = 0 }, func(d *httproute.Descriptor) { d.Handler = nil }, func(d *httproute.Descriptor) { d.Module = "wrong" }} {
			changed := append([]httproute.Descriptor(nil), routes...)
			mutate(&changed[i])
			require.Error(t, validateCurrentApplicationRoutesInternal(changed, false, false, false, false, false, false, false, currentApplicationOptionalRoutes{ProductAgent: true, ContentOnly: true, Collections: true, AIWorkbench: true}))
		}
	}
}
