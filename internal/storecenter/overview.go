package storecenter

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"gorm.io/gorm"
)

type OverviewScope string

const (
	OverviewAll    OverviewScope = "ALL_NON_DELETED"
	OverviewActive OverviewScope = "ACTIVE"
)

// LiveOverviewAuthorizer resolves identity and store read permission afresh on
// every call; a context-only identity snapshot is insufficient.
type LiveOverviewAuthorizer interface {
	AuthorizeLiveOverview(context.Context, string) (StoreMemberAccess, error)
}

// OverviewIdentity is bound by the authenticated Chat request, never model input.
type OverviewIdentity struct{ OrganizationID, ActorID, MemberID string }

func authorizeOverview(ctx context.Context, org string, live LiveOverviewAuthorizer) (StoreMemberAccess, error) {
	if isNilDependency(live) {
		return StoreMemberAccess{}, ErrDependencyUnavailable
	}
	access, err := live.AuthorizeLiveOverview(ctx, org)
	if err != nil {
		return StoreMemberAccess{}, err
	}
	if access.OrganizationID != org {
		return StoreMemberAccess{}, ErrNotFound
	}
	if _, err = validateOpaqueIdentity("actor", access.ActorID, MaxSubjectBytes); err != nil {
		return StoreMemberAccess{}, ErrNotFound
	}
	if _, err = validateOpaqueIdentity("member", access.MemberID, MaxSubjectBytes); err != nil {
		return StoreMemberAccess{}, ErrNotFound
	}
	return access, nil
}

// VisibleStoreOverview is a live, member-authorized projection, not a stored fact.
type VisibleStoreOverview struct {
	Total                int64            `json:"total"`
	Platforms            map[string]int64 `json:"platforms"`
	Statuses             map[string]int64 `json:"statuses"`
	OperatingTypeUnknown int64            `json:"operatingTypeUnknown"`
	ReadStartedAt        time.Time        `json:"readStartedAt"`
	ActivePlatforms      map[string]int64 `json:"-"`
}

// VisibleOverview reads only store identity, platform, and status. It never
// consumes the catalog's order/inventory/connection signals.
func (c *Catalog) VisibleOverview(ctx context.Context, identity OverviewIdentity, scope OverviewScope, live LiveOverviewAuthorizer) (VisibleStoreOverview, error) {
	empty := VisibleStoreOverview{}
	org := identity.OrganizationID
	if c == nil || c.scoped == nil || (scope != OverviewAll && scope != OverviewActive) {
		return empty, ErrDependencyUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	access, err := authorizeOverview(ctx, org, live)
	if err != nil {
		return empty, err
	}
	if access.ActorID != identity.ActorID || access.MemberID != identity.MemberID {
		return empty, ErrNotFound
	}
	records := []workbenchStoreRecord{}
	readStartedAt := time.Now().UTC()
	err = c.scoped.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Select("id", "platform", "record_status").Where("organization_id=? AND deleted_at IS NULL", org)
		if !access.Administrator {
			query = query.Where("EXISTS(SELECT 1 FROM workbench_store_member_grants g WHERE g.organization_id=workbench_stores.organization_id AND g.store_id=workbench_stores.id AND g.member_id=? AND g.active=?)", access.MemberID, true)
		}
		return query.Order("id ASC").Find(&records).Error
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return empty, dependencyError(err)
	}
	out := VisibleStoreOverview{Platforms: map[string]int64{}, Statuses: map[string]int64{}, ActivePlatforms: map[string]int64{}, ReadStartedAt: readStartedAt}
	for _, row := range records {
		platform, platformErr := normalizePlatform(row.Platform)
		status := RecordStatus(row.RecordStatus)
		if platformErr != nil || string(platform) != row.Platform || !validRecordStatus(status) {
			return empty, dependencyError(errors.New("invalid store overview fact"))
		}
		if scope == OverviewActive && status != RecordStatusActive {
			continue
		}
		out.Total++
		out.Platforms[string(platform)]++
		out.Statuses[string(status)]++
		if status == RecordStatusActive {
			out.ActivePlatforms[string(platform)]++
		}
	}
	out.OperatingTypeUnknown = out.Total
	again, err := authorizeOverview(ctx, org, live)
	if err != nil || again != access {
		return empty, ErrNotFound
	}
	current := []workbenchStoreRecord{}
	query := c.scoped.db.WithContext(ctx).Select("id").Where("organization_id=? AND deleted_at IS NULL", org)
	if !access.Administrator {
		query = query.Where("EXISTS(SELECT 1 FROM workbench_store_member_grants g WHERE g.organization_id=workbench_stores.organization_id AND g.store_id=workbench_stores.id AND g.member_id=? AND g.active=?)", access.MemberID, true)
	}
	if err = query.Find(&current).Error; err != nil {
		return empty, dependencyError(err)
	}
	visible := make(map[string]bool, len(current))
	for _, row := range current {
		visible[row.ID] = true
	}
	if len(visible) != len(records) {
		return empty, ErrNotFound
	}
	for _, row := range records {
		if !visible[row.ID] {
			return empty, ErrNotFound
		}
	}
	return out, nil
}
