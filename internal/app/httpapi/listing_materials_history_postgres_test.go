package httpapi

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	supplyapp "task-processor/internal/app/supplychain"
	"task-processor/internal/listing/materials"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/product/collection"
)

// Seed finite historical envelopes only in the existing task-owned fixture DB.
// This is historical test data, never a production migration or old executor.
func testMaterialsV1ReadOnlyHTTP(t *testing.T, value string, parent *productAgentApplication, supply *supplyapp.Application, base record.TargetReceipt, actor context.Context, calls *atomic.Int32, db *gorm.DB, snapshot agentconfig.Snapshot, call func(string, string, string, any, int) []byte, path string) {
	t.Helper()
	ctx := context.Background()
	run, found, err := parent.store.Lookup(ctx, snapshot.Scope, snapshot.Request.Binding, snapshot.Request.Key)
	require.NoError(t, err)
	require.True(t, found)
	identity := agent.ExecutionIdentity{OrganizationID: snapshot.Scope.OrganizationID, ActorID: snapshot.Scope.ActorID, MemberID: snapshot.MemberID}
	mode, err := parent.configuration.LoadModeSnapshot(ctx, identity, snapshot.Request.ModeSnapshotRef)
	require.NoError(t, err)
	version, policy, prompt := "v1.0.0", "listing-materials-policy-v1", "listing-materials-prompt-v1"
	if strings.Contains(value, "historical-v2-") {
		version, policy, prompt = "v2.0.0", "listing-materials-policy-v2", "listing-materials-prompt-v2"
	}
	if strings.Contains(value, "historical-v3-") {
		version, policy, prompt = "v3.0.0", "listing-materials-policy-v3", "listing-materials-prompt-v3"
	}
	mode.Subject.Definition.Version = version
	modeRaw := mustMaterialsJSON(t, mode)
	mode.Digest = collection.Digest(mode)
	require.NoError(t, db.Table("agent_configuration.execution_mode_snapshots").Where("id=?", mode.ID).Updates(map[string]any{"definition_version": version, "payload": modeRaw, "digest": mode.Digest}).Error)
	snapshot.AgentVersion = version
	snapshot.Request.PolicyVersion, snapshot.Request.PromptVersion = policy, prompt
	snapshot.ExecutionModelProfile.PromptVersion = snapshot.Request.PromptVersion
	snapshot.ExecutionModelProfile.OutputSchemaVersion = "listing-materials-action-v1"
	snapshot.Request.ModeSnapshotRef = mode.Ref()
	snapshot.Digest = ""
	snapshotRaw := mustMaterialsJSON(t, snapshot)
	snapshot.Digest = collection.Digest(snapshot)
	require.NoError(t, db.Table("agent_configuration.start_snapshots").Where("id=?", snapshot.ID).Updates(map[string]any{"agent_version": version, "payload": snapshotRaw, "digest": snapshot.Digest}).Error)
	ref := agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: snapshot.ID, Digest: snapshot.Digest}
	run.State.Request = snapshot.Request
	run.State.Request.ConfigurationSnapshotRef = ref
	run.State.DomainProposal.Payload = json.RawMessage(`{"brandCode":"fixture-brand","evidenceIds":["original-v1-evidence"]}`)
	run.State.DomainProposal.Kind = "listing-materials-v1"
	run.State.Validation.PolicyVersion = snapshot.Request.PolicyVersion
	run.State.Validation.CandidateHash = collection.Digest(run.State.DomainProposal)
	for n := range run.State.History {
		if run.State.History[n].Tool.ID == materials.InspectToolRef().ID {
			if version == "v3.0.0" {
				continue
			} // v3's exact inspect/advice format is retained verbatim.
			run.State.History[n].Tool.Version = version
			run.State.History[n].Output = json.RawMessage(`{"evidenceId":"original-v1-evidence"}`)
		}
	}
	if strings.HasSuffix(value, "unknown") {
		run.State.Phase = agent.Stopped
		run.State.StopReason = agent.StopUsageUnknown
		run.State.PendingInvocationID = "original-unknown-invocation"
	}
	// The row fingerprint remains its original historical opaque value; no
	// current runtime is allowed to claim, rebuild or reinterpret that request.
	raw := mustMaterialsJSON(t, run)
	require.NoError(t, db.Table("product_agent_runs").Where("run_id=?", run.State.RunID).Updates(map[string]any{"payload": raw, "phase": string(run.State.Phase)}).Error)
	adopt := materialsAdoptBody{ConfigurationSnapshotRef: ref, CandidateHash: run.State.Validation.CandidateHash, HumanConfirmed: true}
	if strings.HasSuffix(value, "committed") {
		p, err := materials.ParseProposal(run.State.DomainProposal.Payload)
		require.NoError(t, err)
		input, err := materials.ApplyProposal(base.Record, p)
		require.NoError(t, err)
		saved, err := supply.Targets.Create(actor, materialsAdoptionKey(run.State), input)
		require.NoError(t, err)
		for _, method := range []string{"GET", "POST"} {
			endpoint := path + "/runs/" + snapshot.Request.Key + "/adoption"
			var body any
			if method == "POST" {
				endpoint = path + "/runs/" + snapshot.Request.Key + "/adopt"
				body = adopt
			}
			response := call(method, endpoint, "", body, 200)
			require.Contains(t, string(response), saved.Record.ID)
		}
	} else {
		call("POST", path+"/runs/"+snapshot.Request.Key+"/adopt", "", adopt, 409)
	}
	response := call("GET", path+"/runs/"+snapshot.Request.Key, "", nil, 200)
	var out materialsRunDTO
	require.NoError(t, json.Unmarshal(response, &out))
	require.False(t, out.CanRequestAdoption)
	require.False(t, out.CanResume)
	if version == "v3.0.0" {
		require.NotNil(t, out.ImageAdvice, "D3 keeps original v3 advice readable, without recomputation")
		require.NoError(t, out.ImageAdvice.Verify())
	} else {
		require.Nil(t, out.ImageAdvice, "no new advice is fabricated for an old exact Tool version")
	}
	if strings.HasSuffix(value, "committed") {
		require.NotNil(t, out.Adoption)
	}
	if strings.HasSuffix(value, "unknown") {
		require.Equal(t, "unknown_reserved", out.UsageStatus)
	}
	call("POST", path+"/runs", snapshot.Request.Key, materialsStartBody{ConfigurationSnapshotRef: ref}, 200)
	call("POST", path+"/runs/"+snapshot.Request.Key+"/resume", "", materialsResumeBody{ConfigurationSnapshotRef: ref, Revision: strconv.FormatUint(run.State.Revision, 10)}, 409)
	after, found, err := parent.store.Lookup(ctx, snapshot.Scope, snapshot.Request.Binding, snapshot.Request.Key)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, raw, mustMaterialsJSON(t, after), "read/replay/refusal cannot change historical state, usage or UNKNOWN")
	require.Equal(t, int32(2), calls.Load(), "only initial loopback inspect/propose; v1 never redispatches")
	loaded, err := parent.configuration.LoadSnapshot(ctx, snapshot.Scope, ref)
	require.NoError(t, err)
	require.Equal(t, snapshot, loaded)
	_, err = parent.configuration.CheckRegisteredTextExecutionMode(ctx, identity, ref, mode.Ref())
	require.ErrorIs(t, err, agentconfig.ErrConflict)
}
