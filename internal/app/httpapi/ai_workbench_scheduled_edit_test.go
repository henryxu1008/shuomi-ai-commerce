package httpapi

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/aiworkbench"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Reuses the real HTTP/PG/authorization fixture; all model transports in that
// fixture are controlled. Quote/edit/copy must not start an execution.
func testScheduledPlanEditAndCopy(t *testing.T, call func(string, string, string, any, int) []byte, selection aiworkbench.TaskSelection, db *gorm.DB, app *aiWorkbenchApplication) {
	t.Helper()
	rule := map[string]any{"frequency": "daily", "timeZone": "UTC", "hour": 9, "minute": 0}
	input := map[string]any{"selection": selection, "goalSummary": "Original scheduled content", "maxOccurrences": "3", "rule": rule}
	var q scheduledQuote
	require.NoError(t, json.Unmarshal(call("POST", workbenchScheduleBase+"/quote", "", input, 200), &q))
	input["confirmation"] = q.Confirmation
	var result struct{ Plan scheduledPlanView }
	require.NoError(t, json.Unmarshal(call("POST", workbenchScheduleBase, uuid.NewString(), input, 200), &result))
	original := result.Plan
	modified := selection
	modified.Parameters.TargetLanguage = "fr-FR"
	preview := map[string]any{"scheduleId": original.ID, "expectedRevision": original.Revision, "selection": modified, "goalSummary": "Updated future content", "maxOccurrences": "4", "rule": rule}
	require.NoError(t, json.Unmarshal(call("POST", workbenchScheduleBase+"/quote", "", preview, 200), &q))
	changes := map[string]any{"selection": modified, "goalSummary": preview["goalSummary"], "maxOccurrences": "4"}
	patch := map[string]any{"expectedRevision": original.Revision, "rule": rule, "confirmation": q.Confirmation, "changes": changes}
	limits := app.agent.config.Limits
	app.agent.config.Limits.ModelCalls++
	call("PATCH", workbenchScheduleBase+"/"+original.ID, uuid.NewString(), patch, 409)
	app.agent.config.Limits = limits
	key := uuid.NewString()
	require.NoError(t, json.Unmarshal(call("PATCH", workbenchScheduleBase+"/"+original.ID, key, patch, 200), &result))
	require.Equal(t, "Updated future content", result.Plan.GoalSummary)
	require.Equal(t, "fr-FR", result.Plan.Selection.Parameters.TargetLanguage)
	require.Equal(t, "4", result.Plan.MaxOccurrences)
	call("PATCH", workbenchScheduleBase+"/"+original.ID, key, patch, 200)
	call("PATCH", workbenchScheduleBase+"/"+original.ID, uuid.NewString(), patch, 409)
	preview["expectedRevision"] = result.Plan.Revision
	preview["goalSummary"] = "Second future content"
	require.NoError(t, json.Unmarshal(call("POST", workbenchScheduleBase+"/quote", "", preview, 200), &q))
	second := map[string]any{"expectedRevision": result.Plan.Revision, "rule": rule, "confirmation": q.Confirmation, "changes": map[string]any{"selection": modified, "goalSummary": "Second future content", "maxOccurrences": "4"}}
	require.NoError(t, json.Unmarshal(call("PATCH", workbenchScheduleBase+"/"+original.ID, uuid.NewString(), second, 200), &result))
	require.NoError(t, json.Unmarshal(call("PATCH", workbenchScheduleBase+"/"+original.ID, key, patch, 200), &result))
	require.Equal(t, "Second future content", result.Plan.GoalSummary)
	var occurrences int64
	require.NoError(t, db.Table("ai_workbench.scheduled_occurrences").Where("plan_id=?", original.ID).Count(&occurrences).Error)
	require.Zero(t, occurrences)
	var ended struct{ Plan scheduledPlanView }
	require.NoError(t, json.Unmarshal(call("POST", workbenchScheduleBase+"/"+original.ID+"/close", uuid.NewString(), map[string]string{"expectedRevision": result.Plan.Revision}, 200), &ended))
	copyInput := map[string]any{"sourceScheduleId": original.ID, "selection": modified, "goalSummary": "Copied scheduled content", "maxOccurrences": "1", "rule": map[string]any{"frequency": "once", "timeZone": "UTC", "atUtc": time.Now().UTC().Truncate(time.Second).Add(time.Hour)}}
	require.NoError(t, json.Unmarshal(call("POST", workbenchScheduleBase+"/quote", "", copyInput, 200), &q))
	copyInput["confirmation"] = q.Confirmation
	copyKey := uuid.NewString()
	require.NoError(t, json.Unmarshal(call("POST", workbenchScheduleBase, copyKey, copyInput, 200), &result))
	require.NotEqual(t, original.ID, result.Plan.ID)
	require.Equal(t, "0", result.Plan.ConsumedOccurrences)
	call("POST", workbenchScheduleBase, copyKey, copyInput, 200)
}
