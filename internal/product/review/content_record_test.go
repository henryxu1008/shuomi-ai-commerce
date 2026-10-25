package review

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	reviewproposal "task-processor/internal/product/review/proposal"
)

func TestCandidateWireAndReviewHashesStayStableAcrossOwnerExtraction(t *testing.T) {
	input := CandidateInput{Base: CreateInput{ProductKey: "product", BaseVersion: 1}, PublicationID: "publication", PolicyVersion: "title-review-v1",
		Candidate: reviewproposal.Candidate{Changes: []reviewproposal.FieldChange{{Field: "title", Value: "new", EvidenceIDs: []string{"raw"}}}}}
	const wire = `{"Base":{"product_key":"product","base_version":1},"PublicationID":"publication","PolicyVersion":"title-review-v1","Candidate":{"Changes":[{"Field":"title","Value":"new","EvidenceIDs":["raw"]}],"Warnings":null,"Rejections":null}}`
	raw, err := json.Marshal(input)
	require.NoError(t, err)
	require.Equal(t, wire, string(raw), "zero selection/provenance must keep the original command encoding")
	op, err := operation(Scope{Org: "org", Actor: "owner"}, "candidate", "create_candidate", "", &input)
	require.NoError(t, err)
	require.Equal(t, "b6a2ede8a75cc54bea91958d3d0d9f0a38e3e9b0ec4d2e17d2a2787f717be743", op.Fingerprint)
	record := Record{Input: input.Base, BasePublicationID: "publication", Before: "old", Title: "new", Policy: input.PolicyVersion, Revision: 1,
		Original: reviewproposal.Proposal{Changes: input.Candidate.Changes, Quality: reviewproposal.QualityScore{Overall: 100, EvidenceCoverage: 100, RequiredFieldCoverage: 100}, Validation: reviewproposal.ValidationResult{Valid: true, EvaluatedChanges: 1}}}
	require.Equal(t, "a688e751ceb2835430cb25047cf53fba59664c2ba67be9b2cb7d83d61ce516c6", proposalDigest(record))
}

func contentRecordFixture(t *testing.T) Record {
	t.Helper()
	selection, err := reviewproposal.NewContentSelection("en-US", []string{"title", "description"})
	require.NoError(t, err)
	return Record{ID: "00000000-0000-0000-0000-000000000001", Org: "org", Owner: "owner", Input: CreateInput{ProductKey: "product", BaseVersion: 1},
		Execution: &ExecutionProvenance{ContentSelection: selection, Scope: agent.Scope{OrganizationID: "org", ActorID: "owner"},
			RunID: "00000000-0000-0000-0000-000000000002", RequestKey: "00000000-0000-0000-0000-000000000003",
			Binding:       agent.Binding{ContextKind: "acquisition", ContextID: "context", ProductKey: "product", CatalogVersion: "1", PublicationID: "source-publication", TargetPlatform: "product"},
			Configuration: agent.ConfigurationSnapshotRef{Kind: "agent-configuration-v1", ID: "00000000-0000-0000-0000-000000000004", Digest: strings.Repeat("a", 64)},
			Mode:          agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: "00000000-0000-0000-0000-000000000005", Digest: strings.Repeat("b", 64)}},
		Policy: reviewproposal.ContentPolicyVersion, BasePublicationID: "source-publication", State: "pending", Revision: 1,
		Content: &ContentPayload{Selection: selection, Before: []ContentValue{{Field: "title", Value: "source title"}, {Field: "description", Value: "source description"}},
			After: []ContentValue{{Field: "title", Value: "suggested title"}, {Field: "description", Value: "suggested description"}}},
		Original: reviewproposal.Proposal{Validation: reviewproposal.ValidationResult{Valid: true, EvaluatedChanges: 2}, Changes: []reviewproposal.FieldChange{{Field: "title", Value: "suggested title", EvidenceIDs: []string{"source"}}, {Field: "description", Value: "suggested description", EvidenceIDs: []string{"source"}}}}}
}

