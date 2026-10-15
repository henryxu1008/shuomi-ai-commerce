package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/app/productsourcing"
	supplymarketapp "task-processor/internal/app/supplymarket"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	"task-processor/internal/product/catalog/tools/canonicalinspect"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/enrichment"
)

func TestOwnContentRunsThroughRealReviewApplyAndChoiceWithoutShop(t *testing.T) {
	for _, mode := range []string{"default", "read_only", "auto_approve", "full_access"} {
		t.Run(mode, func(t *testing.T) { testProductAgentOwners(t, "own content "+mode) })
	}
}

func TestOwnContentTitleOnlyRunsThroughRealReviewApplyAndChoiceWithoutShop(t *testing.T) {
	for _, mode := range []string{"default", "read_only", "auto_approve", "full_access"} {
		t.Run(mode, func(t *testing.T) { testProductAgentOwners(t, "title-only own content "+mode) })
	}
}

// The controlled provider cites only evidence that the real canonical tool
// disclosed. It does not fabricate an acquisition receipt for an own product.
func ownContentFixtureProposal(t *testing.T, r *http.Request) string {
	t.Helper()
	var wire struct {
		Messages []struct {
			Content string `json:"content"`
		} `json:"messages"`
	}
	require.NoError(t, json.NewDecoder(r.Body).Decode(&wire))
	evidence := ""
	for _, message := range wire.Messages {
		var prompt struct {
			Input struct {
				History []struct{ Output json.RawMessage }
			}
		}
		if json.Unmarshal([]byte(message.Content), &prompt) != nil {
			continue
		}
		for _, observation := range prompt.Input.History {
			var output canonicalinspect.Output
			if json.Unmarshal(observation.Output, &output) == nil && output.ProductKey != "" {
				for _, source := range output.Snapshot.Sources {
					if source.Detail != "" {
						evidence = source.Detail
					}
				}
			}
		}
	}
	require.NotEmpty(t, evidence)
	encoded, err := json.Marshal(agent.Action{Kind: "propose", Candidate: enrichment.Candidate{Changes: []enrichment.FieldChange{{Field: "title", Value: "Reviewed own product title", EvidenceIDs: []string{evidence}}}}})
	require.NoError(t, err)
	return string(encoded)
}

