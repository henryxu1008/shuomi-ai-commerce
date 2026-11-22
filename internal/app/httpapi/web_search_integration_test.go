package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/websearch"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/aiworkbench"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/aicapability/bravesearch"
	resourceadapter "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/orgresource"
	workstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/aiworkbench"
	channelstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/channelcredential"
	websearchstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/websearch"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/workbenchcontext"
)

type searchFixtureGrants struct{ revoked, revokeChat atomic.Bool }

func (g *searchFixtureGrants) Load(_ context.Context, source workbenchcontext.GrantSource, r workbenchcontext.GrantRequest) (workbenchcontext.GrantResult, error) {
	out := workbenchcontext.GrantResult{Source: source}
	for _, org := range []string{"A", "B"} {
		roles := searchFixtureRoles(org, r.Subject, g.revoked.Load())
		out.Grants = append(out.Grants, authidentity.OrganizationGrant{AuthorizationID: "member-" + org + "-" + r.Subject, OrganizationID: org, ProjectID: "project", Roles: roles})
	}
	return out, nil
}

func (*searchFixtureGrants) Invalidate(string, string) {}
func (g *searchFixtureGrants) RoleModules(_ context.Context, org string, keys []string) (map[string][]string, error) {
	out := map[string][]string{}
	for _, key := range keys {
		switch key {
		case authz.EnterpriseRoleKey(org, 1):
			out[key] = []string{"ai-channels", "ai-credentials"}
		case authz.EnterpriseRoleKey(org, 2):
			out[key] = []string{"ai-channels"}
		case authz.EnterpriseRoleKey(org, 3):
			out[key] = []string{"ai-credentials"}
		case authz.EnterpriseRoleKey(org, 4):
			if !g.revokeChat.Load() {
				out[key] = []string{"chat"}
			}
		}
	}
	return out, nil
}

func searchFixtureRoles(org, actor string, revoked bool) []string {
	switch actor {
	case "platform":
		return []string{"platform_admin"}
	case "admin":
		return []string{"listingkit_admin"}
	case "owner":
		if !revoked {
			return []string{authz.EnterpriseRoleKey(org, 1)}
		}
	case "channel-only":
		return []string{authz.EnterpriseRoleKey(org, 2)}
	case "key-only":
		return []string{authz.EnterpriseRoleKey(org, 3)}
	case "viewer":
		return []string{"listingkit_viewer"}
	}
	return []string{authz.EnterpriseRoleKey(org, 4)}
}

