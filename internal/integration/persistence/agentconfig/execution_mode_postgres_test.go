package agentconfigpersistence

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
)

func modeSubject(private bool) agentconfig.ResolvedModeSubject {
	value := agentconfig.ResolvedModeSubject{Subject: agentconfig.ModeSubject{Kind: agentconfig.RegisteredAgentSubject, ID: "product.title.agent"}, Definition: commercetool.AgentRef{ID: "product.title.agent", Version: "v1.0.0"}}
	if private {
		value.Subject = agentconfig.ModeSubject{Kind: agentconfig.PrivateDeliverySubject, ID: uuid.NewString()}
		value.Definition = commercetool.AgentRef{ID: "platform.draft.quality", Version: "v2.0.0"}
	}
	return value
}

func TestExecutionModeCASReceiptsAndPrivateSubject(t *testing.T) {
	db, store, _ := fixture(t)
	var modes agentconfig.ModeRepository = store
	ctx := context.Background()
	identity := agent.ExecutionIdentity{OrganizationID: "mode-org", ActorID: "mode-actor", MemberID: "mode-member"}
	subject := modeSubject(true)
	initial, err := modes.ReadExecutionMode(ctx, identity, subject)
	require.NoError(t, err)
	require.Equal(t, agent.ModeDefault, initial.Mode)
	require.Equal(t, "0", initial.Revision)
	var count int64
	require.NoError(t, db.Table("agent_configuration.execution_modes").Count(&count).Error)
	require.Zero(t, count)
	cmd := agentconfig.ModeCommand{Identity: identity, Subject: subject, Key: uuid.NewString(), Mode: agent.ModeFullAccess}
	receipt, err := modes.ExecuteMode(ctx, cmd)
	require.NoError(t, err)
	require.Equal(t, "1", receipt.Revision)
	replayed, err := modes.ExecuteMode(ctx, cmd, func(context.Context) error { t.Fatal("committed receipt repeated external eligibility"); return nil })
	require.NoError(t, err)
	require.Equal(t, receipt, replayed)
	changed := cmd
	changed.Mode = agent.ModeReadOnly
	_, err = modes.ExecuteMode(ctx, changed)
	require.ErrorIs(t, err, agentconfig.ErrConflict)
	changed.Key = uuid.NewString()
	_, err = modes.ExecuteMode(ctx, changed)
	require.ErrorIs(t, err, agentconfig.ErrRevision)
	changed.ExpectedRevision = 1
	next, err := modes.ExecuteMode(ctx, changed)
	require.NoError(t, err)
	require.Equal(t, "2", next.Revision)
	replayed, err = modes.ExecuteMode(ctx, cmd)
	require.NoError(t, err)
	require.Equal(t, receipt, replayed)
	current, err := modes.ReadExecutionMode(ctx, identity, subject)
	require.NoError(t, err)
	require.Equal(t, agent.ModeReadOnly, current.Mode)
	require.NoError(t, db.Table("agent_configuration.organization_agents").Count(&count).Error)
	require.Zero(t, count, "private mode must not fabricate public activation")
	var ledgerCount int64
	require.NoError(t, db.Table("agent_configuration.commands").Where("operation=?", "set-execution-mode").Count(&ledgerCount).Error)
	require.EqualValues(t, 2, ledgerCount)
}

func TestExecutionModeSnapshotFreezeRevocationAndIdentity(t *testing.T) {
	_, store, _ := fixture(t)
	var modes agentconfig.ModeRepository = store
	ctx := context.Background()
	identity := agent.ExecutionIdentity{OrganizationID: "mode-org", ActorID: "mode-actor", MemberID: "mode-member"}
	subject := modeSubject(true)
	start := agentconfig.ModeStartCommand{Identity: identity, Subject: subject, Binding: agentconfig.ModeBinding{ContextKind: "quality_draft", ContextID: uuid.NewString(), RequestKey: uuid.NewString(), IntentDigest: strings.Repeat("ab", 32)}}
	snapshot, err := modes.PrepareMode(ctx, start)
	require.NoError(t, err)
	require.True(t, snapshot.Ref().Valid())
	require.Equal(t, agent.ModeDefault, snapshot.Mode)
	require.Equal(t, "0", snapshot.Revision)
	loaded, err := modes.LoadModeSnapshot(ctx, identity, snapshot.Ref())
	require.NoError(t, err)
	require.Equal(t, snapshot, loaded)
	_, err = modes.CheckModeSnapshot(ctx, identity, snapshot.Ref())
	require.NoError(t, err)
	_, err = modes.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: identity, Subject: subject, Key: uuid.NewString(), Mode: agent.ModeFullAccess})
	require.NoError(t, err)
	_, err = modes.CheckModeSnapshot(ctx, identity, snapshot.Ref())
	require.ErrorIs(t, err, agentconfig.ErrChanged)
	replay, err := modes.PrepareMode(ctx, start)
	require.NoError(t, err)
	require.Equal(t, snapshot, replay, "old request is never upgraded to full")
	start.ExpectedRevision = 1
	_, err = modes.PrepareMode(ctx, start)
	require.ErrorIs(t, err, agentconfig.ErrConflict)
	start.Binding.RequestKey = uuid.NewString()
	full, err := modes.PrepareMode(ctx, start)
	require.NoError(t, err)
	require.Equal(t, agent.ModeFullAccess, full.Mode)
	other := identity
	other.OrganizationID = "other-org"
	_, err = modes.LoadModeSnapshot(ctx, other, full.Ref())
	require.ErrorIs(t, err, agentconfig.ErrNotFound)
	other = identity
	other.MemberID = "replacement-member"
	_, err = modes.LoadModeSnapshot(ctx, other, full.Ref())
	require.ErrorIs(t, err, agentconfig.ErrNotFound)
	badRef := full.Ref()
	badRef.Digest = strings.Repeat("cd", 32)
	_, err = modes.LoadModeSnapshot(ctx, identity, badRef)
	require.ErrorIs(t, err, agentconfig.ErrNotFound)
	start.Binding.IntentDigest = strings.Repeat("cd", 32)
	_, err = modes.PrepareMode(ctx, start)
	require.ErrorIs(t, err, agentconfig.ErrConflict)
}

