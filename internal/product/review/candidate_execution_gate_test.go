package review

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/product/catalog"
	reviewproposal "task-processor/internal/product/review/proposal"
	"task-processor/internal/product/sourcing"
)

type candidateGateStore struct {
	Store
	Tx
	reader    catalog.VersionedSnapshotReader
	source    SourcePublicationReader
	saved     int
	completed int
	replay    *View
	preflight bool
}

func (s *candidateGateStore) Preflight(context.Context, Operation) (View, bool, error) {
	if s.preflight && s.replay != nil {
		return *s.replay, true, nil
	}
	return View{}, false, nil
}
func (s *candidateGateStore) Run(_ context.Context, _ Operation, fn func(Tx) (View, error)) (View, error) {
	return fn(s)
}
func (s *candidateGateStore) Replay() (View, bool, error) {
	if s.replay != nil {
		return *s.replay, true, nil
	}
	return View{}, false, nil
}
func (s *candidateGateStore) Reader() catalog.VersionedSnapshotReader { return s.reader }
func (s *candidateGateStore) SourceReader() SourcePublicationReader   { return s.source }
func (s *candidateGateStore) Save(Record) error                       { s.saved++; return nil }
func (s *candidateGateStore) Complete(View) error                     { s.completed++; return nil }

func candidateGateFixture(t *testing.T, content bool) (*Service, *candidateGateStore, context.Context, CandidateInput) {
	t.Helper()
	r := contentRecordFixture(t)
	base := catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: r.Org, ProductKey: r.Input.ProductKey}, Version: 1,
		PublicationID: r.BasePublicationID, Snapshot: catalog.ProductSnapshot{Title: "source title", Description: "source description", Brand: "original brand"}}
	source := &sourcePublicationReaderStub{value: sourcing.PersistedPublication{Snapshot: base.Snapshot,
		Receipt:  sourcing.PublicationReceipt{OrganizationID: r.Org, ProductKey: r.Input.ProductKey, CatalogVersion: 1, PublicationID: base.PublicationID, CatalogPublicationID: base.PublicationID},
		Envelope: sourcing.SourceEnvelope{Identity: sourcing.SourceIdentity{SourceType: sourcing.SourceTypeManualImport, SourcePlatform: "fixture", SourceID: "source", SourceVersion: "v1"}, RawReference: sourcing.RawSourceReference{ReferenceID: "source"}}}}
	store := &candidateGateStore{reader: exactCatalogReaderStub{published: base}, source: source}
	auth, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	service, err := NewCandidateService(store.reader, source, store, auth)
	require.NoError(t, err)
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: r.Org, EffectiveOrganizationID: r.Org, UserID: r.Owner, EffectiveMemberID: "member", Roles: []string{"listingkit_admin"}, TokenExpiresAt: time.Now().Add(time.Hour)})
	input := CandidateInput{ContentSelection: r.Content.Selection, Execution: r.Execution, Base: r.Input, PublicationID: r.BasePublicationID,
		PolicyVersion: reviewproposal.ContentPolicyVersion, Candidate: reviewproposal.Candidate{Changes: r.Original.Changes}}
	if !content {
		input.ContentSelection, input.Execution.ContentSelection = reviewproposal.ContentSelection{}, reviewproposal.ContentSelection{}
		input.PolicyVersion = "title-review-v1"
		input.Candidate.Changes = input.Candidate.Changes[:1]
	}
	return service, store, ctx, input
}

func candidateGateProof(execution ExecutionProvenance, mode agent.ExecutionMode) PolicyProvenance {
	action := TitlePolicyAdoptionVersion
	if execution.ContentSelection.Valid() {
		action = ContentPolicyAdoptionVersion
	}
	return PolicyProvenance{Execution: execution, Identity: agent.ExecutionIdentity{OrganizationID: execution.Scope.OrganizationID, ActorID: execution.Scope.ActorID, MemberID: "member"},
		Mode: mode, ModeRevision: "1", PolicyVersion: agent.ExecutionModePolicyVersion, ActionVersion: action}
}

