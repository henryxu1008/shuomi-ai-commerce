package httpapi

// Isolated #1458 integration: the real trial, publication and comparison owners
// run with local response, identity and workflow-startup substitutes.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"golang.org/x/net/html"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"task-processor/internal/agent"
	pc "task-processor/internal/aiworkbench/projectcenter"
	dataservicesapp "task-processor/internal/app/dataservices"
	"task-processor/internal/authidentity"
	datahttp "task-processor/internal/dataservice/httpapi"
	"task-processor/internal/httproute"
	"task-processor/internal/integration/acquisition/amazon"
	resourceadapter "task-processor/internal/integration/orgresource"
	projectstore "task-processor/internal/integration/persistence/aiworkbench/projectcenter"
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
	"task-processor/internal/product/research/analysis"
	rh "task-processor/internal/product/research/httpapi"
	"task-processor/internal/product/selection"
	"task-processor/internal/product/sourcing"
)

const zenRowsSpikeKeyword = "offline sample lamp"

func TestAmazonFirstPageRouteRequiresExplicitAdmission(t *testing.T) {
	var routes []httproute.Descriptor
	for _, r := range currentWorkbenchApplicationRoutes {
		routes = append(routes, httproute.Descriptor{Method: r.Method, Path: r.Path})
	}
	routes = append(routes, rh.Routes(nil)...)
	routes = append(routes, marketResearchAnalysisRoutes(nil, nil)...)
	routes = append(routes, datahttp.BuildRoutes(datahttp.Dependencies{})...)
	base := currentApplicationOptionalRoutes{MarketResearch: true, DataServices: true}
	validate := func(routes []httproute.Descriptor, options currentApplicationOptionalRoutes) error {
		return validateCurrentApplicationRoutesInternal(routes, false, false, false, false, false, false, false, options)
	}
	require.NoError(t, validate(routes, base))
	withTrial := append(routes, amazonFirstPageRoutes(nil)...)
	require.Error(t, validate(withTrial, base))
	base.AmazonFirstPage = true
	require.Error(t, validate(routes, base))
	require.NoError(t, validate(withTrial, base))
	withTrial[len(withTrial)-1].AuthPolicy = httproute.AuthPolicyPublic
	require.Error(t, validate(withTrial, base))
}

type offlineZenRowsPage struct {
	server       *httptest.Server
	starterError error
	starts       int
	browserCalls int
}

type offlineZenRowsProjects struct {
	projects researchProjects
	scope    collection.Scope
}

func (p offlineZenRowsProjects) Authorize(ctx context.Context, scope collection.Scope, project string, active bool) error {
	if scope != p.scope {
		return research.ErrForbidden
	}
	return p.projects.requireResearchProject(ctx, pc.Scope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}, project, active)
}

func (p offlineZenRowsProjects) AuthorizeAnalysis(ctx context.Context, scope collection.Scope, project string, active bool) error {
	return p.Authorize(ctx, scope, project, active)
}

