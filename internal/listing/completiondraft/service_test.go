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
func (r testSources) Read(_ context.Context, _ collection.Scope, id string) (preparation.Preparation, error) {
	if id != r.source.PreparationID {
		return preparation.Preparation{}, preparation.ErrNotFound
	}
	return preparation.Preparation{ID: id}, nil
}

type testCatalog struct{ snapshot catalog.PublishedSnapshot }

func (r testCatalog) GetSnapshot(context.Context, catalog.SnapshotIdentity, uint64) (catalog.PublishedSnapshot, error) {
	return r.snapshot, nil
}

type testRecords struct {
	Repository
	writes       int
	beforeCommit func()
	beforeList   func()
	saved        Prepared
	record       Record
	page         collection.Page[Record]
	listCalls    int
}

func (r *testRecords) ListForCollectionItem(context.Context, collection.Scope, string, collection.Query) (collection.Page[Record], error) {
	r.listCalls++
	if r.beforeList != nil {
		r.beforeList()
	}
	return r.page, nil
}

type itemOwner struct {
	*collection.Service
	item  collection.Item
	reads int
	err   error
}

func (o *itemOwner) ReadItem(context.Context, string) (collection.ItemDetail, error) {
	o.reads++
	return collection.ItemDetail{Item: o.item}, o.err
}

func TestItemDraftListKeepsFixedVersionsAcrossPreparationsAndRechecksCurrentItem(t *testing.T) {
	for _, change := range []string{"none", "archived", "revoked", "member", "cancelled", "wrong-item", "wrong-kind"} {
		t.Run(change, func(t *testing.T) {
			s, a, r, ctx, source := fixture(t)
			owner := &itemOwner{Service: s.owners.(*collection.Service), item: collection.Item{ID: source.CollectionItemID, Source: source.Source}}
			s.owners = owner
			second := source
			second.ID = uuid.NewString()
			second.PreparationID = uuid.NewString()
			second.Source.Version = 2
			r.page = collection.Page[Record]{Items: []Record{{Source: source, Revision: 3}, {Source: second, Revision: 1}}, Total: 2}
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			r.beforeList = func() {
				switch change {
				case "archived":
					owner.err = collection.ErrNotFound
				case "revoked":
					a.denied = true
				case "member":
					a.scope.MemberID = "replacement"
				case "cancelled":
					cancel()
				case "wrong-item":
					r.page.Items[1].Source.CollectionItemID = uuid.NewString()
				case "wrong-kind":
					r.page.Items[1].Source.Source.Kind = "collected"
				}
			}
			page, err := s.ListForCollectionItem(ctx, source.CollectionItemID, collection.Query{Limit: 20})
			if change != "none" {
				require.Error(t, err)
				require.Empty(t, page.Items)
				return
			}
			require.NoError(t, err)
			require.Equal(t, source.CollectionItemID, page.CollectionItemID)
			require.Equal(t, r.page.Items, page.Items)
			require.Equal(t, 2, owner.reads)
			require.Equal(t, 1, r.listCalls)
		})
	}
}

func TestItemDraftListRejectsInvalidQueryOrUnavailableCurrentItemBeforeRepository(t *testing.T) {
	s, _, r, ctx, source := fixture(t)
	owner := &itemOwner{Service: s.owners.(*collection.Service), item: collection.Item{ID: source.CollectionItemID, Source: source.Source}}
	s.owners = owner
	_, err := s.ListForCollectionItem(ctx, source.CollectionItemID, collection.Query{Limit: 101})
	require.ErrorIs(t, err, preparation.ErrInvalid)
	owner.err = collection.ErrNotFound
	_, err = s.ListForCollectionItem(ctx, source.CollectionItemID, collection.Query{Limit: 20})
	require.ErrorIs(t, err, collection.ErrNotFound)
	require.Zero(t, r.listCalls)
}

func (r *testRecords) Save(ctx context.Context, p Prepared) (Receipt, error) {
	r.saved = p
	_, _, _, input, source, err := p.Read(ctx)
	if err != nil {
		return Receipt{}, err
	}
	if r.beforeCommit != nil {
		r.beforeCommit()
	}
	if err := p.ValidateCommit(ctx, source); err != nil {
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
func (r *testRecords) List(context.Context, collection.Scope, string, collection.Query) (collection.Page[Record], error) {
	if r.beforeList != nil {
		r.beforeList()
	}
	return collection.Page[Record]{Items: []Record{r.record}, Total: 1}, nil
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
	require.ErrorIs(t, (Prepared{}).ValidateCommit(ctx, source), preparation.ErrForbidden)
}

func TestCommitRejectsSourceDriftEvenWithCurrentOwner(t *testing.T) {
	s, _, r, ctx, source := fixture(t)
	_, err := s.Save(ctx, uuid.NewString(), SaveInput{SourceID: source.ID})
	require.NoError(t, err)
	source.Source.Version++
	require.ErrorIs(t, r.saved.ValidateCommit(ctx, source), preparation.ErrForbidden)
}
func TestListRejectsAuthorityChangesAfterTheBoundedPageRead(t *testing.T) {
	for _, change := range []string{"revoked", "member-replaced"} {
		t.Run(change, func(t *testing.T) {
			s, a, r, ctx, source := fixture(t)
			_, err := s.Save(ctx, uuid.NewString(), SaveInput{SourceID: source.ID})
			require.NoError(t, err)
			r.beforeList = func() {
				if change == "revoked" {
					a.denied = true
				} else {
					a.scope.MemberID = "new-member"
				}
			}
			page, err := s.List(ctx, source.PreparationID, collection.Query{Limit: 20})
			require.ErrorIs(t, err, preparation.ErrForbidden)
			require.Empty(t, page.Items)
		})
	}
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
