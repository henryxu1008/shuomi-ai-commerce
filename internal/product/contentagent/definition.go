// Package contentagent consumes the current Product, Review and Agent owners
// for bounded content localization. It owns no database, worker or provider.
package contentagent

import (
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agentconfig"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/commercetool"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog/tools/canonicalinspect"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/contentagent/scriptcompute"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/sourcing/tools/sourceevidenceinspect"
)

func Registration() (commercetool.AgentDefinition, []commercetool.Definition) {
	tools := []commercetool.Definition{canonicalinspect.Definition(), sourceevidenceinspect.Definition(), scriptcompute.Definition()}
	definition := commercetool.AgentDefinition{ID: agentconfig.ContentAgentID, Version: agentconfig.ContentAgentVersion}
	for _, tool := range tools {
		definition.AllowedTools = append(definition.AllowedTools, tool.Ref)
	}
	return definition, tools
}

// Only factual read tools are exposed to the governed model; compute belongs
// exclusively to the code-owned fixed graph node.
func ModelTools() []commercetool.ToolRef {
	return []commercetool.ToolRef{canonicalinspect.Definition().Ref, sourceevidenceinspect.Definition().Ref}
}