func (p *offlineZenRowsPage) EnsureExecution(context.Context, dataacquisition.Job) error {
	p.starts++
	return p.starterError
}
func (p *offlineZenRowsPage) Ready(context.Context) error { return nil }
func (p *offlineZenRowsPage) Sites() []dataacquisition.Site {
	site, _ := dataacquisition.ResolveSite("us")
	return []dataacquisition.Site{site}
}
func (p *offlineZenRowsPage) Funding(context.Context, collection.Scope) (orgresource.ResourceFunding, error) {
	return orgresource.FundingEnterprise, nil
}
func (p *offlineZenRowsPage) CurrentDataRowPrice(context.Context) (dataacquisition.PriceSnapshot, error) {
	return dataacquisition.PriceSnapshot{OfferID: "offline-data-row", Version: "v1", Currency: "CNY"}, nil
}
func (p *offlineZenRowsPage) Discover(context.Context, dataacquisition.Query) ([]string, error) {
	p.browserCalls++
	return nil, dataacquisition.ErrUnavailable
}
func (p *offlineZenRowsPage) Fetch(context.Context, string, string) (dataacquisition.Evidence, error) {
	p.browserCalls++
	return dataacquisition.Evidence{}, dataacquisition.ErrUnavailable
}
func (p *offlineZenRowsPage) readPage(ctx context.Context, q dataacquisition.Query) ([]dataacquisition.Evidence, error) {
	if q.Site != "us" || q.Mode != "keyword" || q.Keyword != zenRowsSpikeKeyword || q.CategoryNode != "" {
		return nil, dataacquisition.ErrForbidden
	}
	source := "https://www.amazon.com/s?" + url.Values{"k": {q.Keyword}, "page": {"1"}}.Encode()
	target := p.server.URL + "/v1/?" + url.Values{"url": {source}, "extract": {"auto"}, "proxy_country": {"us"}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.server.Client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, dataacquisition.ErrSourceUnsupported
	}
	const limit = 1 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil || len(body) > limit {
		return nil, dataacquisition.ErrSourceUnsupported
	}
	ids, facts, err := decodeOfflineSearchPage(body, q.Limit, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	out := make([]dataacquisition.Evidence, 0, len(ids))
	for _, id := range ids {
		out = append(out, facts[id])
	}
	return out, nil
}

type offlineSearchCard struct {
	ASIN      string   `json:"asin"`
	Title     string   `json:"title"`
	Price     *float64 `json:"price"`
	Sponsored bool     `json:"is_sponsored"`
}

type offlineSearchPage struct {
	HTML   string `json:"html"`
	Parsed struct {
		Results    []offlineSearchCard `json:"results"`
		Pagination struct {
			NextPageURL string `json:"next_page_url"`
		} `json:"pagination"`
	} `json:"parsed"`
}

func decodeOfflineSearchPage(body []byte, limit int, captured time.Time) ([]string, map[string]dataacquisition.Evidence, error) {
	var page offlineSearchPage
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, nil, err
	}
	ids, err := amazon.ParseDiscovery(page.HTML, limit)
	if err != nil {
		return nil, nil, err
	}
	doc, err := html.Parse(strings.NewReader(page.HTML))
	if err != nil {
		return nil, nil, err
	}
	images := map[string]string{}
	for node := range doc.Descendants() {
		if node.Type != html.ElementNode || node.Data != "img" || offlineAttribute(node, "class") != "s-image" {
			continue
		}
		for card := node.Parent; card != nil; card = card.Parent {
			if offlineAttribute(card, "data-component-type") == "s-search-result" {
				asin := offlineAttribute(card, "data-asin")
				if _, seen := images[asin]; !seen {
					images[asin] = offlineAttribute(node, "src")
				}
				break
			}
		}
	}
	cards := map[string]offlineSearchCard{}
	for _, card := range page.Parsed.Results {
		if _, seen := cards[card.ASIN]; !seen && !card.Sponsored {
			cards[card.ASIN] = card
		}
	}
	facts := make(map[string]dataacquisition.Evidence, len(ids))
	validIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		card := cards[id]
		fact := dataacquisition.Evidence{
			Site: "us", ASIN: id, Title: card.Title, MainImage: images[id],
			ParserVersion: dataacquisition.AmazonZenRowsSearchVersion, CapturedAt: captured.Format(time.RFC3339Nano),
			// A search result alone does not establish stock or offer currency.
			Availability: "unknown",
			Missing:      []string{"availability", "price", "currency", "description", "brand", "attributes", "variants", "rating", "reviewCount"},
		}
		if fact.Validate() == nil {
			facts[id] = fact
			validIDs = append(validIDs, id)
		}
	}
	return validIDs, facts, nil
}

func offlineAttribute(node *html.Node, name string) string {
	for _, attr := range node.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}
	return ""
}

