package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/product/collection"
)

func TestImageAdviceExistingOwnersToReview(t *testing.T) {
	testProductAgentOwners(t, "image advice")
}

func TestImageAdvicePreviousTitleRegistrationIsReadOnly(t *testing.T) {
	for _, state := range []string{"pending", "committed"} {
		t.Run(state, func(t *testing.T) { testProductAgentOwners(t, "title historical "+state) })
	}
}

// Finite historical envelopes are seeded only in the task-owned fixture DB;
// no old executor, migration or compatibility path is installed.
func testTitlePreviousRegistrationReadOnly(t *testing.T, a *productAgentApplication, db *gorm.DB, server *httptest.Server, key, path, startBody, receiptID string, calls *atomic.Int32) {
	t.Helper()
	ctx := context.Background()
	scope := agent.Scope{OrganizationID: "B", ActorID: "admin"}
	operationID := strings.TrimSuffix(strings.TrimPrefix(path, productAcquisitionBase+"/"), "/product-agent/runs")
	run, err := a.store.Read(ctx, scope, operationID, key)
	require.NoError(t, err)
	snapshot, err := a.configuration.LoadSnapshot(ctx, scope, run.State.Request.ConfigurationSnapshotRef)
	require.NoError(t, err)
	identity := agent.ExecutionIdentity{OrganizationID: "B", ActorID: "admin", MemberID: snapshot.MemberID}
	mode, err := a.configuration.LoadModeSnapshot(ctx, identity, snapshot.Request.ModeSnapshotRef)
	require.NoError(t, err)
	mode.Subject.Definition.Version = "v1.0.0"
	modeRaw := mustMaterialsJSON(t, mode)
	mode.Digest = collection.Digest(mode)
	require.NoError(t, db.Table("agent_configuration.execution_mode_snapshots").Where("id=?", mode.ID).Updates(map[string]any{"definition_version": "v1.0.0", "payload": modeRaw, "digest": mode.Digest}).Error)
	snapshot.AgentVersion = "v1.0.0"
	snapshot.Request.PromptVersion = "product-title-agent-v1"
	snapshot.Request.ModeSnapshotRef = mode.Ref()
	snapshot.Digest = ""
	snapshotRaw := mustMaterialsJSON(t, snapshot)
	snapshot.Digest = collection.Digest(snapshot)
	require.NoError(t, db.Table("agent_configuration.start_snapshots").Where("id=?", snapshot.ID).Updates(map[string]any{"agent_version": "v1.0.0", "payload": snapshotRaw, "digest": snapshot.Digest}).Error)
	run.State.Request = snapshot.Request
	run.State.Request.ConfigurationSnapshotRef = agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: snapshot.ID, Digest: snapshot.Digest}
	raw := mustMaterialsJSON(t, run)
	require.NoError(t, db.Table("product_agent_runs").Where("run_id=?", run.State.RunID).Update("payload", raw).Error)
	beforeCalls := calls.Load()
	code, response, err := acquisitionHTTPRequest(server, "GET", path+"/"+key, "admin", "B", "", "")
	require.NoError(t, err)
	require.Equal(t, 200, code, string(response))
	var out productAgentResultDTO
	require.NoError(t, json.Unmarshal(response, &out))
	require.False(t, out.CanResume)
	require.False(t, out.CanSubmitReview)
	require.Nil(t, out.ImageAdvice, "old records are not rebuilt from current facts")
	code, response, err = acquisitionHTTPRequest(server, "POST", path, "admin", "B", key, startBody)
	require.NoError(t, err)
	require.Equal(t, 200, code, string(response), "original key replays the saved run without a new claim")
	code, response, err = acquisitionHTTPRequest(server, "POST", path+"/"+key+"/resume", "admin", "B", "", `{"revision":"`+strconv.FormatUint(run.State.Revision, 10)+`","feedback":"continue"}`)
	require.NoError(t, err)
	require.Equal(t, 409, code, string(response))
	code, response, err = acquisitionHTTPRequest(server, "POST", path+"/"+key+"/review", "admin", "B", "", `{}`)
	require.NoError(t, err)
	if receiptID == "" {
		require.Equal(t, 409, code, string(response))
	} else {
		require.Equal(t, 200, code, string(response))
		require.Contains(t, string(response), receiptID)
	}
	after, err := a.store.Read(ctx, scope, run.State.Request.Binding.ContextID, key)
	require.NoError(t, err)
	require.Equal(t, raw, mustMaterialsJSON(t, after), "read/replay/refusal must not rewrite old state or usage")
	require.Equal(t, beforeCalls, calls.Load())
}
