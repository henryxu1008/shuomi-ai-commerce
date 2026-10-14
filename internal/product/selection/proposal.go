package selection

import (
	"encoding/json"
	sigjson "sigs.k8s.io/json"
	"task-processor/internal/agent"
)

type Assessment struct {
	ItemID      string   `json:"itemId"`
	Advice      string   `json:"advice"`
	EvidenceIDs []string `json:"evidenceIds"`
}
type Proposal struct {
	Overview          string       `json:"overview"`
	RecommendedItemID string       `json:"recommendedItemId,omitempty"`
	Assessments       []Assessment `json:"assessments"`
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
	if !boundedText(p.Overview, 512, true) || len(p.Assessments) < 1 || len(p.Assessments) > len(out.Products) {
		return agent.ErrInvalid
	}
	products := map[string]map[string]bool{}
	for _, product := range out.Products {
		facts := map[string]bool{}
		for _, fact := range product.Facts {
			facts[fact.ID] = true
		}
		products[product.Ref.ItemID] = facts
	}
	if p.RecommendedItemID != "" && products[p.RecommendedItemID] == nil {
		return agent.ErrInvalid
	}
	seen := map[string]bool{}
	for _, assessment := range p.Assessments {
		facts := products[assessment.ItemID]
		if facts == nil || seen[assessment.ItemID] || !boundedText(assessment.Advice, 512, true) || len(assessment.EvidenceIDs) < 1 || len(assessment.EvidenceIDs) > 6 {
			return agent.ErrInvalid
		}
		seen[assessment.ItemID] = true
		citations := map[string]bool{}
		for _, id := range assessment.EvidenceIDs {
			if !facts[id] || citations[id] {
				return agent.ErrInvalid
			}
			citations[id] = true
		}
	}
	if p.RecommendedItemID != "" && !seen[p.RecommendedItemID] {
		return agent.ErrInvalid
	}
	return nil
}
