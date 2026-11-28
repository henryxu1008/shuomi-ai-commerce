package store

import (
	"context"
	"errors"
	"strings"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

func isColorReworkUniqueConflict(err error) bool {
	var postgres *pgconn.PgError
	if errors.As(err, &postgres) {
		return postgres.Code == "23505" && postgres.ConstraintName == "image_agent_color_rework_confirmed"
	}
	// The existing SQLite-backed repository tests use the identical unique index.
	return strings.Contains(err.Error(), "UNIQUE constraint failed: image_agent_v2_runs.tenant_id, image_agent_v2_runs.color_rework_key")
}

func (r *gormRepository) FindConfirmedColorImageRework(ctx context.Context, scope imageagent.RunScope, key string) (imageagent.RunProjection, error) {
	var run runRecord
	err := r.db.WithContext(ctx).Select("id").Where("tenant_id = ? AND owner_user_id = ? AND color_rework_key = ?", scope.TenantID, scope.OwnerUserID, key).Take(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return imageagent.RunProjection{}, imageagent.ErrRunNotFound
	}
	if err != nil {
		return imageagent.RunProjection{}, err
	}
	scope.RunID = run.ID
	return r.GetProjection(ctx, scope)
}

func (r *memoryRepository) FindConfirmedColorImageRework(_ context.Context, scope imageagent.RunScope, key string) (imageagent.RunProjection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, value := range r.projections {
		if value.Run.TenantID == scope.TenantID && value.Run.UserID == scope.OwnerUserID &&
			value.Run.ImageAdmission != nil && imageagent.ConfirmedColorImageReworkKey(value.Plan) == key {
			return cloneProjection(value), nil
		}
	}
	return imageagent.RunProjection{}, imageagent.ErrRunNotFound
}
