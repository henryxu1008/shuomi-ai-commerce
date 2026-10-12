package httpapi

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"sync/atomic"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/authidentity"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	"testing"
	"time"
)

func TestWorkbenchExistingTitleRunFinishesPolicyAdoption(t *testing.T) {
	for _, mode := range []string{"auto_approve", "full_access"} {
		t.Run(mode, func(t *testing.T) { testProductAgentOwners(t, "permission recovery "+mode) })
	}
}

func testWorkbenchTitlePolicyRecovery(t *testing.T, value, operationID string, configuration *configstore.Store, a *productAgentApplication, server *httptest.Server, calls *atomic.Int32) {
	t.Helper()
	identity := agent.ExecutionIdentity{OrganizationID: "B", ActorID: "admin", MemberID: "member-B-admin"}
	receipt, err := configuration.ExecuteMode(context.Background(), agentconfig.ModeCommand{Identity: identity, Subject: a.modeSubject(), Key: uuid.NewString(), Mode: agent.ExecutionMode(value)})
	require.NoError(t, err)
	ctx, err := (productReviewCapabilityBinder{}).Bind(authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: "B", EffectiveOrganizationID: "B", HomeOrganizationID: "B", UserID: "admin", EffectiveMemberID: identity.MemberID, Roles: []string{"listingkit_admin"}, TokenExpiresAt: time.Now().Add(time.Hour)}), "Bearer isolated-test-token")
	require.NoError(t, err)
	binding, err := a.binding(ctx, operationID, "shein")
	require.NoError(t, err)
	request, err := a.startRequest(ctx, binding, uuid.NewString(), "", nil, receipt.Revision)
	require.NoError(t, err)
	run, err := a.runtime.Start(ctx, request)
	require.NoError(t, err)
	require.True(t, agentRunReviewable(run.State))
	beforeCalls := calls.Load()
	require.Positive(t, beforeCalls)
	// The run and pending Review are durable before policy adoption fails.
	policy := a.policyAdoption
	defer func() { a.policyAdoption = policy }()
	a.policyAdoption = nil
	_, err = a.finishTitleCandidate(ctx, run)
	require.ErrorIs(t, err, agentconfig.ErrUnavailable)
	input := agentReviewInput(binding, request.PolicyVersion, run.State.Candidate)
	input.Execution = agentExecutionProvenance(run.State)
	view, err := a.reviews.CreateFromCandidate(ctx, "agent:"+run.State.RunID, input)
	require.NoError(t, err)
	require.Equal(t, "pending", view.State)
	wire, err := json.Marshal(request)
	require.NoError(t, err)
	task := aiworkbench.BusinessTask{Scope: aiworkbench.Scope{OrganizationID: "B", ActorID: "admin"}, OperationID: operationID, TargetPlatform: "shein", ExecutionRequestKey: request.Key, ConfigurationSnapshotRef: request.ConfigurationSnapshotRef, ContextSnapshotRef: request.ContextSnapshotRef, ExecutionRequest: wire}
	crossedOwner, err := (workbenchExecution{agent: a}).startTask(ctx, task)
	require.ErrorIs(t, err, agentconfig.ErrUnavailable, "existing run must retry unfinished adoption rather than report success")
	require.True(t, crossedOwner, "post-processing failure cannot be reported as a pre-owner rejection")
	a.policyAdoption = policy
	_, err = (workbenchExecution{agent: a}).startTask(ctx, task)
	require.NoError(t, err)
	applied := titleCall(t, server, "GET", titleBasePath+"/"+view.ID, "admin", "B", "", "", 200)
	require.Equal(t, "applied", applied.State)
	require.NotNil(t, applied.PolicyAdoption)
	require.Empty(t, applied.History)
	_, err = (workbenchExecution{agent: a}).startTask(ctx, task)
	require.NoError(t, err)
	replayed := titleCall(t, server, "GET", titleBasePath+"/"+view.ID, "admin", "B", "", "", 200)
	require.Equal(t, applied, replayed, "recovery must reuse the original policy adoption and publication")
	recovered, found, err := a.store.Lookup(ctx, run.State.Scope, binding, request.Key)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, run, recovered)
	require.Equal(t, beforeCalls, calls.Load(), "recovering adoption cannot resend a model request")
}

