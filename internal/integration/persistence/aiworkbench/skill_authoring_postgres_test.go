//go:build integration

package aiworkbenchpersistence

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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
	visible, err := store.AuthoringDrafts(ctx, scope, conversation.ID, "member-a")
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

func TestSkillAuthoringRevisionCannotOverwriteConfirmedPackage(t *testing.T) {
	ctx := context.Background()
	store, err := New(workbenchDB(t))
	require.NoError(t, err)
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "user-a"}
	conversation, _, err := store.Create(ctx, scope, uuid.NewString(), aiworkbench.CreateInput{})
	require.NoError(t, err)
	makeJob := func(digit string, expires time.Time) AuthoringDraft {
		deadline := expires.Add(time.Minute)
		return AuthoringDraft{ConversationID: conversation.ID, Key: uuid.NewString(), MemberID: "member-a", Requirement: "Collapse spaces",
			FormatVersion: 2, SkillVersion: 2, SkillDigest: strings.Repeat("a", 64), InputHash: strings.Repeat("b", 64),
			ProfileDigest: strings.Repeat("c", 64), InvocationID: strings.Repeat(digit, 64), ExpiresAt: expires, Deadline: &deadline}
	}
	expires := time.Now().Add(time.Minute)
	root, err := store.StartAuthoring(ctx, scope, makeJob("1", expires))
	require.NoError(t, err)
	root, err = store.FinishAuthoring(ctx, scope, root, "COMPLETE", "generated root")
	require.NoError(t, err)
	makeRevision := func(digit, base string) AuthoringDraft {
		job := makeJob(digit, expires)
		job.RootKey, job.BaseRevision = &root.Key, &base
		job.InputSnapshot = "{\"name\":\"script-skill\"}"
		return job
	}
	first, err := store.StartAuthoring(ctx, scope, makeRevision("2", root.Key))
	require.NoError(t, err)
	late, err := store.StartAuthoring(ctx, scope, makeRevision("3", root.Key))
	require.NoError(t, err)
	require.NoError(t, store.AuthoringDispatch(ctx, scope, late))
	first, err = store.FinishAuthoring(ctx, scope, first, "COMPLETE", "first revision")
	require.NoError(t, err)
	_, err = store.StartAuthoring(ctx, scope, makeRevision("4", root.Key))
	require.ErrorIs(t, err, aiworkbench.ErrProposalStale)
	require.ErrorIs(t, store.AuthoringDispatch(ctx, scope, late), aiworkbench.ErrProposalStale)

	const scriptPackage = `[{"path":"SKILL.md","text":"confirmed rule"},{"path":"scripts/main.py","text":"confirmed source"}]`
	_, err = store.FreezeAuthoringPackage(ctx, scope, root, root.Key, uuid.NewString(), strings.Repeat("a", 64), scriptPackage, 2)
	require.ErrorIs(t, err, aiworkbench.ErrIdempotencyConflict)
	firstSave := uuid.NewString()
	frozen, err := store.FreezeAuthoringPackage(ctx, scope, root, first.Key, firstSave, strings.Repeat("a", 64), scriptPackage, 2)
	require.NoError(t, err)
	_, err = store.FinishAuthoring(ctx, scope, late, "COMPLETE", "late revision")
	require.NoError(t, err)
	recovered, err := store.FreezeAuthoringPackage(ctx, scope, root, first.Key, uuid.NewString(), strings.Repeat("a", 64), scriptPackage, 2)
	require.NoError(t, err)
	require.Equal(t, *frozen.SaveKey, *recovered.SaveKey)
	require.Equal(t, first.Key, recovered.AuthoringRevision())
	require.Equal(t, 2, recovered.SaveFormat)
	require.Equal(t, scriptPackage, recovered.SaveContent)
	_, err = store.StartAuthoring(ctx, scope, makeRevision("4", first.Key))
	require.ErrorIs(t, err, aiworkbench.ErrProposalStale)

	neverStarted := makeJob("5", time.Now().Add(-time.Second))
	_, err = store.StartAuthoring(ctx, scope, neverStarted)
	require.ErrorIs(t, err, aiworkbench.ErrProposalStale)
	_, err = store.AuthoringConfirmation(ctx, scope, conversation.ID, neverStarted.Key, neverStarted.ExpiresAt)
	require.ErrorIs(t, err, aiworkbench.ErrProposalStale)
	confirmed, err := store.AuthoringConfirmation(ctx, scope, conversation.ID, first.Key, time.Now().Add(-time.Second))
	require.NoError(t, err)
	require.Equal(t, first.Key, confirmed.Key)
}

