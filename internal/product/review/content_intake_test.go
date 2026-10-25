package review

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/product/catalog"
	reviewproposal "task-processor/internal/product/review/proposal"
	"task-processor/internal/product/sourcing"
)

func TestContentIntakeRevalidatesFullSelectionAndExactSource(t *testing.T) {
	r := contentRecordFixture(t)
	base := catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: r.Org, ProductKey: r.Input.ProductKey}, Version: 1,
		PublicationID: r.BasePublicationID, Snapshot: catalog.ProductSnapshot{Title: "source title", Description: "source description", Brand: "original brand"}}
	source := &sourcePublicationReaderStub{value: sourcing.PersistedPublication{Snapshot: base.Snapshot,
		Receipt: sourcing.PublicationReceipt{OrganizationID: r.Org, ProductKey: r.Input.ProductKey, CatalogVersion: 1, PublicationID: base.PublicationID, CatalogPublicationID: base.PublicationID},
		Envelope: sourcing.SourceEnvelope{Identity: sourcing.SourceIdentity{SourceType: sourcing.SourceTypeManualImport, SourcePlatform: "fixture", SourceID: "source", SourceVersion: "v1"},
			RawReference: sourcing.RawSourceReference{ReferenceID: "source"}}}}
	auth, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	store := &candidateAttemptStore{}
	service, err := NewCandidateService(exactCatalogReaderStub{published: base}, source, store, auth)
	require.NoError(t, err)
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: r.Org, EffectiveOrganizationID: r.Org, UserID: r.Owner, Roles: []string{"listingkit_admin"}, TokenExpiresAt: time.Now().Add(time.Hour)})
	input := CandidateInput{ContentSelection: r.Content.Selection, Execution: r.Execution, Base: r.Input, PublicationID: r.BasePublicationID,
		PolicyVersion: reviewproposal.ContentPolicyVersion, Candidate: reviewproposal.Candidate{Changes: r.Original.Changes}}
	proposal, err := service.ValidateCandidate(ctx, input)
	require.NoError(t, err)
	require.True(t, proposal.Validation.Valid)
	require.Len(t, proposal.Changes, 2)
	pure := ContentValidationInput{Selection: input.ContentSelection, Base: input.Base, PublicationID: input.PublicationID, Candidate: input.Candidate}
	validated, err := service.ValidateContentCandidate(ctx, pure)
	require.NoError(t, err, "pure runtime validation must not fabricate an execution RunID")
	require.Equal(t, proposal, validated)
	wrong := pure
	wrong.Candidate.Changes = append([]reviewproposal.FieldChange(nil), pure.Candidate.Changes...)
	wrong.Candidate.Changes[1].Field = "price"
	_, err = service.ValidateContentCandidate(ctx, wrong)
	require.Error(t, err)
	_, attempted, err := service.CreateFromCandidateWithTransactionAttempt(ctx, "agent:content-run", input)
	require.ErrorIs(t, err, ErrUnavailable)
	require.True(t, attempted, "valid content must reach the existing Review transaction")
	for _, mutate := range []func(*CandidateInput){
		func(in *CandidateInput) { in.ContentSelection.TargetLanguage = "de" },
		func(in *CandidateInput) { in.Execution = nil },
		func(in *CandidateInput) { in.Candidate.Changes = in.Candidate.Changes[:1] },
		func(in *CandidateInput) { in.Candidate.Changes[1].Field = "price" },
		func(in *CandidateInput) { in.Candidate.Changes[1].EvidenceIDs = []string{"unobserved-source"} },
		func(in *CandidateInput) { in.PublicationID = "different-source" },
		func(in *CandidateInput) { in.PolicyVersion = "title-review-v1" },
	} {
		changed := input
		changed.Candidate.Changes = append([]reviewproposal.FieldChange(nil), input.Candidate.Changes...)
		mutate(&changed)
		_, err := service.ValidateCandidate(ctx, changed)
		require.Error(t, err)
		_, attempted, err := service.CreateFromCandidateWithTransactionAttempt(ctx, "agent:bad-content-run", changed)
		require.Error(t, err)
		require.False(t, attempted)
	}
}
