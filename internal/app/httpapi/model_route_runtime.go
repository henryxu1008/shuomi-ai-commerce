package httpapi

import (
	"context"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/integration/openai"
)

func newProductTextRouteResolver(cfg ProductAgentDependencies, policies map[governed.RouteKey]governed.RoutePolicy) (*governed.OrganizationRouteResolver, error) {
	if cfg.CurrentTextPolicies != nil {
		return governed.NewManagedOrganizationRouteResolver(openai.NewGormCredentialResolver(cfg.RunDB), cfg.CurrentTextPolicies)
	}
	return governed.NewManagedOrganizationRouteResolver(openai.NewGormCredentialResolver(cfg.RunDB), unavailableSharedTextReader{})
}

type unavailableSharedTextReader struct{}

func (unavailableSharedTextReader) ReadPolicy(context.Context, governed.RouteKey) (governed.RoutePolicy, error) {
	return governed.RoutePolicy{}, d.ErrNeedsConfiguration
}
