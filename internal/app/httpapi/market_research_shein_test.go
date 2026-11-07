package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"task-processor/internal/agent"
	dataservicesapp "task-processor/internal/app/dataservices"
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/authidentity"
	"task-processor/internal/dataservice"
	keystore "task-processor/internal/integration/persistence/dataservice"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	jobstore "task-processor/internal/integration/persistence/product/dataacquisition"
	researchstore "task-processor/internal/integration/persistence/product/research"
	sourcestore "task-processor/internal/integration/persistence/product/sourcing"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
	"task-processor/internal/product/research"
	"task-processor/internal/product/selection"
	"task-processor/internal/product/sourcing"
)

type sheinResearchPublication struct {
	Job struct {
		JobID string
		Scope collection.Scope
	}
	Publication struct {
		Source collection.Source
	}
	ObservationUTF8 string
}

type sheinResearchTrialAuthority struct {
	approval dataacquisition.SHEINUSTrialApproval
}

func (a sheinResearchTrialAuthority) AuthorizeSHEINUSTrial(_ context.Context, p dataacquisition.Principal, id string, _ dataacquisition.SHEINUSTrialAction) (dataacquisition.SHEINUSTrialApproval, error) {
	if p.Scope != a.approval.Scope || id != a.approval.ID {
		return dataacquisition.SHEINUSTrialApproval{}, dataacquisition.ErrForbidden
	}
	return a.approval, nil
}

func (a sheinResearchTrialAuthority) CheckSHEINUSTrialBudget(context.Context, dataacquisition.SHEINUSTrialApproval) (dataacquisition.SHEINUSTrialBudget, error) {
	return dataacquisition.SHEINUSTrialBudget{
		Binding: a.approval.Binding(), AvailableCredits: 25, MaximumRequestCredits: 25,
		ZeroIncrementalCash: true, ExclusiveKeyWindow: true,
	}, nil
}

type sheinResearchAccess struct {
	dataservice.Access
	dataservice.SpecialistAccess
	sourcing.PublishedAcquisitionReader
	scope         collection.Scope
	denied        bool
	project       string
	projectDenied bool
}

func (a *sheinResearchAccess) Authorize(context.Context, string) (collection.Scope, error) {
	return a.scope, nil
}

func (a *sheinResearchAccess) CheckRead(_ context.Context, p dataacquisition.Principal) error {
	if a.denied || p.Scope != a.scope {
		return dataacquisition.ErrForbidden
	}
	return nil
}

func (a *sheinResearchAccess) CheckExecution(ctx context.Context, p dataacquisition.Principal, _ orgresource.ResourceFunding) error {
	return a.CheckRead(ctx, p)
}

func (a *sheinResearchAccess) AuthorizeAnalysis(_ context.Context, scope collection.Scope, project string, _ bool) error {
	if a.projectDenied || scope != a.scope || project != a.project {
		return research.ErrForbidden
	}
	return nil
}

// Project authorization has a different signature from Collection authorization.
type sheinResearchProject struct{ access *sheinResearchAccess }

func (p sheinResearchProject) Authorize(ctx context.Context, scope collection.Scope, project string, write bool) error {
	return p.access.AuthorizeAnalysis(ctx, scope, project, write)
}
func (p sheinResearchProject) AuthorizeAnalysis(ctx context.Context, scope collection.Scope, project string, active bool) error {
	return p.access.AuthorizeAnalysis(ctx, scope, project, active)
}

func (p sheinResearchProject) AuthorizeDataMarket(_ context.Context, scope collection.Scope) error {
	if scope != p.access.scope {
		return research.ErrForbidden
	}
	return nil
}

