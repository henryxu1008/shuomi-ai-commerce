package httpapi

import (
	"fmt"
	"net/http"

	"task-processor/internal/core/config"
	"task-processor/internal/httproute"
	kernelmodule "task-processor/internal/kernel/module"
)

type runtimeBundle struct {
	routes           []httproute.Descriptor
	authDependencies *routeAuthDependencies
}

func buildRuntimeBundleFromModules(cfg *config.Config, modules []kernelmodule.Module) (runtimeBundle, error) {
	reg := kernelmodule.NewRegistry()

	for _, mod := range modules {
		if mod == nil {
			continue
		}
		if !mod.Enabled(cfg) {
			continue
		}
		if err := mod.Register(reg); err != nil {
			return runtimeBundle{}, fmt.Errorf("register module %s: %w", mod.Name(), err)
		}
	}

	return runtimeBundle{
		routes: reg.Routes(),
	}, nil
}

func (b runtimeBundle) buildServerBundle(port int, authorization routeAuthorization) (*http.Server, []httproute.Descriptor) {
	if b.authDependencies != nil {
		authorization = authorization.withWorkbench(*b.authDependencies)
	}
	return buildHTTPServerFromRoutes(port, b.routes, authorization), b.routes
}
