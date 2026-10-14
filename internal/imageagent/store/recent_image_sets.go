package store

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"task-processor/internal/agentconfig"
	"task-processor/internal/imageagent"
)

func (r *gormRepository) ListImageSets(ctx context.Context, identity imageagent.ExecutionIdentity, contextID, cursor string, size int) ([]imageagent.ImageSetRunSummary, string, error) {
	return r.listVisualRuns(ctx, identity, contextID, cursor, size, false)
}
func (r *gormRepository) ListArtworkRuns(ctx context.Context, identity imageagent.ExecutionIdentity, cursor string, size int) ([]imageagent.ImageSetRunSummary, string, error) {
	return r.listVisualRuns(ctx, identity, "", cursor, size, true)
}
func (r *gormRepository) listVisualRuns(ctx context.Context, identity imageagent.ExecutionIdentity, contextID, cursor string, size int, artwork bool) ([]imageagent.ImageSetRunSummary, string, error) {
	if r == nil || ctx == nil || r.scopeProtocol != imageagent.OrganizationScopeProtocol || identity.ScopeProtocol != r.scopeProtocol || identity.TenantID == "" || identity.UserID == "" || identity.MemberID == "" || size < 1 || size > 40 || cursor != "" && !agentconfig.UUID(cursor) {
		return nil, "", imageagent.ErrValidation
	}
	query := r.db.WithContext(ctx).Model(&runRecord{}).Where("scope_protocol = ? AND tenant_id = ? AND owner_user_id = ? AND member_id = ?", r.scopeProtocol, identity.TenantID, identity.UserID, identity.MemberID)
	if artwork {
		query = query.Where("target_platform = ?", "sds")
	}
	if contextID != "" {
		query = query.Where("business_task_id = ?", contextID)
	}
	if cursor != "" {
		var previous runRecord
		err := r.db.WithContext(ctx).Where("scope_protocol = ? AND tenant_id = ? AND owner_user_id = ? AND member_id = ? AND id = ?", r.scopeProtocol, identity.TenantID, identity.UserID, identity.MemberID, cursor).Take(&previous).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", imageagent.ErrValidation
		}
		if err != nil {
			return nil, "", err
		}
		if artwork && previous.TargetPlatform != "sds" {
			return nil, "", imageagent.ErrValidation
		}
		if contextID != "" && previous.BusinessTaskID != contextID {
			return nil, "", imageagent.ErrValidation
		}
		query = query.Where("created_at < ? OR (created_at = ? AND id < ?)", previous.CreatedAt, previous.CreatedAt, previous.ID)
	}
	var rows []runRecord
	if err := query.Order("created_at DESC, id DESC").Limit(size + 1).Find(&rows).Error; err != nil {
		return nil, "", err
	}
	next := ""
	if len(rows) > size {
		rows = rows[:size]
		next = rows[len(rows)-1].ID
	}
	result := make([]imageagent.ImageSetRunSummary, 0, len(rows))
	for _, row := range rows {
		projection, err := r.GetProjection(ctx, imageagent.RunScope{TenantID: identity.TenantID, OwnerUserID: identity.UserID, RunID: row.ID})
		if err != nil {
			return nil, "", err
		}
		plan := projection.Plan
		if artwork {
			if plan.Artwork == nil {
				continue
			}
			if err = imageagent.ValidateArtworkPlan(plan); err != nil {
				return nil, "", err
			}
			result = append(result, imageagent.ImageSetRunSummary{RunID: row.ID, ContextKind: imageagent.ImageSourceContextKind(imageagent.ArtworkSchema), ContextID: row.BusinessTaskID, Status: imageagent.RunStatus(row.Status), TargetPlatform: row.TargetPlatform, CreatedAt: row.CreatedAt})
			continue
		}
		if plan.Set == nil {
			continue
		}
		if err = imageagent.ValidateImageSetPlan(plan); err != nil {
			return nil, "", err
		}
		result = append(result, imageagent.ImageSetRunSummary{RunID: row.ID, ContextKind: plan.Set.Source.ContextKind, ContextID: row.BusinessTaskID, Status: imageagent.RunStatus(row.Status), TargetPlatform: row.TargetPlatform, CreatedAt: row.CreatedAt})
	}
	return result, next, nil
}
