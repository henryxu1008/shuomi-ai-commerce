package skillpersistence

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	d "task-processor/internal/skill"
)

func seedOfficialV1(t *testing.T) (*Store, []versionRow) {
	t.Helper()
	db, s := fixture(t)
	releases, err := d.OfficialPackages()
	require.NoError(t, err)
	for _, release := range releases {
		pkg := release.Package
		if release.Previous != nil {
			pkg = *release.Previous
		}
		id := uuid.NewSHA1(uuid.NameSpaceOID, []byte("skill:"+release.Key)).String()
		now := time.Now().UTC()
		require.NoError(t, db.Create(&skillRow{ID: id, OwnerKind: d.PlatformOwner, OwnerID: d.PlatformID, Creator: "original-maintainer", HeadVersion: 1, Revision: 3, CreatedAt: now, UpdatedAt: now}).Error)
		require.NoError(t, insertVersion(db, d.Version{Metadata: d.Metadata{ID: id}, Number: 1, Package: pkg, State: d.Enabled, Visibility: d.Shared, AllowCopy: true, AllowExport: true, CreatedAt: now}))
	}
	var old []versionRow
	require.NoError(t, db.Order("skill_id").Find(&old).Error)
	return s, old
}

func TestOfficialVisualReleaseAppendsOnlyThreeVersionsAndPreservesAllOriginalRows(t *testing.T) {
	s, old := seedOfficialV1(t)
	ctx := context.Background()
	p := d.Principal{UserID: "admin", PlatformMaintainer: true}
	resolve := func(context.Context) (d.Principal, error) { return p, nil }
	r, err := s.InstallOfficial(ctx, p, resolve)
	require.NoError(t, err)
	require.Len(t, r.Installed, 3)
	require.Len(t, r.Existing, 3)
	for _, ref := range r.Installed {
		require.EqualValues(t, 2, ref.Version)
	}
	var after []versionRow
	require.NoError(t, s.db.Where("version=1").Order("skill_id").Find(&after).Error)
	require.Equal(t, old, after)
	r, err = s.InstallOfficial(ctx, p, resolve)
	require.NoError(t, err)
	require.Empty(t, r.Installed)
	require.Len(t, r.Existing, 6)
	var count int64
	require.NoError(t, s.db.Model(&versionRow{}).Count(&count).Error)
	require.EqualValues(t, 9, count)
}

func TestOfficialVisualReleaseNeverOverridesDisabledOrManuallyAdvancedHead(t *testing.T) {
	for _, mode := range []string{"disabled", "head"} {
		t.Run(mode, func(t *testing.T) {
			s, old := seedOfficialV1(t)
			id := uuid.NewSHA1(uuid.NameSpaceOID, []byte("skill:official-method-v1:image-standard")).String()
			if mode == "disabled" {
				require.NoError(t, s.db.Exec("UPDATE skill_capability.versions SET state='DISABLED' WHERE skill_id=?", id).Error)
			} else {
				require.NoError(t, s.db.Exec("UPDATE skill_capability.skills SET head_version=3 WHERE id=?", id).Error)
			}
			p := d.Principal{UserID: "admin", PlatformMaintainer: true}
			_, err := s.InstallOfficial(context.Background(), p, func(context.Context) (d.Principal, error) { return p, nil })
			require.ErrorIs(t, err, d.ErrConflict)
			var count int64
			require.NoError(t, s.db.Model(&versionRow{}).Count(&count).Error)
			require.EqualValues(t, len(old), count, "all earlier appends must roll back")
		})
	}
}

func TestOfficialInstallKeepsExistingVersionsAndStatesAcrossMaintainers(t *testing.T) {
	db, s := fixture(t)
	ctx := context.Background()
	p := d.Principal{UserID: "admin-a", PlatformMaintainer: true}
	resolve := func(context.Context) (d.Principal, error) { return p, nil }
	first, err := s.InstallOfficial(ctx, p, resolve)
	require.NoError(t, err)
	require.Len(t, first.Installed, 6)
	require.Empty(t, first.Existing)
	v, err := s.Read(ctx, p, d.Owner{Kind: d.PlatformOwner, ID: d.PlatformID}, first.Installed[0].ID, 1)
	require.NoError(t, err)
	require.Equal(t, d.Enabled, v.State)
	require.True(t, v.AllowCopy && v.AllowExport)
	_, err = s.Execute(ctx, d.Command{Principal: p, Owner: v.Metadata.Owner, Key: uuid.NewString(), ID: v.Metadata.ID, Expected: v.Metadata.Revision, Version: 1, Operation: "disable"}, resolve)
	require.NoError(t, err)
	secondAdmin := d.Principal{UserID: "admin-b", PlatformMaintainer: true}
	second, err := s.InstallOfficial(ctx, secondAdmin, func(context.Context) (d.Principal, error) { return secondAdmin, nil })
	require.NoError(t, err)
	require.Empty(t, second.Installed)
	require.Len(t, second.Existing, 6)
	unchanged, err := s.Read(ctx, p, v.Metadata.Owner, v.Metadata.ID, 1)
	require.NoError(t, err)
	require.Equal(t, d.Disabled, unchanged.State)
	require.Equal(t, v.Package, unchanged.Package)
	var count int64
	require.NoError(t, db.Raw("SELECT count(*) FROM skill_capability.versions").Scan(&count).Error)
	require.EqualValues(t, 9, count)
	_, err = s.InstallOfficial(ctx, d.Principal{UserID: "ordinary-user"}, resolve)
	require.ErrorIs(t, err, d.ErrForbidden)
}

func TestOfficialInstallConcurrentAndConflictRollsBackEntireRelease(t *testing.T) {
	db, s := fixture(t)
	ctx := context.Background()
	p := d.Principal{UserID: "admin", PlatformMaintainer: true}
	resolve := func(context.Context) (d.Principal, error) { return p, nil }
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := s.InstallOfficial(ctx, p, resolve); results <- err }()
	}
	wg.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
	var rows []skillRow
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 6)
	// Leave a conflicting release identity and remove the others, only inside
	// this fixture DB. The installer must roll back every otherwise-new row.
	conflict := rows[len(rows)-1].ID
	require.NoError(t, db.Exec("DELETE FROM skill_capability.versions WHERE skill_id<>?", conflict).Error)
	require.NoError(t, db.Exec("DELETE FROM skill_capability.skills WHERE id<>?", conflict).Error)
	require.NoError(t, db.Exec("UPDATE skill_capability.skills SET owner_kind='USER',owner_id='other-user' WHERE id=?", conflict).Error)
	_, err := s.InstallOfficial(ctx, p, resolve)
	require.ErrorIs(t, err, d.ErrConflict)
	var count int64
	require.NoError(t, db.Raw("SELECT count(*) FROM skill_capability.skills").Scan(&count).Error)
	require.EqualValues(t, 1, count)
}
