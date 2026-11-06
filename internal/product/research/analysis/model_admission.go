package analysis

import (
	"encoding/json"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
)

type ModelAdmission struct{}

func (ModelAdmission) ProjectEvidence(raw json.RawMessage) (json.RawMessage, error) {
	if _, err := ParseInspection(raw); err != nil {
		return nil, err
	}
	return append(json.RawMessage(nil), raw...), nil
}
func (ModelAdmission) ValidateAction(a agent.Action) error {
	// The native validator can append one diagnostic; reserve its place in
	// the unchanged 4096-entry response rather than truncating paid history.
	if len(a.Candidate.Changes) > 0 || len(a.Candidate.Warnings) > 0 || len(a.Candidate.Rejections) > 0 || len(a.Confidence) > 0 || len(a.ContextCitationIDs) > 0 || len(a.Unresolved) > 4095 {
		return agent.ErrInvalid
	}
	for _, unresolved := range a.Unresolved {
		if !(bounded(unresolved, 8192) && unresolved != "") {
			return agent.ErrInvalid
		}
	}
	switch a.Kind {
	case "tool":
		if a.Tool == agentconfig.ResearchInspectToolRef() && a.DomainProposal == nil {
			return nil
		}
	case "interrupt":
		if a.Tool == (commercetool.ToolRef{}) && a.DomainProposal == nil {
			return nil
		}
	case "propose":
		if a.Tool != (commercetool.ToolRef{}) || a.DomainProposal == nil || a.DomainProposal.Kind != agentconfig.ResearchProposalKind {
			return agent.ErrInvalid
		}
		_, err := ParseProposal(a.DomainProposal.Payload)
		return err
	}
	return agent.ErrInvalid
}
