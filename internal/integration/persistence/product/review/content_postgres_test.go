package reviewpersistence

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/agent"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/enrichment"
	"task-processor/internal/product/review"
	"task-processor/internal/product/sourcing"
)

// All source/candidate values are controlled fixtures; no model is invoked.
type contentSourceFixture struct {
	value  sourcing.PersistedPublication
	denied bool
}

func (s *contentSourceFixture) AuthorizeRead(ctx context.Context) (context.Context, error) {
	if s.denied {
		return ctx, sourcing.ErrPublicationForbidden
	}
	return ctx, nil
}
func (s *contentSourceFixture) Read(_ context.Context, id string) (sourcing.PersistedPublication, error) {
	if s.denied {
		return sourcing.PersistedPublication{}, sourcing.ErrPublicationForbidden
	}
	if id != s.value.Receipt.PublicationID {
		return sourcing.PersistedPublication{}, sourcing.ErrSourcePublicationNotFound
	}
	return s.value, nil
}

func contentPostgresFixture(t *testing.T) (*review.Service, *Repository, catalog.CompleteSnapshotReader, *gorm.DB, context.Context, catalog.PublishedSnapshot, review.CandidateInput) {
	t.Helper()
	repository, db := postgresReviewRepositoryFixture(t)
	require.NoError(t, catalogstore.AutoMigrate(db))
	writer, err := catalogstore.NewRepository(db)
	require.NoError(t, err)
	publisher, err := catalog.NewPublisher(writer)
	require.NoError(t, err)
	snapshots, err := catalogstore.NewBoundedSnapshotReader(db, 2<<20)
	require.NoError(t, err)
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", UserID: "owner", Roles: []string{"listingkit_admin"}, TokenExpiresAt: time.Now().Add(time.Hour)})
	base, err := publisher.Publish(ctx, catalog.PublishRequest{Identity: catalog.SnapshotIdentity{TenantID: "org", ProductKey: "product"}, PublicationID: "source-publication", Snapshot: catalog.ProductSnapshot{
		Title: "source title", Description: "source description", SellingPoints: []string{"source point"}, SEOKeywords: []string{"source keyword"}, Brand: "preserved brand",
		Images: []catalog.Image{{URL: "https://fixture.invalid/source.png"}}, Variants: []catalog.Variant{{SKU: "preserved-sku", Stock: 7, Price: &catalog.Price{Currency: "CNY", Amount: 12.5}}}}})
	require.NoError(t, err)
	source := &contentSourceFixture{value: sourcing.PersistedPublication{Snapshot: base.Snapshot, Receipt: sourcing.PublicationReceipt{OrganizationID: "org", ActorID: "owner", ProductKey: "product", CatalogVersion: base.Version, PublicationID: base.PublicationID, CatalogPublicationID: base.PublicationID},
		Envelope: sourcing.SourceEnvelope{Identity: sourcing.SourceIdentity{SourceType: sourcing.SourceTypeManualImport, SourcePlatform: "fixture", SourceID: "source", SourceVersion: "v1"}, RawReference: sourcing.RawSourceReference{ReferenceID: "source"}}}}
	repository.sourceReaderFactory = func(*gorm.DB) (review.SourcePublicationReader, error) { return source, nil }
	auth, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	service, err := review.NewCandidateService(snapshots, source, repository, auth)
	require.NoError(t, err)
	selection, err := enrichment.NewContentSelection("en-US", []string{"title", "description", "selling_points", "seo_keywords"})
	require.NoError(t, err)
	input := review.CandidateInput{ContentSelection: selection, Base: review.CreateInput{ProductKey: "product", BaseVersion: base.Version}, PublicationID: base.PublicationID, PolicyVersion: enrichment.ContentPolicyVersion,
		Execution: &review.ExecutionProvenance{ContentSelection: selection, Scope: agent.Scope{OrganizationID: "org", ActorID: "owner"}, RunID: uuid.NewString(), RequestKey: uuid.NewString(),
			Binding:       agent.Binding{ContextKind: "acquisition", ContextID: "context", ProductKey: "product", CatalogVersion: "1", PublicationID: base.PublicationID, TargetPlatform: "product"},
			Configuration: agent.ConfigurationSnapshotRef{Kind: "agent-configuration-v1", ID: uuid.NewString(), Digest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			Mode:          agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: uuid.NewString(), Digest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},
		Candidate: enrichment.Candidate{Changes: []enrichment.FieldChange{{Field: "title", Value: "translated title", EvidenceIDs: []string{"source"}}, {Field: "description", Value: "translated description", EvidenceIDs: []string{"source"}}, {Field: "selling_points", Value: `["translated point"]`, EvidenceIDs: []string{"source"}}, {Field: "seo_keywords", Value: `["translated keyword"]`, EvidenceIDs: []string{"source"}}}}}
	return service, repository, snapshots, db, ctx, base, input
}

func TestPostgresContentHumanApplyIsAtomicAndPreservesOtherFacts(t *testing.T) {
	service, repository, snapshots, _, ctx, base, input := contentPostgresFixture(t)
	proposal, err := service.CreateFromCandidate(ctx, "agent:"+input.Execution.RunID, input)
	require.NoError(t, err)
	require.Len(t, proposal.Content.Before, 4)
	require.Equal(t, "pending", proposal.State)
	accepted, err := service.Decide(ctx, "content-accept", proposal.ID, review.DecisionInput{Action: "accept", ExpectedRevision: proposal.Revision})
	require.NoError(t, err)
	view, err := service.Apply(ctx, "content-apply", proposal.ID, review.ApplyInput{ExpectedRevision: accepted.Revision})
	require.NoError(t, err)
	require.Equal(t, "applied", view.State)
	current, err := snapshots.GetCurrentSnapshot(ctx, base.Identity)
	require.NoError(t, err)
	require.EqualValues(t, 2, current.Version)
	want := base.Snapshot
	want.Title, want.Description = "translated title", "translated description"
	want.SellingPoints, want.SEOKeywords = []string{"translated point"}, []string{"translated keyword"}
	require.Equal(t, want, current.Snapshot)
	replayed, err := service.Apply(ctx, "content-apply", proposal.ID, review.ApplyInput{ExpectedRevision: accepted.Revision})
	require.NoError(t, err)
	require.Equal(t, view, replayed)
	stored, err := repository.Read(ctx, review.Scope{Org: "org", Actor: "owner"}, view.ID)
	require.NoError(t, err)
	require.Equal(t, view.Receipt, stored.Receipt)
}

func TestPostgresContentCompletionFailureRollsBackPublicationAndReview(t *testing.T) {
	service, repository, snapshots, db, ctx, base, input := contentPostgresFixture(t)
	proposal, err := service.CreateFromCandidate(ctx, "agent:"+input.Execution.RunID, input)
	require.NoError(t, err)
	accepted, err := service.Decide(ctx, "content-accept", proposal.ID, review.DecisionInput{Action: "accept", ExpectedRevision: proposal.Revision})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE FUNCTION reject_content_receipt() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.operation_key='content-apply-fault' THEN RAISE EXCEPTION 'controlled receipt failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_content_receipt BEFORE INSERT ON product_title_operations FOR EACH ROW EXECUTE FUNCTION reject_content_receipt()`).Error)
	_, err = service.Apply(ctx, "content-apply-fault", proposal.ID, review.ApplyInput{ExpectedRevision: accepted.Revision})
	require.Error(t, err)
	current, err := snapshots.GetCurrentSnapshot(ctx, base.Identity)
	require.NoError(t, err)
	require.Equal(t, base, current)
	stored, err := repository.Read(ctx, review.Scope{Org: "org", Actor: "owner"}, proposal.ID)
	require.NoError(t, err)
	require.Equal(t, "accepted", stored.State)
	require.Nil(t, stored.Receipt)
}

func contentPolicyFixture(t *testing.T, repository *Repository, snapshots catalog.VersionedSnapshotReader, mode *agent.ExecutionMode) (*review.PolicyAdoptionService, review.ExecutionPolicyResolver) {
	t.Helper()
	source, err := repository.sourceReaderFactory(repository.db)
	require.NoError(t, err)
	auth, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	resolve := func(_ context.Context, execution review.ExecutionProvenance, _ enrichment.Candidate, _ bool) (review.PolicyProvenance, error) {
		return review.PolicyProvenance{Execution: execution, Identity: agent.ExecutionIdentity{OrganizationID: "org", ActorID: "owner", MemberID: "member"}, Mode: *mode, ModeRevision: "1", PolicyVersion: agent.ExecutionModePolicyVersion, ActionVersion: review.ContentPolicyAdoptionVersion}, nil
	}
	adopter, err := review.NewPolicyAdoptionService(snapshots, source.(review.SourcePublicationGateway), repository, auth, func(context.Context) (review.CandidateExecutionScope, error) {
		return review.CandidateExecutionScope{OrganizationID: "org", ActorID: "owner", MemberID: "member"}, nil
	}, resolve)
	require.NoError(t, err)
	return adopter, resolve
}

func TestPostgresContentPolicyUsesWholeCandidateAndFourModes(t *testing.T) {
	for _, test := range []struct {
		name      string
		mode      agent.ExecutionMode
		titleOnly bool
		outcome   agent.PolicyOutcome
	}{
		{"full all fields", agent.ModeFullAccess, false, agent.PolicyAllow},
		{"auto all fields needs human", agent.ModeAutoApprove, false, agent.PolicyHumanRequired},
		{"auto title only", agent.ModeAutoApprove, true, agent.PolicyAllow},
		{"default needs human", agent.ModeDefault, false, agent.PolicyHumanRequired},
		{"read only denied", agent.ModeReadOnly, false, agent.PolicyDeny},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, repository, snapshots, _, ctx, base, input := contentPostgresFixture(t)
			if test.titleOnly {
				selection, err := enrichment.NewContentSelection("en-US", []string{"title"})
				require.NoError(t, err)
				input.ContentSelection, input.Execution.ContentSelection = selection, selection
				input.Candidate.Changes = input.Candidate.Changes[:1]
			}
			proposal, err := service.CreateFromCandidate(ctx, "agent:"+input.Execution.RunID, input)
			require.NoError(t, err)
			mode := test.mode
			adopter, resolve := contentPolicyFixture(t, repository, snapshots, &mode)
			command := review.PolicyAdoptionCommand{ProposalID: proposal.ID, ExpectedRevision: proposal.Revision, Execution: *input.Execution}
			view, decision, err := adopter.Adopt(context.Background(), "content-native", command)
			require.NoError(t, err)
			require.Equal(t, test.outcome, decision.Outcome)
			current, err := snapshots.GetCurrentSnapshot(ctx, base.Identity)
			require.NoError(t, err)
			if test.outcome != agent.PolicyAllow {
				require.Equal(t, "pending", view.State)
				require.Equal(t, base, current, "automatic approval cannot apply only the title part")
				if mode == agent.ModeReadOnly {
					service.WithExecutionPolicy(resolve)
					_, err = service.Decide(ctx, "manual-readonly", proposal.ID, review.DecisionInput{Action: "accept", ExpectedRevision: proposal.Revision})
					require.ErrorIs(t, err, review.ErrForbidden)
				}
				return
			}
			require.Equal(t, "applied", view.State)
			require.Empty(t, view.History)
			require.Empty(t, view.Content.Decisions, "native policy cannot manufacture human approval")
			require.Equal(t, review.ContentPolicyAdoptionVersion, view.PolicyAdoption.ActionVersion)
			require.Equal(t, input.ContentSelection, view.PolicyAdoption.Execution.ContentSelection)
			require.Equal(t, !test.titleOnly, view.PolicyAdoption.RequiresHuman)
			require.EqualValues(t, 2, current.Version)
			require.Equal(t, "translated title", current.Snapshot.Title)
			if !test.titleOnly {
				require.Equal(t, "translated description", current.Snapshot.Description)
				require.Equal(t, []string{"translated point"}, current.Snapshot.SellingPoints)
				require.Equal(t, []string{"translated keyword"}, current.Snapshot.SEOKeywords)
			}
			mode = agent.ModeReadOnly // old receipt readback causes no new policy effect
			replayed, replayDecision, err := adopter.Adopt(context.Background(), "content-native", command)
			require.NoError(t, err)
			require.Equal(t, agent.PolicyAllow, replayDecision.Outcome)
			require.Equal(t, view, replayed)
			_, _, err = adopter.Adopt(context.Background(), "different-key", command)
			require.ErrorIs(t, err, review.ErrConflict)
		})
	}
}

