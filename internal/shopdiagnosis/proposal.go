package shopdiagnosis

import (
	"encoding/json"
	sigjson "sigs.k8s.io/json"
	"task-processor/internal/agent"
)

type Recommendation struct {
	Advice      string   `json:"advice"`
	EvidenceIDs []string `json:"evidenceIds"`
}
type Proposal struct {
	Overview        string           `json:"overview"`
	Recommendations []Recommendation `json:"recommendations"`
}

func ParseProposal(raw json.RawMessage) (Proposal, error) {
	var p Proposal
	if len(raw) > agent.MaxModelOutputBytes {
		return p, agent.ErrInvalid
	}
	errors, err := sigjson.UnmarshalStrict(raw, &p)
	if err != nil || len(errors) > 0 {
		return Proposal{}, agent.ErrInvalid
	}
	return p, nil
}
func ValidateProposal(out Inspection, p Proposal) error {
	if !boundedText(p.Overview, 512, true) || len(p.Recommendations) < 1 || len(p.Recommendations) > 5 {
		return agent.ErrInvalid
	}
	facts := map[string]bool{}
	for _, f := range out.Facts {
		facts[f.ID] = true
	}
	for _, r := range p.Recommendations {
		if !boundedText(r.Advice, 512, true) || len(r.EvidenceIDs) < 1 || len(r.EvidenceIDs) > 6 {
			return agent.ErrInvalid
		}
		seen := map[string]bool{}
		for _, id := range r.EvidenceIDs {
			if !facts[id] || seen[id] {
				return agent.ErrInvalid
			}
			seen[id] = true
		}
	}
	return nil
}
