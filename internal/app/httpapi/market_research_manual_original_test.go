package httpapi

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	pc "task-processor/internal/aiworkbench/projectcenter"
	dataservicesapp "task-processor/internal/app/dataservices"
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	researchstore "task-processor/internal/integration/persistence/product/research"
	kernel "task-processor/internal/kernel/module"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/research"
)

// The old fixture publishes test setup only in this task-owned PostgreSQL.
// Normal application composition receives only the new read capability.
func TestResearchManualOriginalApplicationAssessmentRecoveryAndIsolation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("manual1205"), tcpostgres.WithUsername("fixture_owner"), tcpostgres.WithPassword("isolated-manual-only"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pg.Terminate(context.Background())) })
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	open := func(dsn string) *gorm.DB {
		db, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, e)
		pool, e := db.DB()
		require.NoError(t, e)
		t.Cleanup(func() { require.NoError(t, pool.Close()) })
		return db
	}
	owner := open(dsn)
	require.NoError(t, productsourcing.InstallAcquisitionSchema(owner))
	require.NoError(t, collectionstore.InstallSchema(owner))
	require.NoError(t, researchstore.InstallSchema(owner))
	require.NoError(t, owner.Exec("CREATE ROLE product_research_runtime LOGIN PASSWORD 'isolated-manual-runtime'; REVOKE CREATE ON SCHEMA public FROM PUBLIC").Error)
	require.NoError(t, researchstore.GrantRuntime(owner))
	require.NoError(t, researchstore.InstallSupplierReviewSchema(owner))
	require.NoError(t, researchstore.InstallAssessmentSchema(owner))
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	u.User = url.UserPassword(researchstore.RuntimeRole, "isolated-manual-runtime")
	repo, err := researchstore.New(ctx, open(u.String()))
	require.NoError(t, err)
	scope := collection.Scope{OrganizationID: "200", ActorID: "fixture-actor", MemberID: "fixture-member"}
	project := uuid.NewString()
	role := authz.EnterpriseRoleKey(scope.OrganizationID, 3)
	modules := &fixtureRoleModules{key: role, modules: []string{"projects", "supply-mine", "acquisition"}}
	policy, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	policy.SetRolePolicyReader(modules)
	id := authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, HomeOrganizationID: "home", UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, Roles: []string{role}, TokenExpiresAt: time.Now().Add(time.Hour)}
	resolver := &fixtureLiveResolver{productReviewResolverStub: productReviewResolverStub{value: id}}
	auth := &projectSources{resolver: resolver, policy: policy}
	projects := researchProjects{auth: auth, service: &pc.Service{Store: &fixtureProjectStore{project: pc.Project{ID: project, Scope: pc.Scope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}, Fields: pc.Fields{Kind: "PRODUCT_RESEARCH"}}}, Authorize: auth.authorize}}
	ctx, err = (productReviewCapabilityBinder{}).Bind(authidentity.WithAuthenticatedIdentity(ctx, id), "Bearer synthetic-only")
	require.NoError(t, err)
	fixture, err := dataservicesapp.NewResearchFixture(ctx, owner, scope, project, projects.authorizeFixture, false)
	require.NoError(t, err)
	batch, err := fixture.Import(ctx, scope, project)
	require.NoError(t, err)
	entry := batch.Amazon[1]
	collectionsRepo, err := collectionstore.NewRepository(ctx, owner, func(*gorm.DB) (collectionstore.OwnPublisher, error) { return nil, collection.ErrUnavailable })
	require.NoError(t, err)
	access := &sheinResearchAccess{scope: scope, project: project}
	collections, err := collection.NewService(collectionsRepo, access, access)
	require.NoError(t, err)
	snapshots, err := catalogstore.NewBoundedSnapshotReader(owner, 2<<20)
	require.NoError(t, err)
	sources := &researchSources{collections: collections.WithSnapshots(snapshots), fixture: fixture}
	svc := research.New(repo, projects, sources)
	selection := research.Selection{ItemID: entry.Item.ID, Revision: entry.Item.Revision, PublicationID: entry.Item.Source.PublicationID, Version: entry.Item.Source.Version}
	observation, err := sources.Observe(ctx, scope, project, selection)
	require.NoError(t, err)
	linked, err := svc.Execute(ctx, scope, project, uuid.NewString(), research.Command{Action: "link", Selections: []research.Selection{selection}})
	require.NoError(t, err)
	candidate, err := repo.Load(ctx, scope, project, linked.CandidateIDs[0])
	require.NoError(t, err)
	binding := dataservicesapp.ResearchManualOriginalBinding{Scope: scope, Project: project, CandidateID: candidate.ID, LinkID: candidate.SelectedLinkID, ItemID: selection.ItemID, ItemRevision: selection.Revision, PublicationID: selection.PublicationID, CatalogVersion: selection.Version, Checksum: entry.Checksum, EnvelopeHash: entry.Receipt.EnvelopeHash, SnapshotHash: entry.Receipt.SnapshotHash}
	// The independent manual profile coexists with enabled execution but is
	// mutually exclusive with the full fixture profile at composition time.
	options := currentApplicationOptions{researchManualOriginal: &binding, marketResearchConfigs: 1, marketResearchSupplierReview: true, productAcquisitionDB: owner, productCollections: 1, dataServices: &DataServicesDependencies{ExecutionEnabled: true}}
	require.NoError(t, options.validateResearchManualOriginal("https://localhost:35444"))
	options.researchFixture = &ResearchFixtureBinding{}
	require.Error(t, options.validateResearchManualOriginal("https://localhost:35444"))
	options.researchFixture = nil
	require.Error(t, options.validateResearchManualOriginal("https://identity.example.com"))
	sources.fixture = nil
	m := researchModule{service: svc, projects: projects}
	require.NoError(t, m.enableManualOriginal(ctx, owner, binding))
	require.Nil(t, m.fixture)
	svc.SupplierSources = sources
	svc.AssessmentExpiry = supplierEvidenceExpiry
	registry := kernel.NewRegistry()
	require.NoError(t, m.Register(registry))
	for _, route := range registry.Routes() {
		require.NotContains(t, route.Path, "acceptance-fixture")
	}
	got, err := sources.Observe(ctx, scope, project, selection)
	require.NoError(t, err)
	require.Equal(t, observation.Ref, got.Ref)
	other := batch.Amazon[0]
	_, err = sources.Observe(ctx, scope, project, research.Selection{ItemID: other.Item.ID, Revision: other.Item.Revision, PublicationID: other.Item.Source.PublicationID, Version: other.Item.Source.Version})
	require.Error(t, err)
	cmd := research.Command{Action: research.SaveAssessment, CandidateID: candidate.ID, ExpectedRevision: candidate.Revision, LinkID: candidate.SelectedLinkID, EvidenceDigest: observation.Ref.Digest(), Assessment: &research.AssessmentInput{Cost: research.CostInput{Rule: "fixed_percent", PurchaseCurrency: "USD"}}}
	preview, err := svc.PreviewAssessment(ctx, scope, project, cmd)
	require.NoError(t, err)
	require.Equal(t, "INSUFFICIENT_INPUT", preview.Result.Status)
	key := uuid.NewString()
	saved, err := svc.Execute(ctx, scope, project, key, cmd)
	require.NoError(t, err)
	require.NotNil(t, saved.AssessmentRef)
	read, err := svc.ReadAssessment(ctx, scope, project, candidate.ID, key)
	require.NoError(t, err)
	require.NotNil(t, read.Detail)
	require.Equal(t, *saved.AssessmentRef, read.Detail.Ref)
	replay, err := svc.Execute(ctx, scope, project, key, cmd)
	require.NoError(t, err)
	require.True(t, replay.Replayed)
	require.Equal(t, saved.AssessmentRef, replay.AssessmentRef)
	lookup, err := svc.Lookup(ctx, scope, project, key)
	require.NoError(t, err)
	require.Equal(t, saved.AssessmentRef, lookup.AssessmentRef)
	review, err := svc.Execute(ctx, scope, project, uuid.NewString(), research.Command{Action: "review", CandidateID: candidate.ID, ExpectedRevision: saved.Revisions[candidate.ID], EvidenceDigest: cmd.EvidenceDigest})
	require.NoError(t, err)
	decision := research.Command{Action: "decide", CandidateID: candidate.ID, ExpectedRevision: review.Revisions[candidate.ID], EvidenceDigest: cmd.EvidenceDigest, State: "APPROVED_FOR_TEST", Reason: "合成验收，不转入", AssessmentRef: saved.AssessmentRef}
	_, err = svc.Execute(ctx, scope, project, uuid.NewString(), decision)
	require.ErrorIs(t, err, research.ErrUnsupported)
	decision.State = "REJECTED"
	decided, err := svc.Execute(ctx, scope, project, uuid.NewString(), decision)
	require.NoError(t, err)
	require.EqualValues(t, 5, decided.Revisions[candidate.ID])
	current, err := repo.Load(ctx, scope, project, candidate.ID)
	require.NoError(t, err)
	require.Equal(t, saved.AssessmentRef, current.History[0].AssessmentRef)
	// Revoke current Sourcing permission without changing the underlying data.
	modules.modules = []string{"projects", "supply-mine"}
	read, err = svc.ReadAssessment(ctx, scope, project, candidate.ID, key)
	require.NoError(t, err)
	require.Nil(t, read.Detail)
	page, err := svc.List(ctx, scope, project, "")
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Nil(t, page.Items[0].Observation)
	require.Empty(t, page.Items[0].Reason)
	for table, expected := range map[string]int64{"product_source_publications": 3, "product_snapshot_versions": 3, "product_collection_items": 3, "research_assessments": 1, "research_commands": 4} {
		var n int64
		require.NoError(t, owner.Table(table).Count(&n).Error)
		require.Equal(t, expected, n, table)
	}
	// An incomplete source binding never degrades to the old fixture mode.
	bad := binding
	bad.SnapshotHash = strings.Repeat("0", 64)
	reader, err := dataservicesapp.NewResearchManualOriginalReader(ctx, owner, bad, func(context.Context, collection.Scope, string, bool) error { return nil })
	require.NoError(t, err)
	_, err = reader.Read(ctx, scope, project, collection.SelectionInput{ItemID: selection.ItemID, ExpectedRevision: selection.Revision, OriginalPublicationID: selection.PublicationID, OriginalVersion: selection.Version})
	require.Error(t, err)
}
