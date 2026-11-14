package skillpersistence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"task-processor/internal/agent"
	d "task-processor/internal/skill"
)

// InstallOfficial is a bounded release command. Its identity is global, not
// actor-derived. A declared successor appends only to its exact enabled original
// head. Existing releases and manually advanced heads are never overwritten.
func (s *Store) InstallOfficial(ctx context.Context, p d.Principal, resolve d.ResolvePrincipal) (d.OfficialInstallResult, error) {
	return s.installOfficial(ctx, p, resolve, false)
}

// PrepareOfficial is called only by the trusted application startup path.
// It never acts on behalf of a user and never modifies an existing skill.
func (s *Store) PrepareOfficial(ctx context.Context) error {
	_, err := s.installOfficial(ctx, d.Principal{UserID: "system:official-skills"}, nil, true)
	return err
}

func (s *Store) installOfficial(ctx context.Context, p d.Principal, resolve d.ResolvePrincipal, automatic bool) (d.OfficialInstallResult, error) {
	out := d.OfficialInstallResult{Installed: []agent.SkillVersionRef{}, Existing: []agent.SkillVersionRef{}}
	if !automatic && (!p.PlatformMaintainer || !agent.ValidID(p.UserID) || resolve == nil) {
		return out, d.ErrForbidden
	}
	packages, err := d.OfficialPackages()
	if err != nil {
		return out, err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		live := p
		if !automatic {
			live, err = resolve(ctx)
			if err != nil {
				return err
			}
			if live != p || !live.PlatformMaintainer {
				return d.ErrForbidden
			}
		}
		if err = tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", "skill:official-method-v1").Error; err != nil {
			return err
		}
		for _, release := range packages {
			id := uuid.NewSHA1(uuid.NameSpaceOID, []byte("skill:"+release.Key)).String()
			if automatic {
				var row skillRow
				result := tx.Where("id=?", id).Take(&row)
				if result.Error == nil {
					if row.OwnerKind != d.PlatformOwner || row.OwnerID != d.PlatformID {
						return d.ErrConflict
					}
					if _, err := readVersion(tx, id, 0, true); err != nil {
						return d.ErrConflict
					}
					continue
				}
				if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
					return result.Error
				}
			}
			existing, readErr := readVersion(tx, id, release.Version, true)
			if readErr == nil {
				if existing.Metadata.Owner != (d.Owner{Kind: d.PlatformOwner, ID: d.PlatformID}) || existing.Package.Digest != release.Package.Digest || existing.Source != nil {
					return d.ErrConflict
				}
				out.Existing = append(out.Existing, existing.Ref())
				continue
			}
			if !errors.Is(readErr, d.ErrNotFound) {
				return readErr
			}
			original, originalErr := readVersion(tx, id, 1, true)
			basePackage := release.Package
			if release.Previous != nil {
				basePackage = *release.Previous
			}
			now := time.Now().UTC()
			if errors.Is(originalErr, d.ErrNotFound) {
				// A row without its original immutable version is a conflict too.
				var count int64
				if err = tx.Model(&skillRow{}).Where("id=?", id).Count(&count).Error; err != nil {
					return err
				}
				if count != 0 {
					return d.ErrConflict
				}
				original = d.Version{Metadata: d.Metadata{ID: id, Owner: d.Owner{Kind: d.PlatformOwner, ID: d.PlatformID}, Creator: p.UserID, HeadVersion: 1, Revision: 1, CreatedAt: now, UpdatedAt: now}, Number: 1, Package: basePackage, State: d.Draft, Visibility: d.Private, AllowCopy: true, AllowExport: true, CreatedAt: now}
				if automatic {
					original.State, original.Visibility = d.Enabled, d.Shared
					original.Metadata.Revision += 2
				} else {
					for _, action := range []string{"publish", "enable"} {
						original.State, original.Visibility, err = d.Transition(live, original, action)
						if err != nil {
							return err
						}
						original.Metadata.Revision++
					}
				}
				row := skillRow{ID: id, OwnerKind: d.PlatformOwner, OwnerID: d.PlatformID, Creator: p.UserID, HeadVersion: 1, Revision: original.Metadata.Revision, CreatedAt: now, UpdatedAt: now}
				if err = tx.Create(&row).Error; err != nil {
					return err
				}
				if err = insertVersion(tx, original); err != nil {
					return err
				}
			} else if originalErr != nil {
				return originalErr
			}
			if original.Metadata.Owner != (d.Owner{Kind: d.PlatformOwner, ID: d.PlatformID}) || original.Package.Digest != basePackage.Digest || original.Source != nil {
				return d.ErrConflict
			}
			if release.Version == 1 {
				out.Installed = append(out.Installed, original.Ref())
				continue
			}
			if release.Version != 2 || release.Previous == nil || original.Metadata.HeadVersion != 1 || original.State != d.Enabled || original.Visibility != d.Shared {
				return d.ErrConflict
			}
			v := d.Version{Metadata: original.Metadata, Number: 2, Package: release.Package, State: d.Draft, Visibility: d.Private, AllowCopy: true, AllowExport: true, CreatedAt: now}
			v.Metadata.HeadVersion, v.Metadata.UpdatedAt = 2, now
			v.Metadata.Revision++
			if automatic {
				v.State, v.Visibility = d.Enabled, d.Shared
				v.Metadata.Revision += 2
			} else {
				for _, action := range []string{"publish", "enable"} {
					v.State, v.Visibility, err = d.Transition(live, v, action)
					if err != nil {
						return err
					}
					v.Metadata.Revision++
				}
			}
			if err = insertVersion(tx, v); err != nil {
				return err
			}
			updated := tx.Model(&skillRow{}).Where("id=? AND head_version=1 AND revision=?", id, original.Metadata.Revision).Updates(map[string]any{"head_version": 2, "revision": v.Metadata.Revision, "updated_at": now})
			if updated.Error != nil {
				return updated.Error
			}
			if updated.RowsAffected != 1 {
				return d.ErrConflict
			}
			out.Installed = append(out.Installed, v.Ref())
		}
		return nil
	})
	if err != nil {
		return d.OfficialInstallResult{}, err
	}
	return out, nil
}
