package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	listingapp "task-processor/internal/app/listing"
	"task-processor/internal/listing/materials"
	record "task-processor/internal/listing/record/target"
)

func testCandidateRulesNativeOwners(t *testing.T, value string, a *listingMaterialsApplication, parent *productAgentApplication, supply *listingapp.Application, base record.TargetReceipt, actor context.Context, deps routeAuthDependencies, calls *atomic.Int32, input materialsPrepareInput, path string, call func(string, string, string, any, int) []byte) {
	t.Helper()
	scope := agent.Scope{OrganizationID: "B", ActorID: "admin"}
	prepared := call("POST", path+"/prepare", "", input, 200)
	var out struct {
		ConfigurationSnapshotRef agent.ConfigurationSnapshotRef `json:"configurationSnapshotRef"`
		Purpose                  string                         `json:"purpose"`
		Candidate                string                         `json:"candidateCategoryId"`
	}
	require.NoError(t, json.Unmarshal(prepared, &out))
	require.Equal(t, materials.CategoryRulesPurpose, out.Purpose)
	require.Equal(t, "123", out.Candidate)
	frozen, err := parent.configuration.LoadSnapshot(context.Background(), scope, out.ConfigurationSnapshotRef)
	require.NoError(t, err)
	require.Equal(t, "123", frozen.Request.Binding.CandidateRules.CandidateCategoryID)
	changed := input
	changed.CandidateCategoryID = ""
	call("POST", path+"/prepare", "", changed, 409)
	changed = input
	changed.CandidateCategoryID = "124"
	call("POST", path+"/prepare", "", changed, 409)
	if strings.Contains(value, "preclaim") {
		_, err = parent.configuration.Execute(context.Background(), agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.MaterialsAgentID, Operation: "disable", Expected: 1})
		require.NoError(t, err)
		replay := call("POST", path+"/prepare", "", input, 200)
		require.JSONEq(t, string(prepared), string(replay), "original committed preparation is readable before Agent claim even after disable")
		call("POST", path+"/runs", input.RequestKey, materialsStartBody{ConfigurationSnapshotRef: out.ConfigurationSnapshotRef}, 409)
		require.Zero(t, calls.Load())
		return
	}
	raw := call("POST", path+"/runs", input.RequestKey, materialsStartBody{ConfigurationSnapshotRef: out.ConfigurationSnapshotRef}, 200)
	var result materialsRunDTO
	require.NoError(t, json.Unmarshal(raw, &result))
	require.Equal(t, agent.HumanReviewRequired, result.Phase, string(raw))
	require.Equal(t, materials.CategoryRulesPurpose, result.Purpose)
	require.NotNil(t, result.QueryRules)
	require.NotNil(t, result.RuleExplanation)
	require.Equal(t, "123", result.QueryRules.Binding.CandidateRules.CandidateCategoryID)
	require.Equal(t, "not_evaluated", result.QueryRules.Coverage[4].Status)
	require.False(t, result.CanRequestAdoption)
	require.Nil(t, result.Proposal)
	require.Nil(t, result.After)
	require.Empty(t, result.CandidateHash)
	require.Nil(t, result.Adoption)
	require.NotContains(t, string(raw), `"before":`)
	require.Equal(t, int32(2), calls.Load(), "one original governed read/explain sequence")
	run, found, err := parent.store.LookupContext(context.Background(), scope, agent.CandidateRulesContextKind, base.Record.ID, input.RequestKey)
	require.NoError(t, err)
	require.True(t, found)
	// Exercise the real failure projection from the existing runtime contract,
	// without mutating this committed successful run or its known usage.
	failed := run
	failed.State.StopReason = agent.StopRepairLimit
	validation := *failed.State.Validation
	validation.Valid = false
	failed.State.Validation = &validation
	failedDTO, err := a.projectRun(actor, base.Record.ID, failed, frozen)
	require.NoError(t, err)
	require.Equal(t, agent.HumanReviewRequired, failedDTO.Phase)
	require.Equal(t, agent.StopRepairLimit, failedDTO.StopReason)
	require.Equal(t, "observed", failedDTO.UsageStatus)
	require.NotNil(t, failedDTO.QueryRules)
	require.Nil(t, failedDTO.RuleExplanation)
	require.False(t, failedDTO.CanRequestAdoption)
	failedJSON := string(mustMaterialsJSON(t, failedDTO))
	require.NotContains(t, failedJSON, `"ruleExplanation":`)
	require.NotContains(t, failedJSON, `"before":`)
	call("POST", path+"/runs/"+input.RequestKey+"/adopt", "", materialsAdoptBody{ConfigurationSnapshotRef: out.ConfigurationSnapshotRef, CandidateHash: run.State.Validation.CandidateHash, HumanConfirmed: true}, 409)
	head, err := supply.ReadTarget(actor, base.Record.Source.ID, base.Record.Input.StoreID)
	require.NoError(t, err)
	require.Equal(t, base.Record.ID, head.ID)
	// Advance only the synthetic task's Target and disable the Agent. Historical
	// rules/explanation must still come from the exact recorded payload.
	next := base.Record.Input
	next.ExpectedRevision = base.Record.Revision
	_, err = supply.Targets.Create(actor, uuid.NewString(), next)
	require.NoError(t, err)
	_, err = parent.configuration.Execute(context.Background(), agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.MaterialsAgentID, Operation: "disable", Expected: 1})
	require.NoError(t, err)
	rebuilt, err := buildListingMaterialsApplication(context.Background(), parent, supply)
	require.NoError(t, err)
	server := httptest.NewServer(buildCurrentApplicationHTTPServer(listingMaterialsRoutes(rebuilt, rebuilt.bindRequest), deps).Handler)
	defer server.Close()
	for _, method := range []string{"GET", "POST"} {
		endpoint := path + "/runs/" + input.RequestKey
		body := ""
		key := ""
		if method == "POST" {
			endpoint = path + "/runs"
			key = input.RequestKey
			body = string(mustMaterialsJSON(t, materialsStartBody{ConfigurationSnapshotRef: out.ConfigurationSnapshotRef}))
		}
		code, data, e := acquisitionHTTPRequest(server, method, endpoint, "admin", "B", key, body)
		require.NoError(t, e)
		require.Equal(t, 200, code, string(data))
		var retained materialsRunDTO
		require.NoError(t, json.Unmarshal(data, &retained))
		require.Equal(t, result.QueryRules, retained.QueryRules)
		require.Equal(t, result.RuleExplanation, retained.RuleExplanation)
	}
	require.Equal(t, int32(2), calls.Load(), "history/replay/reconstruction cannot redispatch a model")
}
