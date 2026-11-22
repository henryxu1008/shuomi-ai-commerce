package httpapi

import (
	"fmt"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/core/config"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	kernelmodule "github.com/henryxu1008/shuomi-ai-commerce/internal/kernel/module"
)

type runtimeBundle struct {
	routes []httproute.Descriptor
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
