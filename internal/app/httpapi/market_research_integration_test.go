package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"task-processor/internal/app/productsourcing"
	supplyhttp "task-processor/internal/app/supplychain/httpapi"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	prepstore "task-processor/internal/integration/persistence/listing/preparation"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	researchstore "task-processor/internal/integration/persistence/product/research"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/research"
	"task-processor/internal/product/sourcing"
	"testing"
	"time"
)

func TestResearchHTTPUsesOriginalConfigurationToolLedgerAndRecovery(t *testing.T) {
	testProductAgentOwners(t, "research")
}

func TestResearchHTTPReferenceZeroUsageKeepsOriginalReadonlyJourney(t *testing.T) {
	testProductAgentOwners(t, "research zero")
}

// Only the upstream project/provenance adapters are synthetic here. The new
// research repository, Collection, bounded inspector, tool audit, Agent runtime,
// model HTTP adapter, point ledger and configuration owner are their real code.
type analysisProjectsFixture struct {
	project string
	scope   collection.Scope
}

func (p analysisProjectsFixture) Authorize(ctx context.Context, s collection.Scope, id string, _ bool) error {
	if s != p.scope || id != p.project {
		return research.ErrNotFound
	}
	return nil
}
func (p analysisProjectsFixture) AuthorizeAnalysis(ctx context.Context, s collection.Scope, id string, b bool) error {
	return p.Authorize(ctx, s, id, b)
}

type analysisSourcesFixture struct {
	observations map[string]research.Observation
	denied       bool
}

