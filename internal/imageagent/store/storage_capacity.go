package store

import (
	"context"
	"task-processor/internal/imageagent"
)

// This internal bounded lookup resolves the original owner from its immutable
// run. It creates no projection or generation fact and grants no dispatch.
func (r *gormRepository) ReadGenerationFactForStorage(ctx context.Context, org, ownerHash, run string, revision int64, slot string, attempt int64) (imageagent.GenerationFact, error) {
	var runs []runRecord
	if e := r.db.WithContext(ctx).Select("tenant_id, owner_user_id, id").Where("tenant_id = ? AND id = ?", org, run).Limit(2).Find(&runs).Error; e != nil {
		return imageagent.GenerationFact{}, e
	}
	if len(runs) != 1 {
		return imageagent.GenerationFact{}, imageagent.ErrRunNotFound
	}
	key, e := imageagent.ArtifactOwnerKey(runs[0].UserID)
	if e != nil || key != ownerHash {
		return imageagent.GenerationFact{}, imageagent.ErrIdentityRequired
	}
	return r.ReadGenerationFact(ctx, imageagent.SlotExternalEffectIdentity{RunScope: imageagent.RunScope{TenantID: org, OwnerUserID: runs[0].UserID, RunID: run}, PlanRevision: revision, SlotID: slot, Attempt: int(attempt)})
}
