package skillpersistence

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	"testing"
)

func TestSkillAdmissionUsesExistingConfigurationOwnerAndReceipt(t *testing.T) {
	db, _ := fixture(t)
	repo, e := configstore.New(db)
	require.NoError(t, e)
	ctx := context.Background()
	scope := agent.Scope{OrganizationID: "org-a", ActorID: "admin"}
	_, e = repo.Execute(ctx, agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.ContentAgentID, Operation: "enable", Absent: true})
	require.NoError(t, e)
	current, e := repo.ReadSkillAdmission(ctx, scope)
	require.NoError(t, e)
	require.Equal(t, "0", current.Revision)
	require.False(t, current.AllowPersonal)
	cmd := agentconfig.SkillAdmissionCommand{Scope: scope, Key: uuid.NewString(), AllowPersonal: true}
	check := func(context.Context) error { return nil }
	first, e := repo.SetSkillAdmission(ctx, cmd, check)
	require.NoError(t, e)
	require.True(t, first.AllowPersonal)
	require.Equal(t, "1", first.Revision)
	replay, e := repo.SetSkillAdmission(ctx, cmd, check)
	require.NoError(t, e)
	require.Equal(t, first, replay)
	conflict := cmd
	conflict.AllowPersonal = false
	_, e = repo.SetSkillAdmission(ctx, conflict, check)
	require.ErrorIs(t, e, agentconfig.ErrConflict)
	stale := cmd
	stale.Key = uuid.NewString()
	stale.AllowPersonal = false
	_, e = repo.SetSkillAdmission(ctx, stale, check)
	require.ErrorIs(t, e, agentconfig.ErrRevision)
	denied := stale
	denied.Expected = 1
	_, e = repo.SetSkillAdmission(ctx, denied, func(context.Context) error { return agentconfig.ErrForbidden })
	require.ErrorIs(t, e, agentconfig.ErrForbidden)
	after, e := repo.ReadSkillAdmission(ctx, scope)
	require.NoError(t, e)
	require.Equal(t, first, after)
}
