package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
	"github.com/stretchr/testify/require"
)

type unavailableScriptControl struct {
	agent.ScriptExecutor
	check  func() error
	checks int
}

func (s *unavailableScriptControl) CheckControl(context.Context) error {
	s.checks++
	return s.check()
}

func testScriptControlBeforeModel(t *testing.T, f *acquisitionHTTPFixture, server *httptest.Server, app *productAgentApplication, path string, request any, calls *atomic.Int32) {
	t.Helper()
	worker := app.script
	defer func() { app.script = worker }()
	count := func(table string) int64 {
		var n int64
		require.NoError(t, f.owner.Table(table).Count(&n).Error)
		return n
	}
	beforeCalls := calls.Load()
	beforeInvocations := count("ai_invocations")
	beforeReservations := count("saas_organization_resource_reservations")
	blocked := &unavailableScriptControl{ScriptExecutor: worker, check: func() error { return agent.ErrUnavailable }}
	app.script = blocked
	skillHTTP(t, server, "POST", path, "admin", "B", uuid.NewString(), "", request, 503)
	require.Positive(t, blocked.checks)
	require.Equal(t, beforeCalls, calls.Load())
	require.Equal(t, beforeInvocations, count("ai_invocations"))
	require.Equal(t, beforeReservations, count("saas_organization_resource_reservations"))
	// The connection can fail after Quote/prepare. The existing before-send
	// permit must still stop the provider and settle its original reservation.
	late := &unavailableScriptControl{ScriptExecutor: worker, check: func() error {
		if count("ai_invocations") > beforeInvocations {
			return agent.ErrUnavailable
		}
		return nil
	}}
	app.script = late
	key := uuid.NewString()
	raw := skillHTTP(t, server, "POST", path, "admin", "B", key, "", request, 200)
	var result productAgentResultDTO
	require.NoError(t, json.Unmarshal(raw, &result))
	require.Equal(t, agent.Stopped, result.Phase)
	require.Equal(t, agent.StopDependency, result.StopReason)
	require.Zero(t, result.Tokens)
	require.Zero(t, result.EstimatedCostMicros)
	require.False(t, result.CanSubmitReview)
	require.False(t, result.CanResume)
	require.Nil(t, result.ScriptReceipt, "preflight must not pretend that script execution occurred")
	require.Equal(t, beforeCalls, calls.Load(), "late control failure must not send to provider")
	var inv struct {
		Outcome     string
		UsageKnown  bool
		TotalTokens int
		ErrorCode   string
	}
	require.NoError(t, f.owner.Table("ai_invocations").Select("outcome,usage_known,total_tokens,error_code").Where("agent_run_id=?", result.RunID).Take(&inv).Error)
	require.Equal(t, "failed", inv.Outcome)
	require.True(t, inv.UsageKnown)
	require.Zero(t, inv.TotalTokens)
	var bucket struct {
		Reserved int64
		Consumed int64
	}
	require.NoError(t, f.owner.Table("saas_organization_resource_buckets").Select("reserved,consumed").Where("organization_id=? AND resource_type=?", "B", "ai_point").Take(&bucket).Error)
	require.Zero(t, bucket.Reserved)
	require.Zero(t, bucket.Consumed)
	checks := late.checks
	skillHTTP(t, server, "POST", path, "admin", "B", key, "", request, 200)
	skillHTTP(t, server, "GET", path+"/"+key, "admin", "B", "", "", nil, 200)
	require.Equal(t, checks, late.checks, "same-key terminal reads must not require control readiness")
	require.Equal(t, beforeCalls, calls.Load())
}

func testScriptReceiptWithoutControl(t *testing.T, server *httptest.Server, app *productAgentApplication, path, key string, request any, expected productAgentResultDTO, calls *atomic.Int32) {
	t.Helper()
	worker := app.script
	defer func() { app.script = worker }()
	blocked := &unavailableScriptControl{ScriptExecutor: worker, check: func() error { return agent.ErrUnavailable }}
	app.script = blocked
	before := calls.Load()
	for _, method := range []string{"POST", "GET"} {
		endpoint, payload := path, request
		if method == "GET" {
			endpoint, payload = path+"/"+key, nil
		}
		raw := skillHTTP(t, server, method, endpoint, "admin", "B", key, "", payload, 200)
		var got productAgentResultDTO
		require.NoError(t, json.Unmarshal(raw, &got))
		require.Equal(t, expected.RunID, got.RunID)
		require.Equal(t, expected.ScriptReceipt, got.ScriptReceipt)
		require.Equal(t, expected.Candidate, got.Candidate)
	}
	require.Zero(t, blocked.checks, "reading a successful receipt must not probe or dispatch")
	require.Equal(t, before, calls.Load())
}
