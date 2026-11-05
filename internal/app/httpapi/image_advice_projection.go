package httpapi

import (
	"encoding/json"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
	"task-processor/internal/listing/materials"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/asset/tools/assetinspect"
	proposal "task-processor/internal/product/review/proposal"
)

// Projection reads only an immutable, successfully audited observation.
// Missing historical data is never reconstructed from current owner facts.
func recordedImageAdvice(s agent.State, native bool) (*asset.ImageAdvice, error) {
	ref := assetinspect.ImageAdviceDefinition().Ref
	if native {
		current := s.Request.PolicyVersion == materials.PolicyVersion && s.Request.PromptVersion == materials.PromptVersion
		originalV3 := s.Request.PolicyVersion == "listing-materials-policy-v3" && s.Request.PromptVersion == "listing-materials-prompt-v3"
		if !current && !originalV3 {
			return nil, nil
		}
		ref = materials.InspectToolRef()
	} else if !agentconfig.CurrentTitleRequest(s.Request.PolicyVersion, s.Request.PromptVersion) {
		return nil, nil
	}
	var result *asset.ImageAdvice
	for _, o := range s.History {
		if o.Tool != ref {
			continue
		}
		if o.AuditStatus != commercetool.AuditStatusRecorded || len(o.Output) == 0 {
			continue // A failed Tool remains readable history, never advice.
		}
		if !agent.ValidID(o.CallID) || o.InvocationID != "" || len(o.Output) > asset.MaxImageAdviceBytes {
			return nil, agent.ErrInvalid
		}
		var advice asset.ImageAdvice
		if native {
			raw, e := (materials.ModelAdmission{}).ProjectEvidence(o.Output)
			if e != nil {
				return nil, e
			}
			var d materials.Disclosure
			if json.Unmarshal(raw, &d) != nil || d.Binding != s.Request.Binding {
				return nil, agent.ErrInvalid
			}
			advice = d.ImageAdvice
			if advice.Binding.Target == nil || advice.Binding.Target.RecordID != s.Request.Binding.ContextID {
				return nil, agent.ErrInvalid
			}
		} else {
			var e error
			advice, e = asset.DecodeImageAdvice(o.Output)
			if e != nil {
				return nil, e
			}
			if advice.Binding.Target != nil {
				return nil, agent.ErrInvalid
			}
		}
		b := s.Request.Binding
		if advice.Verify() != nil || advice.Evidence.ToolID != ref.ID || advice.Evidence.ToolVersion != ref.Version || advice.Evidence.CallID != o.CallID ||
			advice.Binding.ProductKey != b.ProductKey || advice.Binding.CatalogVersion != b.CatalogVersion || advice.Binding.PublicationID != b.PublicationID || advice.Binding.TargetPlatform != b.TargetPlatform {
			return nil, agent.ErrInvalid
		}
		value := advice
		result = &value
	}
	return result, nil
}
func currentAgentRequest(s agent.State) bool {
	return agentconfig.CurrentTitleRequest(s.Request.PolicyVersion, s.Request.PromptVersion) ||
		s.Request.PolicyVersion == proposal.ContentPolicyVersion && agentconfig.ValidContentPrompt(s.Request.PromptVersion, s.Request.WorkbenchContextRef)
}