func TestSHEINUSResearchConsumesExactPublishedCards(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("shein_research"), tcpostgres.WithUsername("test_owner"), tcpostgres.WithPassword("isolated-research"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	for _, install := range []func(*gorm.DB) error{keystore.InstallSchema, keystore.InstallCustomSchema, catalogstore.AutoMigrate, sourcestore.InstallSchema, collectionstore.InstallSchema, jobstore.InstallSchema, researchstore.InstallSchema} {
		require.NoError(t, install(db))
	}
	require.ErrorIs(t, jobstore.VerifySHEINUSTrialRuntimePermissions(ctx, pool), dataacquisition.ErrUnavailable)
	require.NoError(t, jobstore.InstallSHEINUSTrialSchema(ctx, pool))
	raw, err := os.ReadFile("../../../docs/architecture/evidence/shein-us-data-v1-p1-p2.json")
	require.NoError(t, err)
	var fixture struct {
		ProjectID     string
		Publications  map[string]sheinResearchPublication
		ConsumerCases struct {
			LinkP1ThenP2AndReplayP2 struct{ CandidateID string }
		}
	}
	require.NoError(t, json.Unmarshal(raw, &fixture))
	scope := fixture.Publications["P1"].Job.Scope
	ctx = authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute)})
	access := &sheinResearchAccess{scope: scope, project: fixture.ProjectID}
	module, err := dataservicesapp.NewModule(ctx, dataservicesapp.Dependencies{ProductDB: db, Access: access, Live: access, Specialist: access})
	require.NoError(t, err)
	require.Nil(t, module.Runner())
	require.Nil(t, module.ChargeOwner())
	cr, err := collectionstore.NewRepository(ctx, db, func(*gorm.DB) (collectionstore.OwnPublisher, error) { return nil, collection.ErrUnavailable })
	require.NoError(t, err)
	collections, err := collection.NewService(cr, access, access)
	require.NoError(t, err)
	snapshots, err := catalogstore.NewBoundedSnapshotReader(db, sourcing.MaxEncodedSnapshotBytes)
	require.NoError(t, err)
	collections.WithSnapshots(snapshots)
	repo, err := researchstore.New(ctx, db)
	require.NoError(t, err)
	reader := researchSources{collections: collections, data: module}
	svc := research.New(repo, sheinResearchProject{access}, reader)
	linkKeys := make(map[string]string)
	selected := make(map[string]research.Selection)
	for _, name := range []string{"P1", "P2"} {
		f := fixture.Publications[name]
		publishSHEINResearchTrial(t, ctx, module, access, f)
		batch, err := collection.NewPublicationBatch(scope, f.Job.JobID, collection.SourceSHEINUSData, "SHEIN · US")
		require.NoError(t, err)
		items, err := cr.ListItems(ctx, scope, batch.ID, collection.Query{Limit: 10})
		require.NoError(t, err)
		require.Len(t, items.Items, 1)
		item := items.Items[0]
		require.Equal(t, f.Publication.Source.ProductKey, item.Source.ProductKey)
		selection := research.Selection{ItemID: item.ID, Revision: item.Revision, PublicationID: item.Source.PublicationID, Version: item.Source.Version}
		selected[name] = selection
		linkKeys[name] = uuid.NewString()
		receipt, err := svc.Execute(ctx, scope, fixture.ProjectID, linkKeys[name], research.Command{Action: "link", Selections: []research.Selection{selection}})
		require.NoError(t, err)
		require.Equal(t, []string{fixture.ConsumerCases.LinkP1ThenP2AndReplayP2.CandidateID}, receipt.CandidateIDs)
	}
	page, err := svc.List(ctx, scope, fixture.ProjectID, "")
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Len(t, page.Items[0].Evidence, 2)
	require.Equal(t, selected["P1"], page.Items[0].Observation.Ref.Selection)
	replay, err := svc.Execute(ctx, scope, fixture.ProjectID, linkKeys["P2"], research.Command{Action: "link", Selections: []research.Selection{selected["P2"]}})
	require.NoError(t, err)
	require.True(t, replay.Replayed)
	view := page.Items[0]
	require.Equal(t, "38.88", view.Observation.SHEIN.Price.Amount)
	require.Equal(t, "after coupon", *view.Observation.SHEIN.Price.PromotionNote)
	require.Equal(t, "1727", view.Observation.Query.CategoryRef)
	require.Empty(t, view.Observation.CapturedAt)
	_, err = svc.Execute(ctx, scope, fixture.ProjectID, uuid.NewString(), research.Command{Action: "link", Selections: []research.Selection{selected["P2"]}})
	require.NoError(t, err)
	page, err = svc.List(ctx, scope, fixture.ProjectID, "")
	require.NoError(t, err)
	require.Equal(t, view, page.Items[0], "a second command for the same evidence cannot change the candidate")

	wrong := selected["P1"]
	wrong.Version = selected["P2"].Version
	_, err = reader.Observe(ctx, scope, "", wrong)
	require.Error(t, err)
	for _, action := range []string{"review", "decide", "reset"} {
		intent := research.Command{Action: action, CandidateID: view.ID, ExpectedRevision: view.Revision, EvidenceDigest: view.EvidenceDigest}
		if action == "decide" {
			intent.State, intent.Reason = "APPROVED_FOR_TEST", "forged assessment"
		}
		key := uuid.NewString()
		_, err = svc.Execute(ctx, scope, fixture.ProjectID, key, intent)
		require.ErrorIs(t, err, research.ErrUnsupported)
		_, err = svc.Lookup(ctx, scope, fixture.ProjectID, key)
		require.ErrorIs(t, err, research.ErrNotFound)
	}
	analysis := []research.AnalysisChoice{{CandidateID: view.ID, ExpectedRevision: view.Revision, LinkID: view.SelectedLinkID, EvidenceDigest: view.EvidenceDigest, Selection: selected["P1"]},
		{CandidateID: uuid.NewString(), ExpectedRevision: 1, LinkID: uuid.NewString(), EvidenceDigest: strings.Repeat("b", 64), Selection: research.Selection{ItemID: "ffffffff-ffff-4fff-8fff-ffffffffffff", Revision: 1, PublicationID: uuid.NewString(), Version: 1}}}
	_, err = svc.InspectSelection(ctx, scope, fixture.ProjectID, analysis)
	require.ErrorIs(t, err, research.ErrUnsupported)
	p1, p2 := selected["P1"], selected["P2"]
	amazon := publishComparisonFixture(t, ctx, db, scope, access, cr)
	require.NotEqual(t, page.Items[0].Observation.Ref.Source.ProductKey, amazon.Source.ProductKey)
	_, _, err = (selection.Inspector{Collections: collections}).Prepare(ctx, fixture.ProjectID, []collection.SelectionInput{
		{ItemID: p1.ItemID, ExpectedRevision: p1.Revision, OriginalPublicationID: p1.PublicationID, OriginalVersion: p1.Version},
		{ItemID: amazon.ID, ExpectedRevision: amazon.Revision, OriginalPublicationID: amazon.Source.PublicationID, OriginalVersion: amazon.Source.Version},
	})
	require.ErrorIs(t, err, agent.ErrInvalid)
	require.False(t, collection.ListingTransferKind(collection.SourceSHEINUSData))

	// A synthetic preexisting decision verifies masking without opening a SHEIN review path.
	reason := "reason bound to the original P1"
	intent := research.Command{Action: "decide", CandidateID: view.ID, EvidenceDigest: view.EvidenceDigest, Reason: reason, State: "APPROVED_FOR_TEST", ExpectedRevision: view.Revision - 1}
	decision := research.Decision{State: intent.State, Reason: reason, EvidenceDigest: view.EvidenceDigest, Ref: view.Observation.Ref, Revision: view.Revision, At: time.Now()}
	intentJSON, err := json.Marshal(intent)
	require.NoError(t, err)
	decisionJSON, err := json.Marshal(decision)
	require.NoError(t, err)
	require.NoError(t, db.Exec("UPDATE research_opportunities SET state=?,reason=? WHERE id=?", intent.State, reason, view.ID).Error)
	require.NoError(t, db.Exec(`INSERT INTO research_commands (organization_id,actor_id,project_id,key,input_hash,candidate_id,intent_json,receipt_json,decision_json,created_at)
		VALUES (?,?,?,?,?,?,?,'{}',?,now())`, scope.OrganizationID, scope.ActorID, fixture.ProjectID, uuid.NewString(), collection.Digest(intent), view.ID, string(intentJSON), string(decisionJSON)).Error)
	page, err = svc.List(ctx, scope, fixture.ProjectID, "")
	require.NoError(t, err)
	require.Equal(t, reason, page.Items[0].Reason)
	require.Equal(t, reason, page.Items[0].History[0].Reason)
	p1Job := fixture.Publications["P1"].Job.JobID
	var original []byte
	require.NoError(t, db.Raw("SELECT evidence_json FROM data_acquisition_items WHERE job_id=?", p1Job).Row().Scan(&original))
	tampered := strings.Replace(string(original), "38.88", "39.88", 1)
	require.NotEqual(t, string(original), tampered)
	require.NoError(t, db.Exec("UPDATE data_acquisition_items SET evidence_json=? WHERE job_id=?", []byte(tampered), p1Job).Error)
	_, err = reader.Observe(ctx, scope, "", p1)
	require.ErrorIs(t, err, research.ErrUnavailable)
	require.NoError(t, db.Exec("UPDATE data_acquisition_items SET evidence_json=? WHERE job_id=?", original, p1Job).Error)
	require.NoError(t, db.Exec("UPDATE product_collection_items SET archived_at=now() WHERE id=?", p1.ItemID).Error)
	page, err = svc.List(ctx, scope, fixture.ProjectID, "")
	require.NoError(t, err)
	masked := page.Items[0]
	require.Equal(t, view.SelectedLinkID, masked.SelectedLinkID)
	require.Nil(t, masked.Observation)
	require.Empty(t, masked.Reason)
	require.Empty(t, masked.History[0].Reason)
	var newLink string
	for _, evidence := range masked.Evidence {
		if evidence.ID == view.SelectedLinkID {
			require.Nil(t, evidence.Observation)
		} else {
			require.NotNil(t, evidence.Observation)
			require.Equal(t, p2, evidence.Observation.Ref.Selection)
			newLink = evidence.ID
		}
	}
	require.NotEmpty(t, newLink)
	_, err = svc.Execute(ctx, scope, fixture.ProjectID, uuid.NewString(), research.Command{Action: "select", CandidateID: view.ID, ExpectedRevision: masked.Revision, LinkID: newLink})
	require.NoError(t, err)
	page, err = svc.List(ctx, scope, fixture.ProjectID, "")
	require.NoError(t, err)
	require.Equal(t, "36.50", page.Items[0].Observation.SHEIN.Price.Amount)
	stored, err := repo.Load(ctx, scope, fixture.ProjectID, view.ID)
	require.NoError(t, err)
	require.True(t, func() bool {
		for _, record := range stored.History {
			if record.Reason == reason && record.Ref.Selection == p1 {
				return true
			}
		}
		return false
	}(), "masking retains the original decision")
	t.Run("trial entry recovery", func(t *testing.T) {
		verifySHEINTrialRecovery(t, ctx, module, access)
	})
	t.Run("trial entry authorization lost after metering", func(t *testing.T) {
		verifySHEINTrialPublicationDenied(t, ctx, module, access, fixture.Publications["P1"])
	})
	access.denied = true
	_, err = reader.Observe(ctx, scope, "", p2)
	require.ErrorIs(t, err, research.ErrForbidden)
	access.projectDenied = true
	_, err = svc.List(ctx, scope, fixture.ProjectID, "")
	require.ErrorIs(t, err, research.ErrForbidden)
}