func startOfflineZenRows(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var page offlineSearchPage
	var raw strings.Builder
	raw.WriteString(`<div class="s-main-slot">`)
	for _, n := range []int{1, 2, 1, 3, 9, 4, 5, 6} {
		asin := fmt.Sprintf("B%09d", n)
		card := offlineSearchCard{ASIN: asin, Title: fmt.Sprintf("Synthetic lamp %d", n), Sponsored: n == 9}
		if n == 1 {
			price := 19.95
			card.Price = &price
		}
		page.Parsed.Results = append(page.Parsed.Results, card)
		fmt.Fprintf(&raw, `<div data-component-type="s-search-result" data-asin="%s"><img class="s-image" src="https://m.media-amazon.com/images/I/sample-%d.jpg">`, asin, n)
		if card.Sponsored {
			raw.WriteString(`<span class="puis-sponsored-label-text">Sponsored</span>`)
		}
		raw.WriteString("</div>")
	}
	raw.WriteString("</div>")
	page.HTML = raw.String()
	page.Parsed.Pagination.NextPageURL = "/s?k=offline+sample+lamp&page=2"
	payload, err := json.Marshal(page)
	require.NoError(t, err)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/v1/" ||
			r.URL.Query().Get("url") != "https://www.amazon.com/s?k=offline+sample+lamp&page=1" ||
			r.URL.Query().Get("proxy_country") != "us" || r.URL.Query().Get("extract") != "auto" {
			http.Error(w, "expected the first Amazon US keyword page", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

func openZenRowsSpikePostgres(t *testing.T, ctx context.Context, c *tcpostgres.PostgresContainer, name string) *gorm.DB {
	t.Helper()
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	address, err := url.Parse(dsn)
	require.NoError(t, err)
	address.Path = "/" + name
	db, err := gorm.Open(postgres.Open(address.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	return db
}

func TestZenRowsAmazonUSFirstPageOriginalSaveAndComparison(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	c, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("spike_product"), tcpostgres.WithUsername("test_owner"), tcpostgres.WithPassword("test-only-zenrows"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, c.Terminate(context.Background())) })
	db := openZenRowsSpikePostgres(t, ctx, c, "spike_product")
	for _, install := range []func(*gorm.DB) error{keystore.InstallSchema, keystore.InstallCustomSchema, catalogstore.AutoMigrate, sourcestore.InstallSchema, collectionstore.InstallSchema, jobstore.InstallSchema, researchstore.InstallSchema} {
		require.NoError(t, install(db))
	}
	require.NoError(t, db.Exec("CREATE DATABASE spike_resource").Error)
	require.NoError(t, db.Exec("CREATE DATABASE spike_project").Error)
	resourceDB := openZenRowsSpikePostgres(t, ctx, c, "spike_resource")
	projectDB := openZenRowsSpikePostgres(t, ctx, c, "spike_project")
	require.NoError(t, resourceadapter.AutoMigrate(resourceDB))
	require.NoError(t, projectstore.InstallSchema(projectDB))
	require.NoError(t, resourceDB.Exec(`INSERT INTO saas_organization_resource_buckets
		(organization_id,resource_type,available,allocated,reserved,consumed,created_at,updated_at)
		VALUES ('offline-org','data_row',10,0,0,0,now(),now())`).Error)
	resourceRepo, err := resourceadapter.NewGormConsumerChargeRepository(resourceDB, resourceadapter.TransactionConfig{})
	require.NoError(t, err)

	scope := collection.Scope{OrganizationID: "offline-org", ActorID: "offline-actor", MemberID: "offline-member"}
	ctx = authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{
		TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID,
		EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute),
	})
	projectRepo, err := projectstore.New(projectDB)
	require.NoError(t, err)
	projectScope := pc.Scope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}
	projectService := &pc.Service{Store: projectRepo, Authorize: func(_ context.Context, candidate pc.Scope, _ bool) error {
		if candidate != projectScope {
			return pc.ErrForbidden
		}
		return nil
	}}
	projectReceipt, err := projectService.Execute(ctx, projectScope, uuid.NewString(), pc.Command{Operation: "create", Fields: &pc.Fields{
		Title: "Synthetic Amazon US first page", Goal: "Compare saved samples", Kind: "PRODUCT_RESEARCH",
	}})
	require.NoError(t, err)
	project := projectReceipt.ID
	access := &sheinResearchAccess{scope: scope, project: project}
	server, pageRequests := startOfflineZenRows(t)
	provider := &offlineZenRowsPage{server: server}
	module, err := dataservicesapp.NewModule(ctx, dataservicesapp.Dependencies{
		ProductDB: db, Access: access, Live: access, Specialist: access, ExecutionEnabled: true,
		Prices: provider, Funding: provider, Provider: provider, Starter: provider,
		Charges: func(owner orgresource.ConsumerChargeOwner) (dataacquisition.Charges, error) {
			return orgresource.NewConsumerChargeService(resourceRepo, map[orgresource.ResourceConsumer]orgresource.ConsumerChargeOwner{
				orgresource.ConsumerAmazonData: owner,
			})
		},
	})
	require.NoError(t, err)

	principal := dataacquisition.Principal{Scope: scope}
	approval := dataacquisition.AmazonFirstPageApproval{
		ID: uuid.NewString(), Scope: scope, ProjectID: project, Keyword: zenRowsSpikeKeyword,
		AccountRef: uuid.NewString(), KeyDigest: strings.Repeat("a", 64), BudgetRef: uuid.NewString(),
		BudgetExpiresAt: time.Now().Add(time.Minute), ExpiresAt: time.Now().Add(time.Minute),
		MaximumCredits: 25, AvailableCredits: 25, NoOverageCharges: true, Funding: orgresource.FundingEnterprise,
		Price: dataacquisition.PriceConfirmation{Version: "v1"},
	}
	projectReader := offlineZenRowsProjects{projects: researchProjects{service: projectService}, scope: scope}
	send := func(ctx context.Context, a dataacquisition.AmazonFirstPageApproval) dataacquisition.AmazonFirstPageSubmission {
		facts, err := provider.readPage(ctx, a.Query())
		require.NoError(t, err)
		return dataacquisition.AmazonFirstPageSubmission{Receipt: dataacquisition.AmazonFirstPageReceipt{
			ApprovalID: a.ID, State: dataacquisition.AmazonFirstPageUnknown, RequestID: "offline-request-1",
		}, Evidence: facts}
	}
	trial, err := module.AmazonFirstPageTrial(approval, projectReader.Authorize, send)
	require.NoError(t, err)
	_, err = trial.Read(ctx, principal, project, approval.ID)
	require.ErrorIs(t, err, dataacquisition.ErrNotFound)
	_, err = trial.Submit(ctx, principal, uuid.NewString(), approval.ID)
	require.ErrorIs(t, err, dataacquisition.ErrForbidden)
	require.Zero(t, pageRequests.Load())
	provider.starterError = dataacquisition.ErrUnavailable
	_, err = trial.Submit(ctx, principal, project, approval.ID)
	require.ErrorIs(t, err, dataacquisition.ErrUnknown)
	receipt, err := trial.Read(ctx, principal, project, approval.ID)
	require.NoError(t, err)
	require.Equal(t, dataacquisition.AmazonFirstPagePrepared, receipt.State)
	require.Nil(t, receipt.Credits)
	require.Equal(t, 1, provider.starts)
	require.EqualValues(t, 1, pageRequests.Load())
	provider.starterError = nil
	restartedTrial, err := module.AmazonFirstPageTrial(approval, projectReader.Authorize, send)
	require.NoError(t, err)
	replay, err := restartedTrial.Submit(ctx, principal, project, approval.ID)
	require.NoError(t, err)
	require.Equal(t, receipt, replay)
	require.Equal(t, 2, provider.starts)

	runner := module.Runner()
	require.NoError(t, runner.Run(ctx, scope, receipt.JobID))
	job, err := runner.Read(ctx, principal, receipt.JobID)
	require.NoError(t, err)
	require.Equal(t, "SUCCEEDED", job.State)
	require.Equal(t, 5, job.Saved)
	require.EqualValues(t, 1, pageRequests.Load())

	require.NoError(t, runner.Run(ctx, scope, receipt.JobID))
	require.EqualValues(t, 1, pageRequests.Load())
	require.Zero(t, provider.browserCalls)
	var consumed int
	require.NoError(t, resourceDB.Raw("SELECT consumed FROM saas_organization_resource_buckets WHERE organization_id=? AND resource_type='data_row'", scope.OrganizationID).Scan(&consumed).Error)
	require.Equal(t, 5, consumed)

	collectionsRepo, err := collectionstore.NewRepository(ctx, db, func(*gorm.DB) (collectionstore.OwnPublisher, error) { return nil, collection.ErrUnavailable })
	require.NoError(t, err)
	collections, err := collection.NewService(collectionsRepo, access, access)
	require.NoError(t, err)
	snapshots, err := catalogstore.NewBoundedSnapshotReader(db, sourcing.MaxEncodedSnapshotBytes)
	require.NoError(t, err)
	collections.WithSnapshots(snapshots)
	items, err := collectionsRepo.ListItems(ctx, scope, job.BatchID, collection.Query{Limit: 10})
	require.NoError(t, err)
	require.Len(t, items.Items, 5)

	researchRepo, err := researchstore.New(ctx, db)
	require.NoError(t, err)
	researchService := research.New(researchRepo, projectReader, researchSources{collections: collections, data: module})
	selections := make([]research.Selection, 0, len(items.Items))
	for _, item := range items.Items {
		require.Equal(t, "amazon_data", item.Source.Kind)
		selections = append(selections, research.Selection{ItemID: item.ID, Revision: item.Revision, PublicationID: item.Source.PublicationID, Version: item.Source.Version})
	}
	_, err = researchService.Execute(ctx, scope, project, uuid.NewString(), research.Command{Action: "link", Selections: selections})
	require.NoError(t, err)
	page, err := researchService.List(ctx, scope, project, "")
	require.NoError(t, err)
	require.Len(t, page.Items, 5)
	choices := make([]research.AnalysisChoice, 0, len(page.Items))
	var asins []string
	for _, c := range page.Items {
		require.NotNil(t, c.Observation)
		require.Equal(t, job.ID, c.Observation.Ref.JobID)
		require.Equal(t, zenRowsSpikeKeyword, c.Observation.Query.Keyword)
		require.Equal(t, dataacquisition.AmazonZenRowsSearchVersion, c.Observation.ParserVersion)
		require.Nil(t, c.Observation.Price, "the search sample does not establish a stocked offer")
		asins = append(asins, c.Observation.ASIN)
		choices = append(choices, research.AnalysisChoice{CandidateID: c.ID, ExpectedRevision: c.Revision, LinkID: c.SelectedLinkID, EvidenceDigest: c.EvidenceDigest, Selection: c.Observation.Ref.Selection})
	}
	require.ElementsMatch(t, []string{"B000000001", "B000000002", "B000000003", "B000000004", "B000000005"}, asins)
	inspector := analysis.Inspector{Research: researchService, Products: selection.Inspector{Collections: collections}}
	for _, count := range []int{2, 3, 4, 5} {
		_, result, err := inspector.Prepare(ctx, project, choices[:count])
		require.NoError(t, err)
		require.Len(t, result.Candidates, count)
		require.Equal(t, count, result.Summary.Unpriced)
		for _, candidate := range result.Candidates {
			require.NotEmpty(t, candidate.Observation.Checksum)
			require.Contains(t, candidate.Missing, "observation.price")
			require.NotEmpty(t, candidate.Facts)
		}
	}
	_, _, err = inspector.Prepare(ctx, project, choices[:1])
	require.ErrorIs(t, err, agent.ErrInvalid)
	_, _, err = inspector.Prepare(ctx, project, append(choices, choices[0]))
	require.ErrorIs(t, err, agent.ErrInvalid)
	_, _, err = inspector.Prepare(ctx, uuid.NewString(), choices[:2])
	require.ErrorIs(t, err, research.ErrNotFound)
	require.EqualValues(t, 1, pageRequests.Load())
}
