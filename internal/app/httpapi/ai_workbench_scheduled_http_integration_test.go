package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"net/http/httptest"
	"task-processor/internal/aiworkbench"
	"testing"
	"time"
)

type scheduledBackendFixture struct {
	failed    bool
	calls     int
	afterSync func()
}

func (s *scheduledBackendFixture) Ready() bool { return true }
func (s *scheduledBackendFixture) Start(aiworkbench.ScheduledDispatcher) (func(), error) {
	return func() {}, nil
}
func (s *scheduledBackendFixture) Synchronize(context.Context, aiworkbench.ScheduledPlan) error {
	s.calls++
	if s.afterSync != nil {
		s.afterSync()
	}
	if s.failed {
		return errors.New("isolated response loss")
	}
	return nil
}

func testScheduledHTTPManagement(t *testing.T, w *aiWorkbenchApplication, deps routeAuthDependencies, selection aiworkbench.TaskSelection, db *gorm.DB) {
	backend := &scheduledBackendFixture{failed: true}
	w.schedules = backend
	server := httptest.NewServer(buildCurrentApplicationHTTPServer(aiWorkbenchScheduleRoutes(w), deps).Handler)
	defer server.Close()
	call := func(method, path, key string, body any, status int) []byte {
		wire := ""
		if body != nil {
			raw, err := json.Marshal(body)
			require.NoError(t, err)
			wire = string(raw)
		}
		code, raw, err := acquisitionHTTPRequest(server, method, path, "admin", "B", key, wire)
		require.NoError(t, err)
		require.Equal(t, status, code, string(raw))
		return raw
	}
	input := map[string]any{"selection": selection, "goalSummary": "Localize product", "maxOccurrences": "1", "rule": map[string]any{"frequency": "once", "timeZone": "Asia/Shanghai", "atUtc": time.Now().UTC().Truncate(time.Second).Add(2 * time.Minute)}}
	var quote struct {
		Confirmation                                  string
		PerRunUpperBoundMicros, TotalUpperBoundMicros string
	}
	require.NoError(t, json.Unmarshal(call("POST", workbenchScheduleBase+"/quote", "", input, 200), &quote))
	require.NotEmpty(t, quote.Confirmation)
	input["confirmation"] = quote.Confirmation
	key := uuid.NewString()
	var saved struct {
		Plan      struct{ ID, State, RegistrationState, Revision string }
		PlanSaved bool
	}
	require.NoError(t, json.Unmarshal(call("POST", workbenchScheduleBase, key, input, 200), &saved))
	require.True(t, saved.PlanSaved)
	require.Equal(t, "UNKNOWN", saved.Plan.RegistrationState)
	require.NotEqual(t, "ACTIVE", saved.Plan.State)
	backend.failed = false
	require.NoError(t, json.Unmarshal(call("POST", workbenchScheduleBase, key, input, 200), &saved))
	require.Equal(t, "ACTIVE", saved.Plan.State)
	previous := backend.calls
	call("POST", workbenchScheduleBase, key, input, 200)
	require.Equal(t, previous, backend.calls, "a matched durable create receipt needs no new registration")
	input["goalSummary"] = "A different goal"
	call("POST", workbenchScheduleBase, key, input, 409)
	path := workbenchScheduleBase + "/" + saved.Plan.ID
	code, _, err := acquisitionHTTPRequest(server, "GET", path, "reviewer", "B", "", "")
	require.NoError(t, err)
	require.NotEqual(t, 200, code)
	controlKey := uuid.NewString()
	control := map[string]any{"expectedRevision": saved.Plan.Revision}
	var paused struct {
		Plan struct{ ID, State, Revision string }
	}
	require.NoError(t, json.Unmarshal(call("POST", path+"/pause", controlKey, control, 200), &paused))
	require.Equal(t, "PAUSED", paused.Plan.State)
	var replayed struct{ Plan struct{ Revision string } }
	require.NoError(t, json.Unmarshal(call("POST", path+"/pause", controlKey, control, 200), &replayed))
	require.Equal(t, paused.Plan.Revision, replayed.Plan.Revision)
	call("POST", path+"/resume", uuid.NewString(), control, 409)
	control["expectedRevision"] = paused.Plan.Revision
	require.NoError(t, json.Unmarshal(call("POST", path+"/resume", uuid.NewString(), control, 200), &saved))
	editInput := map[string]any{"scheduleId": saved.Plan.ID, "rule": map[string]any{"frequency": "once", "timeZone": "Asia/Shanghai", "atUtc": time.Now().UTC().Truncate(time.Second).Add(5 * time.Minute)}, "maxOccurrences": "1"}
	require.NoError(t, json.Unmarshal(call("POST", workbenchScheduleBase+"/quote", "", editInput, 200), &quote))
	edit := map[string]any{"expectedRevision": saved.Plan.Revision, "rule": editInput["rule"], "confirmation": quote.Confirmation}
	require.NoError(t, json.Unmarshal(call("PATCH", path, uuid.NewString(), edit, 200), &saved))
	control["expectedRevision"] = saved.Plan.Revision
	call("POST", path+"/close", uuid.NewString(), control, 200)

	// A committed create can still return 503 during the final authoritative
	// read. The original key must recover that one plan, never a second intent.
	input["goalSummary"] = "Localize product"
	failRead := false
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register("scheduled-final-read-loss", func(tx *gorm.DB) {
		if failRead && tx.Statement.Table == "scheduled_plans" {
			failRead = false
			_ = tx.AddError(errors.New("synthetic final read unavailable"))
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Query().Remove("scheduled-final-read-loss") })
	backend.afterSync = func() { failRead = true }
	lostKey := uuid.NewString()
	var before int64
	require.NoError(t, db.Table("ai_workbench.scheduled_plans").Count(&before).Error)
	call("POST", workbenchScheduleBase, lostKey, input, 503)
	backend.afterSync = nil
	require.NoError(t, json.Unmarshal(call("POST", workbenchScheduleBase, lostKey, input, 200), &saved))
	var after int64
	require.NoError(t, db.Table("ai_workbench.scheduled_plans").Count(&after).Error)
	require.Equal(t, before+1, after)
}
