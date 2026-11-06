package store

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/imageagent"
	"testing"
	"time"
)

func TestRecentImageSetsBindOrganizationActorMemberAndCursor(t *testing.T) {
	repo := NewOrganizationRepository(newConcurrentSQLite(t)).(*gormRepository)
	ids := map[string]string{}
	for _, owner := range []struct{ org, actor, member string }{{"tenant-a", "user-1", "member-1"}, {"tenant-a", "user-1", "member-1"}, {"tenant-b", "user-1", "member-1"}, {"tenant-a", "other", "member-1"}, {"tenant-a", "user-1", "replacement"}} {
		run := manualRun(uuid.NewString(), owner.org)
		run.UserID = owner.actor
		run.MemberID = owner.member
		run.ScopeProtocol = imageagent.OrganizationScopeProtocol
		run.BusinessTaskID = "operation-1"
		plan := imageSetPlanForStore(t)
		scope := imageagent.ScopeForRun(*run)
		_, err := repo.InitializeRun(context.Background(), imageagent.ProjectionInitialization{Scope: scope, Run: *run, Plan: plan, Catalog: imageagent.AssetCatalog{Assets: []imageagent.AuthorizedAsset{{ID: "source-1", Type: imageagent.AuthorizedAssetSource, URL: "https://images.example.org/source.png"}}}, Snapshot: imageagent.RunProjection{Run: *run, Plan: plan}, CommitID: "start", EventType: "run.initialized", EventPayload: json.RawMessage(`{}`)})
		require.NoError(t, err)
		ids[owner.member] = run.ID
	}
	identity := imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, TenantID: "tenant-a", UserID: "user-1", MemberID: "member-1"}
	first, next, unavailable, err := repo.ListImageSets(context.Background(), identity, "operation-1", "", 1)
	require.NoError(t, err)
	require.Zero(t, unavailable)
	require.Len(t, first, 1)
	require.NotEmpty(t, next)
	second, last, unavailable, err := repo.ListImageSets(context.Background(), identity, "operation-1", next, 1)
	require.NoError(t, err)
	require.Zero(t, unavailable)
	require.Len(t, second, 1)
	require.Empty(t, last)
	require.NotEqual(t, first[0].RunID, second[0].RunID)
	_, _, _, err = repo.ListImageSets(context.Background(), identity, "operation-1", ids["replacement"], 1)
	require.Error(t, err)
}

func TestRecentImageSetsKeepValidRowsWhenOneProjectionCannotBeRead(t *testing.T) {
	db := newConcurrentSQLite(t)
	repo := NewOrganizationRepository(db).(*gormRepository)
	var ids []string
	for i := 0; i < 3; i++ {
		run := manualRun(uuid.NewString(), "tenant-a")
		run.MemberID = "member-1"
		run.ScopeProtocol = imageagent.OrganizationScopeProtocol
		run.BusinessTaskID = "operation-1"
		plan := imageSetPlanForStore(t)
		scope := imageagent.ScopeForRun(*run)
		_, err := repo.InitializeRun(context.Background(), imageagent.ProjectionInitialization{Scope: scope, Run: *run, Plan: plan, Catalog: imageagent.AssetCatalog{Assets: []imageagent.AuthorizedAsset{{ID: "source-1", Type: imageagent.AuthorizedAssetSource, URL: "https://images.example.org/source.png"}}}, Snapshot: imageagent.RunProjection{Run: *run, Plan: plan}, CommitID: "start", EventType: "run.initialized", EventPayload: json.RawMessage(`{}`)})
		require.NoError(t, err)
		require.NoError(t, db.Model(&runRecord{}).Where("id = ?", run.ID).Update("created_at", time.Unix(int64(i+1), 0)).Error)
		ids = append(ids, run.ID)
	}
	var row projectionRecord
	require.NoError(t, db.Where("run_id = ?", ids[1]).First(&row).Error)
	var snapshot imageagent.RunProjection
	require.NoError(t, json.Unmarshal(row.SnapshotJSON, &snapshot))
	snapshot.Plan.Set.Text = nil
	invalid, err := json.Marshal(snapshot)
	require.NoError(t, err)
	require.NoError(t, db.Model(&row).Update("snapshot_json", invalid).Error)
	identity := imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, TenantID: "tenant-a", UserID: manualRun("unused", "tenant-a").UserID, MemberID: "member-1"}
	artworkID := uuid.NewString()
	require.NoError(t, db.Create(&runRecord{ScopeProtocol: identity.ScopeProtocol, TenantID: identity.TenantID, UserID: identity.UserID, MemberID: identity.MemberID, ID: artworkID, BusinessTaskID: "artwork-context", TargetPlatform: "sds", Mode: "standard", IdempotencyKey: artworkID, Status: "blocked", CreatedAt: time.Unix(4, 0)}).Error)
	_, _, _, err = repo.ListImageSets(context.Background(), identity, "", artworkID, 2)
	require.ErrorIs(t, err, imageagent.ErrValidation, "artwork cursors cannot select an image page")
	artwork, _, unavailableArtwork, err := repo.ListArtworkRuns(context.Background(), identity, "", 2)
	require.NoError(t, err)
	require.Empty(t, artwork)
	require.Equal(t, 1, unavailableArtwork)
	allImages, _, unavailableImages, err := repo.ListImageSets(context.Background(), identity, "", "", 20)
	require.NoError(t, err)
	require.Len(t, allImages, 2)
	require.Equal(t, 1, unavailableImages, "unreadable artwork must not be counted in product image history")
	items, next, unavailable, err := repo.ListImageSets(context.Background(), identity, "operation-1", "", 2)
	require.NoError(t, err, "one invalid local record must not make the task list unavailable")
	require.Equal(t, 1, unavailable)
	require.Len(t, items, 1)
	require.Equal(t, ids[2], items[0].RunID)
	require.Equal(t, ids[1], next, "cursor follows the scanned page including unavailable rows")
	var after projectionRecord
	require.NoError(t, db.Where("run_id = ?", ids[1]).First(&after).Error)
	require.Equal(t, invalid, after.SnapshotJSON, "reading must preserve the unavailable record")
	_, err = repo.GetProjection(context.Background(), imageagent.RunScope{TenantID: identity.TenantID, OwnerUserID: identity.UserID, RunID: ids[1]})
	require.ErrorIs(t, err, imageagent.ErrValidation, "direct reads and commands keep their full validation")
	items, next, unavailable, err = repo.ListImageSets(context.Background(), identity, "operation-1", next, 2)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, ids[0], items[0].RunID)
	require.Empty(t, next)
	require.Zero(t, unavailable)

	// An unavailable-only page remains a real page, including its cursor.
	require.NoError(t, db.Where("run_id = ?", ids[2]).Delete(&projectionRecord{}).Error)
	items, next, unavailable, err = repo.ListImageSets(context.Background(), identity, "operation-1", "", 2)
	require.NoError(t, err)
	require.Empty(t, items)
	require.Equal(t, 2, unavailable)
	require.Equal(t, ids[1], next)

	dependency := errors.New("synthetic projection dependency failure")
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register("recent_dependency", func(tx *gorm.DB) {
		if tx.Statement.Table == (projectionRecord{}).TableName() {
			tx.AddError(dependency)
		}
	}))
	defer db.Callback().Query().Remove("recent_dependency")
	items, next, unavailable, err = repo.ListImageSets(context.Background(), identity, "operation-1", "", 2)
	require.ErrorIs(t, err, dependency)
	require.Nil(t, items)
	require.Empty(t, next)
	require.Zero(t, unavailable, "a real dependency failure is not a partial success")
}
