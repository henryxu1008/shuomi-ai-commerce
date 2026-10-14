package selection

import (
	"encoding/json"
	sigjson "sigs.k8s.io/json"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
)

type ModelAdmission struct{}

func ParseInspection(raw json.RawMessage) (Inspection, error) {
	var out Inspection
	if len(raw) == 0 || len(raw) > MaxInspectionBytes {
		return out, agent.ErrInvalid
	}
	strict, err := sigjson.UnmarshalStrict(raw, &out)
	if err != nil || len(strict) > 0 || len(out.Products) < 2 || len(out.Products) > agent.MaxSelectionProducts {
		return Inspection{}, agent.ErrInvalid
	}
	previous := ""
	products := map[string]bool{}
	for _, p := range out.Products {
		if !p.Ref.Valid() || p.Ref.ItemID <= previous || products[p.Ref.ProductKey] || len(p.Facts) > MaxFacts || len(p.Missing) > 32 || !boundedText(p.SourceKind, 128, true) {
			return Inspection{}, agent.ErrInvalid
		}
		previous = p.Ref.ItemID
		products[p.Ref.ProductKey] = true
		seen := map[string]bool{}
		for _, f := range p.Facts {
			if !agent.ValidID(f.Path) || len(f.Value) > 512 || len(f.Value) == 0 || !json.Valid(f.Value) || f.ID != factID(p.Ref, f.Path, f.Value) || seen[f.Path] {
				return Inspection{}, agent.ErrInvalid
			}
			seen[f.Path] = true
		}
		for _, missing := range p.Missing {
			if !agent.ValidID(missing) {
				return Inspection{}, agent.ErrInvalid
			}
		}
	}
	return out, nil
}
func (ModelAdmission) ProjectEvidence(raw json.RawMessage) (json.RawMessage, error) {
	if _, err := ParseInspection(raw); err != nil {
		return nil, err
	}
	return append(json.RawMessage(nil), raw...), nil
}
func (ModelAdmission) ValidateAction(a agent.Action) error {
	if len(a.Candidate.Changes) > 0 || len(a.Candidate.Warnings) > 0 || len(a.Candidate.Rejections) > 0 || len(a.Confidence) > 0 || len(a.ContextCitationIDs) > 0 || len(a.Unresolved) > 4096 {
		return agent.ErrInvalid
	}
	for _, unresolved := range a.Unresolved {
		if !boundedText(unresolved, 8192, true) {
			return agent.ErrInvalid
		}
	}
	switch a.Kind {
	case "tool":
		if a.Tool == agentconfig.SelectionInspectToolRef() && a.DomainProposal == nil {
			return nil
		}
	case "interrupt":
		if a.Tool == (commercetool.ToolRef{}) && a.DomainProposal == nil {
			return nil
		}
	case "propose":
		if a.Tool != (commercetool.ToolRef{}) || a.DomainProposal == nil || a.DomainProposal.Kind != agentconfig.SelectionProposalKind {
			return agent.ErrInvalid
		}
		_, err := ParseProposal(a.DomainProposal.Payload)
		return err
	}
	return agent.ErrInvalid
}