func TestContentRecordHasTypedDecisionAndWireInsteadOfEmptyTitleApproval(t *testing.T) {
	r := contentRecordFixture(t)
	require.NoError(t, r.Decide("owner", DecisionInput{Action: "accept", ExpectedRevision: 1}))
	require.Equal(t, "accepted", r.State)
	require.Empty(t, r.History)
	require.Len(t, r.Content.Decisions, 1)
	require.Equal(t, r.Content.After, r.Content.Decisions[0].After)
	require.Equal(t, "", r.Title)
	require.NoError(t, r.Decide("owner", DecisionInput{Action: "edit", ExpectedRevision: 2, Content: []ContentValue{{Field: "title", Value: "edited title"}, {Field: "description", Value: "edited description"}}}))
	require.Equal(t, "pending", r.State)
	require.Equal(t, "suggested title", r.Original.Changes[0].Value)
	require.Equal(t, "edited title", r.Content.After[0].Value)
	require.Equal(t, "suggested title", r.Content.Decisions[1].Before[0].Value)
	v := r.View()
	require.Equal(t, r.Content, v.Content)
	v.Content.After[0].Value = "cannot mutate stored record"
	require.Equal(t, "edited title", r.Content.After[0].Value)
	wire, err := json.Marshal(v)
	require.NoError(t, err)
	require.Contains(t, string(wire), `"content":`)
	require.NotContains(t, string(wire), `"policy":"title-review-v1"`)
}

func TestContentRecordRejectsExpandedOrPartialEditsWithoutMutation(t *testing.T) {
	for _, input := range []DecisionInput{
		{Action: "edit", ExpectedRevision: 1, Title: "title-only bypass"},
		{Action: "edit", ExpectedRevision: 1, Content: []ContentValue{{Field: "title", Value: "partial"}}},
		{Action: "edit", ExpectedRevision: 1, Content: []ContentValue{{Field: "title", Value: "changed"}, {Field: "price", Value: "100"}}},
		{Action: "accept", ExpectedRevision: 1, Content: []ContentValue{{Field: "title", Value: "changed"}}},
	} {
		r := contentRecordFixture(t)
		before, err := json.Marshal(r)
		require.NoError(t, err)
		require.Error(t, r.Decide("owner", input))
		after, err := json.Marshal(r)
		require.NoError(t, err)
		require.Equal(t, before, after)
	}
	r := Record{Policy: "title-review-v1", Revision: 1, State: "pending", Title: "title"}
	require.Error(t, r.Decide("owner", DecisionInput{Action: "edit", ExpectedRevision: 1, Title: "changed", Content: []ContentValue{{Field: "description", Value: "not a title field"}}}))
}

func TestContentPayloadWireRejectsPolicyOrSelectionConfusion(t *testing.T) {
	r := contentRecordFixture(t)
	// Wire validation does not trust an empty legacy title projection.
	require.NoError(t, ValidateView(r.View()))
	v := r.View()
	v.Content = nil
	require.Error(t, ValidateView(v))
	v = r.View()
	v.Policy = "title-review-v1"
	require.Error(t, ValidateView(v))
	v = r.View()
	v.Content.Selection.Description = false
	require.Error(t, ValidateView(v))
	v = r.View()
	v.Content.After[1].Field = "price"
	require.Error(t, ValidateView(v))
}

func TestContentRecordStorageReservesWholeDecisionReceiptAndNativeProof(t *testing.T) {
	r := contentRecordFixture(t)
	// Current content fits; a future accept must repeat the entire selected
	// before/after values, never fail only after a Product publication.
	r.Content.After[1].Value = strings.Repeat("x", 28000)
	r.Original.Changes[1].Value = r.Content.After[1].Value
	raw, err := json.Marshal(r)
	require.NoError(t, err)
	require.Less(t, len(raw), MaxRecordBytes)
	require.ErrorIs(t, r.ValidateStorage(), ErrTooLarge)
	r.Content.After[1].Value = strings.Repeat("x", 12000)
	r.Original.Changes[1].Value = r.Content.After[1].Value
	require.NoError(t, r.ValidateStorage())
	require.NoError(t, r.Decide(strings.Repeat("<", 128), DecisionInput{Action: "accept", ExpectedRevision: 1}))
	require.NoError(t, r.ValidateStorage())
}