func TestTitlePermissionModesUseExistingOwners(t *testing.T) {
	for _, mode := range []string{"default", "auto_approve", "read_only", "full_access"} {
		t.Run(mode, func(t *testing.T) { testProductAgentOwners(t, "permission "+mode) })
	}
}
func testTitlePermissionMode(t *testing.T, value string, f *acquisitionHTTPFixture, operationID string, configuration *configstore.Store, a *productAgentApplication, server *httptest.Server, calls *atomic.Int32) {
	t.Helper()
	identity := agent.ExecutionIdentity{OrganizationID: "B", ActorID: "admin", MemberID: "member-B-admin"}
	receipt, err := configuration.ExecuteMode(context.Background(), agentconfig.ModeCommand{Identity: identity, Subject: a.modeSubject(), Key: uuid.NewString(), Mode: agent.ExecutionMode(value)})
	require.NoError(t, err)
	key := uuid.NewString()
	path := productAcquisitionBase + "/" + operationID + "/product-agent/runs"
	body := `{"expectedModeRevision":"` + receipt.Revision + `","targetPlatform":"shein"}`
	missingCode, missingRaw, missingErr := acquisitionHTTPRequest(server, "POST", path, "admin", "B", uuid.NewString(), `{"targetPlatform":"shein"}`)
	require.NoError(t, missingErr)
	require.Equal(t, 428, missingCode, string(missingRaw))
	require.Zero(t, calls.Load())
	code, raw, err := acquisitionHTTPRequest(server, "POST", path, "admin", "B", key, body)
	require.NoError(t, err)
	require.Equal(t, 200, code, string(raw))
	var result productAgentResultDTO
	require.NoError(t, json.Unmarshal(raw, &result))
	require.Equal(t, value != "auto_approve" && value != "full_access", result.CanSubmitReview)
	require.Equal(t, value != "auto_approve" && value != "full_access", result.HumanReviewRequired)
	require.Equal(t, agent.ExecutionMode(value), result.ExecutionMode)
	require.NotNil(t, result.ModeSnapshotRef)
	code, raw, err = acquisitionHTTPRequest(server, "POST", path+"/"+key+"/review", "admin", "B", "", `{}`)
	require.NoError(t, err)
	require.Equal(t, 200, code, string(raw))
	var link struct {
		ProposalID string `json:"proposalId"`
	}
	require.NoError(t, json.Unmarshal(raw, &link))
	view := titleCall(t, server, "GET", titleBasePath+"/"+link.ProposalID, "admin", "B", "", "", 200)
	if value == "auto_approve" || value == "full_access" {
		require.Equal(t, "applied", view.State)
		require.NotNil(t, view.PolicyAdoption)
		require.Empty(t, view.History)
		require.Equal(t, agent.ExecutionMode(value), view.PolicyAdoption.Mode)
	} else {
		require.Equal(t, "pending", view.State)
		require.Nil(t, view.PolicyAdoption)
		status := 200
		if value == "read_only" {
			status = 403
		}
		accepted := titleCall(t, server, "POST", titleBasePath+"/"+view.ID+"/decisions", "admin", "B", uuid.NewString(), `{"action":"accept","expected_revision":1}`, status)
		if value == "default" {
			applied := titleCall(t, server, "POST", titleBasePath+"/"+view.ID+"/apply", "admin", "B", uuid.NewString(), `{"expected_revision":2}`, 200)
			require.NotEmpty(t, accepted.History)
			require.Nil(t, applied.PolicyAdoption)
		}
	}
	before := calls.Load()
	_, err = configuration.ExecuteMode(context.Background(), agentconfig.ModeCommand{Identity: identity, Subject: a.modeSubject(), Key: uuid.NewString(), Mode: agent.ModeReadOnly, ExpectedRevision: 1})
	require.NoError(t, err)
	code, raw, err = acquisitionHTTPRequest(server, "POST", path, "admin", "B", key, body)
	require.NoError(t, err)
	require.Equal(t, 200, code, string(raw))
	require.Equal(t, before, calls.Load(), "old exact run cannot be sent again after a mode change")
	if value != "read_only" {
		code, raw, err = acquisitionHTTPRequest(server, "POST", path, "admin", "B", uuid.NewString(), body)
		require.NoError(t, err)
		require.Equal(t, 409, code, string(raw))
		require.Equal(t, before, calls.Load(), "stale confirmation must fail before any new dispatch")
	}
}
