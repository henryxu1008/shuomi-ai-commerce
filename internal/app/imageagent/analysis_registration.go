package imageagentapp

import (
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
	"task-processor/internal/product/asset/tools/assetinspect"
	"task-processor/internal/product/catalog/tools/canonicalinspect"
)

// Internal execution registration under the existing image responsibility;
// never a second enterprise activation/template or top-level Agent card.
func ImageAnalysisRegistration(approved bool) (commercetool.AgentDefinition, []commercetool.Definition) {
	tools := []commercetool.Definition{canonicalinspect.Definition()}
	if approved {
		tools = append(tools, assetinspect.Definition())
	}
	d := commercetool.AgentDefinition{ID: agentconfig.ImageAnalysisAgentID, Version: agentconfig.ImageAnalysisVersion}
	for _, tool := range tools {
		d.AllowedTools = append(d.AllowedTools, tool.Ref)
	}
	return d, tools
}
