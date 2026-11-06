package agent

import "encoding/hex"

const ResearchSelectionContextKind = "market-research-selection-v1"

// ResearchSelectionRef pins a current research choice and the exact bounded
// disclosure. It carries no access grant or independent research result.
type ResearchSelectionRef struct {
	Candidates       [MaxSelectionProducts]ResearchCandidateRef
	InspectionDigest string
}
type ResearchCandidateRef struct {
	CandidateID, CandidateRevision, SelectedLinkID, EvidenceDigest string
}

func canonicalResearchDigest(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == 32 && hex.EncodeToString(b) == s
}
func (b Binding) validResearchSelection() bool {
	if b.StoreDiagnosis != (StoreDiagnosisRef{}) || !b.validSelection() || !canonicalResearchDigest(b.ResearchSelection.InspectionDigest) {
		return false
	}
	seen := map[string]bool{}
	for i, p := range b.Selection {
		r := b.ResearchSelection.Candidates[i]
		if p == (SelectionProductRef{}) {
			if r != (ResearchCandidateRef{}) {
				return false
			}
			continue
		}
		if !selectionUUID(r.CandidateID) || !selectionVersion(r.CandidateRevision) || !selectionUUID(r.SelectedLinkID) || !canonicalResearchDigest(r.EvidenceDigest) || seen[r.CandidateID] {
			return false
		}
		seen[r.CandidateID] = true
	}
	return true
}