func TestPostgresContentReplayKeepsCurrentSourceAuthorization(t *testing.T) {
	for _, native := range []bool{false, true} {
		t.Run(map[bool]string{false: "human", true: "native"}[native], func(t *testing.T) {
			service, repository, snapshots, _, ctx, base, input := contentPostgresFixture(t)
			key := "agent:" + input.Execution.RunID
			proposal, err := service.CreateFromCandidate(ctx, key, input)
			require.NoError(t, err)
			var replay func() error
			if native {
				mode := agent.ModeFullAccess
				adopter, _ := contentPolicyFixture(t, repository, snapshots, &mode)
				command := review.PolicyAdoptionCommand{ProposalID: proposal.ID, ExpectedRevision: proposal.Revision, Execution: *input.Execution}
				_, _, err = adopter.Adopt(context.Background(), "native-replay", command)
				require.NoError(t, err)
				replay = func() error { _, _, err := adopter.Adopt(context.Background(), "native-replay", command); return err }
			} else {
				accepted, err := service.Decide(ctx, "content-accept", proposal.ID, review.DecisionInput{Action: "accept", ExpectedRevision: proposal.Revision})
				require.NoError(t, err)
				command := review.ApplyInput{ExpectedRevision: accepted.Revision}
				_, err = service.Apply(ctx, "human-replay", proposal.ID, command)
				require.NoError(t, err)
				replay = func() error { _, err := service.Apply(ctx, "human-replay", proposal.ID, command); return err }
			}
			source, err := repository.sourceReaderFactory(repository.db)
			require.NoError(t, err)
			source.(*contentSourceFixture).denied = true
			require.ErrorIs(t, replay(), review.ErrForbidden, "a committed receipt does not grant revoked source access")
			_, err = service.CreateFromCandidate(ctx, key, input)
			require.ErrorIs(t, err, review.ErrForbidden)
			current, err := snapshots.GetCurrentSnapshot(ctx, base.Identity)
			require.NoError(t, err)
			require.EqualValues(t, 2, current.Version, "replay never creates a publication")
		})
	}
}

