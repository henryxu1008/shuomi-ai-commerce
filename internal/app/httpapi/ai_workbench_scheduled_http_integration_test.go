package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"net/http/httptest"
	"strings"
	"task-processor/internal/aicapability"
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
	resumeKey := uuid.NewString()
	require.NoError(t, json.Unmarshal(call("POST", path+"/resume", resumeKey, control, 200), &saved))
	limits := w.agent.config.Limits
	w.agent.config.Limits.Steps++
	call("POST", path+"/resume", resumeKey, control, 200)
	call("POST", path+"/resume", uuid.NewString(), control, 409)
	w.agent.config.Limits = limits
	profileReader := w.agent.selectTitleProfile
	failAgentRead := false
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register("scheduled-agent-read-outage", func(tx *gorm.DB) {
		if failAgentRead && tx.Statement.Table == "organization_agents" {
			_ = tx.AddError(errors.New("synthetic agent configuration read unavailable"))
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Query().Remove("scheduled-agent-read-outage") })
	for _, dependency := range []string{"agent", "profile"} {
		input := map[string]any{"scheduleId": saved.Plan.ID, "maxOccurrences": "1", "rule": map[string]any{"frequency": "once", "timeZone": "UTC", "atUtc": time.Now().UTC().Truncate(time.Second).Add(5 * time.Minute)}}
		require.NoError(t, json.Unmarshal(call("POST", workbenchScheduleBase+"/quote", "", input, 200), &quote))
		command := map[string]any{"expectedRevision": saved.Plan.Revision, "rule": input["rule"], "confirmation": quote.Confirmation}
		key := uuid.NewString()
		if dependency == "agent" {
			failAgentRead = true
		} else {
			w.agent.selectTitleProfile = func(context.Context, string) (aicapability.ModelProfile, error) {
				return aicapability.ModelProfile{}, errors.New("synthetic profile read unavailable")
			}
		}
		call("PATCH", path, key, command, 503)
		failAgentRead = false
		w.agent.selectTitleProfile = profileReader
		var receipts int64
		require.NoError(t, db.Table("ai_workbench.scheduled_plan_commands").Where("idempotency_key=?", key).Count(&receipts).Error)
		require.Zero(t, receipts, "dependency failure must not retire the command")
		require.NoError(t, json.Unmarshal(call("PATCH", path, key, command, 200), &saved))
	}
	editAt := time.Now().UTC().Truncate(time.Second).Add(3 * time.Second)
	editInput := map[string]any{"scheduleId": saved.Plan.ID, "rule": map[string]any{"frequency": "once", "timeZone": "Asia/Shanghai", "atUtc": editAt}, "maxOccurrences": "1"}
	require.NoError(t, json.Unmarshal(call("POST", workbenchScheduleBase+"/quote", "", editInput, 200), &quote))
	edit := map[string]any{"expectedRevision": saved.Plan.Revision, "rule": editInput["rule"], "confirmation": quote.Confirmation}
	editKey := uuid.NewString()
	require.NoError(t, json.Unmarshal(call("PATCH", path, editKey, edit, 200), &saved))
	w.agent.config.Limits.Steps++
	time.Sleep(time.Until(editAt.Add(50 * time.Millisecond)))
	call("PATCH", path, editKey, edit, 200)
	call("PATCH", path, uuid.NewString(), edit, 409)
	edit["confirmation"] = strings.Repeat("b", 64)
	call("PATCH", path, editKey, edit, 409)
	w.agent.config.Limits = limits
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
	owner := aiworkbench.Scope{OrganizationID: "B", ActorID: "admin"}
	current, err := w.store.ReadScheduledPlan(context.Background(), owner, saved.Plan.ID)
	require.NoError(t, err)
	expired := current.ScheduledPlanDraft
	expired.ID = uuid.NewString()
	expired.CreatedAt = time.Now().UTC().Add(-time.Hour)
	expired.RuleStartAt = expired.CreatedAt
	expired.Rule.AtUTC = time.Now().UTC().Truncate(time.Second).Add(-30 * time.Minute)
	expired.CreationFingerprint = ""
	require.NoError(t, w.store.CreateScheduledPlan(context.Background(), expired))
	require.NoError(t, w.store.MarkScheduleRegistered(context.Background(), owner, expired.ID, 1))
	var tasksBefore int64
	require.NoError(t, db.Table("ai_workbench.business_tasks").Count(&tasksBefore).Error)
	require.NoError(t, w.recoverScheduledPlans(context.Background()))
	receipt, err := w.store.ReadScheduledOccurrence(context.Background(), owner, expired.ID, expired.Rule.AtUTC)
	require.NoError(t, err)
	require.Equal(t, "SKIPPED", receipt.State)
	require.Empty(t, receipt.BusinessTaskID)
	current, err = w.store.ReadScheduledPlan(context.Background(), owner, expired.ID)
	require.NoError(t, err)
	require.Equal(t, "CLOSED", current.State)
	require.Zero(t, current.ConsumedOccurrences)
	var tasksAfter int64
	require.NoError(t, db.Table("ai_workbench.business_tasks").Count(&tasksAfter).Error)
	require.Equal(t, tasksBefore, tasksAfter, "expiry recovery must never dispatch paid work")

	// A request that never committed can be safely retired only after the
	// durable negative receipt also fences its older in-flight attempt.
	invalidInput := map[string]any{"selection": selection, "goalSummary": "Localize product", "maxOccurrences": "1", "confirmation": strings.Repeat("a", 64), "rule": map[string]any{"frequency": "once", "timeZone": "UTC", "atUtc": time.Now().UTC().Add(-time.Minute)}}
	rejectedKey := uuid.NewString()
	var rejected struct {
		Code            string
		CommandAccepted *bool
	}
	require.NoError(t, json.Unmarshal(call("POST", workbenchScheduleBase, rejectedKey, invalidInput, 409), &rejected))
	require.Equal(t, "SCHEDULE_COMMAND_REJECTED", rejected.Code)
	require.NotNil(t, rejected.CommandAccepted)
	require.False(t, *rejected.CommandAccepted)
	call("POST", workbenchScheduleBase, rejectedKey, invalidInput, 409)
	invalidInput["goalSummary"] = "Changed intent"
	raw := call("POST", workbenchScheduleBase, rejectedKey, invalidInput, 409)
	require.Contains(t, string(raw), "IDEMPOTENCY_CONFLICT")
	require.NotContains(t, string(raw), "commandAccepted")
}

