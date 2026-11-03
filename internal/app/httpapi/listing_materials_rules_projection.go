package httpapi

import (
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
	"task-processor/internal/listing/materials"
	"task-processor/internal/product/collection"
)

// History consumes recorded facts only. It must not query today's rules or
// regenerate an explanation, including after head/configuration has changed.
func recordedCandidateRules(s agent.State) (*materials.CandidateRulesDisclosure, *materials.RuleExplanation, error) {
	var rules *materials.CandidateRulesDisclosure
	reads := map[string]materials.CandidateRulesDisclosure{}
	if !s.Request.Binding.Valid() || s.Request.Binding.ContextKind != agent.CandidateRulesContextKind {
		return nil, nil, agent.ErrInvalid
	}
	for _, o := range s.History {
		if o.Tool != materials.RulesToolRef() || o.AuditStatus != commercetool.AuditStatusRecorded || len(o.Output) == 0 {
			continue
		}
		if !agent.ValidID(o.CallID) || o.InvocationID != "" {
			return nil, nil, agent.ErrInvalid
		}
		value, err := materials.ParseCandidateRulesDisclosure(o.Output)
		if err != nil || value.Binding != s.Request.Binding || value.Evidence.CallID != o.CallID {
			return nil, nil, agent.ErrInvalid
		}
		rules = &value
		if _, duplicate := reads[o.CallID]; duplicate {
			return nil, nil, agent.ErrInvalid
		}
		reads[o.CallID] = value
	}
	if s.Phase != agent.HumanReviewRequired || s.StopReason != "" || s.RejectedPayload != nil || s.DomainProposal == nil || s.Validation == nil || !s.Validation.Valid || s.Validation.PolicyVersion != s.Request.PolicyVersion || s.Validation.CandidateHash != collection.Digest(s.DomainProposal) {
		return rules, nil, nil
	}
	if s.DomainProposal.Kind != agentconfig.MaterialsStoredProposalKind(s.Request.PolicyVersion, s.Request.PromptVersion) {
		return nil, nil, agent.ErrInvalid
	}
	result, err := materials.ParseDomainResult(s.DomainProposal.Payload)
	if err != nil || result.Purpose != materials.CategoryRulesPurpose || rules == nil {
		return nil, nil, agent.ErrInvalid
	}
	for n, ref := range result.Explanation.References {
		observed, found := reads[ref.CallID]
		if !found || ref.EvidenceID != observed.EvidenceID || ref.RuleDigest != observed.Evidence.RuleDigest {
			return nil, nil, agent.ErrInvalid
		}
		if n == 0 {
			rules = &observed
		} else if observed.EvidenceID != rules.EvidenceID || observed.Evidence.RuleDigest != rules.Evidence.RuleDigest {
			return nil, nil, agent.ErrInvalid
		}
	}
	return rules, result.Explanation, nil
}
