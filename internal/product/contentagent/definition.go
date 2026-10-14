// Package contentagent consumes the current Product, Review and Agent owners
// for bounded content localization. It owns no database, worker or provider.
package contentagent

import (
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
	"task-processor/internal/product/catalog/tools/canonicalinspect"
	"task-processor/internal/product/sourcing/tools/sourceevidenceinspect"
)

func Registration() (commercetool.AgentDefinition, []commercetool.Definition) {
	tools := []commercetool.Definition{canonicalinspect.Definition(), sourceevidenceinspect.Definition()}
	definition := commercetool.AgentDefinition{ID: agentconfig.ContentAgentID, Version: agentconfig.ContentAgentVersion}
	for _, tool := range tools {
		definition.AllowedTools = append(definition.AllowedTools, tool.Ref)
	}
	return definition, tools
}