func testScheduledTaskListing(t *testing.T, w *aiWorkbenchApplication, deps routeAuthDependencies, taskID, filter string, want bool) {
	t.Helper()
	server := httptest.NewServer(buildCurrentApplicationHTTPServer(aiWorkbenchRoutes(w), deps).Handler)
	defer server.Close()
	path := workbenchTaskBase + "?limit=50&scopeKind=" + filter
	status, raw, err := acquisitionHTTPRequest(server, "GET", path, "admin", "B", "", "")
	require.NoError(t, err)
	require.Equal(t, 200, status, string(raw))
	var page struct {
		Tasks      []workbenchTaskView
		Incomplete bool
	}
	require.NoError(t, json.Unmarshal(raw, &page))
	found := false
	for _, task := range page.Tasks {
		if task.ID == taskID {
			found = true
			require.NotEmpty(t, task.ScheduleID)
			require.NotEmpty(t, task.ScheduledOccurrenceID)
			require.Empty(t, task.ConversationID)
		}
	}
	require.Equal(t, want, found, string(raw))
	if !want {
		require.True(t, page.Incomplete)
	}
	_, other, err := acquisitionHTTPRequest(server, "GET", path, "reviewer", "B", "", "")
	require.NoError(t, err)
	require.NotContains(t, string(other), taskID)
}
