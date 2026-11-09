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
	preparationstore "task-processor/internal/integration/persistence/listing/preparation"
	acquisitionstore "task-processor/internal/integration/persistence/product/acquisition"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	jobstore "task-processor/internal/integration/persistence/product/dataacquisition"
	researchstore "task-processor/internal/integration/persistence/product/research"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
	"task-processor/internal/product/research"
	rh "task-processor/internal/product/research/httpapi"
	"task-processor/internal/product/sourcing"
)

type fixture1688Provider struct{}

func (fixture1688Provider) Acquire(_ context.Context, source sourcing.AcquisitionSource) (sourcing.AcquisitionEvidence, error) {
	title, sku := "Fixture supplier original", "sku-adjustable-white-01"
	return sourcing.AcquisitionEvidence{
		SchemaVersion: 1, SourceURL: source.URL, OfferID: source.OfferID, Title: &title,
		Variants:      []sourcing.AcquisitionVariant{{SourceID: &sku, Attributes: []sourcing.AcquisitionAttribute{{Name: "width", Value: "40–55 cm"}}}},
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

func publishResearchAmazonFixture(t *testing.T, ctx context.Context, db *gorm.DB, access *sheinResearchAccess, keyword string, samples ...dataacquisition.Evidence) map[string]collection.Source {
	t.Helper()
	publisher, err := dataservicesapp.NewProductPublisher(access)
	require.NoError(t, err)
	repo, err := jobstore.NewRepository(ctx, db, access, publisher, func(*gorm.DB) (dataacquisition.CapturedResultReader, error) {
		return nil, dataacquisition.ErrUnavailable
	})
	require.NoError(t, err)
	q := dataacquisition.Query{Site: "us", Mode: "keyword", Keyword: keyword, Limit: 5, Fields: []string{"attributes", "price", "currency"}}
	asins := make([]string, 0, len(samples))
	evidenceByASIN := make(map[string]dataacquisition.Evidence, len(samples))
	for _, sample := range samples {
		asins = append(asins, sample.ASIN)
		evidenceByASIN[sample.ASIN] = sample
	}
	repo.ConfigurePrices(testSupplierDataPrice{})
	job, err := repo.Admit(ctx, dataacquisition.Principal{Scope: access.scope}, uuid.NewString(), q, orgresource.FundingMember, dataacquisition.PriceConfirmation{Version: "test-price", MaximumCostFen: int64(q.Limit) * 5})
	require.NoError(t, err)
	job, err = repo.Discover(ctx, job, asins)
	require.NoError(t, err)
	items, err := repo.Items(ctx, job)
	require.NoError(t, err)
	require.Len(t, items, len(samples))
	sources := make(map[string]collection.Source, len(samples))
	for _, item := range items {
		intent, err := repo.ChargeIntent(ctx, orgresource.ConsumerChargeIdentity{OrganizationID: access.scope.OrganizationID, Consumer: orgresource.ConsumerAmazonData, OperationID: item.ID})
		require.NoError(t, err)
		_, err = repo.BindReservation(ctx, job, item.ID, orgresource.ConsumerChargeReceipt{Intent: intent, ReservationID: uuid.NewString(), State: orgresource.ReservationReserved})
		require.NoError(t, err)
		item, err = repo.Claim(ctx, job, item.ID)
		require.NoError(t, err)
		item, err = repo.PrepareEvidence(ctx, job, item, evidenceByASIN[item.ASIN])
		require.NoError(t, err)
		item, err = repo.Publish(ctx, job, item)
		require.NoError(t, err)
		sources[item.ASIN] = *item.Source
	}
	return sources
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
	for _, install := range []func(*gorm.DB) error{productsourcing.InstallAcquisitionSchema, collectionstore.InstallSchema, keystore.InstallSchema, keystore.InstallCustomSchema, jobstore.InstallSchema, researchstore.InstallSchema, preparationstore.InstallSchema} {
		require.NoError(t, install(db))
	}
	require.NoError(t, db.Exec("CREATE ROLE product_research_runtime LOGIN PASSWORD 'fixture-runtime'; REVOKE CREATE ON SCHEMA public FROM PUBLIC").Error)
	require.NoError(t, researchstore.GrantRuntime(db))
	require.NoError(t, researchstore.InstallSupplierReviewSchema(db))
	require.NoError(t, researchstore.InstallAssessmentSchema(db))

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
	sample := func(asin, title, width, installation string, price float64) dataacquisition.Evidence {
		return dataacquisition.Evidence{Site: "us", ASIN: asin, Title: title,
			MainImage: "https://m.media-amazon.com/images/I/fixture.jpg", Availability: "available", Price: price, Currency: "USD",
			CapturedAt: time.Now().UTC().Format(time.RFC3339Nano), ParserVersion: "amazon-v1",
			Attributes: map[string]string{"width": width, "installation": installation, "pack": "1 piece"}}
	}
	a := sample("B000123456", "Adjustable under sink organizer", "38–55 cm", "no drilling", 29.99)
	b := sample("B000123457", "Fixed under sink organizer", "42 cm", "requires drilling", 24.99)
	previous := publishResearchAmazonFixture(t, ctx, db, access, "under sink shelf", a)[a.ASIN]
	publishResearchAmazonFixture(t, ctx, db, access, "under sink rack", a)
	amazon := publishResearchAmazonFixture(t, ctx, db, access, "under sink organizer", a, b)
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
	require.Len(t, batches.Items, 4)
	var saved []collection.Item
	for _, batch := range batches.Items {
		items, err := cr.ListItems(ctx, scope, batch.ID, collection.Query{Limit: 10})
		require.NoError(t, err)
		saved = append(saved, items.Items...)
	}
	require.Len(t, saved, 5)
	var market, alternative, oldMarket, supplier research.Selection
	var marketBatchID string
	for _, item := range saved {
		sel := research.Selection{ItemID: item.ID, Revision: item.Revision, PublicationID: item.Source.PublicationID, Version: item.Source.Version}
		switch {
		case item.Source == amazon[a.ASIN]:
			market, marketBatchID = sel, item.BatchID
		case item.Source == amazon[b.ASIN]:
			alternative = sel
		case item.Source == previous:
			oldMarket = sel
		case item.Source.PublicationID == acquired.Publication.Receipt.PublicationID:
			supplier = sel
		}
	}
	require.True(t, research.ValidSelection(market))
	require.EqualValues(t, 3, market.Version)
	require.EqualValues(t, 1, alternative.Version)
	require.True(t, research.ValidSelection(oldMarket))
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
	svc.AssessmentExpiry = supplierEvidenceExpiry
	link := func(selections ...research.Selection) research.Receipt {
		t.Helper()
		receipt, err := svc.Execute(ctx, scope, access.project, uuid.NewString(), research.Command{Action: "link", Selections: selections})
		require.NoError(t, err)
		return receipt
	}
	candidate := func(id string) research.CandidateView {
		t.Helper()
		page, err := svc.List(ctx, scope, access.project, "")
		require.NoError(t, err)
		for _, row := range page.Items {
			if row.ID == id {
				return row
			}
		}
		t.Fatalf("missing candidate %s", id)
		return research.CandidateView{}
	}
	approve := func(row research.CandidateView) {
		t.Helper()
		receipt, err := svc.Execute(ctx, scope, access.project, uuid.NewString(), research.Command{
			Action: "review", CandidateID: row.ID, ExpectedRevision: row.Revision, EvidenceDigest: row.EvidenceDigest,
		})
		require.NoError(t, err)
		_, err = svc.Execute(ctx, scope, access.project, uuid.NewString(), research.Command{
			Action: "decide", CandidateID: row.ID, ExpectedRevision: receipt.Revisions[row.ID], EvidenceDigest: row.EvidenceDigest,
			State: "APPROVED_FOR_TEST", Reason: "fixture evidence accepted",
		})
		require.NoError(t, err)
	}
	id := link(oldMarket).CandidateIDs[0]
	approve(candidate(id))
	oldDecision := candidate(id).History[0]
	require.Equal(t, "APPROVED_FOR_TEST", oldDecision.State)
	link(market, alternative)
	current := candidate(id)
	require.Equal(t, oldMarket.PublicationID, current.Observation.Ref.Selection.PublicationID)
	require.Equal(t, "APPROVED_FOR_TEST", current.State)
	var linkID string
	for _, evidence := range current.Evidence {
		if evidence.Observation != nil && evidence.Observation.Ref.Selection == market {
			linkID = evidence.ID
		}
	}
	require.NotEmpty(t, linkID)
	_, err = svc.Execute(ctx, scope, access.project, uuid.NewString(), research.Command{
		Action: "select", CandidateID: id, ExpectedRevision: current.Revision, LinkID: linkID,
	})
	require.NoError(t, err)
	current = candidate(id)
	require.Equal(t, "TO_RESEARCH", current.State)
	require.Equal(t, "under sink organizer", current.Observation.Query.Keyword)
	require.NotNil(t, current.Observation.Price)
	require.Equal(t, 29.99, *current.Observation.Price)
	require.Equal(t, market, current.Observation.Ref.Selection)
	require.Contains(t, current.History, oldDecision)
	preview, err := svc.SupplierPreview(ctx, scope, access.project, id, supplier)
	require.NoError(t, err)
	require.Len(t, preview.Offer.Variants, 1)
	require.Equal(t, "38–55 cm", preview.Market.Attributes["width"])
	require.Equal(t, "sku-adjustable-white-01", preview.Offer.Variants[0].ID)
	cmd := research.Command{Action: research.RecordSupplierReview, CandidateID: id, ExpectedRevision: preview.Revision, LinkID: preview.LinkID, EvidenceDigest: preview.EvidenceDigest,
		Supplier: &research.SupplierInput{Selection: supplier, SKUID: "sku-adjustable-white-01", Quantity: 20, Unit: "piece", Status: "CONFIRMED", Reason: "minimum 40 cm accepted",
			Comparisons: []research.SupplierComparison{{MarketKey: "width", SupplierKey: "width", SupplierLevel: "sku", Result: "DIFFERENT"}}}}
	key := uuid.NewString()
	_, err = svc.Execute(ctx, scope, access.project, key, cmd)
	require.NoError(t, err)
	replay, err := svc.Execute(ctx, scope, access.project, key, cmd)
	require.NoError(t, err)
	require.True(t, replay.Replayed)

	assessmentCandidate := candidate(id)
	assessmentKey := uuid.NewString()
	assessmentCommand := research.Command{Action: research.SaveAssessment, CandidateID: id, ExpectedRevision: assessmentCandidate.Revision, LinkID: assessmentCandidate.SelectedLinkID, EvidenceDigest: assessmentCandidate.EvidenceDigest,
		Assessment: &research.AssessmentInput{Cost: research.CostInput{Rule: "unsupported", PurchaseCurrency: "USD", Quantity: "20", Specification: "sku-adjustable-white-01"}, SupplierReview: &research.AssessmentSupplierRef{Key: key, Revision: replay.Revisions[id]}}}
	assessmentReceipt, err := svc.Execute(ctx, scope, access.project, assessmentKey, assessmentCommand)
	require.NoError(t, err)
	assessment, err := svc.ReadAssessment(ctx, scope, access.project, id, assessmentKey)
	require.NoError(t, err)
	require.NotNil(t, assessment.Detail)
	require.Equal(t, "CONFIRMED", assessment.Detail.SupplierBinding)
	require.Equal(t, "sku-adjustable-white-01", assessment.Detail.SupplierSKU)
	require.Equal(t, 20, assessment.Detail.SupplierQuantity)
	require.Equal(t, "piece", assessment.Detail.SupplierUnit)
	require.Equal(t, "UNSUPPORTED_COST_RULE", assessment.Detail.Result.Status)
	require.Equal(t, *assessmentReceipt.AssessmentRef, assessment.Detail.Ref)
	mismatched := assessmentCommand
	changed := *assessmentCommand.Assessment
	changed.Cost.Quantity = "21"
	mismatched.Assessment = &changed
	mismatched.ExpectedRevision = assessmentReceipt.Revisions[id]
	_, err = svc.Execute(ctx, scope, access.project, uuid.NewString(), mismatched)
	require.ErrorIs(t, err, research.ErrConflict, "changed quantity cannot inherit the saved supplier confirmation")

	approve(candidate(id))
	require.Equal(t, "APPROVED_FOR_TEST", candidate(id).State)
	page, err := svc.List(ctx, scope, access.project, "")
	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	for _, row := range page.Items {
		if row.ID != id {
			require.Equal(t, alternative, row.Observation.Ref.Selection)
			require.NotNil(t, row.Observation.Price)
			require.Equal(t, 24.99, *row.Observation.Price)
			require.Equal(t, "TO_RESEARCH", row.State)
		}
	}
	originalBatch, err := cs.ReadBatch(ctx, marketBatchID)
	require.NoError(t, err)
	preparationRepo, err := preparationstore.NewRepository(ctx, db)
	require.NoError(t, err)
	preparations, err := preparation.NewService(preparationRepo, cs, access)
	require.NoError(t, err)
	transferKey := uuid.NewString()
	input := preparation.TransferInput{
		BatchID: marketBatchID, ExpectedRevision: originalBatch.Revision, ItemIDs: []string{market.ItemID},
	}
	transfer, err := preparations.Transfer(ctx, transferKey, input)
	require.NoError(t, err)
	require.EqualValues(t, 1, transfer.Preparation.Count)
	found, err := preparations.ReadByKey(ctx, transferKey)
	require.NoError(t, err)
	require.Equal(t, transfer.Preparation, found.Preparation)
	repeated, err := preparations.Transfer(ctx, transferKey, input)
	require.NoError(t, err)
	require.True(t, repeated.Replayed)
	require.Equal(t, found.Preparation, repeated.Preparation)
	originals, err := preparations.ListSources(ctx, found.Preparation.ID, collection.Query{Limit: 10})
	require.NoError(t, err)
	require.Len(t, originals.Items, 1)
	require.Equal(t, market.ItemID, originals.Items[0].CollectionItemID)
	require.Equal(t, amazon[a.ASIN], originals.Items[0].Source)

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
	require.Contains(t, body, "minimum 40 cm accepted")
	require.Contains(t, body, "40–55 cm")
	require.Contains(t, body, "sku-adjustable-white-01")
	for _, denied := range []*bool{&roles.denied, &access.denied} {
		*denied = true
		require.JSONEq(t, `{"items":[{"current":true}]}`, read())
		protected, e := svc.ReadAssessment(ctx, scope, access.project, id, assessmentKey)
		require.NoError(t, e)
		require.Nil(t, protected.Detail)
		*denied = false
	}
	var count int64
	require.NoError(t, db.Table("research_supplier_reviews").Count(&count).Error)
	require.EqualValues(t, 1, count)
}

type testSupplierDataPrice struct{}

func (testSupplierDataPrice) CurrentDataRowPrice(context.Context) (dataacquisition.PriceSnapshot, error) {
	return dataacquisition.PriceSnapshot{OfferID: "test-offer", Version: "test-price", Currency: "CNY", UnitPriceFen: 5}, nil
}
