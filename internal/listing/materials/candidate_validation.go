package materials

import (
	"context"
	"encoding/json"

	"task-processor/internal/agent"
	"task-processor/internal/commercetool"
)

func (v Validator) validateCandidateRules(ctx context.Context, request agent.Request, candidate agent.DomainProposal, history []agent.Observation) (agent.Validation, error) {
	invalid := agent.Validation{PolicyVersion: PolicyVersion, Unresolved: []string{"CANDIDATE_RULE_EVIDENCE_INVALID"}}
	reader, ok := v.Reader.(CandidateRulesReader)
	if !ok || !request.Binding.Valid() {
		return invalid, nil
	}
	result, err := ParseDomainResult(candidate.Payload)
	if err != nil || result.Purpose != CategoryRulesPurpose {
		return invalid, nil
	}
	current, err := reader.ReadCandidateRules(ctx, request.Binding)
	if err != nil {
		return agent.Validation{}, err
	}
	raw, err := json.Marshal(current)
	if err != nil || current.Binding != request.Binding {
		return agent.Validation{}, ErrInspectionChanged
	}
	if _, err = ParseCandidateRulesDisclosure(raw); err != nil {
		return agent.Validation{}, ErrInspectionChanged
	}
	for _, ref := range result.Explanation.References {
		found := false
		for _, o := range history {
			if o.Tool != RulesToolRef() || o.AuditStatus != commercetool.AuditStatusRecorded || o.InvocationID != "" || !agent.ValidID(o.CallID) || o.CallID != ref.CallID {
				continue
			}
			d, e := ParseCandidateRulesDisclosure(o.Output)
			if e == nil && d.Binding == request.Binding && d.Evidence.CallID == o.CallID && d.EvidenceID == current.EvidenceID && d.EvidenceID == ref.EvidenceID && d.Evidence.RuleDigest == ref.RuleDigest {
				found = true
				break
			}
		}
		if !found {
			return invalid, nil
		}
	}
	return agent.Validation{Valid: true, PolicyVersion: PolicyVersion, Unresolved: append([]string{}, current.Unknowns...)}, nil
}
