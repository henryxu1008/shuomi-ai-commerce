// Package contentagent consumes the current Product, Review and Agent owners
// for bounded content localization. It owns no database, worker or provider.
package contentagent

import (
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
	"task-processor/internal/product/catalog/tools/canonicalinspect"
	"task-processor/internal/product/contentagent/scriptcompute"
	"task-processor/internal/product/sourcing/tools/sourceevidenceinspect"
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
