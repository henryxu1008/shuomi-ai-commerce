package projectcenterpersistence

import (
	"context"
	"database/sql"
	"gorm.io/gorm"
	pc "task-processor/internal/aiworkbench/projectcenter"
	"time"
)

func (s *Store) Overview(ctx context.Context, scope pc.Scope, mode string, start, end time.Time) (pc.OverviewSnapshot, error) {
	out := pc.OverviewSnapshot{Scope: scope, TaskIDs: []string{}, StoreIDs: []string{}}
	e := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var counts struct{ ActiveProjects, TodayUpdatedProjects, StoreProjects, GeneralProjects int64 }
		if e := tx.Raw(`SELECT count(*) AS active_projects,
   count(*) FILTER (WHERE updated_at >= ? AND updated_at < ?) AS today_updated_projects,
   count(*) FILTER (WHERE store_id <> '') AS store_projects,
   count(*) FILTER (WHERE store_id = '') AS general_projects
		   FROM ai_workbench_projects.projects WHERE organization_id=? AND actor_id=? AND NOT archived`, start, end, scope.OrganizationID, scope.ActorID).Scan(&counts).Error; e != nil {
			return e
		}
		out.ActiveProjects = counts.ActiveProjects
		out.TodayUpdatedProjects = counts.TodayUpdatedProjects
		out.StoreProjects = counts.StoreProjects
		out.GeneralProjects = counts.GeneralProjects
		if e := tx.Raw(`SELECT DISTINCT r.target_id FROM ai_workbench_projects.references r
   JOIN ai_workbench_projects.projects p ON p.id=r.project_id AND p.organization_id=r.organization_id AND p.actor_id=r.actor_id
   WHERE p.organization_id=? AND p.actor_id=? AND NOT p.archived AND r.active AND r.kind='BUSINESS_TASK'
   ORDER BY r.target_id LIMIT 201`, scope.OrganizationID, scope.ActorID).Scan(&out.TaskIDs).Error; e != nil {
			return e
		}
		stores := tx.Table("ai_workbench_projects.projects p").Where("p.organization_id=? AND p.actor_id=? AND p.archived=? AND p.store_id<>''", scope.OrganizationID, scope.ActorID, mode == "archived")
		if mode == "recent" {
			stores = stores.Joins("JOIN ai_workbench_projects.visits v ON v.project_id=p.id AND v.organization_id=p.organization_id AND v.actor_id=p.actor_id")
		}
		return stores.Distinct("p.store_id").Order("p.store_id").Limit(101).Pluck("p.store_id", &out.StoreIDs).Error
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if e != nil {
		return pc.OverviewSnapshot{}, pc.ErrUnavailable
	}
	return out, nil
}
