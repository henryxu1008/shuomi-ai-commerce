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
	configstore "task-processor/internal/integration/persistence/agentconfig"
	"testing"
)

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
	require.True(t, result.CanSubmitReview)
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
