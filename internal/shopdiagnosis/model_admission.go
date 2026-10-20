package shopdiagnosis

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
	strict, e := sigjson.UnmarshalStrict(raw, &out)
	if e != nil || len(strict) > 0 || !out.Ref.Valid() || len(out.Facts) > MaxFacts || len(out.Missing) > 32 {
		return Inspection{}, agent.ErrInvalid
	}
	seen := map[string]bool{}
	plain := append([]Fact{}, out.Facts...)
	for n, f := range out.Facts {
		if !agent.ValidID(f.Path) || !boundedText(f.Value, 512, true) || f.ID != factID(out.Ref, f.Path, f.Value) || seen[f.Path] {
			return Inspection{}, agent.ErrInvalid
		}
		seen[f.Path] = true
		plain[n].ID = ""
	}
	for _, m := range out.Missing {
		if !agent.ValidID(m) {
			return Inspection{}, agent.ErrInvalid
		}
	}
	ref := out.Ref
	ref.EvidenceDigest = ""
	if out.Ref.EvidenceDigest != digest(struct {
		Ref     agent.StoreDiagnosisRef
		Facts   []Fact
		Missing []string
	}{ref, plain, out.Missing}) {
		return Inspection{}, agent.ErrInvalid
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
	// The public run allows 4096 notes; the native validator owns at most one.
	if len(a.Unresolved) > 4095 {
		return agent.ErrInvalid
	}
	for _, note := range a.Unresolved {
		if !boundedText(note, 8192, true) {
			return agent.ErrInvalid
		}
	}
	if len(a.Candidate.Changes) > 0 || len(a.Candidate.Warnings) > 0 || len(a.Candidate.Rejections) > 0 || len(a.Confidence) > 0 || len(a.ContextCitationIDs) > 0 {
		return agent.ErrInvalid
	}
	switch a.Kind {
	case "tool":
		if a.Tool == agentconfig.DiagnosisInspectToolRef() && a.DomainProposal == nil {
			return nil
		}
	case "interrupt":
		if a.Tool == (commercetool.ToolRef{}) && a.DomainProposal == nil {
			return nil
		}
	case "propose":
		if a.Tool != (commercetool.ToolRef{}) || a.DomainProposal == nil || a.DomainProposal.Kind != agentconfig.DiagnosisProposalKind {
			return agent.ErrInvalid
		}
		_, err := ParseProposal(a.DomainProposal.Payload)
		return err
	}
	return agent.ErrInvalid
}
