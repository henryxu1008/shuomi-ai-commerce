package completiondraft

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/authidentity"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
)

type testAuth struct {
	scope  collection.Scope
	denied bool
}

func (a *testAuth) Authorize(context.Context, string) (collection.Scope, error) {
	if a.denied {
		return collection.Scope{}, preparation.ErrForbidden
	}
	return a.scope, nil
}

type testCollections struct{ collection.Repository }
type testAcquisition struct {
	sourcing.PublishedAcquisitionReader
}
type testSources struct {
	preparation.Repository
	source preparation.SourceItem
}

func (r testSources) ReadRetainedSource(_ context.Context, _ collection.Scope, id string) (preparation.SourceItem, error) {
	if id != r.source.ID {
		return preparation.SourceItem{}, preparation.ErrNotFound
	}
	return r.source, nil
}

type testCatalog struct{ snapshot catalog.PublishedSnapshot }

func (r testCatalog) GetSnapshot(context.Context, catalog.SnapshotIdentity, uint64) (catalog.PublishedSnapshot, error) {
	return r.snapshot, nil
}

type testRecords struct {
	Repository
	writes       int
	beforeCommit func()
	saved        Prepared
	record       Record
}

func (r *testRecords) Save(ctx context.Context, p Prepared) (Receipt, error) {
	r.saved = p
	if r.beforeCommit != nil {
		r.beforeCommit()
	}
	_, _, _, input, source, err := p.Read(ctx)
	if err != nil {
		return Receipt{}, err
	}
	r.writes++
	r.record = Record{ID: uuid.NewString(), DraftID: uuid.NewString(), Revision: 1, Source: source, Content: input.Content, Status: StatusPending, CreatedAt: time.Now()}
	return Receipt{Record: r.record}, nil
}
func (r *testRecords) Read(_ context.Context, _ collection.Scope, id string) (Record, error) {
	if id != r.record.Source.ID {
		return Record{}, preparation.ErrNotFound
	}
	return r.record, nil
}
func (r *testRecords) ReadCommand(context.Context, collection.Scope, string) (Receipt, error) {
	return Receipt{Record: r.record, Replayed: true}, nil
}

func fixture(t *testing.T) (*Service, *testAuth, *testRecords, context.Context, preparation.SourceItem) {
	t.Helper()
	a := &testAuth{scope: collection.Scope{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a"}}
	cs, err := collection.NewService(testCollections{}, a, testAcquisition{})
	require.NoError(t, err)
	source := preparation.SourceItem{ID: uuid.NewString(), PreparationID: uuid.NewString(), CollectionItemID: uuid.NewString(), CollectionRevision: 1, Source: collection.Source{ProductKey: "original-product", PublicationID: uuid.NewString(), Version: 1, Kind: "own"}}
	ps, err := preparation.NewService(testSources{source: source}, cs, a)
	require.NoError(t, err)
	sel, err := preparation.NewSourceSelector(ps, cs, testSources{source: source}, testCatalog{catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: "org-a", ProductKey: source.Source.ProductKey}, PublicationID: source.Source.PublicationID, Version: 1, Snapshot: catalog.ProductSnapshot{Title: "Original title"}}})
	require.NoError(t, err)
	r := &testRecords{}
	s, err := NewService(sel, cs, ps, a, r)
	require.NoError(t, err)
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: "org-a", EffectiveOrganizationID: "org-a", UserID: "actor-a", EffectiveMemberID: "member-a", TokenExpiresAt: time.Now().Add(time.Minute)})
	return s, a, r, ctx, source
}

func TestEmptyIncompleteContentIsSavedWithoutStoreAndOriginalSourceIsRetained(t *testing.T) {
	s, _, r, ctx, source := fixture(t)
	receipt, err := s.Save(ctx, uuid.NewString(), SaveInput{SourceID: source.ID, Content: Content{}})
	require.NoError(t, err)
	require.Equal(t, source, receipt.Record.Source)
	require.Equal(t, StatusPending, receipt.Record.Status)
	require.Equal(t, Content{}, receipt.Record.Content)
	require.Equal(t, 1, r.writes)
	read, err := s.Read(ctx, source.ID)
	require.NoError(t, err)
	require.Equal(t, receipt.Record, read)
}

func TestPreparedSaveRechecksLiveAuthorityBeforeRepositoryCommitAndReceiptRecovery(t *testing.T) {
	s, a, r, ctx, source := fixture(t)
	r.beforeCommit = func() { a.denied = true }
	_, err := s.Save(ctx, uuid.NewString(), SaveInput{SourceID: source.ID, Content: Content{Title: "candidate"}})
	require.ErrorIs(t, err, preparation.ErrForbidden)
	require.Zero(t, r.writes)
	_, err = s.ReadCommand(ctx, uuid.NewString())
	require.ErrorIs(t, err, preparation.ErrForbidden)
	_, _, _, _, _, err = (Prepared{}).Read(ctx)
	require.ErrorIs(t, err, preparation.ErrForbidden)
}

func TestPreparedSaveRejectsChangedMemberAndExpiredContext(t *testing.T) {
	s, a, r, ctx, source := fixture(t)
	r.beforeCommit = func() { a.scope.MemberID = "member-b" }
	_, err := s.Save(ctx, uuid.NewString(), SaveInput{SourceID: source.ID})
	require.ErrorIs(t, err, preparation.ErrForbidden)
	expired := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: "org-a", EffectiveOrganizationID: "org-a", UserID: "actor-a", EffectiveMemberID: "member-a", TokenExpiresAt: time.Now().Add(-time.Second)})
	_, err = s.Save(expired, uuid.NewString(), SaveInput{SourceID: source.ID})
	require.ErrorIs(t, err, preparation.ErrForbidden)
}

func TestContentAndRevisionBoundsBeforePersistence(t *testing.T) {
	s, _, r, ctx, source := fixture(t)
	for _, in := range []SaveInput{
		{SourceID: source.ID, ExpectedRevision: -1},
		{SourceID: source.ID, Content: Content{Title: strings.Repeat("中", 342)}},
		{SourceID: source.ID, Content: Content{Description: strings.Repeat("a", 32769)}},
		{SourceID: source.ID, Content: Content{Description: strings.Repeat("\x01", 12000)}}, // JSON escaping exceeds the whole-request budget.
		{SourceID: source.ID, Content: Content{Title: "invalid\x00text"}},
		{SourceID: source.ID, Content: Content{Description: string([]byte{0xff})}},
		{SourceID: "not-a-source"},
	} {
		_, err := s.Save(ctx, uuid.NewString(), in)
		require.ErrorIs(t, err, preparation.ErrInvalid)
	}
	require.Zero(t, r.writes)
}
