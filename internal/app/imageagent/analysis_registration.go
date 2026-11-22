package imageagentapp

import (
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agentconfig"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/commercetool"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset/tools/assetinspect"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog/tools/canonicalinspect"
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