func publishComparisonFixture(t *testing.T, ctx context.Context, db *gorm.DB, scope collection.Scope, auth collection.Authorizer, repo *collectionstore.Repository) collection.Item {
	t.Helper()
	operation := uuid.NewString()
	envelope, err := collection.OwnEnvelope(operation, collection.OwnProduct{Title: "Independent comparison fixture"})
	require.NoError(t, err)
	batch, err := collection.NewPublicationBatch(scope, operation, "amazon_data", "Synthetic comparison")
	require.NoError(t, err)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		publisher, err := productsourcing.NewOwnProductWriter(tx, auth)
		if err != nil {
			return err
		}
		source, err := publisher.PublishOwn(ctx, scope, operation, envelope)
		if err != nil {
			return err
		}
		source.Kind, source.OperationID = "amazon_data", operation
		_, err = collectionstore.AppendDataPublication(ctx, tx, scope, batch, source, time.Now())
		return err
	}))
	items, err := repo.ListItems(ctx, scope, batch.ID, collection.Query{Limit: 10})
	require.NoError(t, err)
	require.Len(t, items.Items, 1)
	return items.Items[0]
}

// Use the two fixed fixture cards as synthetic responses to separate approved category trials.
func publishSHEINResearchTrial(t *testing.T, ctx context.Context, module *dataservicesapp.Module, access *sheinResearchAccess, f sheinResearchPublication) {
	t.Helper()
	var observation dataacquisition.SHEINUSObservation
	require.NoError(t, json.Unmarshal([]byte(f.ObservationUTF8), &observation))
	approval := dataacquisition.SHEINUSTrialApproval{
		ID: f.Job.JobID, Scope: f.Job.Scope, ProjectID: access.project,
		AccountRef: uuid.NewString(), KeyRef: uuid.NewString(), Funding: orgresource.FundingMember,
		ExpiresAt: time.Now().Add(time.Minute),
	}
	card := dataacquisition.SHEINUSTrialCard{
		Position: 1, GoodsID: observation.NativeGoodsID, Title: *observation.Title, URL: observation.SourceURL,
		Amount: observation.Price.Amount, Currency: observation.Price.Currency, PromotionNote: "after coupon",
	}
	sends := 0
	deps := sheinTrialDependencies{
		authority: sheinResearchTrialAuthority{approval},
		send: func(_ context.Context, a dataacquisition.SHEINUSTrialApproval, request dataacquisition.SHEINUSTrialRequest) (dataacquisition.SHEINUSTrialSubmission, error) {
			sends++
			require.Equal(t, "https://us.shein.com/Women-Dresses-c-1727.html", request.URL)
			credits := 5
			return dataacquisition.SHEINUSTrialSubmission{
				SHEINUSTrialUsage: dataacquisition.SHEINUSTrialUsage{ApprovalID: a.ID, Binding: a.Binding(), RequestID: a.ID, Credits: &credits},
				Cards:             []dataacquisition.SHEINUSTrialCard{card},
			}, nil
		},
		lookup: func(context.Context, dataacquisition.SHEINUSTrialApproval, dataacquisition.SHEINUSTrialRequest, dataacquisition.SHEINUSTrialAttempt) (dataacquisition.SHEINUSTrialUsage, error) {
			return dataacquisition.SHEINUSTrialUsage{}, dataacquisition.ErrNotFound
		},
	}
	server := sheinTrialTestHTTP(t, ctx, module, deps, sheinResearchProject{access}, map[string]collection.Scope{"researcher": f.Job.Scope})
	path := sheinTrialTestPath(approval.ProjectID, approval.ID)
	require.JSONEq(t, `{"code":"NOT_FOUND"}`, sheinTrialTestRequest(t, server, http.MethodGet, path, "researcher", "", http.StatusNotFound))
	require.Zero(t, sends)
	first := sheinTrialTestRequest(t, server, http.MethodPost, path+"/submit", "researcher", "", http.StatusOK)
	require.JSONEq(t, `{"approvalId":"`+approval.ID+`","requestId":"`+approval.ID+`","credits":5,"reservedCredits":0,"state":"CONFIRMED","publishedCount":1}`, first)
	require.JSONEq(t, first, sheinTrialTestRequest(t, server, http.MethodPost, path+"/submit", "researcher", "", http.StatusOK))
	require.Equal(t, 1, sends)
}
