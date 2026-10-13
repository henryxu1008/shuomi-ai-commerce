package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	supplyapp "task-processor/internal/app/supplychain"
	"task-processor/internal/commercetool"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/shein/goods"
	sheinmodel "task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

// The original acquisition/supply fixtures create a fresh synthetic database.
// Only the rules/model protocol are controlled loopback fixtures; all run,
// configuration, member, point, source and Target owners are actual adapters.
func TestMaterialsHTTPRunAndNativeDraftUseFourExistingModes(t *testing.T) {
	for _, mode := range []string{"default", "read_only", "auto_approve", "full_access"} {
		t.Run(mode, func(t *testing.T) { testProductAgentOwners(t, "supply materials "+mode) })
	}
}

func TestMaterialsNewAdoptionRechecksOriginalModeConfigurationAndTemplate(t *testing.T) {
	for _, change := range []string{"mode changed", "disabled", "archived"} {
		t.Run(change, func(t *testing.T) { testProductAgentOwners(t, "supply materials default "+change) })
	}
}

type materialsOwnerFixtureRules struct{ supplyAgentFixtureRules }

func (r materialsOwnerFixtureRules) ReadTargetRules(context.Context, collection.Scope, string, goods.OfficialDraftInput) (storecenter.ProductMerchantBinding, goods.OfficialRuleSnapshot, error) {
	return r.merchant, goods.OfficialRuleSnapshot{ApplicationMode: sheinmodel.ModeSelfOperated, Brands: []sheinmodel.Brand{{Code: "fixture-brand", Name: "Disclosed fixture brand"}}}, nil
}

