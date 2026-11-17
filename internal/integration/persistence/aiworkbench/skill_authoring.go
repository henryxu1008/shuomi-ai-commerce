package aiworkbenchpersistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"task-processor/internal/aiworkbench"
)

// AuthoringDraft is the workbench receipt and pending save intent.
// The Skill store owns the saved version and its create receipt.
type AuthoringDraft struct {
	OrganizationID string
	ActorID        string
	ConversationID string
	Key            string
	MemberID       string
	State          string
	Requirement    string
	SkillVersion   uint64
	SkillDigest    string
	InputHash      string
	ProfileDigest  string
	MaximumPoints  int64
	InvocationID   string
	ExpiresAt      time.Time
	Deadline       *time.Time
	Generated      string
	SaveKey        *string
	SaveDigest     string
	SaveContent    string
	SavedSkillID   *string
	CreatedAt      time.Time
}

func (AuthoringDraft) TableName() string { return "ai_workbench.skill_authoring_drafts" }

func (s *Store) CanAuthorSkills(ctx context.Context) bool {
	return s != nil && s.db.WithContext(ctx).Model(&AuthoringDraft{}).Select("key").Limit(0).Find(&[]AuthoringDraft{}).Error == nil
}

func (s *Store) authoringQuery(ctx context.Context, scope aiworkbench.Scope, conversation string) *gorm.DB {
	return s.db.WithContext(ctx).Model(&AuthoringDraft{}).
		Where("organization_id = ? AND actor_id = ? AND conversation_id = ?", scope.OrganizationID, scope.ActorID, conversation)
}

func (s *Store) AuthoringDraft(ctx context.Context, scope aiworkbench.Scope, conversation, key string) (AuthoringDraft, error) {
	var row AuthoringDraft
	if !validScope(scope) || !validKey(conversation) || !validKey(key) {
		return row, aiworkbench.ErrInvalid
	}
	err := s.authoringQuery(ctx, scope, conversation).Where("key = ?", key).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return row, aiworkbench.ErrNotFound
	}
	if err != nil {
		return row, aiworkbench.ErrUnavailable
	}
	return row, nil
}

func (s *Store) AuthoringDrafts(ctx context.Context, scope aiworkbench.Scope, conversation string) ([]AuthoringDraft, error) {
	rows := []AuthoringDraft{}
	err := s.authoringQuery(ctx, scope, conversation).Where("state <> 'QUOTED'").
		Order("(save_key IS NOT NULL AND saved_skill_id IS NULL OR state = 'READY') DESC, created_at DESC, key").Limit(8).Find(&rows).Error
	if err != nil {
		return nil, aiworkbench.ErrUnavailable
	}
	return rows, nil
}

func (s *Store) QuoteAuthoring(ctx context.Context, scope aiworkbench.Scope, quote AuthoringDraft) (AuthoringDraft, error) {
	quote.OrganizationID, quote.ActorID = scope.OrganizationID, scope.ActorID
	quote.State = "QUOTED"
	result := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&quote)
	if result.Error != nil {
		return AuthoringDraft{}, aiworkbench.ErrUnavailable
	}
	if result.RowsAffected == 1 {
		return quote, nil
	}
	original, err := s.AuthoringDraft(ctx, scope, quote.ConversationID, quote.Key)
	if err != nil {
		return original, err
	}
	if original.MemberID != quote.MemberID || original.InputHash != quote.InputHash || original.SkillDigest != quote.SkillDigest ||
		original.SkillVersion != quote.SkillVersion || original.MaximumPoints != quote.MaximumPoints || original.State != "QUOTED" {
		return original, aiworkbench.ErrIdempotencyConflict
	}
	return original, nil
}

func (s *Store) ConfirmAuthoring(ctx context.Context, scope aiworkbench.Scope, row AuthoringDraft, now, deadline time.Time) (AuthoringDraft, error) {
	result := s.authoringQuery(ctx, scope, row.ConversationID).
		Where("key = ? AND member_id = ? AND state = 'QUOTED' AND expires_at > ?", row.Key, row.MemberID, now).
		Updates(map[string]any{"state": "READY", "deadline": deadline})
	if result.Error != nil {
		return row, aiworkbench.ErrUnavailable
	}
	return s.AuthoringDraft(ctx, scope, row.ConversationID, row.Key)
}

func (s *Store) FinishAuthoring(ctx context.Context, scope aiworkbench.Scope, row AuthoringDraft, state, generated string) (AuthoringDraft, error) {
	result := s.authoringQuery(ctx, scope, row.ConversationID).Where("key = ? AND state = 'READY'", row.Key).
		Updates(map[string]any{"state": state, "generated": generated})
	if result.Error != nil {
		return row, aiworkbench.ErrUnavailable
	}
	return s.AuthoringDraft(ctx, scope, row.ConversationID, row.Key)
}

func (s *Store) FreezeAuthoringSave(ctx context.Context, scope aiworkbench.Scope, row AuthoringDraft, saveKey, contentDigest, content string) (AuthoringDraft, error) {
	result := s.authoringQuery(ctx, scope, row.ConversationID).Where("key = ? AND state = 'COMPLETE' AND save_key IS NULL", row.Key).
		Updates(map[string]any{"save_key": saveKey, "save_digest": contentDigest, "save_content": content})
	if result.Error != nil {
		return row, aiworkbench.ErrUnavailable
	}
	current, err := s.AuthoringDraft(ctx, scope, row.ConversationID, row.Key)
	if err != nil {
		return current, err
	}
	if current.SaveKey == nil || current.SaveDigest != contentDigest {
		return current, aiworkbench.ErrIdempotencyConflict
	}
	return current, nil
}

func (s *Store) FinishAuthoringSave(ctx context.Context, scope aiworkbench.Scope, row AuthoringDraft, skillID string) (AuthoringDraft, error) {
	result := s.authoringQuery(ctx, scope, row.ConversationID).Where("key = ? AND save_key = ? AND saved_skill_id IS NULL", row.Key, row.SaveKey).
		Updates(map[string]any{"saved_skill_id": skillID, "save_content": ""})
	if result.Error != nil {
		return row, aiworkbench.ErrUnavailable
	}
	current, err := s.AuthoringDraft(ctx, scope, row.ConversationID, row.Key)
	if err != nil {
		return current, err
	}
	if current.SavedSkillID == nil || *current.SavedSkillID != skillID {
		return current, aiworkbench.ErrIdempotencyConflict
	}
	return current, nil
}
