package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	dataservicesapp "task-processor/internal/app/dataservices"
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	keystore "task-processor/internal/integration/persistence/dataservice"
	acquisitionstore "task-processor/internal/integration/persistence/product/acquisition"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	jobstore "task-processor/internal/integration/persistence/product/dataacquisition"
	researchstore "task-processor/internal/integration/persistence/product/research"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
	"task-processor/internal/product/research"
	rh "task-processor/internal/product/research/httpapi"
	"task-processor/internal/product/sourcing"
)

type fixture1688Provider struct{}

func (fixture1688Provider) Acquire(_ context.Context, source sourcing.AcquisitionSource) (sourcing.AcquisitionEvidence, error) {
	title, sku := "Fixture supplier original", "sku-blue-01"
	return sourcing.AcquisitionEvidence{
		SchemaVersion: 1, SourceURL: source.URL, OfferID: source.OfferID, Title: &title,
		Variants:      []sourcing.AcquisitionVariant{{SourceID: &sku, Attributes: []sourcing.AcquisitionAttribute{{Name: "pack", Value: "2 pieces"}}}},
		ContentSHA256: strings.Repeat("a", 64), CapturedAt: time.Now().UTC(), ParserVersion: "synthetic-v1",
	}, nil
}

type fixture1688Roles struct{ denied bool }

func (r *fixture1688Roles) ResolveLiveRoles(context.Context, string, string) ([]string, error) {
	if r.denied {
		return nil, nil
	}
	return []string{"listingkit_admin"}, nil
}

type fixture1688Charges struct{ operations *acquisitionstore.Repository }

func (c fixture1688Charges) Reserve(ctx context.Context, id orgresource.ConsumerChargeIdentity) (orgresource.ConsumerChargeReceipt, error) {
	intent, err := c.operations.ReadChargeIntent(ctx, id.OrganizationID, id.OperationID)
	if err != nil {
		return orgresource.ConsumerChargeReceipt{}, err
	}
	return orgresource.ConsumerChargeReceipt{
		Intent: orgresource.ConsumerChargeIntent{Identity: id, ActorID: intent.Scope.ActorID, MemberID: intent.MemberID,
			Funding: orgresource.ResourceFunding(intent.Funding), ResourceType: orgresource.ResourceDataRow, Quantity: 1,
			Fingerprint: intent.Fingerprint, BusinessScope: intent.Source.URL},
		ReservationID: uuid.NewSHA1(uuid.NameSpaceOID, []byte(id.OperationID)).String(), State: orgresource.ReservationReserved,
	}, nil
}

func (fixture1688Charges) Reconcile(context.Context, orgresource.ConsumerChargeIdentity) (orgresource.ConsumerChargeReceipt, error) {
	return orgresource.ConsumerChargeReceipt{}, nil
}

func publishResearchAmazonFixture(t *testing.T, ctx context.Context, db *gorm.DB, access *sheinResearchAccess) collection.Source {
	t.Helper()
	publisher, err := dataservicesapp.NewProductPublisher(access)
	require.NoError(t, err)
	repo, err := jobstore.NewRepository(ctx, db, access, publisher, func(*gorm.DB) (dataacquisition.CapturedResultReader, error) {
		return nil, dataacquisition.ErrUnavailable
	})
	require.NoError(t, err)
	q := dataacquisition.Query{Site: "us", Mode: "asin", ASINs: []string{"B000123456"}, Limit: 1}
	job, err := repo.Admit(ctx, dataacquisition.Principal{Scope: access.scope}, uuid.NewString(), q, orgresource.FundingMember)
	require.NoError(t, err)
	job, err = repo.Discover(ctx, job, q.ASINs)
	require.NoError(t, err)
	items, err := repo.Items(ctx, job)
	require.NoError(t, err)
	require.Len(t, items, 1)
	item := items[0]
	intent, err := repo.ChargeIntent(ctx, orgresource.ConsumerChargeIdentity{OrganizationID: access.scope.OrganizationID, Consumer: orgresource.ConsumerAmazonData, OperationID: item.ID})
	require.NoError(t, err)
	_, err = repo.BindReservation(ctx, job, item.ID, orgresource.ConsumerChargeReceipt{Intent: intent, ReservationID: uuid.NewString(), State: orgresource.ReservationReserved})
	require.NoError(t, err)
	item, err = repo.Claim(ctx, job, item.ID)
	require.NoError(t, err)
	item, err = repo.PrepareEvidence(ctx, job, item, dataacquisition.Evidence{Site: "us", ASIN: item.ASIN, Title: "Fixture Amazon original",
		MainImage: "https://m.media-amazon.com/images/I/fixture.jpg", Availability: "available", Price: 10, Currency: "USD",
		CapturedAt: time.Now().UTC().Format(time.RFC3339Nano), ParserVersion: "amazon-v1", Attributes: map[string]string{"pack": "1 piece"}})
	require.NoError(t, err)
	item, err = repo.Publish(ctx, job, item)
	require.NoError(t, err)
	return *item.Source
}

