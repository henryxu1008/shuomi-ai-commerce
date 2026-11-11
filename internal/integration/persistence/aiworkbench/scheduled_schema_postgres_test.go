//go:build integration

package aiworkbenchpersistence

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// The existing Workbench installer owns this schema. It must be repeatable,
// tenant-scoped and refuse cross-plan occurrence pointers.
func TestScheduledPlanSchemaBoundaries(t *testing.T) {
	db := workbenchDB(t)
	require.NoError(t, InstallSchema(db))
	planID, otherPlanID := uuid.NewString(), uuid.NewString()
	slotID := uuid.NewString()
	now := time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)
	insertPlan := func(id, owner string, total int64) error {
		return db.Exec(`INSERT INTO ai_workbench.scheduled_plans
			(id,organization_id,owner_user_id,member_id,kind,task_selection,goal_summary,
			 frequency,time_zone,once_at_utc,max_occurrences,per_run_upper_bound_micros,
			 total_upper_bound_micros,state,revision,created_at,updated_at)
			VALUES (?,?,?,?,?,'{}'::jsonb,'Localize the selected product','once','Asia/Shanghai',?,3,2500,?,
			 'ACTIVE',1,?,?)`,
			id, "org-a", owner, "member-"+owner, "product.content.localize",
			now.Add(time.Hour), total, now, now).Error
	}
	require.NoError(t, insertPlan(planID, "alice", 7500))
	require.NoError(t, insertPlan(otherPlanID, "bob", 7500))
	require.Error(t, insertPlan(uuid.NewString(), "mallory", 7499), "total ceiling must agree with count times per-run bound")
	insertOccurrence := func(id, plan, owner string, slot time.Time) error {
		return db.Exec(`INSERT INTO ai_workbench.scheduled_occurrences
			(id,organization_id,owner_user_id,plan_id,scheduled_at_utc,plan_revision,payload_digest,state,created_at,updated_at)
			VALUES (?,?,?,?,?,1,?,'CLAIMED',?,?)`,
			id, "org-a", owner, plan, slot, strings.Repeat("a", 64), now, now).Error
	}
	slot := now.Add(time.Hour)
	require.NoError(t, insertOccurrence(slotID, planID, "alice", slot))
	require.Error(t, insertOccurrence(uuid.NewString(), planID, "alice", slot), "same plan+UTC slot is unique")
	require.Error(t, insertOccurrence(uuid.NewString(), planID, "bob", slot.Add(time.Hour)), "plan actor identity cannot be switched")
	require.NoError(t, db.Exec(`UPDATE ai_workbench.scheduled_plans SET active_occurrence_id = ? WHERE id = ?`,
		slotID, planID).Error)
	require.Error(t, db.Exec(`UPDATE ai_workbench.scheduled_plans SET active_occurrence_id = ? WHERE id = ?`,
		slotID, otherPlanID).Error, "active occurrence must belong to that exact plan/actor")
}

func TestScheduledPlanRuntimePrivileges(t *testing.T) {
	db := workbenchDB(t)
	const role = "ai_workbench_schedule_runtime"
	require.NoError(t, db.Exec("CREATE ROLE "+role+" LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS").Error)
	require.NoError(t, GrantRuntime(db, role))
	for _, table := range []string{"scheduled_plans", "scheduled_occurrences"} {
		var hasDelete bool
		require.NoError(t, db.Raw("SELECT has_table_privilege(?,?,'DELETE')",
			role, "ai_workbench."+table).Scan(&hasDelete).Error)
		require.False(t, hasDelete)
	}
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE " + role).Error; err != nil {
			return err
		}
		return VerifySchema(context.Background(), tx)
	}))
}
