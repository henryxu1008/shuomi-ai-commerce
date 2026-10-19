package httpapi

import (
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/integration/openai"
)

func newProductTextRouteResolver(cfg ProductAgentDependencies, policies map[governed.RouteKey]governed.RoutePolicy) (*governed.OrganizationRouteResolver, error) {
	if cfg.CurrentTextPolicies != nil {
		return governed.NewManagedOrganizationRouteResolver(openai.NewGormCredentialResolver(cfg.RunDB), cfg.CurrentTextPolicies)
	}
	return governed.NewOrganizationRouteResolver(openai.NewGormCredentialResolver(cfg.RunDB), policies)
}