func (s *analysisSourcesFixture) Observe(_ context.Context, _ collection.Scope, in research.Selection) (research.Observation, error) {
	if s.denied {
		return research.Observation{}, research.ErrForbidden
	}
	v, ok := s.observations[in.ItemID]
	if !ok || v.Ref.Selection != in {
		return v, research.ErrNotFound
	}
	return v, nil
}
func testResearchHTTPThroughOwners(t *testing.T, f *acquisitionHTTPFixture, parent *productAgentApplication, deps routeAuthDependencies, permissions *authz.ListingKitAuthorizer, calls *atomic.Int32) {
	t.Helper()
	ctx := context.Background()
	live := &productReviewLiveOrganizationAccess{resolver: deps.organizationResolver, now: time.Now}
	authority, e := collection.NewContextAuthorizer(live, permissions)
	require.NoError(t, e)
	require.NoError(t, collectionstore.InstallSchema(f.owner))
	repo, e := collectionstore.NewRepository(ctx, f.owner, func(tx *gorm.DB) (collectionstore.OwnPublisher, error) {
		return productsourcing.NewOwnProductWriter(tx, authority)
	})
	require.NoError(t, e)
	sources, e := productsourcing.NewPublishedAcquisitionReader(ctx, f.owner, live, permissions)
	require.NoError(t, e)
	collections, e := collection.NewService(repo, authority, sources)
	require.NoError(t, e)
	snapshots, e := catalogstore.NewBoundedSnapshotReader(f.owner, sourcing.MaxEncodedSnapshotBytes)
	require.NoError(t, e)
	collections.WithSnapshots(snapshots)
	actor := authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: "B", EffectiveOrganizationID: "B", UserID: "admin", EffectiveMemberID: "member-B-admin", Roles: []string{"listingkit_admin"}, TokenExpiresAt: time.Now().Add(time.Minute)})
	actor, e = (productReviewCapabilityBinder{}).Bind(actor, "Bearer admin")
	require.NoError(t, e)
	scope := collection.Scope{OrganizationID: "B", ActorID: "admin", MemberID: "member-B-admin"}
	project := uuid.NewString()
	require.NoError(t, researchstore.InstallSchema(f.owner))
	researchRepo, e := researchstore.New(ctx, f.owner)
	require.NoError(t, e)
	provenance := &analysisSourcesFixture{observations: map[string]research.Observation{}}
	service := research.New(researchRepo, analysisProjectsFixture{project, scope}, provenance)
	// The offline producer publishes fixed sample bodies through the original
	// Product transaction. Collection membership/source kinds remain real, so
	// the final handoff exercises the same references as the research analysis.
	batch, e := collection.NewPublicationBatch(scope, uuid.NewString(), "amazon_data", "Offline research samples")
	require.NoError(t, e)
	selections := []research.Selection{}
	for n := 0; n < 2; n++ {
		op := uuid.NewString()
		envelope, e := collection.OwnEnvelope(op, collection.OwnProduct{Title: fmt.Sprintf("Saved sample %d", n)})
		require.NoError(t, e)
		var itemID string
		require.NoError(t, f.owner.Transaction(func(tx *gorm.DB) error {
			publisher, err := productsourcing.NewOwnProductWriter(tx, authority)
			if err != nil {
				return err
			}
			source, err := publisher.PublishOwn(actor, scope, op, envelope)
			if err != nil {
				return err
			}
			source.Kind, source.OperationID = "amazon_data", op
			itemID, err = collectionstore.AppendDataPublication(actor, tx, scope, batch, source, time.Now())
			return err
		}))
		item, e := collections.ReadItem(actor, itemID)
		require.NoError(t, e)
		selection := research.Selection{ItemID: item.Item.ID, Revision: item.Item.Revision, PublicationID: item.Item.Source.PublicationID, Version: item.Item.Source.Version}
		selections = append(selections, selection)
		source := item.Item.Source
		price := 19.99 + float64(n)
		provenance.observations[selection.ItemID] = research.Observation{Item: item.Item, Ref: research.EvidenceRef{Selection: selection, Source: source, JobID: batch.OperationID, Checksum: strings.Repeat("c", 64)}, BatchID: batch.ID, Site: "US", ASIN: fmt.Sprintf("B%09d", n), Title: item.Item.Title, CapturedAt: "2026-11-02T00:00:00Z", ParserVersion: "synthetic-v1", Query: research.QuerySummary{Mode: "keyword", Keyword: "sample", Limit: 2, Fields: []string{"title", "price"}}, Price: &price, Currency: "USD", PriceBasis: "displayed_listing_offer"}
	}
	_, e = service.Execute(actor, scope, project, uuid.NewString(), research.Command{Action: "link", Selections: selections})
	require.NoError(t, e)
	page, e := service.List(actor, scope, project, "")
	require.NoError(t, e)
	require.Len(t, page.Items, 2)
	in := researchInput{ProjectID: project}
	for _, c := range page.Items {
		o := c.Observation
		require.NotNil(t, o)
		in.Candidates = append(in.Candidates, researchChoice{CandidateID: c.ID, ExpectedRevision: strconv.FormatInt(c.Revision, 10), LinkID: c.SelectedLinkID, EvidenceDigest: c.EvidenceDigest, Selection: selectionChoice{ItemID: o.Ref.Selection.ItemID, ExpectedRevision: strconv.FormatInt(o.Ref.Selection.Revision, 10), OriginalPublicationID: o.Ref.Selection.PublicationID, OriginalVersion: strconv.FormatUint(o.Ref.Selection.Version, 10)}})
	}
	a, e := buildMarketResearchApplication(ctx, parent, collections, service, f.owner, deps.organizationResolver, permissions)
	require.NoError(t, e)
	noModel, e := buildMarketResearchApplication(ctx, nil, collections, service, f.owner, deps.organizationResolver, permissions)
	require.NoError(t, e)
	_, facts, e := noModel.inspectInput(actor, in)
	require.NoError(t, e)
	require.Len(t, facts.Candidates, 2)
	require.Nil(t, noModel.runtime)
	require.Zero(t, calls.Load())
	routes := marketResearchAnalysisRoutes(a, a.bindRequest)
	require.NoError(t, prepstore.InstallSchema(f.owner))
	core, e := composePreparationCore(ctx, f.owner, deps, permissions, collections)
	require.NoError(t, e)
	routes = append(routes, supplyhttp.PreparationRoutes(core.app, (productReviewCapabilityBinder{}).Bind)...)
	app := buildCurrentApplicationHTTPServer(routes, deps)
	server := httptest.NewServer(app.Handler)
	defer server.Close()
	base := strings.ReplaceAll(marketResearchAnalysisBase, ":project_id", project)
	call := func(method, path, key string, body any, status int) []byte {
		t.Helper()
		raw := ""
		if body != nil {
			raw = string(mustSelectionJSON(t, body))
		}
		code, data, e := acquisitionHTTPRequest(server, method, path, "admin", "B", key, raw)
		require.NoError(t, e)
		require.Equal(t, status, code, string(data))
		return data
	}
	call("POST", base+"/inspect", "", in, http.StatusOK)
	require.Zero(t, calls.Load())
	_, e = a.configuration.Execute(ctx, agentconfig.Command{Scope: agent.Scope{OrganizationID: "B", ActorID: "admin"}, Key: uuid.NewString(), AgentID: agentconfig.ResearchAgentID, Operation: "enable", Absent: true})
	require.NoError(t, e)
	key := uuid.NewString()
	prepare := researchPrepareInput{researchInput: in, RequestKey: key, ExpectedModeRevision: "0", GoalSummary: "Compare selected saved observations"}
	prepared := call("POST", base+"/prepare", key, prepare, http.StatusOK)
	var dto struct {
		ConfigurationSnapshotRef agent.ConfigurationSnapshotRef `json:"configurationSnapshotRef"`
		ReferencePriceVersion    string                         `json:"referencePriceVersion"`
	}
	require.NoError(t, json.Unmarshal(prepared, &dto))
	require.NotEmpty(t, dto.ReferencePriceVersion)
	start := materialsStartBody{ConfigurationSnapshotRef: dto.ConfigurationSnapshotRef}
	oldProfile := a.profile
	a.profile = func(context.Context, string) (aicapability.ModelProfile, error) {
		return aicapability.ModelProfile{}, agent.ErrUnavailable
	}
	require.JSONEq(t, string(prepared), string(call("POST", base+"/prepare", key, prepare, http.StatusOK)))
	require.JSONEq(t, string(prepared), string(call("GET", base+"/preparations/"+key, "", nil, http.StatusOK)))
	require.Zero(t, calls.Load())
	a.profile = oldProfile
	mismatch := prepare
	mismatch.GoalSummary = "different intent"
	call("POST", base+"/prepare", key, mismatch, http.StatusConflict)
	raw := call("POST", base+"/runs", key, start, http.StatusOK)
	var result researchRunDTO
	require.NoError(t, json.Unmarshal(raw, &result))
	require.True(t, result.Validated, string(raw))
	require.Equal(t, dto.ReferencePriceVersion, result.ReferencePriceVersion)
	require.Len(t, result.Inspection.Candidates, 2)
	require.EqualValues(t, 2, calls.Load())
	require.Equal(t, agent.HumanReviewRequired, result.Phase)
	before, e := core.app.Preparations.List(actor, collection.Query{Limit: 20})
	require.NoError(t, e)
	require.Empty(t, before.Items, "read-only advice must not transfer products")
	require.JSONEq(t, string(raw), string(call("POST", base+"/runs", key, start, http.StatusOK)))
	require.EqualValues(t, 2, calls.Load())
	recent := call("GET", base+"/runs", "", nil, http.StatusOK)
	require.Contains(t, string(recent), result.RunID)
	require.NotContains(t, string(recent), "Sample")
	require.NotContains(t, string(recent), "evidenceDigest")
	// A later human decision changes the candidate revision but not the saved AI observation.
	c := page.Items[0]
	_, e = service.Execute(actor, scope, project, uuid.NewString(), research.Command{Action: "review", CandidateID: c.ID, ExpectedRevision: c.Revision, EvidenceDigest: c.EvidenceDigest})
	require.NoError(t, e)
	_, e = service.Execute(actor, scope, project, uuid.NewString(), research.Command{Action: "decide", CandidateID: c.ID, ExpectedRevision: c.Revision + 1, EvidenceDigest: c.EvidenceDigest, State: "APPROVED_FOR_TEST", Reason: "human confirmed saved facts"})
	require.NoError(t, e)
	// Reload the same project before the user confirms its precise approved
	// subset in the original Collection -> Preparation flow. No Agent action
	// performs this write and the other candidate remains unselected.
	reloaded, e := researchstore.New(ctx, f.owner)
	require.NoError(t, e)
	current, e := research.New(reloaded, analysisProjectsFixture{project, scope}, provenance).List(actor, scope, project, "")
	require.NoError(t, e)
	var chosen research.CandidateView
	for _, candidate := range current.Items {
		if candidate.ID == c.ID {
			chosen = candidate
		} else {
			require.Equal(t, "TO_RESEARCH", candidate.State)
		}
	}
	require.Equal(t, "APPROVED_FOR_TEST", chosen.State)
	require.Equal(t, "human confirmed saved facts", chosen.Reason)
	require.NotNil(t, chosen.Observation)
	currentBatch, e := collections.ReadBatch(actor, chosen.Observation.BatchID)
	require.NoError(t, e)
	require.True(t, currentBatch.SupplyTransferSupported)
	transferInput := collection.BatchSelectionInput{BatchID: currentBatch.ID, ExpectedRevision: currentBatch.Revision, ItemIDs: []string{chosen.Observation.Ref.Selection.ItemID}}
	transferKey := uuid.NewString()
	var transferred, replay preparation.TransferReceipt
	require.NoError(t, json.Unmarshal(call("POST", supplyhttp.SupplyBasePath+"/transfer", transferKey, transferInput, http.StatusOK), &transferred))
	require.NoError(t, json.Unmarshal(call("POST", supplyhttp.SupplyBasePath+"/transfer", transferKey, transferInput, http.StatusOK), &replay))
	require.True(t, replay.Replayed)
	require.Equal(t, transferred.Preparation.ID, replay.Preparation.ID)
	var preparedSources collection.Page[preparation.SourceItem]
	require.NoError(t, json.Unmarshal(call("GET", supplyhttp.SupplyBasePath+"/"+transferred.Preparation.ID+"/sources", "", nil, http.StatusOK), &preparedSources))
	require.Len(t, preparedSources.Items, 1)
	require.Equal(t, chosen.Observation.Item.Source, preparedSources.Items[0].Source)
	require.Equal(t, transferInput.ItemIDs[0], preparedSources.Items[0].CollectionItemID)
	after, e := core.app.Preparations.List(actor, collection.Query{Limit: 20})
	require.NoError(t, e)
	require.Len(t, after.Items, 1)
	require.EqualValues(t, 2, calls.Load(), "human choice and transfer never dispatch another model call")
	require.JSONEq(t, string(raw), string(call("GET", base+"/runs/"+key, "", nil, http.StatusOK)))
	require.JSONEq(t, string(prepared), string(call("POST", base+"/prepare", key, prepare, http.StatusOK)))
	call("POST", base+"/inspect", "", in, http.StatusConflict)
	provenance.denied = true
	for _, path := range []string{"/preparations/" + key, "/runs/" + key} {
		masked := call("GET", base+path, "", nil, http.StatusForbidden)
		require.Contains(t, string(masked), `"operationMayExist":true`)
		require.NotContains(t, string(masked), "configurationSnapshotRef")
		require.NotContains(t, string(masked), "inspection")
	}
	masked := call("POST", base+"/prepare", key, prepare, http.StatusForbidden)
	require.Contains(t, string(masked), `"operationMayExist":true`)
	hidden := call("GET", base+"/runs", "", nil, http.StatusOK)
	require.NotContains(t, string(hidden), result.RunID)
	require.EqualValues(t, 2, calls.Load())
	// This is the same original frozen configuration, not a new history owner.
	frozen, e := a.configuration.LoadSnapshot(ctx, agent.Scope{OrganizationID: "B", ActorID: "admin"}, dto.ConfigurationSnapshotRef)
	require.NoError(t, e)
	require.Equal(t, agentconfig.ResearchAgentID, frozen.AgentID)
	require.Equal(t, 2, frozen.Request.Limits.ModelCalls)
	_, _, e = a.configuration.LoadResearchPreparationByKey(ctx, frozen.Scope, "other-member", project, key)
	require.ErrorIs(t, e, agentconfig.ErrForbidden)
}
