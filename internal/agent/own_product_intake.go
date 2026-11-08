package agent

import "github.com/google/uuid"

const OwnProductIntakeContextKind = "own-product-intake-v1"

type OwnProductIntakeRef struct {
	InputSHA256 string
	ImageSHA256 string `json:",omitempty"`
}

func (b Binding) validOwnProductIntake() bool {
	id, err := uuid.Parse(b.ContextID)
	r := b.OwnProductIntake
	return err == nil && id != uuid.Nil && id.String() == b.ContextID && b.TargetPlatform == "product" &&
		canonicalResearchDigest(r.InputSHA256) && (r.ImageSHA256 == "" || canonicalResearchDigest(r.ImageSHA256)) &&
		b.ProductKey == "" && b.CatalogVersion == "" && b.PublicationID == "" &&
		b.Selection == ([MaxSelectionProducts]SelectionProductRef{}) && b.StoreDiagnosis == (StoreDiagnosisRef{}) &&
		b.ResearchSelection == (ResearchSelectionRef{}) && b.CandidateRules == (CandidateRulesRef{})
}