func TestPostgresContentAndTitleLineagePreservesSourceAndRejectsOtherChanges(t *testing.T) {
	service, repository, snapshots, _, ctx, base, input := contentPostgresFixture(t)
	proposal, err := service.CreateFromCandidate(ctx, "agent:"+input.Execution.RunID, input)
	require.NoError(t, err)
	accepted, err := service.Decide(ctx, "content-accept", proposal.ID, review.DecisionInput{Action: "accept", ExpectedRevision: 1})
	require.NoError(t, err)
	applied, err := service.Apply(ctx, "content-apply", proposal.ID, review.ApplyInput{ExpectedRevision: accepted.Revision})
	require.NoError(t, err)
	title, err := service.CreateFromCandidate(ctx, "next-title", review.CandidateInput{Base: review.CreateInput{ProductKey: "product", BaseVersion: applied.Receipt.ProductVersion}, PublicationID: applied.Receipt.PublicationID, PolicyVersion: "title-review-v1", Candidate: enrichment.Candidate{Changes: []enrichment.FieldChange{{Field: "title", Value: "next reviewed title", EvidenceIDs: []string{"source"}}}}})
	require.NoError(t, err, "a title consumer must read genuine content lineage")
	accepted, err = service.Decide(ctx, "title-accept", title.ID, review.DecisionInput{Action: "accept", ExpectedRevision: 1})
	require.NoError(t, err)
	applied, err = service.Apply(ctx, "title-apply", title.ID, review.ApplyInput{ExpectedRevision: accepted.Revision})
	require.NoError(t, err)
	input.Base.BaseVersion, input.PublicationID = applied.Receipt.ProductVersion, applied.Receipt.PublicationID
	input.Execution.Binding.CatalogVersion = strconv.FormatUint(input.Base.BaseVersion, 10)
	input.Execution.Binding.PublicationID = input.PublicationID
	input.Execution.RunID, input.Execution.RequestKey = uuid.NewString(), uuid.NewString()
	proposal, err = service.CreateFromCandidate(ctx, "agent:"+input.Execution.RunID, input)
	require.NoError(t, err)
	accepted, err = service.Decide(ctx, "third-accept", proposal.ID, review.DecisionInput{Action: "accept", ExpectedRevision: 1})
	require.NoError(t, err)
	_, err = service.Apply(ctx, "third-apply", proposal.ID, review.ApplyInput{ExpectedRevision: accepted.Revision})
	require.NoError(t, err)
	current, err := snapshots.GetCurrentSnapshot(ctx, base.Identity)
	require.NoError(t, err)
	source, err := repository.sourceReaderFactory(repository.db)
	require.NoError(t, err)
	lineage, err := review.ResolveAppliedSource(ctx, review.Scope{Org: "org", Actor: "owner"}, current, snapshots, source, repository)
	require.NoError(t, err)
	require.Equal(t, base, lineage.Original)
	require.Equal(t, base.Snapshot, lineage.Source.Snapshot)
	require.Len(t, lineage.Applied, 3)
	require.Equal(t, enrichment.ContentPolicyVersion, lineage.Applied[0].Policy)
	require.Equal(t, input.ContentSelection, lineage.Applied[0].ContentSelection)
	require.Equal(t, "title-review-v1", lineage.Applied[1].Policy)
	require.True(t, lineage.Applied[1].ContentSelection.Absent())
	for _, field := range []string{"brand", "price", "image"} {
		t.Run(field, func(t *testing.T) {
			changed, err := catalog.CloneProductSnapshot(current.Snapshot)
			require.NoError(t, err)
			switch field {
			case "brand":
				changed.Brand = "unreviewed brand"
			case "price":
				changed.Variants[0].Price.Amount = 99
			case "image":
				changed.Images[0].URL = "https://fixture.invalid/unreviewed.png"
			}
			bad := current
			bad.Snapshot = changed
			_, err = review.ResolveAppliedSource(ctx, review.Scope{Org: "org", Actor: "owner"}, bad, snapshots, source, repository)
			require.ErrorIs(t, err, review.ErrConflict)
		})
	}
}