func testOwnContentThroughOwners(t *testing.T, f *acquisitionHTTPFixture, module productAgentModule, deps routeAuthDependencies, permissions *authz.ListingKitAuthorizer, configuration *configstore.Store, mode agent.ExecutionMode, calls *atomic.Int32) {
	t.Helper()
	ctx := context.Background()
	live := &productReviewLiveOrganizationAccess{resolver: deps.organizationResolver, now: time.Now}
	authority, err := collection.NewContextAuthorizer(live, permissions)
	require.NoError(t, err)
	require.NoError(t, collectionstore.InstallSchema(f.owner))
	repository, err := collectionstore.NewRepository(ctx, f.owner, func(tx *gorm.DB) (collectionstore.OwnPublisher, error) {
		return productsourcing.NewOwnProductWriter(tx, authority)
	})
	require.NoError(t, err)
	sources, err := productsourcing.NewPublishedAcquisitionReader(ctx, f.owner, live, permissions)
	require.NoError(t, err)
	collections, err := collection.NewService(repository, authority, sources)
	require.NoError(t, err)
	snapshots, err := catalogstore.NewBoundedSnapshotReader(f.owner, 1<<20)
	require.NoError(t, err)
	collections.WithSnapshots(snapshots)
	actor := authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: "B", EffectiveOrganizationID: "B", UserID: "admin", EffectiveMemberID: "member-B-admin", Roles: []string{"listingkit_admin"}, TokenExpiresAt: time.Now().Add(time.Minute)})
	actor, err = (productReviewCapabilityBinder{}).Bind(actor, "Bearer admin")
	require.NoError(t, err)
	receipt, err := collections.Mutate(actor, uuid.NewString(), collection.Mutation{Action: "create_product", Product: &collection.OwnProduct{Title: "Saved own product", Images: []string{"https://images.example.org/own.png"}}})
	require.NoError(t, err)
	item, err := collections.ReadItem(actor, receipt.ItemID)
	require.NoError(t, err)
	a := module.application.content
	require.NotNil(t, a)
	a.ownProducts = collections
	_, err = configuration.Execute(ctx, agentconfig.Command{Scope: agent.Scope{OrganizationID: "B", ActorID: "admin"}, Key: uuid.NewString(), AgentID: agentconfig.ContentAgentID, Operation: "enable", Absent: true})
	require.NoError(t, err)
	revision := "0"
	if mode != agent.ModeDefault {
		saved, e := configuration.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: agent.ExecutionIdentity{OrganizationID: "B", ActorID: "admin", MemberID: "member-B-admin"}, Subject: a.modeSubject(), Key: uuid.NewString(), Mode: mode})
		require.NoError(t, e)
		revision = saved.Revision
	}
	routes := append(module.routes, ownContentAgentRoutes(a)...)
	app := buildIsolatedApplicationHTTPServer(routes, deps, 2*time.Minute)
	server := httptest.NewServer(app.Handler)
	defer server.Close()
	base := "/api/v1/workbench/collections/items/" + item.Item.ID + "/content-agent"
	code, raw, err := acquisitionHTTPRequest(server, "GET", base+"/source", "admin", "B", "", "")
	require.NoError(t, err)
	require.Equal(t, 200, code, string(raw))
	var preview struct{ CatalogVersion, PublicationID string }
	require.NoError(t, json.Unmarshal(raw, &preview))
	require.Equal(t, "1", preview.CatalogVersion)
	input := `{"expectedModeRevision":"` + revision + `","targetPlatform":"product","contentSelection":{"targetLanguage":"en-US","title":true},"ownSource":{"catalogVersion":"` + preview.CatalogVersion + `","publicationId":"` + preview.PublicationID + `"}}`
	key := uuid.NewString()
	path := base + "/runs"
	code, raw, err = acquisitionHTTPRequest(server, "POST", path, "admin", "B", key, input)
	require.NoError(t, err)
	require.Equal(t, 200, code, string(raw))
	var result productAgentResultDTO
	require.NoError(t, json.Unmarshal(raw, &result))
	require.Equal(t, "own_product", result.ContextKind)
	require.Equal(t, agent.HumanReviewRequired, result.Phase)
	require.Len(t, result.Candidate.Changes, 1)
	require.Equal(t, int32(3), calls.Load())
	reader := supplymarketapp.EffectiveProductReader{Snapshots: snapshots, Applied: a.reviewLookup}
	choice, err := reader.ReadChoice(actor, collection.Scope{OrganizationID: "B", ActorID: "admin", MemberID: "member-B-admin"}, item)
	require.NoError(t, err)
	if mode == agent.ModeFullAccess || mode == agent.ModeAutoApprove {
		require.True(t, choice.Optimized)
	} else {
		require.False(t, choice.Optimized)
	}
	code, raw, err = acquisitionHTTPRequest(server, "POST", path+"/"+key+"/review", "admin", "B", "", `{}`)
	require.NoError(t, err)
	require.Equal(t, 200, code, string(raw))
	var link struct{ ProposalID string }
	require.NoError(t, json.Unmarshal(raw, &link))
	view := titleCall(t, server, "GET", titleBasePath+"/"+link.ProposalID, "admin", "B", "", "", 200)
	if mode == agent.ModeReadOnly {
		titleDecision(t, server, view, "accept", "admin", "", 403)
	} else if mode != agent.ModeFullAccess && mode != agent.ModeAutoApprove {
		accepted := titleDecision(t, server, view, "accept", "admin", "", 200)
		view = titleApply(t, server, accepted, uuid.NewString(), 200)
	}
	choice, err = reader.ReadChoice(actor, collection.Scope{OrganizationID: "B", ActorID: "admin", MemberID: "member-B-admin"}, item)
	require.NoError(t, err)
	require.Equal(t, mode != agent.ModeReadOnly, choice.Optimized)
	if mode != agent.ModeReadOnly {
		require.Equal(t, uint64(2), choice.Selection.EffectiveVersion)
		require.Equal(t, "Reviewed own product title", choice.Product.Title)
	}
	before := calls.Load()
	if choice.Optimized {
		code, raw, err = acquisitionHTTPRequest(server, "POST", path, "admin", "B", uuid.NewString(), input)
		require.NoError(t, err)
		require.Equal(t, 409, code, string(raw))
		require.Equal(t, before, calls.Load(), "a stale source preview cannot start a new run")
	}
	// Original-key replay keeps its source even after Catalog advances.
	for _, action := range []string{"read", "start"} {
		method, url, body := "GET", path+"/"+key, ""
		if action == "start" {
			method, url, body = "POST", path, input
		}
		code, raw, err = acquisitionHTTPRequest(server, method, url, "admin", "B", key, body)
		require.NoError(t, err)
		require.Equal(t, 200, code, string(raw))
		var replay productAgentResultDTO
		require.NoError(t, json.Unmarshal(raw, &replay))
		require.Equal(t, result.RunID, replay.RunID)
		require.Equal(t, "1", replay.CatalogVersion)
	}
	require.Equal(t, before, calls.Load())
	for index, wrong := range []string{strings.Replace(input, `"title":true`, `"description":true`, 1), strings.Replace(input, preview.PublicationID, uuid.NewString(), 1)} {
		code, raw, err = acquisitionHTTPRequest(server, "POST", path, "admin", "B", key, wrong)
		require.NoError(t, err)
		want := http.StatusConflict
		if index == 0 && a.config.ContentTitleOnly {
			// The title-only gate rejects description before idempotency lookup.
			want = http.StatusBadRequest
		}
		require.Equal(t, want, code, string(raw))
	}
	require.Equal(t, before, calls.Load(), "invalid selection/source must not call the model")
	// The acquisition namespace cannot read the same UUID/key.
	code, _, err = acquisitionHTTPRequest(server, "GET", productAcquisitionBase+"/"+item.Item.ID+"/content-agent/runs/"+key, "admin", "B", "", "")
	require.NoError(t, err)
	require.NotEqual(t, 200, code)
	for _, who := range [][2]string{{"reviewer", "B"}, {"admin", "A"}} {
		code, _, err = acquisitionHTTPRequest(server, "GET", path+"/"+key, who[0], who[1], "", "")
		require.NoError(t, err)
		require.NotEqual(t, 200, code)
	}
	recent, _, err := configuration.Recent(actor, agent.Scope{OrganizationID: "B", ActorID: "admin"}, "member-B-admin", agentconfig.ContentAgentID, "", 20, a.store, func(b agent.Binding) error { _, e := a.Authorize(actor, b); return e })
	require.NoError(t, err)
	require.Len(t, recent, 1)
	require.Equal(t, "own_product", recent[0].ContextKind)
	// A manual source revision invalidates the old binding without another call.
	_, err = collections.Mutate(actor, uuid.NewString(), collection.Mutation{Action: "revise_product", ItemID: item.Item.ID, ExpectedRevision: item.Item.Revision, ExpectedSourceVersion: item.Item.Source.Version, ExpectedPublicationID: item.Item.Source.PublicationID, Product: &collection.OwnProduct{Title: "Manually revised own product"}})
	require.NoError(t, err)
	code, _, err = acquisitionHTTPRequest(server, "GET", path+"/"+key, "admin", "B", "", "")
	require.NoError(t, err)
	require.Equal(t, 409, code)
	require.Equal(t, before, calls.Load())
	updated, err := collections.ReadItem(actor, item.Item.ID)
	require.NoError(t, err)
	require.Equal(t, item.Item.Source.Version+1+boolVersion(mode != agent.ModeReadOnly), updated.Item.Source.Version)
}

func boolVersion(value bool) uint64 {
	if value {
		return 1
	}
	return 0
}