func TestResearchSupplierReviewUsesLiveOriginalOwners(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("supplier_research_fixture"), tcpostgres.WithUsername("fixture_owner"), tcpostgres.WithPassword("fixture-only"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pg.Terminate(context.Background())) })
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	for _, install := range []func(*gorm.DB) error{productsourcing.InstallAcquisitionSchema, collectionstore.InstallSchema, keystore.InstallSchema, keystore.InstallCustomSchema, jobstore.InstallSchema, researchstore.InstallSchema} {
		require.NoError(t, install(db))
	}
	require.NoError(t, db.Exec("CREATE ROLE product_research_runtime LOGIN PASSWORD 'fixture-runtime'; REVOKE CREATE ON SCHEMA public FROM PUBLIC").Error)
	require.NoError(t, researchstore.GrantRuntime(db))
	require.NoError(t, researchstore.InstallSupplierReviewSchema(db))

	scope := collection.Scope{OrganizationID: "org-fixture", ActorID: "actor-fixture", MemberID: "member-fixture"}
	ctx = authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID,
		UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute)})
	access := &sheinResearchAccess{scope: scope, project: uuid.NewString()}
	roles := &fixture1688Roles{}
	permissions, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	operations, err := acquisitionstore.NewRepository(ctx, db)
	require.NoError(t, err)
	producer, err := productsourcing.NewPublicAcquisition(ctx, db, roles, permissions, fixture1688Provider{}, fixture1688Charges{operations}, productsourcing.WithCollectionPublication())
	require.NoError(t, err)
	acquired, err := producer.Acquire(ctx, uuid.NewString(), "https://detail.1688.com/offer/981645030344.html")
	require.NoError(t, err)
	amazon := publishResearchAmazonFixture(t, ctx, db, access)
	sourceReader, err := productsourcing.NewPublishedAcquisitionReader(ctx, db, roles, permissions)
	require.NoError(t, err)
	cr, err := collectionstore.NewRepository(ctx, db, func(*gorm.DB) (collectionstore.OwnPublisher, error) { return nil, collection.ErrUnavailable })
	require.NoError(t, err)
	cs, err := collection.NewService(cr, access, sourceReader)
	require.NoError(t, err)
	snapshots, err := catalogstore.NewBoundedSnapshotReader(db, sourcing.MaxEncodedSnapshotBytes)
	require.NoError(t, err)
	cs.WithSnapshots(snapshots)
	batches, err := cr.ListBatches(ctx, scope, collection.Query{Limit: 10})
	require.NoError(t, err)
	require.Len(t, batches.Items, 2)
	var saved []collection.Item
	for _, batch := range batches.Items {
		items, err := cr.ListItems(ctx, scope, batch.ID, collection.Query{Limit: 10})
		require.NoError(t, err)
		saved = append(saved, items.Items...)
	}
	require.Len(t, saved, 2)
	var market, supplier research.Selection
	for _, item := range saved {
		sel := research.Selection{ItemID: item.ID, Revision: item.Revision, PublicationID: item.Source.PublicationID, Version: item.Source.Version}
		if item.Source == amazon {
			market = sel
		}
		if item.Source.PublicationID == acquired.Publication.Receipt.PublicationID {
			supplier = sel
		}
	}
	require.True(t, research.ValidSelection(market))
	require.True(t, research.ValidSupplierSelection(supplier))
	data, err := dataservicesapp.NewModule(ctx, dataservicesapp.Dependencies{ProductDB: db, Access: access, Live: access, Specialist: access})
	require.NoError(t, err)
	runtimeURL, err := url.Parse(dsn)
	require.NoError(t, err)
	runtimeURL.User = url.UserPassword(researchstore.RuntimeRole, "fixture-runtime")
	runtimeDB, err := gorm.Open(postgres.Open(runtimeURL.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	runtimePool, err := runtimeDB.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, runtimePool.Close()) })
	require.NoError(t, researchstore.VerifyRuntimePermissions(ctx, runtimeDB))
	repo, err := researchstore.New(ctx, runtimeDB)
	require.NoError(t, err)
	sources := researchSources{collections: cs, data: data}
	svc := research.New(repo, sheinResearchProject{access}, sources)
	svc.SupplierSources = sources
	svc.SupplierExpiry = supplierEvidenceExpiry
	linked, err := svc.Execute(ctx, scope, access.project, uuid.NewString(), research.Command{Action: "link", Selections: []research.Selection{market}})
	require.NoError(t, err)
	id := linked.CandidateIDs[0]
	preview, err := svc.SupplierPreview(ctx, scope, access.project, id, supplier)
	require.NoError(t, err)
	require.Len(t, preview.Offer.Variants, 1)
	require.Equal(t, "1 piece", preview.Market.Attributes["pack"])
	require.Equal(t, "sku-blue-01", preview.Offer.Variants[0].ID)
	cmd := research.Command{Action: research.RecordSupplierReview, CandidateID: id, ExpectedRevision: preview.Revision, LinkID: preview.LinkID, EvidenceDigest: preview.EvidenceDigest,
		Supplier: &research.SupplierInput{Selection: supplier, SKUID: "sku-blue-01", Quantity: 20, Unit: "piece", Status: "CONFIRMED", Reason: "packaging differs",
			Comparisons: []research.SupplierComparison{{MarketKey: "pack", SupplierKey: "pack", SupplierLevel: "sku", Result: "DIFFERENT"}}}}
	key := uuid.NewString()
	_, err = svc.Execute(ctx, scope, access.project, key, cmd)
	require.NoError(t, err)
	replay, err := svc.Execute(ctx, scope, access.project, key, cmd)
	require.NoError(t, err)
	require.True(t, replay.Replayed)
	batch, err := cs.Mutate(ctx, uuid.NewString(), collection.Mutation{Action: "create_batch", Name: "Moved originals"})
	require.NoError(t, err)
	for _, item := range []research.Selection{market, supplier} {
		_, err = cs.Mutate(ctx, uuid.NewString(), collection.Mutation{Action: "move_item", ItemID: item.ItemID, TargetBatchID: batch.BatchID, ExpectedRevision: item.Revision})
		require.NoError(t, err)
	}
	_, err = svc.SupplierPreview(ctx, scope, access.project, id, supplier)
	require.ErrorIs(t, err, research.ErrConflict)
	handler := &rh.Handler{Service: svc, Bind: func(_ context.Context, _ string) (context.Context, collection.Scope, error) { return ctx, scope, nil }}
	router := gin.New()
	for _, route := range rh.Routes(handler) {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	path := rh.Base + "/" + access.project + "/candidates/" + id + "/supplier-review"
	read := func() string {
		t.Helper()
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, 200, rec.Code, rec.Body.String())
		require.Contains(t, rec.Header().Get("Cache-Control"), "no-store")
		return rec.Body.String()
	}
	body := read()
	require.Contains(t, body, "packaging differs")
	require.Contains(t, body, "2 pieces")
	require.Contains(t, body, "sku-blue-01")
	for _, denied := range []*bool{&roles.denied, &access.denied} {
		*denied = true
		require.JSONEq(t, `{"items":[{"current":true}]}`, read())
		*denied = false
	}
	var count int64
	require.NoError(t, db.Table("research_supplier_reviews").Count(&count).Error)
	require.EqualValues(t, 1, count)
}