func TestContentNativeProofCannotClaimTitleOnlyOrAutoApproveMultipleFields(t *testing.T) {
	r := contentRecordFixture(t)
	proof := PolicyProvenance{Execution: *r.Execution, Identity: agent.ExecutionIdentity{OrganizationID: "org", ActorID: "owner", MemberID: "member"},
		Mode: agent.ModeFullAccess, ModeRevision: "1", PolicyVersion: agent.ExecutionModePolicyVersion, ActionVersion: ContentPolicyAdoptionVersion, ProposalDigest: strings.Repeat("c", 64)}
	require.True(t, proof.Valid())
	changed := proof
	changed.ActionVersion = TitlePolicyAdoptionVersion
	require.False(t, changed.Valid())
	changed = proof
	changed.Mode = agent.ModeAutoApprove
	require.False(t, changed.Valid())
	changed.Execution.ContentSelection.Description = false
	require.True(t, changed.Valid())
	r.State = "applied"
	proof.ProposalDigest = proposalDigest(r)
	r.PolicyAdoption = &proof
	r.Receipt = &Receipt{ProposalID: r.ID, Revision: r.Revision, ProductVersion: 2, PublicationID: "review:publication", Actor: "owner", At: time.Now().UTC()}
	require.NoError(t, ValidateStoredRecord(r))
	require.NoError(t, ValidateView(r.View()))
	r.Content.Selection.TargetLanguage = "de"
	require.Error(t, ValidateStoredRecord(r))
	require.Error(t, ValidateView(r.View()))
}

func TestContentStoredHistoryMustExplainFinalValuesAndState(t *testing.T) {
	r := contentRecordFixture(t)
	require.NoError(t, r.Decide("owner", DecisionInput{Action: "edit", ExpectedRevision: 1, Content: []ContentValue{{"title", "edited title"}, {"description", "edited description"}}}))
	require.NoError(t, r.Decide("owner", DecisionInput{Action: "accept", ExpectedRevision: 2}))
	require.NoError(t, ValidateStoredRecord(r))
	for _, change := range []func(*Record){
		func(r *Record) { r.Content.Decisions[1].Before[0].Value = "unrecorded intervening edit" },
		func(r *Record) { r.Content.Decisions = nil },
		func(r *Record) { r.State = "pending" },
		func(r *Record) { r.Content.After[0].Value = "unrecorded final edit" },
		func(r *Record) { r.Content.Decisions[0].Before[0].Value = "not the original candidate" },
		func(r *Record) { r.Content.Decisions[0].Revision = 1 },
	} {
		raw, err := json.Marshal(r)
		require.NoError(t, err)
		var bad Record
		require.NoError(t, json.Unmarshal(raw, &bad))
		change(&bad)
		require.ErrorIs(t, ValidateStoredRecord(bad), ErrUnavailable)
	}
}

func TestContentNativeProofBindsFullCandidateAndSelection(t *testing.T) {
	r := contentRecordFixture(t)
	r.State = "applied"
	r.PolicyAdoption = &PolicyProvenance{Execution: *r.Execution, Identity: agent.ExecutionIdentity{OrganizationID: "org", ActorID: "owner", MemberID: "member"}, Mode: agent.ModeFullAccess, ModeRevision: "1", PolicyVersion: agent.ExecutionModePolicyVersion, ActionVersion: ContentPolicyAdoptionVersion, ProposalDigest: proposalDigest(r), RequiresHuman: true}
	r.Receipt = &Receipt{ProposalID: r.ID, Revision: r.Revision, ProductVersion: 2, PublicationID: "review:publication", Actor: "owner", At: time.Now().UTC()}
	require.NoError(t, ValidateStoredRecord(r))
	r.Content.After[1].Value = "modified after policy confirmation"
	require.ErrorIs(t, ValidateStoredRecord(r), ErrUnavailable)
}

func TestContentNativeAdoptionRejectsDiagnosticsEvenWhenFieldValidationPassed(t *testing.T) {
	for _, rejection := range []bool{false, true} {
		r := contentRecordFixture(t)
		warning := reviewproposal.Warning{Code: "missing_material", Message: "material specification is not confirmed"}
		if rejection {
			r.Original.Rejections = []reviewproposal.Rejection{{Code: warning.Code, Message: warning.Message}}
		} else {
			r.Original.Warnings = []reviewproposal.Warning{warning}
		}
		require.True(t, r.Original.Validation.Valid, "a structurally valid fieldset is not permission to ignore diagnostic facts")
		require.False(t, originalNativeCandidate(r))
	}
}
