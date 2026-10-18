package agentconfigpersistence

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/agentconfig"
)

// Skill AcquireUseForTarget keeps this availability fence until its consumer
// finishes. Exercise the actual PostgreSQL lock, which SQLite cannot reproduce.
func holdImageSkillFence(t *testing.T, db *gorm.DB, organizationID string) *gorm.DB {
	t.Helper()
	guard := db.Begin()
	require.NoError(t, guard.Error)
	t.Cleanup(func() { _ = guard.Rollback().Error })
	var activation string
	require.NoError(t, guard.Raw("SELECT activation FROM agent_configuration.organization_agents WHERE organization_id=? AND agent_id=? FOR SHARE", organizationID, agentconfig.ImageAgentID).Scan(&activation).Error)
	require.Equal(t, "ENABLED", activation)
	return guard
}

func imageSkillAdmission(start agentconfig.ImageStartCommand, snapshot agentconfig.ImageConfigurationSnapshot) agentconfig.ImageRunAdmissionCommand {
	return agentconfig.ImageRunAdmissionCommand{Scope: start.Scope, Snapshot: snapshot.Ref(), MemberID: start.MemberID, RunID: start.RunID, ConfirmActionID: uuid.NewString(), SourceDigest: start.SourceDigest, InputDigest: start.InputDigest, PlanDigest: digest([]byte("skill-plan")), QuoteDigest: digest([]byte("skill-quote")), Limits: agentconfig.ImageRunLimits{Images: 1, Points: 1, ElapsedSeconds: 60}}
}

func TestVisualConfigurationWithSkillFence(t *testing.T) {
	for _, operation := range []string{"prepare", "admit"} {
		t.Run(operation, func(t *testing.T) {
			db, store, _ := fixture(t)
			start := imageStartFixture(t, store)
			var snapshot agentconfig.ImageConfigurationSnapshot
			if operation == "admit" {
				var err error
				snapshot, err = store.PrepareImageConfiguration(context.Background(), start)
				require.NoError(t, err)
			}
			holdImageSkillFence(t, db, start.Scope.OrganizationID)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if operation == "prepare" {
				_, err := store.PrepareImageConfiguration(ctx, start)
				require.NoError(t, err, "configuration preparation must coexist with its caller's Skill fence")
			} else {
				_, err := store.AdmitImageRun(ctx, imageSkillAdmission(start, snapshot), start.HardLimits)
				require.NoError(t, err, "confirmation must coexist with its caller's Skill fence")
			}
		})
	}
}

func TestVisualSkillFencePreservesConcurrentReceiptsAndDisable(t *testing.T) {
	db, store, _ := fixture(t)
	start := imageStartFixture(t, store)
	guard := holdImageSkillFence(t, db, start.Scope.OrganizationID)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	type prepared struct {
		snapshot agentconfig.ImageConfigurationSnapshot
		err      error
	}
	results := make(chan prepared, 6)
	var workers sync.WaitGroup
	for range 6 {
		workers.Go(func() {
			value, err := store.PrepareImageConfiguration(ctx, start)
			results <- prepared{value, err}
		})
	}
	workers.Wait()
	close(results)
	var snapshot agentconfig.ImageConfigurationSnapshot
	for result := range results {
		require.NoError(t, result.err)
		if snapshot.ID == "" {
			snapshot = result.snapshot
		} else {
			require.Equal(t, snapshot, result.snapshot, "same preparation key must return the original snapshot")
		}
	}
	changed := start
	changed.SourceDigest = digest([]byte("changed-source"))
	_, err := store.PrepareImageConfiguration(ctx, changed)
	require.ErrorIs(t, err, agentconfig.ErrConflict)

	command := imageSkillAdmission(start, snapshot)
	type admitted struct {
		receipt agentconfig.ImageRunAdmissionReceipt
		err     error
	}
	admissions := make(chan admitted, 6)
	for range 6 {
		workers.Go(func() {
			value, err := store.AdmitImageRun(ctx, command, start.HardLimits)
			admissions <- admitted{value, err}
		})
	}
	workers.Wait()
	close(admissions)
	var original agentconfig.ImageRunAdmissionReceipt
	for result := range admissions {
		require.NoError(t, result.err)
		if original.ID == "" {
			original = result.receipt
		} else {
			require.Equal(t, original, result.receipt, "same confirmation must not refresh the budget or deadline")
		}
	}
	conflict := command
	conflict.ConfirmActionID = uuid.NewString()
	_, err = store.AdmitImageRun(ctx, conflict, start.HardLimits)
	require.ErrorIs(t, err, agentconfig.ErrConflict)

	disable := commandForImageDisable(start)
	blocked, stop := context.WithTimeout(ctx, 200*time.Millisecond)
	_, err = store.Execute(blocked, disable)
	stop()
	require.Error(t, err, "the Skill fence must still prevent concurrent disable")
	require.NoError(t, guard.Rollback().Error)
	_, err = store.Execute(ctx, disable)
	require.NoError(t, err, "the same disable operation succeeds after its original fence is released")
}

func commandForImageDisable(start agentconfig.ImageStartCommand) agentconfig.Command {
	value := command(start.Scope, "disable", 1)
	value.AgentID = agentconfig.ImageAgentID
	return value
}
