package agent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func researchBindingFixture() Binding {
	b := Binding{ContextKind: ResearchSelectionContextKind, ContextID: "11111111-1111-4111-8111-111111111111", TargetPlatform: "product"}
	for n, id := range []string{"22222222-2222-4222-8222-222222222222", "33333333-3333-4333-8333-333333333333"} {
		b.Selection[n] = SelectionProductRef{ItemID: id, ItemRevision: "1", ProductKey: id, CatalogVersion: "1", PublicationID: id}
		b.ResearchSelection.Candidates[n] = ResearchCandidateRef{CandidateID: id, CandidateRevision: "2", SelectedLinkID: id, EvidenceDigest: strings.Repeat("a", 64)}
	}
	b.ResearchSelection.InspectionDigest = strings.Repeat("b", 64)
	return b
}

func TestResearchAndCandidateRulesBindingsCannotBorrowEachOthersReferences(t *testing.T) {
	research, rules := researchBindingFixture(), candidateRulesBindingFixture(t, "22")
	require.True(t, research.Valid())
	require.True(t, rules.Valid())
	research.CandidateRules = rules.CandidateRules
	require.False(t, research.Valid(), "research input cannot carry a category selector")
	rules.ResearchSelection = research.ResearchSelection
	require.False(t, rules.Valid(), "category input cannot carry research evidence")
}

func TestResearchBindingPinsProjectCandidateAndObservationWithoutBorrowingOtherContexts(t *testing.T) {
	b := researchBindingFixture()
	require.True(t, b.Valid())
	for _, mutate := range []func(*Binding){
		func(b *Binding) { b.ContextKind = SelectionContextKind },
		func(b *Binding) { b.ContextKind = "acquisition" },
		func(b *Binding) { b.ContextID = "project-from-browser" },
		func(b *Binding) { b.ProductKey = "single-product" },
		func(b *Binding) { b.ResearchSelection.Candidates[1].CandidateRevision = "01" },
		func(b *Binding) {
			b.ResearchSelection.Candidates[1].CandidateID = b.ResearchSelection.Candidates[0].CandidateID
		},
		func(b *Binding) { b.ResearchSelection.Candidates[1].EvidenceDigest = strings.Repeat("A", 64) },
		func(b *Binding) { b.ResearchSelection.Candidates[1] = ResearchCandidateRef{} },
		func(b *Binding) { b.ResearchSelection.InspectionDigest = "not-a-digest" },
		func(b *Binding) { b.StoreDiagnosis.StoreID = "22222222-2222-4222-8222-222222222222" },
	} {
		changed := b
		mutate(&changed)
		require.False(t, changed.Valid(), "%+v", changed)
	}
	changed := b
	changed.ResearchSelection.Candidates[0].EvidenceDigest = strings.Repeat("c", 64)
	require.NotEqual(t, b, changed, "same-key request equality must include original evidence")
	raw, err := json.Marshal(Binding{ContextKind: "acquisition", ContextID: "context", ProductKey: "product", CatalogVersion: "1", PublicationID: "pub", TargetPlatform: "shein"})
	require.NoError(t, err)
	require.NotContains(t, string(raw), "ResearchSelection", "unrelated current consumers do not gain a research variant")
}
