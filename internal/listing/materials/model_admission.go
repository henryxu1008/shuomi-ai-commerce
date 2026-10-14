package materials

import (
	"encoding/json"
	"strings"

	"task-processor/internal/agent"
	"task-processor/internal/commercetool"

	sigjson "sigs.k8s.io/json"
)

// ModelAdmission is the Listing-owned shape boundary injected into the fixed
// model consumer. It validates data, never authorizes/saves platform facts.
type ModelAdmission struct{}

func (ModelAdmission) ProjectEvidence(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 || len(raw) > MaxInspectionBytes {
		return nil, agent.ErrInvalid
	}
	var d Disclosure
	strict, err := sigjson.UnmarshalStrict(raw, &d)
	if err != nil || len(strict) != 0 || !d.Binding.Valid() || d.Binding.ContextKind != "listing-target-record" || d.Binding.TargetPlatform != "shein" || d.Revision < 1 || !agent.ValidID(d.StoreID) || !strings.HasPrefix(d.EvidenceID, "listing-evidence:") || len(d.EvidenceID) != len("listing-evidence:")+64 {
		return nil, agent.ErrInvalid
	}
	return append(json.RawMessage(nil), raw...), nil
}

func (ModelAdmission) ValidateAction(a agent.Action) error {
	if len(a.Candidate.Changes) != 0 || len(a.Candidate.Warnings) != 0 || len(a.Candidate.Rejections) != 0 || len(a.Confidence) != 0 || len(a.ContextCitationIDs) != 0 {
		return agent.ErrInvalid
	}
	switch a.Kind {
	case "tool":
		if a.Tool == InspectToolRef() && a.DomainProposal == nil {
			return nil
		}
	case "interrupt":
		if a.Tool == (commercetool.ToolRef{}) && a.DomainProposal == nil {
			return nil
		}
	case "propose":
		if a.Tool != (commercetool.ToolRef{}) || a.DomainProposal == nil || a.DomainProposal.Kind != ProposalKind {
			return agent.ErrInvalid
		}
		_, err := ParseProposal(a.DomainProposal.Payload)
		return err
	}
	return agent.ErrInvalid
}
