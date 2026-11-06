package materials

import (
	"encoding/json"

	"task-processor/internal/agent"
	"task-processor/internal/commercetool"

	sigjson "sigs.k8s.io/json"
)

// ModelAdmission is the Listing-owned shape boundary injected into the fixed
// model consumer. It validates data, never authorizes/saves platform facts.
type ModelAdmission struct{}

func (m ModelAdmission) ProjectBoundEvidence(b agent.Binding, raw json.RawMessage) (json.RawMessage, error) {
	projected, err := m.ProjectEvidence(raw)
	if err != nil {
		return nil, err
	}
	var header struct {
		Binding       agent.Binding `json:"binding"`
		SchemaVersion string        `json:"schemaVersion"`
	}
	if json.Unmarshal(projected, &header) != nil || header.Binding != b || (b.ContextKind == agent.CandidateRulesContextKind) != (header.SchemaVersion == CandidateRulesSchema) {
		return nil, agent.ErrInvalid
	}
	return projected, nil
}

func (ModelAdmission) ProjectEvidence(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 || len(raw) > MaxInspectionBytes {
		return nil, agent.ErrInvalid
	}
	var header struct {
		SchemaVersion string `json:"schemaVersion"`
	}
	if json.Unmarshal(raw, &header) != nil {
		return nil, agent.ErrInvalid
	}
	if header.SchemaVersion == CandidateRulesSchema {
		if _, err := ParseCandidateRulesDisclosure(raw); err != nil {
			return nil, agent.ErrInvalid
		}
		return append(json.RawMessage(nil), raw...), nil
	}
	var d Disclosure
	strict, err := sigjson.UnmarshalStrict(raw, &d)
	if err != nil || len(strict) != 0 || !d.Binding.Valid() || d.Binding.ContextKind != "listing-target-record" || d.Binding.TargetPlatform != "shein" || d.Revision < 1 || !agent.ValidID(d.StoreID) {
		return nil, agent.ErrInvalid
	}
	fingerprint, err := disclosureFingerprint(d)
	if err != nil || fingerprint != d.EvidenceID {
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
		if (a.Tool == InspectToolRef() || a.Tool == RulesToolRef()) && a.DomainProposal == nil {
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
		_, err := ParseDomainResult(a.DomainProposal.Payload)
		return err
	}
	return agent.ErrInvalid
}

func (m ModelAdmission) ValidateBoundAction(b agent.Binding, a agent.Action) error {
	if !b.Valid() || m.ValidateAction(a) != nil {
		return agent.ErrInvalid
	}
	purpose, tool := MaterialsEditPurpose, InspectToolRef()
	if b.ContextKind == agent.CandidateRulesContextKind {
		purpose, tool = CategoryRulesPurpose, RulesToolRef()
	} else if b.ContextKind != "listing-target-record" {
		return agent.ErrInvalid
	}
	if a.Kind == "tool" && a.Tool != tool {
		return agent.ErrInvalid
	}
	if a.Kind == "propose" {
		p, err := ParseDomainResult(a.DomainProposal.Payload)
		if err != nil || p.Purpose != purpose {
			return agent.ErrInvalid
		}
	}
	return nil
}
