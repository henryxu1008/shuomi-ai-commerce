package agent

import (
	"encoding/hex"
	"time"
)

const StoreDiagnosisContextKind = "shop-operations-diagnosis-v1"

// StoreDiagnosisRef identifies an authorized native disclosure, never a Product
// or an alternative operating fact owner. Its presence grants no permission.
type StoreDiagnosisRef struct {
	StoreID, StartDate, EndDate, EvidenceDigest string
}

func (r StoreDiagnosisRef) Valid() bool {
	start, se := time.Parse(time.DateOnly, r.StartDate)
	end, ee := time.Parse(time.DateOnly, r.EndDate)
	digest, de := hex.DecodeString(r.EvidenceDigest)
	return selectionUUID(r.StoreID) && se == nil && ee == nil && start.Year() > 0 && end.Year() > 0 && r.StartDate == start.Format(time.DateOnly) && r.EndDate == end.Format(time.DateOnly) && !end.Before(start) && end.Sub(start)/(24*time.Hour) < 366 && de == nil && len(digest) == 32 && hex.EncodeToString(digest) == r.EvidenceDigest
}
func (b Binding) validStoreDiagnosis() bool {
	return selectionUUID(b.ContextID) && b.TargetPlatform == "business" && b.ProductKey == "" && b.CatalogVersion == "" && b.PublicationID == "" && b.Selection == ([MaxSelectionProducts]SelectionProductRef{}) && b.StoreDiagnosis.Valid()
}
