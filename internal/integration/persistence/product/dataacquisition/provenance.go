package dataacquisitionpersistence

import (
	"context"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
)

func (r *Repository) ReadCaptured(ctx context.Context, scope collection.Scope, id string) (dataacquisition.Job, dataacquisition.Item, error) {
	if scope.Validate() != nil || !collection.ValidID(id) {
		return dataacquisition.Job{}, dataacquisition.Item{}, dataacquisition.ErrInvalid
	}
	var row itemRow
	result := r.db.WithContext(ctx).Raw("SELECT "+itemColumns+" FROM data_acquisition_items WHERE organization_id=? AND actor_id=? AND id=? AND state='SAVED'", scope.OrganizationID, scope.ActorID, id).Scan(&row)
	if result.Error != nil {
		return dataacquisition.Job{}, dataacquisition.Item{}, dataacquisition.ErrUnavailable
	}
	if result.RowsAffected != 1 {
		return dataacquisition.Job{}, dataacquisition.Item{}, dataacquisition.ErrNotFound
	}
	job, err := r.Read(ctx, scope, row.JobID)
	if err != nil || job.Scope != scope {
		return dataacquisition.Job{}, dataacquisition.Item{}, dataacquisition.ErrNotFound
	}
	item, err := row.item()
	return job, item, err
}