func TestExecutionModeConcurrentFirstWritesHaveOneWinner(t *testing.T) {
	_, store, _ := fixture(t)
	var modes agentconfig.ModeRepository = store
	ctx := context.Background()
	identity := agent.ExecutionIdentity{OrganizationID: "mode-org", ActorID: "mode-actor", MemberID: "mode-member"}
	subject := modeSubject(true)
	var wait sync.WaitGroup
	errors := make(chan error, 2)
	for _, mode := range []agent.ExecutionMode{agent.ModeFullAccess, agent.ModeReadOnly} {
		wait.Add(1)
		go func(mode agent.ExecutionMode) {
			defer wait.Done()
			_, err := modes.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: identity, Subject: subject, Key: uuid.NewString(), Mode: mode})
			errors <- err
		}(mode)
	}
	wait.Wait()
	close(errors)
	winners := 0
	for err := range errors {
		if err == nil {
			winners++
		} else {
			require.ErrorIs(t, err, agentconfig.ErrRevision)
		}
	}
	require.Equal(t, 1, winners)
}

func TestExecutionModeRuntimeGrantPreservesSnapshotAndSubjectIdentity(t *testing.T) {
	db, _, _ := fixture(t)
	role := "mode656_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, db.Exec("CREATE ROLE "+role+" NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS").Error)
	t.Cleanup(func() { _ = db.Exec("DROP OWNED BY " + role).Error; _ = db.Exec("DROP ROLE " + role).Error })
	require.NoError(t, GrantRuntime(db, role))
	var allowed bool
	for _, table := range []string{"execution_modes", "execution_mode_snapshots"} {
		require.NoError(t, db.Raw("SELECT has_table_privilege(?,?,'SELECT,INSERT')", role, "agent_configuration."+table).Scan(&allowed).Error)
		require.True(t, allowed, table)
		require.NoError(t, db.Raw("SELECT has_table_privilege(?,?,'DELETE,TRUNCATE')", role, "agent_configuration."+table).Scan(&allowed).Error)
		require.False(t, allowed, table)
	}
	require.NoError(t, db.Raw("SELECT has_any_column_privilege(?,'agent_configuration.execution_mode_snapshots','UPDATE')", role).Scan(&allowed).Error)
	require.False(t, allowed)
	require.NoError(t, db.Raw("SELECT has_column_privilege(?,'agent_configuration.execution_modes','subject_id','UPDATE')", role).Scan(&allowed).Error)
	require.False(t, allowed)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE " + role).Error; err != nil {
			return err
		}
		store, err := New(tx)
		if err != nil {
			return err
		}
		identity := agent.ExecutionIdentity{OrganizationID: "mode-org", ActorID: "mode-actor", MemberID: "mode-member"}
		subject := modeSubject(true)
		_, err = store.ExecuteMode(context.Background(), agentconfig.ModeCommand{Identity: identity, Subject: subject, Key: uuid.NewString(), Mode: agent.ModeReadOnly})
		if err != nil {
			return err
		}
		_, err = store.PrepareMode(context.Background(), agentconfig.ModeStartCommand{Identity: identity, Subject: subject, ExpectedRevision: 1, Binding: agentconfig.ModeBinding{ContextKind: "quality_draft", ContextID: uuid.NewString(), RequestKey: uuid.NewString(), IntentDigest: strings.Repeat("ab", 32)}})
		return err
	}))
}