func TestWebSearchMountedPlatformOwnedAndBudget(t *testing.T) {
	f := newAcquisitionHTTPFixture(t)
	ctx := context.Background()
	require.NoError(t, installHTTPChannelFixture(f.owner))
	require.NoError(t, f.owner.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE product_agent_owner").Error; err != nil {
			return err
		}
		return websearchstore.Install(ctx, tx)
	}))
	require.NoError(t, workstore.InstallSchema(f.owner))
	require.NoError(t, resourceadapter.AutoMigrate(f.owner))
	require.NoError(t, f.owner.Exec("DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='model_route_config_writer') THEN CREATE ROLE model_route_config_writer LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD 'search-writer-test-only'; END IF; END $$").Error)
	var searchDBName string
	require.NoError(t, f.owner.Raw("SELECT current_database()").Scan(&searchDBName).Error)
	require.NoError(t, f.owner.Exec("GRANT CONNECT ON DATABASE "+searchDBName+" TO model_route_config_writer").Error)
	require.NoError(t, f.owner.Exec("DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='title_credential_writer') THEN CREATE ROLE title_credential_writer LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD 'search-credential-test-only'; END IF; END $$").Error)
	require.NoError(t, f.owner.Exec("GRANT CONNECT ON DATABASE "+searchDBName+" TO title_credential_writer").Error)
	require.NoError(t, f.owner.Exec(`GRANT USAGE ON SCHEMA public TO title_credential_writer;
GRANT SELECT, INSERT ON public.ai_channel_connections, public.ai_channel_connection_receipts TO title_credential_writer;
GRANT UPDATE (revision, enabled, protection_key_id, secret_ciphertext) ON public.ai_channel_connections TO title_credential_writer`).Error)
	require.NoError(t, websearchstore.GrantRuntime(ctx, f.owner))
	routeDB, err := gorm.Open(postgres.Open(os.Getenv("ISSUE398_TEST_DSN")+" dbname="+searchDBName+" user=model_route_config_writer password=search-writer-test-only"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	searchPool, err := routeDB.DB()
	require.NoError(t, err)
	defer searchPool.Close()
	repo := websearchstore.New(routeDB)
	require.NoError(t, repo.VerifyRuntime(ctx))
	credentialDB, err := gorm.Open(postgres.Open(os.Getenv("ISSUE398_TEST_DSN")+" dbname="+searchDBName+" user=title_credential_writer password=search-credential-test-only"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	credentialPool, err := credentialDB.DB()
	require.NoError(t, err)
	defer credentialPool.Close()
	require.NoError(t, channelstore.VerifyManagementWriter(ctx, credentialDB))
	protection, err := channelstore.NewProtection("synthetic-http-v1", []byte(strings.Repeat("h", 32)))
	require.NoError(t, err)
	searchWriter, err := channelstore.New(credentialDB, protection)
	require.NoError(t, err)
	channels := newHTTPChannelFixture(f.owner).owner
	var sent atomic.Int64
	var mu sync.Mutex
	var vendorKeys []string
	grants := &searchFixtureGrants{}
	provider := bravesearch.New(acquisitionHTTPTransport(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "api.search.brave.com" || req.URL.Path != "/res/v1/web/search" {
			return nil, errors.New("unexpected fixture destination")
		}
		sent.Add(1)
		mu.Lock()
		vendorKeys = append(vendorKeys, req.Header.Get("X-Subscription-Token"))
		mu.Unlock()
		switch req.URL.Query().Get("q") {
		case "unknown fixture":
			return nil, errors.New("fixture response lost")
		case "revoke fixture":
			grants.revokeChat.Store(true)
			fallthrough
		case "empty fixture":
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"type":"search","web":{"results":[]}}`))}, nil
		default:
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"type":"search","web":{"results":[{"title":"<b>Official fixture</b>","url":"https://example.org/public-rule","description":"unpersisted-web-snippet"},{"title":"Unsafe","url":"javascript:alert(1)"}]}}`))}, nil
		}
	}))
	service := &websearch.Service{Repo: repo, Metadata: channels, Writer: searchWriter, Secrets: channels, Provider: provider, FingerprintKey: []byte(strings.Repeat("s", 32))}
	require.NoError(t, f.owner.Exec("DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='ai_workbench_runtime') THEN CREATE ROLE ai_workbench_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD 'search-test-only'; END IF; END $$").Error)
	require.NoError(t, workstore.GrantRuntime(f.owner, "ai_workbench_runtime"))
	var database string
	require.NoError(t, f.owner.Raw("SELECT current_database()").Scan(&database).Error)
	require.NoError(t, f.owner.Exec("GRANT CONNECT ON DATABASE "+database+" TO ai_workbench_runtime").Error)
	chatDB, err := gorm.Open(postgres.Open(os.Getenv("ISSUE398_TEST_DSN")+" dbname="+database+" user=ai_workbench_runtime password=search-test-only"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := chatDB.DB()
	require.NoError(t, err)
	defer pool.Close()
	auth, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	auth.SetRolePolicyReader(grants)
	deps := newRouteAuthDependencies()
	deps.workbenchVerifier = titleVerifier{}
	deps.organizationResolver = workbenchcontext.NewResolver(grants, "project", "v1", nil)
	deps.authorizer = auth
	module, err := buildAIWorkbenchModule(ctx, AIWorkbenchDependencies{Search: service, DB: chatDB, ConversationOnly: true, AllowedOrganizationIDs: []string{"A", "B"}, resolver: deps.organizationResolver, authorizer: auth}, nil)
	require.NoError(t, err)
	pricing := modelRoutePricingModule{authorizer: auth, services: ModelRoutePricingDependencies{
		Search: service, SearchOrganizations: []string{"A", "B"}, PlatformChannels: &PlatformModelChannels{MaintainerOrganization: "A", ProjectID: "project"},
	}}
	routes := append(pricing.searchSettingsRoutes(), module.(aiWorkbenchModule).routes...)
	server := httptest.NewServer(buildCurrentApplicationHTTPServer(routes, deps).Handler)
	defer server.Close()

	call := func(method, path, actor, org, key string, body any, code int) []byte {
		t.Helper()
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		if body == nil {
			raw = nil
		}
		status, raw, err := acquisitionHTTPRequest(server, method, path, actor, org, key, string(raw))
		require.NoError(t, err)
		require.Equal(t, code, status, string(raw))
		return raw
	}
	member := func(org, actor string) string { return "member-" + org + "-" + actor }
	newChat := func(org, actor string) string {
		body := workbenchCreateFixtureBody(t, server, actor, org, aiworkbench.ScopeSelection{Kind: "general"}, false)
		status, raw, err := acquisitionHTTPRequest(server, "POST", workbenchChatBase, actor, org, uuid.NewString(), body)
		require.NoError(t, err)
		require.Equal(t, 200, status, string(raw))
		var created struct{ Conversation aiworkbench.Conversation }
		require.NoError(t, json.Unmarshal(raw, &created))
		require.NotEmpty(t, created.Conversation.ID)
		return workbenchChatBase + "/" + created.Conversation.ID + "/public-search"
	}

	path := modelRouteBase + "/search/"
	require.NoError(t, f.owner.Exec(`INSERT INTO saas_organization_resource_buckets (quantity_unit,organization_id,resource_type,available,allocated,reserved,consumed,created_at,updated_at) VALUES ('ai-micropoint-v1','A','ai_point',0,0,0,0,now(),now()), ('','A','data_row',8,0,0,0,now(),now())`).Error)
	points := snapshotOrganizationResourceLedger(t, f.owner, "A")
	keyBody := map[string]any{"revision": 0, "apiKey": "fixture-platform", "enabled": true, "expectedMemberId": member("A", "platform")}
	keyRaw := call("POST", path+"platform-key", "platform", "A", uuid.NewString(), keyBody, 200)
	require.NotContains(t, string(keyRaw), "fixture-platform")
	cfg := websearch.Publication{Name: "Platform Brave", Country: "US", Language: "en", MaxUSDMicros: 5000, TotalLimit: 1, TenantLimit: 1, Organizations: []string{"A"}}
	pub := func(actor, org string, want int) {
		call("POST", path+"platform", actor, org, uuid.NewString(), struct {
			websearch.Publication
			ExpectedMemberID string `json:"expectedMemberId"`
		}{cfg, member(org, actor)}, want)
	}
	pub("platform", "A", 200)
	require.NotContains(t, string(call("GET", path+"enterprise", "admin", "A", "", nil, 200)), "Platform Brave")
	cfg.Revision, cfg.Published = 1, true
	pub("platform", "B", 403)
	pub("admin", "A", 403)
	pub("platform", "A", 200)
	require.Contains(t, string(call("GET", path+"platform", "platform", "A", "", nil, 200)), "Platform Brave")
	selectBody := func(org, source string, rev int) map[string]any {
		return map[string]any{"revision": rev, "source": source, "expectedMemberId": member(org, "admin")}
	}
	call("POST", path+"enterprise", "admin", "B", uuid.NewString(), selectBody("B", "platform", 0), 404)
	call("POST", path+"enterprise", "admin", "A", uuid.NewString(), selectBody("A", "platform", 0), 200)
	for _, actor := range []string{"key-only", "channel-only", "admin"} {
		call("POST", path+"owned-key", actor, "A", uuid.NewString(), map[string]any{"revision": 0, "apiKey": "no-write", "enabled": true, "expectedMemberId": member("A", actor)}, 403)
	}
	keyCommand := uuid.NewString()
	ownedKeyResult := call("POST", path+"owned-key", "owner", "A", keyCommand, map[string]any{"revision": 0, "apiKey": "fixture-owned", "enabled": true, "expectedMemberId": member("A", "owner")}, 200)
	require.Contains(t, string(ownedKeyResult), `"updatedAt":"`)
	require.JSONEq(t, string(ownedKeyResult), string(call("GET", path+"owned-key/receipts/"+keyCommand, "owner", "A", "", nil, 200)))
	grants.revoked.Store(true)
	call("GET", path+"owned-key", "owner", "A", "", nil, 403)
	call("POST", path+"owned-key", "owner", "A", uuid.NewString(), map[string]any{"revision": 1, "enabled": false, "expectedMemberId": member("A", "owner")}, 403)
	grants.revoked.Store(false)
	require.Contains(t, string(call("GET", path+"owned-key", "owner", "B", "", nil, 200)), `"keyPresent":false`)
	call("GET", path+"owned-key/receipts/"+keyCommand, "owner", "B", "", nil, 404)
	require.Zero(t, sent.Load())

	chat := newChat("A", "operator")
	route := func() websearch.Option {
		var dto struct{ Option websearch.Option }
		require.NoError(t, json.Unmarshal(call("GET", chat, "operator", "A", "", nil, 200), &dto))
		require.True(t, dto.Option.Available)
		return dto.Option
	}
	search := func(key, q, preview string, code int) []byte {
		return call("POST", chat, "operator", "A", key, map[string]any{"query": q, "preview": preview, "expectedMemberId": member("A", "operator")}, code)
	}
	first, key := route(), uuid.NewString()
	require.Equal(t, "platform", first.Source)
	raw := search(key, "public fixture", first.Preview, 200)
	require.Contains(t, string(raw), "unpersisted-web-snippet")
	require.NotContains(t, string(raw), "javascript:")
	require.NotContains(t, string(search(key, "public fixture", first.Preview, 200)), "unpersisted-web-snippet")
	require.NotContains(t, string(call("GET", chat+"/"+key, "operator", "A", "", nil, 200)), "unpersisted-web-snippet")
	require.NotContains(t, string(call("GET", chat, "operator", "A", "", nil, 200)), "unpersisted-web-snippet")
	call("GET", chat+"/"+key, "operator", "B", "", nil, 403)
	search(key, "different fixture", first.Preview, 409)
	search(uuid.NewString(), "no remaining quota", first.Preview, 429)
	keyBody["revision"], keyBody["apiKey"] = 1, "fixture-platform-rotated"
	call("POST", path+"platform-key", "platform", "A", uuid.NewString(), keyBody, 200)
	search(uuid.NewString(), "rotation cannot reset quota", route().Preview, 429)
	require.EqualValues(t, 1, sent.Load())

	call("POST", path+"enterprise", "admin", "A", uuid.NewString(), selectBody("A", "owned", 1), 200)
	search(uuid.NewString(), "stale preview", first.Preview, 409)
	owned := route()
	require.Equal(t, "owned", owned.Source)
	require.Contains(t, string(search(uuid.NewString(), "empty fixture", owned.Preview, 200)), "no_results")
	unknown := uuid.NewString()
	require.Contains(t, string(search(unknown, "unknown fixture", owned.Preview, 200)), "unknown")
	require.Contains(t, string(search(unknown, "unknown fixture", owned.Preview, 200)), "unknown")
	revokedKey := uuid.NewString()
	require.Contains(t, string(search(revokedKey, "revoke fixture", owned.Preview, 503)), "OUTCOME_UNKNOWN")
	grants.revokeChat.Store(false)
	require.Contains(t, string(call("GET", chat+"/"+revokedKey, "operator", "A", "", nil, 200)), "no_results")
	require.Contains(t, string(search(revokedKey, "revoke fixture", owned.Preview, 200)), "no_results")
	call("POST", path+"owned-key", "owner", "A", uuid.NewString(), map[string]any{"revision": 1, "apiKey": "fixture-rotated", "enabled": true, "expectedMemberId": member("A", "owner")}, 200)
	search(uuid.NewString(), "old credential", owned.Preview, 409)
	fresh := route()
	require.Contains(t, string(search(uuid.NewString(), "empty fixture", fresh.Preview, 200)), "no_results")
	require.JSONEq(t, string(ownedKeyResult), string(call("GET", path+"owned-key/receipts/"+keyCommand, "owner", "A", "", nil, 200)),
		"an original Key receipt must retain its timestamp after rotation")
	call("POST", path+"owned-key", "owner", "A", uuid.NewString(), map[string]any{"revision": 2, "enabled": false, "expectedMemberId": member("A", "owner")}, 200)
	search(uuid.NewString(), "disabled credential", fresh.Preview, 404)
	mu.Lock()
	require.Equal(t, []string{"fixture-platform", "fixture-owned", "fixture-owned", "fixture-owned", "fixture-rotated"}, vendorKeys)
	mu.Unlock()
	var usage struct{ Platform, Owned, Unknown int }
	require.NoError(t, f.owner.Raw(`SELECT
		count(*) FILTER (WHERE source='platform' AND usage=1) AS platform,
		count(*) FILTER (WHERE source='owned' AND usage=1) AS owned,
		count(*) FILTER (WHERE source='owned' AND status='unknown' AND usage IS NULL) AS unknown
		FROM public.ai_web_search_requests WHERE organization='A'`).Scan(&usage).Error)
	require.Equal(t, struct{ Platform, Owned, Unknown int }{1, 3, 1}, usage)
	require.Equal(t, points, snapshotOrganizationResourceLedger(t, f.owner, "A"))
}

func TestWebSearchLedgerFencesExpiredAndClaimedRequests(t *testing.T) {
	f := newAcquisitionHTTPFixture(t)
	ctx := context.Background()
	require.NoError(t, websearchstore.Install(ctx, f.owner))
	repo := websearchstore.New(f.owner)
	attempt := func(actor string) websearch.Attempt {
		return websearch.Attempt{
			Scope: websearch.Scope{Organization: "A", Actor: actor, Member: actor, Conversation: "chat-" + actor},
			Key:   uuid.NewString(), Query: "public rules", Fingerprint: "fixture-fingerprint",
			Route: websearch.Route{Owner: "platform-fixture", Source: websearch.Platform, TotalLimit: 1, TenantLimit: 1},
		}
	}
	old := attempt("old")
	_, fresh, _, err := repo.Hold(ctx, old)
	require.NoError(t, err)
	require.True(t, fresh)
	require.NoError(t, f.owner.Exec(`UPDATE ai_web_search_requests SET day_start=(date_trunc('day',clock_timestamp() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC')-interval '1 day', started_at=(date_trunc('day',clock_timestamp() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC')-interval '2 seconds', deadline=(date_trunc('day',clock_timestamp() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC')-interval '1 second' WHERE request_key=?`, old.Key).Error)
	claimed, err := repo.Claim(ctx, old)
	require.NoError(t, err)
	require.False(t, claimed)
	closed, err := repo.Find(ctx, old.Scope, old.Key, old.Fingerprint)
	require.NoError(t, err)
	require.Equal(t, websearch.NotSent, closed.Status)

	type output struct {
		attempt websearch.Attempt
		err     error
		fresh   bool
	}
	done := make(chan output, 2)
	start := make(chan struct{})
	for _, user := range []string{"first", "second"} {
		go func() {
			next := attempt(user)
			<-start
			_, fresh, _, err := repo.Hold(ctx, next)
			done <- output{next, err, fresh}
		}()
	}
	close(start)
	var winner websearch.Attempt
	rejected := 0
	for range 2 {
		result := <-done
		if result.err != nil {
			require.ErrorIs(t, result.err, websearch.ErrQuota)
			rejected++
		} else {
			require.True(t, result.fresh)
			winner = result.attempt
		}
	}
	require.Equal(t, 1, rejected)
	claimed, err = repo.Claim(ctx, winner)
	require.NoError(t, err)
	require.True(t, claimed)
	reopened := websearchstore.New(f.owner)
	claimed, err = reopened.Claim(ctx, winner)
	require.NoError(t, err)
	require.False(t, claimed)
	_, fresh, _, err = reopened.Hold(ctx, winner)
	require.NoError(t, err)
	require.False(t, fresh)
	closed, err = reopened.Finish(ctx, winner, websearch.Unknown, nil)
	require.NoError(t, err)
	require.Equal(t, websearch.Unknown, closed.Status)
	_, _, _, err = reopened.Hold(ctx, attempt("third"))
	require.ErrorIs(t, err, websearch.ErrQuota)
	one := 1
	for range 2 {
		closed, err = reopened.Finish(ctx, winner, websearch.Complete, &one)
		require.NoError(t, err)
		require.Equal(t, websearch.Complete, closed.Status)
		require.NotNil(t, closed.Usage)
		require.Equal(t, 1, *closed.Usage)
	}
	closed, err = reopened.Finish(ctx, winner, websearch.Unknown, nil)
	require.NoError(t, err)
	require.Equal(t, websearch.Complete, closed.Status)
}