func testMaterialsNativeOwners(t *testing.T, value string, parent *productAgentApplication, supply *supplyapp.Application, base record.TargetReceipt, actor context.Context, deps routeAuthDependencies, calls *atomic.Int32) {
	t.Helper()
	ctx := context.Background()
	scope := agent.Scope{OrganizationID: "B", ActorID: "admin"}
	_, err := parent.configuration.Execute(ctx, agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.MaterialsAgentID, Operation: "enable", Absent: true})
	require.NoError(t, err)
	subject := agentconfig.ResolvedModeSubject{Subject: agentconfig.ModeSubject{Kind: agentconfig.RegisteredAgentSubject, ID: agentconfig.MaterialsAgentID}, Definition: commercetool.AgentRef{ID: agentconfig.MaterialsAgentID, Version: agentconfig.MaterialsAgentVersion}}
	mode, err := parent.configuration.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: agent.ExecutionIdentity{OrganizationID: "B", ActorID: "admin", MemberID: "member-B-admin"}, Subject: subject, Key: uuid.NewString(), Mode: agent.ExecutionMode(strings.Fields(value)[0])})
	require.NoError(t, err)
	template, err := parent.configuration.Execute(ctx, agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.MaterialsAgentID, Operation: "create-template", Input: agentconfig.TemplateInput{Name: "固定平台资料测试", TargetPlatform: "shein"}})
	require.NoError(t, err)
	a, err := buildListingMaterialsApplication(ctx, parent, supply)
	require.NoError(t, err)
	require.NotNil(t, a.runtime)
	parent.materials = a
	server := httptest.NewServer(buildIsolatedApplicationHTTPServer(listingMaterialsRoutes(a, a.bindRequest), deps, 2*time.Minute).Handler)
	defer server.Close()
	key := uuid.NewString()
	path := strings.Replace(listingMaterialsBase, ":record_id", base.Record.ID, 1)
	call := func(method, path, key string, body any, want int) []byte {
		t.Helper()
		text := ""
		if body != nil {
			text = string(mustMaterialsJSON(t, body))
		}
		code, raw, e := acquisitionHTTPRequest(server, method, path, "admin", "B", key, text)
		require.NoError(t, e)
		require.Equal(t, want, code, string(raw))
		return raw
	}
	raw := call("POST", path+"/prepare", "", materialsPrepareInput{RequestKey: key, ExpectedModeRevision: mode.Revision, GoalSummary: "核对品牌资料", Template: &agentconfig.TemplateRef{TemplateID: template.TemplateID, Revision: "1"}}, 200)
	var prepared struct {
		ConfigurationSnapshotRef agent.ConfigurationSnapshotRef `json:"configurationSnapshotRef"`
	}
	require.NoError(t, json.Unmarshal(raw, &prepared))
	require.True(t, materialsRefValid(prepared.ConfigurationSnapshotRef))
	start := materialsStartBody{ConfigurationSnapshotRef: prepared.ConfigurationSnapshotRef}
	raw = call("POST", path+"/runs", key, start, 200)
	var result materialsRunDTO
	require.NoError(t, json.Unmarshal(raw, &result))
	require.Equal(t, agent.HumanReviewRequired, result.Phase, string(raw))
	require.Equal(t, agent.ExecutionMode(strings.Fields(value)[0]), result.ExecutionMode)
	require.NotNil(t, result.Proposal)
	require.NotNil(t, result.After)
	require.Equal(t, "fixture-brand", *result.Proposal.BrandCode)
	require.Equal(t, base.Record.Input, result.Before)
	require.Equal(t, int32(2), calls.Load(), "one governed inspect/propose sequence")
	require.Nil(t, result.Adoption, "incomplete rules never automatically adopt, even full")
	adopt := materialsAdoptBody{ConfigurationSnapshotRef: prepared.ConfigurationSnapshotRef, CandidateHash: result.CandidateHash}
	if value != "default" && strings.HasPrefix(value, "default ") {
		switch strings.TrimPrefix(value, "default ") {
		case "mode changed":
			_, err = parent.configuration.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: agent.ExecutionIdentity{OrganizationID: "B", ActorID: "admin", MemberID: "member-B-admin"}, Subject: subject, Key: uuid.NewString(), ExpectedRevision: 1, Mode: agent.ModeReadOnly})
		case "disabled":
			_, err = parent.configuration.Execute(ctx, agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.MaterialsAgentID, Operation: "disable", Expected: 1})
		case "archived":
			_, err = parent.configuration.Execute(ctx, agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.MaterialsAgentID, Operation: "archive-template", TemplateID: template.TemplateID, Expected: 1})
		}
		require.NoError(t, err)
		adopt.HumanConfirmed = true
		call("POST", path+"/runs/"+key+"/adopt", "", adopt, 409)
		head, e := supply.ReadTarget(actor, base.Record.Source.ID, base.Record.Input.StoreID)
		require.NoError(t, e)
		require.Equal(t, base.Record.ID, head.ID)
		call("GET", path+"/runs/"+key, "", nil, 200)
		require.Equal(t, int32(2), calls.Load())
		return
	}
	status := 409
	if value == "read_only" {
		status = 403
	}
	raw = call("POST", path+"/runs/"+key+"/adopt", "", adopt, status)
	if value != "read_only" {
		require.Contains(t, string(raw), "HUMAN_CONFIRMATION_REQUIRED")
	}
	head, e := supply.ReadTarget(actor, base.Record.Source.ID, base.Record.Input.StoreID)
	require.NoError(t, e)
	require.Equal(t, base.Record.ID, head.ID, "denial cannot advance Target head")
	adopt.HumanConfirmed = true
	if value == "read_only" {
		call("POST", path+"/runs/"+key+"/adopt", "", adopt, 403)
		raw = call("GET", path+"/runs/"+key, "", nil, 200)
		require.NoError(t, json.Unmarshal(raw, &result))
		require.False(t, result.CanRequestAdoption)
		return
	}
	raw = call("POST", path+"/runs/"+key+"/adopt", "", adopt, 200)
	var saved record.TargetReceipt
	require.NoError(t, json.Unmarshal(raw, &saved))
	require.Equal(t, int64(2), saved.Record.Revision)
	require.Equal(t, "fixture-brand", saved.Record.Input.Draft.Product.BrandCode)
	// The Target owner alone persists a new record. Canonical content and all
	// protected commerce/source fields are byte-equivalent to the original.
	expected := base.Record.Input
	expected.ExpectedRevision = base.Record.Revision
	expected.Draft.Product.BrandCode = "fixture-brand"
	require.Equal(t, expected, saved.Record.Input)
	_, err = parent.configuration.Execute(ctx, agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.MaterialsAgentID, Operation: "disable", Expected: 1})
	require.NoError(t, err)
	_, err = parent.configuration.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: agent.ExecutionIdentity{OrganizationID: "B", ActorID: "admin", MemberID: "member-B-admin"}, Subject: subject, Key: uuid.NewString(), ExpectedRevision: 1, Mode: agent.ModeReadOnly})
	require.NoError(t, err)
	for _, method := range []string{"GET", "POST"} {
		endpoint := path + "/runs/" + key + "/adoption"
		var body any
		if method == "POST" {
			endpoint = path + "/runs/" + key + "/adopt"
			body = adopt
		}
		raw = call(method, endpoint, "", body, 200)
		var replay record.TargetReceipt
		require.NoError(t, json.Unmarshal(raw, &replay))
		require.Equal(t, saved.Record.ID, replay.Record.ID)
	}
	raw = call("GET", path+"/runs/"+key, "", nil, 200)
	require.NoError(t, json.Unmarshal(raw, &result))
	require.NotNil(t, result.Adoption)
	require.Equal(t, saved.Record.ID, result.Adoption.Record.ID)
	raw = call("POST", path+"/runs", key, start, 200)
	require.Contains(t, string(raw), saved.Record.ID, "old head/config/mode cannot hide committed history")
	require.Equal(t, int32(2), calls.Load(), "replay/read never invokes model again")
}
