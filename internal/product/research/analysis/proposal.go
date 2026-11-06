package analysis

import (
	"encoding/json"
	strict "sigs.k8s.io/json"
	"strings"
	"task-processor/internal/agent"
)

type Assessment struct {
	CandidateID    string   `json:"candidateId"`
	Recommendation string   `json:"recommendation"`
	Advice         string   `json:"advice"`
	EvidenceIDs    []string `json:"evidenceIds"`
}
type Proposal struct {
	Overview               string       `json:"overview"`
	OverviewEvidenceIDs    []string     `json:"overviewEvidenceIds"`
	RecommendedCandidateID string       `json:"recommendedCandidateId,omitempty"`
	Assessments            []Assessment `json:"assessments"`
}

func ParseProposal(raw json.RawMessage) (Proposal, error) {
	var p Proposal
	if len(raw) == 0 || len(raw) > agent.MaxModelOutputBytes {
		return p, agent.ErrInvalid
	}
	errs, e := strict.UnmarshalStrict(raw, &p, strict.DisallowDuplicateFields, strict.DisallowUnknownFields)
	if e != nil || len(errs) > 0 {
		return Proposal{}, agent.ErrInvalid
	}
	return p, nil
}
func adviceText(s string) bool { return strings.TrimSpace(s) != "" && bounded(s, 512) }
func ValidateProposal(out Inspection, p Proposal) error {
	if !adviceText(p.Overview) || len(p.Assessments) < 1 || len(p.Assessments) > len(out.Candidates) || len(p.OverviewEvidenceIDs) < 1 || len(p.OverviewEvidenceIDs) > 12 {
		return agent.ErrInvalid
	}
	all := map[string]bool{}
	candidates := map[string]map[string]bool{}
	for _, c := range out.Candidates {
		facts := map[string]bool{}
		for _, f := range c.Facts {
			facts[f.ID] = true
			all[f.ID] = true
		}
		candidates[c.Research.CandidateID] = facts
	}
	seenRefs := map[string]bool{}
	for _, id := range p.OverviewEvidenceIDs {
		if !all[id] || seenRefs[id] {
			return agent.ErrInvalid
		}
		seenRefs[id] = true
	}
	seen := map[string]bool{}
	for _, a := range p.Assessments {
		facts, ok := candidates[a.CandidateID]
		if !ok || seen[a.CandidateID] || !adviceText(a.Advice) || len(a.EvidenceIDs) < 1 || len(a.EvidenceIDs) > 6 || (a.Recommendation != "TEST_CANDIDATE" && a.Recommendation != "RESEARCH_MORE" && a.Recommendation != "NOT_RECOMMENDED") {
			return agent.ErrInvalid
		}
		seen[a.CandidateID] = true
		refs := map[string]bool{}
		for _, id := range a.EvidenceIDs {
			if !facts[id] || refs[id] {
				return agent.ErrInvalid
			}
			refs[id] = true
		}
	}
	if p.RecommendedCandidateID != "" && !seen[p.RecommendedCandidateID] {
		return agent.ErrInvalid
	}
	return nil
}
