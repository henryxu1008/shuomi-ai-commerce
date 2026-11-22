package httpapi

import (
	customhttp "github.com/henryxu1008/shuomi-ai-commerce/internal/agentcustomization/httpapi"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCurrentApplicationCustomizationRequiresOptInAndExactAuthority(t *testing.T) {
	routes := append(agentAndPointRoutes(false, false), customhttp.Routes(nil)...)
	validate := func(r []httproute.Descriptor, enabled bool) error {
		return validateCurrentApplicationRoutesInternal(r, false, false, false, false, false, false, false, currentApplicationOptionalRoutes{AgentCustomization: enabled})
	}
	require.Error(t, validate(routes, false))
	require.NoError(t, validate(routes, true))
	for i, route := range routes {
		if route.Module != "agent-customization" {
			continue
		}
		for name, mutate := range map[string]func(*httproute.Descriptor){
			"module":     func(d *httproute.Descriptor) { d.Module = "wrong" },
			"public":     func(d *httproute.Descriptor) { d.AuthPolicy = httproute.AuthPolicyPublic },
			"permission": func(d *httproute.Descriptor) { d.Permission = "" },
			"organization": func(d *httproute.Descriptor) {
				d.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyCachedRead
			},
			"deadline": func(d *httproute.Descriptor) { d.RequestTimeout = 0 },
			"body":     func(d *httproute.Descriptor) { d.RejectUnreadRequestBody = !d.RejectUnreadRequestBody },
		} {
			t.Run(route.Method+route.Path+name, func(t *testing.T) {
				changed := append([]httproute.Descriptor(nil), routes...)
				mutate(&changed[i])
				require.Error(t, validate(changed, true))
			})
		}
	}
}