func TestAgentCandidateNewSaveRequiresCurrentExecutionAuthority(t *testing.T) {
	for _, content := range []bool{false, true} {
		for _, test := range []struct {
			name    string
			resolve ExecutionPolicyResolver
			want    error
		}{
			{"missing resolver", nil, ErrForbidden},
			{"changed mode or configuration", func(context.Context, ExecutionProvenance, reviewproposal.Candidate, bool) (PolicyProvenance, error) {
				return PolicyProvenance{}, ErrConflict
			}, ErrConflict},
			{"revoked member", func(context.Context, ExecutionProvenance, reviewproposal.Candidate, bool) (PolicyProvenance, error) {
				return PolicyProvenance{}, ErrForbidden
			}, ErrForbidden},
			{"foreign proof", func(_ context.Context, e ExecutionProvenance, _ reviewproposal.Candidate, _ bool) (PolicyProvenance, error) {
				p := candidateGateProof(e, agent.ModeDefault)
				p.Identity.OrganizationID = "foreign"
				return p, nil
			}, ErrForbidden},
			{"wrong run", func(_ context.Context, e ExecutionProvenance, _ reviewproposal.Candidate, _ bool) (PolicyProvenance, error) {
				p := candidateGateProof(e, agent.ModeDefault)
				p.Execution.RequestKey = "other"
				return p, nil
			}, ErrForbidden},
			{"unknown mode", func(_ context.Context, e ExecutionProvenance, _ reviewproposal.Candidate, _ bool) (PolicyProvenance, error) {
				return candidateGateProof(e, "unrecognized"), nil
			}, ErrForbidden},
		} {
			t.Run(test.name+map[bool]string{false: "/title", true: "/content"}[content], func(t *testing.T) {
				service, store, ctx, input := candidateGateFixture(t, content)
				service.WithExecutionPolicy(test.resolve)
				_, err := service.CreateFromCandidate(ctx, "agent:new-run", input)
				require.ErrorIs(t, err, test.want)
				require.Zero(t, store.saved, "a stale original run cannot create a new durable suggestion")
				require.Zero(t, store.completed)
			})
		}
	}
}

func TestAgentCandidateEveryModeCanSaveSuggestionWithoutAdoptionProof(t *testing.T) {
	for _, mode := range []agent.ExecutionMode{agent.ModeDefault, agent.ModeReadOnly, agent.ModeAutoApprove, agent.ModeFullAccess} {
		t.Run(string(mode), func(t *testing.T) {
			service, store, ctx, input := candidateGateFixture(t, true)
			checks := 0
			service.WithExecutionPolicy(func(_ context.Context, e ExecutionProvenance, c reviewproposal.Candidate, human bool) (PolicyProvenance, error) {
				checks++
				require.True(t, human, "saving a suggestion does not require native adoption/admin authority")
				require.Equal(t, input.Candidate, c)
				return candidateGateProof(e, mode), nil
			})
			v, err := service.CreateFromCandidate(ctx, "agent:new-run", input)
			require.NoError(t, err)
			require.Equal(t, 1, checks)
			require.Equal(t, 1, store.saved)
			require.Equal(t, 1, store.completed)
			require.Equal(t, "pending", v.State)
			require.Nil(t, v.PolicyAdoption)
			require.Nil(t, v.Receipt)
		})
	}
}

func TestAgentCandidateCommittedReplaySkipsNewEffectModeCheck(t *testing.T) {
	for _, preflight := range []bool{false, true} {
		service, store, ctx, input := candidateGateFixture(t, true)
		committed := contentRecordFixture(t).View()
		store.replay, store.preflight = &committed, preflight
		service.WithExecutionPolicy(func(context.Context, ExecutionProvenance, reviewproposal.Candidate, bool) (PolicyProvenance, error) {
			t.Fatal("a committed receipt replay must not perform a new effect mode check")
			return PolicyProvenance{}, ErrForbidden
		})
		v, err := service.CreateFromCandidate(ctx, "agent:committed-run", input)
		require.NoError(t, err)
		require.Equal(t, committed, v)
		require.Zero(t, store.saved)
		require.Zero(t, store.completed)
	}
}