func TestPostgresContentNativeRejectsEditsAndPolicyChangeBeforeCommit(t *testing.T) {
	for _, edited := range []bool{true, false} {
		t.Run(map[bool]string{true: "edited candidate", false: "policy changed before commit"}[edited], func(t *testing.T) {
			service, repository, snapshots, _, ctx, base, input := contentPostgresFixture(t)
			proposal, err := service.CreateFromCandidate(ctx, "agent:"+input.Execution.RunID, input)
			require.NoError(t, err)
			if edited {
				values := append([]review.ContentValue(nil), proposal.Content.After...)
				values[1].Value = "human edited description"
				proposal, err = service.Decide(ctx, "manual-edit", proposal.ID, review.DecisionInput{Action: "edit", ExpectedRevision: 1, Content: values})
				require.NoError(t, err)
			}
			mode := agent.ModeFullAccess
			_, resolve := contentPolicyFixture(t, repository, snapshots, &mode)
			calls := 0
			changed := func(ctx context.Context, execution review.ExecutionProvenance, candidate enrichment.Candidate, human bool) (review.PolicyProvenance, error) {
				proof, err := resolve(ctx, execution, candidate, human)
				calls++
				if calls == 2 {
					proof.ModeRevision = "2"
				}
				return proof, err
			}
			source, err := repository.sourceReaderFactory(repository.db)
			require.NoError(t, err)
			auth, err := authz.NewListingKitAuthorizer(nil, nil)
			require.NoError(t, err)
			adopter, err := review.NewPolicyAdoptionService(snapshots, source.(review.SourcePublicationGateway), repository, auth, func(context.Context) (review.CandidateExecutionScope, error) {
				return review.CandidateExecutionScope{OrganizationID: "org", ActorID: "owner", MemberID: "member"}, nil
			}, changed)
			require.NoError(t, err)
			_, _, err = adopter.Adopt(context.Background(), "native-conflict", review.PolicyAdoptionCommand{ProposalID: proposal.ID, ExpectedRevision: proposal.Revision, Execution: *input.Execution})
			require.ErrorIs(t, err, review.ErrConflict)
			current, err := snapshots.GetCurrentSnapshot(ctx, base.Identity)
			require.NoError(t, err)
			require.Equal(t, base, current)
			stored, err := repository.Read(ctx, review.Scope{Org: "org", Actor: "owner"}, proposal.ID)
			require.NoError(t, err)
			require.Equal(t, "pending", stored.State)
			require.Nil(t, stored.Receipt)
		})
	}
}
