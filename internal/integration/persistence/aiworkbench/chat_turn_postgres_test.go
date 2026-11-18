//go:build integration

package aiworkbenchpersistence

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/aiworkbench"
)

func TestChatTurnSchemaUpgradeKeepsExistingRequestsAndHasBoundedGrants(t *testing.T) {
	db := workbenchDB(t)
	ctx := context.Background()
	require.NoError(t, db.Exec("CREATE ROLE ai_workbench_runtime NOLOGIN").Error)
	require.NoError(t, GrantRuntime(db, "ai_workbench_runtime"))
	store, err := New(db)
	require.NoError(t, err)
	scope := aiworkbench.Scope{OrganizationID: "org", ActorID: "actor"}
	chat, _, err := store.CreateScoped(ctx, scope, uuid.NewString(), aiworkbench.CreateInput{MemberID: "member"}, &aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"}, nil)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`ALTER TABLE ai_workbench.messages DROP CONSTRAINT ai_workbench_message_turn,DROP CONSTRAINT ai_workbench_message_presentation,DROP COLUMN turn_key;
	ALTER TABLE ai_workbench.commands DROP CONSTRAINT ai_workbench_turn_authoring,DROP CONSTRAINT ai_workbench_turn_tool,DROP CONSTRAINT ai_workbench_command_shape,DROP COLUMN turn_input,DROP COLUMN tool_state,DROP COLUMN tool_key`).Error)
	require.NoError(t, db.Exec(chatTurnOriginalChecks).Error)
	require.False(t, store.CanChatTurns(ctx))
	require.NoError(t, InstallChatTurnSchema(ctx, db, "ai_workbench_runtime"))
	require.NoError(t, InstallChatTurnSchema(ctx, db, "ai_workbench_runtime"))
	require.True(t, store.CanChatTurns(ctx))
	retained, err := store.Get(ctx, scope, chat.ID)
	require.NoError(t, err)
	require.Equal(t, chat.ID, retained.ID)
	require.NoError(t, validateBoundedWorkbenchPrivileges(db, "ai_workbench_runtime"))
	var allowed bool
	require.NoError(t, db.Raw("SELECT has_column_privilege('ai_workbench_runtime','ai_workbench.commands','turn_input','UPDATE')").Scan(&allowed).Error)
	require.False(t, allowed)
	require.NoError(t, db.Raw("SELECT has_column_privilege('ai_workbench_runtime','ai_workbench.commands','tool_key','UPDATE')").Scan(&allowed).Error)
	require.True(t, allowed)
	require.NoError(t, db.Exec("ALTER TABLE ai_workbench.commands DROP CONSTRAINT ai_workbench_turn_tool,ADD CONSTRAINT ai_workbench_turn_tool CHECK (true)").Error)
	require.False(t, store.CanChatTurns(ctx), "same-named weakened constraint cannot enable new Chat traffic")
}

func TestChatTurnManualBindingIsAtomicScopedAndExpiresOnlyBeforeBinding(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	ctx := context.Background()
	scope := aiworkbench.Scope{OrganizationID: "org", ActorID: "actor"}
	conversation, _, err := store.CreateScoped(ctx, scope, uuid.NewString(), aiworkbench.CreateInput{MemberID: "member"}, &aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"}, nil)
	require.NoError(t, err)
	ref := &aiworkbench.ChatSkillReference{Version: 2, Digest: strings.Repeat("a", 64)}
	envelope := aiworkbench.ChatTurnEnvelope{Version: 1, Input: aiworkbench.ChatTurnInput{Content: "创建一个保留型号的标题技能", Skill: ref}, Creator: ref, GeneralScope: true}
	key := uuid.NewString()
	command, _, err := store.BeginChatTurn(ctx, scope, conversation.ID, key, envelope, aiworkbench.StoreRouteRequest{MemberID: "member", Content: envelope.Input.Content})
	require.NoError(t, err)
	require.Equal(t, "COMPLETE", command.Route.State, "manual selection completes routing without a model")
	require.Empty(t, command.Route.InvocationID)
	require.Equal(t, "PENDING", command.ToolState)
	_, _, err = store.ReplayChatTurn(ctx, scope, conversation.ID, key, "other-member", envelope.Input)
	require.ErrorIs(t, err, aiworkbench.ErrNotFound)
	changed := envelope.Input
	changed.Skill = &aiworkbench.ChatSkillReference{Version: 2, Digest: strings.Repeat("b", 64)}
	_, _, err = store.ReplayChatTurn(ctx, scope, conversation.ID, key, "member", changed)
	require.ErrorIs(t, err, aiworkbench.ErrIdempotencyConflict)
	candidate := func() AuthoringDraft {
		id, e := uuid.NewV7()
		require.NoError(t, e)
		deadline := time.Now().UTC().Add(time.Minute)
		return AuthoringDraft{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, ConversationID: conversation.ID, Key: id.String(), MemberID: "member", Requirement: envelope.Input.Content, SkillVersion: 2, SkillDigest: ref.Digest, InputHash: strings.Repeat("c", 64), ProfileDigest: strings.Repeat("d", 64), MaximumPoints: 1000, InvocationID: strings.ReplaceAll(id.String(), "-", "") + strings.ReplaceAll(id.String(), "-", ""), ExpiresAt: time.Now().UTC().Add(4 * time.Minute), Deadline: &deadline, FormatVersion: 2}
	}
	first, second := candidate(), candidate()
	results := make(chan aiworkbench.ChatTurnCommand, 2)
	errors := make(chan error, 2)
	var wg sync.WaitGroup
	for _, child := range []AuthoringDraft{first, second} {
		wg.Add(1)
		go func(row AuthoringDraft) {
			defer wg.Done()
			result, e := store.BindChatTurnAuthoring(ctx, command, row)
			results <- result
			errors <- e
		}(child)
	}
	wg.Wait()
	close(results)
	close(errors)
	for e := range errors {
		require.NoError(t, e)
	}
	var childKey string
	for result := range results {
		require.Equal(t, "BOUND", result.ToolState)
		if childKey != "" {
			require.Equal(t, childKey, result.ToolKey)
		}
		childKey = result.ToolKey
	}
	var count int64
	require.NoError(t, db.Model(&AuthoringDraft{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
	history, _, err := store.ListMessagesPage(ctx, scope, conversation.ID, 0, 50)
	require.NoError(t, err)
	require.Len(t, history, 2)
	require.Equal(t, "AGENT_SKILL_REF", history[1].Kind)
	require.Equal(t, key, history[1].TurnKey)
	read, err := store.ReadChatTurn(ctx, scope, conversation.ID, key, "member")
	require.NoError(t, err)
	require.Equal(t, childKey, read.ToolKey)
	_, err = store.ReadChatTurn(ctx, aiworkbench.Scope{OrganizationID: "other", ActorID: scope.ActorID}, conversation.ID, key, "member")
	require.ErrorIs(t, err, aiworkbench.ErrNotFound)
	// Moving the database-clock fence cannot turn an already bound child absent.
	require.NoError(t, db.Exec(`UPDATE ai_workbench.commands SET created_at=clock_timestamp()-interval '6 minutes' WHERE idempotency_key=?`, key).Error)
	read, err = store.BindChatTurnAuthoring(ctx, command, candidate())
	require.NoError(t, err)
	require.Equal(t, childKey, read.ToolKey)
	expiredKey := uuid.NewString()
	expired, _, err := store.BeginChatTurn(ctx, scope, conversation.ID, expiredKey, envelope, aiworkbench.StoreRouteRequest{MemberID: "member", Content: envelope.Input.Content})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`UPDATE ai_workbench.commands SET created_at=clock_timestamp()-interval '6 minutes' WHERE idempotency_key=?`, expiredKey).Error)
	expired, err = store.BindChatTurnAuthoring(ctx, expired, candidate())
	require.NoError(t, err)
	require.Equal(t, "NOT_STARTED", expired.ToolState)
	require.Empty(t, expired.ToolKey)
	require.NoError(t, db.Model(&AuthoringDraft{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
	raw, _ := json.Marshal(read.Envelope.Input)
	require.Contains(t, string(raw), envelope.Input.Content)
}

func TestChatTurnRevisionReplaysTheOriginalSnapshotFromPostgreSQL(t *testing.T) {
	ctx := context.Background()
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	scope := aiworkbench.Scope{OrganizationID: "org", ActorID: "actor"}
	conversation, _, err := store.CreateScoped(ctx, scope, uuid.NewString(), aiworkbench.CreateInput{MemberID: "member"}, &aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"}, nil)
	require.NoError(t, err)
	ref := &aiworkbench.ChatSkillReference{Version: 2, Digest: strings.Repeat("a", 64)}
	draft := aiworkbench.TitleSkillDraft{
		Name: "title", DisplayName: "Title", Description: "Title rules",
		Instructions: strings.Repeat("中", 16384),
		Script:       &aiworkbench.TitleSkillScript{Description: "Format title"},
	}
	raw, err := json.Marshal(draft)
	require.NoError(t, err)
	draft.Script.Source = strings.Repeat("a", 65536-len(raw))
	raw, err = json.Marshal(draft)
	require.NoError(t, err)
	require.Len(t, raw, 65536)
	input := aiworkbench.ChatTurnInput{
		Content: "保留型号",
		Skill:   ref,
		Revision: &aiworkbench.ChatTurnRevision{
			Root:         uuid.NewString(),
			BaseRevision: uuid.NewString(),
			Snapshot:     raw,
		},
	}
	key := uuid.NewString()
	envelope := aiworkbench.ChatTurnEnvelope{Version: 1, Input: input, Creator: ref, GeneralScope: true}
	command, _, err := store.BeginChatTurn(ctx, scope, conversation.ID, key, envelope, aiworkbench.StoreRouteRequest{MemberID: "member", Content: input.Content})
	var storedSize int
	require.NoError(t, db.Raw("SELECT octet_length((turn_input->'input'->'revision'->'snapshot')::text) FROM ai_workbench.commands WHERE idempotency_key=?", key).Scan(&storedSize).Error)
	require.Greater(t, storedSize, 65536, "JSONB rendering adds whitespace to a valid snapshot")
	require.NoError(t, err)
	require.Equal(t, input.Revision.Snapshot, command.Envelope.Input.Revision.Snapshot)
	require.Equal(t, "PENDING", command.ToolState)
	replayed, found, err := store.ReplayChatTurn(ctx, scope, conversation.ID, key, "member", input)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, command.Route.Fingerprint, replayed.Route.Fingerprint)
	changed := *input.Revision
	changed.Snapshot = json.RawMessage(strings.ReplaceAll(string(changed.Snapshot), "Title rules", "Other rules"))
	input.Revision = &changed
	_, _, err = store.ReplayChatTurn(ctx, scope, conversation.ID, key, "member", input)
	require.ErrorIs(t, err, aiworkbench.ErrIdempotencyConflict)
}
