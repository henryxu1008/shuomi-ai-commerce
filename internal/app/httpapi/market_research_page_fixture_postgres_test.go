package httpapi

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	dataservicesapp "task-processor/internal/app/dataservices"
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/authidentity"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/research"
	"task-processor/internal/product/sourcing"
)

type fixturePageStore struct {
	research.Repository
	rows []research.Candidate
}

func (s fixturePageStore) List(context.Context, collection.Scope, string, string) ([]research.Candidate, string, error) {
	return s.rows, "", nil
}

type fixturePageCollectionAccess struct {
	*sheinResearchAccess
	delay time.Duration
	calls atomic.Int64
}

func (a *fixturePageCollectionAccess) Authorize(ctx context.Context, permission string) (collection.Scope, error) {
	a.calls.Add(1)
	if err := waitFixtureIdentity(ctx, a.delay); err != nil {
		return collection.Scope{}, err
	}
	return a.sheinResearchAccess.Authorize(ctx, permission)
}

func waitFixtureIdentity(ctx context.Context, delay time.Duration) error {
	if delay == 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func TestResearchFixturePageReadsFiftyWithinOriginalBudgetAndMasksStaleEvidence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("research_page_fixture"), tcpostgres.WithUsername("fixture_owner"), tcpostgres.WithPassword("fixture-only"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pg.Terminate(context.Background())) })
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	require.NoError(t, productsourcing.InstallAcquisitionSchema(db))
	require.NoError(t, collectionstore.InstallSchema(db))

	scope := collection.Scope{OrganizationID: "fixture-org", ActorID: "fixture-actor", MemberID: "fixture-member"}
	project := uuid.NewString()
	ctx = authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{
		TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID,
		UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute),
	})
	access := &fixturePageCollectionAccess{sheinResearchAccess: &sheinResearchAccess{scope: scope, project: project}}
	var fixtureChecks atomic.Int64
	var fixtureDelay time.Duration
	var revokeAfterAdmission bool
	fixture, err := dataservicesapp.NewResearchFixture(ctx, db, scope, project, func(ctx context.Context, gotScope collection.Scope, gotProject string, _ bool) error {
		call := fixtureChecks.Add(1)
		if gotScope != scope || gotProject != project || (revokeAfterAdmission && call > 1) {
			return collection.ErrForbidden
		}
		return waitFixtureIdentity(ctx, fixtureDelay)
	}, true)
	require.NoError(t, err)
	batch, err := fixture.ImportPagination(ctx, scope, project)
	require.NoError(t, err)
	repo, err := collectionstore.NewRepository(ctx, db, func(*gorm.DB) (collectionstore.OwnPublisher, error) {
		return nil, collection.ErrUnavailable
	})
	require.NoError(t, err)
	items, err := repo.ListItems(ctx, scope, batch.BatchID, collection.Query{Limit: 50})
	require.NoError(t, err)
	require.Len(t, items.Items, 50)
	collections, err := collection.NewService(repo, access, access)
	require.NoError(t, err)
	snapshots, err := catalogstore.NewBoundedSnapshotReader(db, sourcing.MaxEncodedSnapshotBytes)
	require.NoError(t, err)
	sources := researchSources{collections: collections.WithSnapshots(snapshots), fixture: fixture}
	rows := make([]research.Candidate, 0, 50)
	for _, item := range items.Items {
		selection := research.Selection{ItemID: item.ID, Revision: item.Revision, PublicationID: item.Source.PublicationID, Version: item.Source.Version}
		original, err := sources.Observe(ctx, scope, project, selection)
		require.NoError(t, err)
		link := research.Link{ID: uuid.NewString(), Ref: original.Ref}
		rows = append(rows, research.Candidate{ID: uuid.NewString(), ProjectID: project, SelectedLinkID: link.ID, Links: []research.Link{link}, Reason: "saved reason"})
	}

	service := research.New(fixturePageStore{rows: rows}, sheinResearchProject{access.sheinResearchAccess}, sources)
	fixtureChecks.Store(0)
	access.calls.Store(0)
	fixtureDelay, access.delay = 100*time.Millisecond, 100*time.Millisecond
	started := time.Now()
	page, err := service.List(ctx, scope, project, "")
	require.NoError(t, err)
	require.Len(t, page.Items, 50)
	require.False(t, page.Incomplete, "all 50 originals must fit within the unchanged page budget under synthetic identity latency")
	require.EqualValues(t, 2, fixtureChecks.Load(), "one current fixture authorization at admission and one before returning evidence")
	require.Zero(t, access.calls.Load(), "the verified fixture read must not re-read full Collection details per row")
	t.Logf("50 originals in %s, %d fixture authorizations, %d repeated Collection detail authorizations", time.Since(started), fixtureChecks.Load(), access.calls.Load())
	for _, item := range page.Items {
		require.NotNil(t, item.Observation)
	}

	fixtureDelay, access.delay = 0, 0
	rows[0].Links[0].Ref.Checksum = "different-original-digest"
	rows[1].Links[0].Ref.Selection.Version++
	page, err = service.List(ctx, scope, project, "")
	require.NoError(t, err)
	for _, item := range page.Items[:2] {
		require.Nil(t, item.Observation)
		require.Nil(t, item.Evidence[0].Observation)
		require.Empty(t, item.EvidenceDigest)
		require.Empty(t, item.Reason)
	}
	require.NotNil(t, page.Items[2].Observation)

	revokeAfterAdmission = true
	fixtureChecks.Store(0)
	page, err = service.List(ctx, scope, project, "")
	require.NoError(t, err)
	require.Len(t, page.Items, 50)
	require.EqualValues(t, 2, fixtureChecks.Load())
	for _, item := range page.Items {
		require.Nil(t, item.Observation)
		require.Nil(t, item.Evidence[0].Observation)
		require.Empty(t, item.EvidenceDigest)
		require.Empty(t, item.Reason)
	}
}
