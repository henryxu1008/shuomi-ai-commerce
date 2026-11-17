//go:build integration

package aiworkbenchpersistence

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/aiworkbench"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	skillstore "task-processor/internal/integration/persistence/skill"
	"task-processor/internal/skill"
)

func TestSkillAuthoringConcurrentSaveAndLostResponseReuseOneSkill(t *testing.T) {
	ctx := context.Background()
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	require.NoError(t, configstore.InstallSchema(db))
	require.NoError(t, skillstore.InstallSchema(ctx, db))
	skillRepo, err := skillstore.New(db)
	require.NoError(t, err)
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "user-a"}
	skills, err := skill.NewService(skillRepo, func(context.Context) (skill.Principal, error) {
		return skill.Principal{UserID: scope.ActorID}, nil
	})
	require.NoError(t, err)
	conversation, _, err := store.Create(ctx, scope, uuid.NewString(), aiworkbench.CreateInput{})
	require.NoError(t, err)
	now := time.Now().UTC()
	job, err := store.QuoteAuthoring(ctx, scope, AuthoringDraft{
		ConversationID: conversation.ID, Key: uuid.NewString(), MemberID: "member-a", Requirement: "Keep the verified model number",
		SkillVersion: 1, SkillDigest: strings.Repeat("a", 64), InputHash: strings.Repeat("b", 64), ProfileDigest: strings.Repeat("c", 64),
		InvocationID: strings.Repeat("d", 64), MaximumPoints: 1000, ExpiresAt: now.Add(time.Minute),
	})
	require.NoError(t, err)
	require.Equal(t, "QUOTED", job.State)
	visible, err := store.AuthoringDrafts(ctx, scope, conversation.ID)
	require.NoError(t, err)
	require.Empty(t, visible)
	job, err = store.ConfirmAuthoring(ctx, scope, job, now, now.Add(time.Minute))
	require.NoError(t, err)
	job, err = store.FinishAuthoring(ctx, scope, job, "COMPLETE", "{}")
	require.NoError(t, err)

	const edited = "---\nname: title-for-market\ndescription: Uses verified facts.\n---\nPreserve the verified model number.\n"
	pkg, err := skill.ValidatePackage([]skill.File{{Path: "SKILL.md", Text: edited}})
	require.NoError(t, err)
	type result struct {
		row AuthoringDraft
		err error
	}
	results := make(chan result, 2)
	for range 2 {
		go func() {
			row, err := store.FreezeAuthoringSave(ctx, scope, job, uuid.NewString(), pkg.Digest, edited)
			results <- result{row, err}
		}()
	}
	first, second := <-results, <-results
	require.NoError(t, first.err)
	require.NoError(t, second.err)
	require.NotNil(t, first.row.SaveKey)
	require.Equal(t, first.row.SaveKey, second.row.SaveKey)
	_, err = store.AuthoringDraft(ctx, aiworkbench.Scope{OrganizationID: "org-b", ActorID: scope.ActorID}, conversation.ID, job.Key)
	require.ErrorIs(t, err, aiworkbench.ErrNotFound)

	// The Skill write commits before the Workbench receipt is updated.
	command := skill.Command{Operation: "create", Key: *first.row.SaveKey, Package: pkg}
	saved, err := skills.Execute(ctx, "personal", command)
	require.NoError(t, err)
	reloaded, err := store.AuthoringDraft(ctx, scope, conversation.ID, job.Key)
	require.NoError(t, err)
	require.Equal(t, edited, reloaded.SaveContent)
	require.Nil(t, reloaded.SavedSkillID)
	replayed, err := skills.Execute(ctx, "personal", command)
	require.NoError(t, err)
	require.Equal(t, saved, replayed)
	finished, err := store.FinishAuthoringSave(ctx, scope, reloaded, replayed.ID)
	require.NoError(t, err)
	require.NotNil(t, finished.SavedSkillID)
	require.Empty(t, finished.SaveContent)

	late, err := store.FreezeAuthoringSave(ctx, scope, job, uuid.NewString(), pkg.Digest, edited)
	require.NoError(t, err)
	require.Equal(t, saved.ID, *late.SavedSkillID)
	_, err = store.FreezeAuthoringSave(ctx, scope, job, uuid.NewString(), strings.Repeat("e", 64), "different")
	require.ErrorIs(t, err, aiworkbench.ErrIdempotencyConflict)
	items, _, _, err := skills.List(ctx, "personal", "", 20)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, skill.Draft, items[0].State)
	require.Equal(t, skill.Private, items[0].Visibility)
}
