package httpapi

import (
	"context"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
)

func newProductTextRouteResolver(cfg ProductAgentDependencies, policies map[governed.RouteKey]governed.RoutePolicy) (*governed.OrganizationRouteResolver, error) {
	if cfg.CurrentTextPolicies != nil {
		return governed.NewManagedOrganizationRouteResolver(cfg.ChannelCredentials, cfg.CurrentTextPolicies)
	}
	return governed.NewManagedOrganizationRouteResolver(cfg.ChannelCredentials, unavailableSharedTextReader{})
}

type unavailableSharedTextReader struct{}

func (unavailableSharedTextReader) ReadPolicy(context.Context, governed.RouteKey) (governed.RoutePolicy, error) {
	return governed.RoutePolicy{}, d.ErrNeedsConfiguration
}