func TestSkillAuthoringUpgradeRetainsOriginalSaveAndRestrictedRole(t *testing.T) {
	ctx := context.Background()
	db := workbenchDB(t)
	const role = "ai_workbench_runtime"
	require.NoError(t, db.Exec("CREATE ROLE ai_workbench_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS").Error)
	require.NoError(t, GrantRuntime(db, role))
	store, err := New(db)
	require.NoError(t, err)
	scope := aiworkbench.Scope{OrganizationID: "org-a", ActorID: "user-a"}
	conversation, _, err := store.Create(ctx, scope, uuid.NewString(), aiworkbench.CreateInput{})
	require.NoError(t, err)
	now := time.Now().UTC()
	row, err := store.QuoteAuthoring(ctx, scope, AuthoringDraft{ConversationID: conversation.ID, Key: uuid.NewString(), MemberID: "member-a",
		Requirement: "Keep model number", SkillVersion: 1, SkillDigest: strings.Repeat("a", 64), InputHash: strings.Repeat("b", 64),
		ProfileDigest: strings.Repeat("c", 64), InvocationID: strings.Repeat("d", 64), ExpiresAt: now.Add(time.Minute)})
	require.NoError(t, err)
	row, err = store.ConfirmAuthoring(ctx, scope, row, now, now.Add(time.Minute))
	require.NoError(t, err)
	row, err = store.FinishAuthoring(ctx, scope, row, "COMPLETE", "historical draft")
	require.NoError(t, err)
	key := uuid.NewString()
	const originalBody = "original SKILL.md bytes awaiting recovery"
	row, err = store.FreezeAuthoringSave(ctx, scope, row, key, strings.Repeat("a", 64), originalBody)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`ALTER TABLE ai_workbench.skill_authoring_drafts
		DROP COLUMN format_version, DROP COLUMN input_snapshot, DROP COLUMN root_key,
		DROP COLUMN base_revision, DROP COLUMN current_revision, DROP COLUMN save_format,
		DROP CONSTRAINT skill_authoring_drafts_save_content_check,
		ADD CHECK (octet_length(save_content)<=16384)`).Error)
	require.NoError(t, validateBoundedWorkbenchPrivileges(db, role))
	require.NoError(t, InstallSkillAuthoringUpgrade(ctx, db, role))
	require.NoError(t, InstallSkillAuthoringUpgrade(ctx, db, role))
	// Start the runtime connection after the owner DDL, as serving does.
	// PostgreSQL cached result plans from before the DDL cannot be reused.
	runtimeDB, err := gorm.Open(postgres.Open(db.Dialector.(*postgres.Dialector).Config.DSN))
	require.NoError(t, err)
	rawRuntime, err := runtimeDB.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, rawRuntime.Close()) })
	require.NoError(t, runtimeDB.Transaction(func(tx *gorm.DB) error {
		require.NoError(t, tx.Exec("SET LOCAL ROLE ai_workbench_runtime").Error)
		require.NoError(t, VerifySchema(ctx, tx))
		runtime, err := New(tx)
		require.NoError(t, err)
		recovered, err := runtime.AuthoringDraft(ctx, scope, conversation.ID, row.Key)
		require.NoError(t, err)
		require.Equal(t, 1, recovered.FormatVersion)
		require.Equal(t, 1, recovered.SaveFormat)
		require.Equal(t, key, *recovered.SaveKey)
		require.Equal(t, originalBody, recovered.SaveContent)
		fresh := AuthoringDraft{ConversationID: conversation.ID, Key: uuid.NewString(), MemberID: "member-a", Requirement: "New script",
			FormatVersion: 2, SkillVersion: 2, SkillDigest: strings.Repeat("a", 64), InputHash: strings.Repeat("b", 64),
			ProfileDigest: strings.Repeat("c", 64), InvocationID: strings.Repeat("e", 64), ExpiresAt: now.Add(time.Minute), Deadline: &now}
		fresh, err = runtime.StartAuthoring(ctx, scope, fresh)
		require.NoError(t, err)
		fresh, err = runtime.FinishAuthoring(ctx, scope, fresh, "COMPLETE", "new draft")
		require.NoError(t, err)
		child := fresh
		child.Key, child.InvocationID, child.RootKey, child.BaseRevision, child.CurrentRevision = uuid.NewString(), strings.Repeat("f", 64), &fresh.Key, &fresh.Key, nil
		child.State, child.Generated = "", ""
		child, err = runtime.StartAuthoring(ctx, scope, child)
		require.NoError(t, err)
		require.NoError(t, runtime.AuthoringDispatch(ctx, scope, child))
		child, err = runtime.FinishAuthoring(ctx, scope, child, "COMPLETE", "edited draft")
		require.NoError(t, err)
		_, err = runtime.FreezeAuthoringPackage(ctx, scope, fresh, child.Key, uuid.NewString(), strings.Repeat("b", 64), "frozen package", 2)
		require.NoError(t, err)
		return nil
	}))
}
