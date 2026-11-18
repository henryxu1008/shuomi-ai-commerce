package aiworkbenchpersistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/skill"
)

// AuthoringDraft is the workbench receipt and pending save intent.
// The Skill store owns the saved version and its create receipt.
type AuthoringDraft struct {
	OrganizationID  string
	ActorID         string
	ConversationID  string
	Key             string
	MemberID        string
	State           string
	Requirement     string
	SkillVersion    uint64
	SkillDigest     string
	InputHash       string
	ProfileDigest   string
	MaximumPoints   int64
	InvocationID    string
	ExpiresAt       time.Time
	Deadline        *time.Time
	Generated       string
	FormatVersion   int `gorm:"default:1"`
	InputSnapshot   string
	RootKey         *string
	BaseRevision    *string
	CurrentRevision *string
	SaveKey         *string
	SaveDigest      string
	SaveContent     string
	SaveFormat      int `gorm:"default:1"`
	SavedSkillID    *string
	CreatedAt       time.Time
}

func (AuthoringDraft) TableName() string { return "ai_workbench.skill_authoring_drafts" }

func (s *Store) CanAuthorSkills(ctx context.Context) bool {
	return s != nil && s.db.WithContext(ctx).Model(&AuthoringDraft{}).Select("key, format_version, current_revision, save_format").Limit(0).Find(&[]AuthoringDraft{}).Error == nil
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

func (s *Store) AuthoringDrafts(ctx context.Context, scope aiworkbench.Scope, conversation, member string) ([]AuthoringDraft, error) {
	rows := []AuthoringDraft{}
	err := s.authoringQuery(ctx, scope, conversation).Where("member_id = ? AND state <> 'QUOTED' AND (root_key IS NULL OR state IN ('READY', 'UNKNOWN'))", member).
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
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		store := &Store{db: tx}
		query := store.authoringQuery(ctx, scope, row.ConversationID)
		result := query.Where("key = ? AND state = 'READY'", row.Key).Updates(map[string]any{"state": state, "generated": generated})
		if result.Error != nil || result.RowsAffected != 1 || state != "COMPLETE" || row.RootKey == nil {
			return result.Error
		}
		return store.authoringQuery(ctx, scope, row.ConversationID).
			Where("key = ? AND member_id = ? AND root_key IS NULL AND state = 'COMPLETE' AND save_key IS NULL AND COALESCE(current_revision, key) = ?", *row.RootKey, row.MemberID, row.BaseRevision).
			Update("current_revision", row.Key).Error
	})
	if err != nil {
		return row, aiworkbench.ErrUnavailable
	}
	return s.AuthoringDraft(ctx, scope, row.ConversationID, row.Key)
}

// FinishAuthoringRejected projects a confirmed post-dispatch rejection.
// Late commercial finalization can close the original UNKNOWN. The caller
// must verify the original zero-charge receipt before using this transition.
func (s *Store) FinishAuthoringRejected(ctx context.Context, scope aiworkbench.Scope, row AuthoringDraft) (AuthoringDraft, error) {
	result := s.authoringQuery(ctx, scope, row.ConversationID).
		Where("key = ? AND member_id = ? AND invocation_id = ? AND state IN ?", row.Key, row.MemberID, row.InvocationID, []string{"READY", "UNKNOWN"}).
		Update("state", "FAILED")
	if result.Error != nil {
		return row, aiworkbench.ErrUnavailable
	}
	return s.AuthoringDraft(ctx, scope, row.ConversationID, row.Key)
}

// AuthoringRevision returns the active generation key for a draft family.
func (row AuthoringDraft) AuthoringRevision() string {
	if row.CurrentRevision != nil {
		return *row.CurrentRevision
	}
	return row.Key
}

func (s *Store) checkAuthoringBase(ctx context.Context, scope aiworkbench.Scope, row AuthoringDraft, lock bool) error {
	if row.RootKey == nil {
		return nil
	}
	var root AuthoringDraft
	query := s.authoringQuery(ctx, scope, row.ConversationID).Where("key = ? AND root_key IS NULL AND member_id = ?", *row.RootKey, row.MemberID)
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := query.Take(&root).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return aiworkbench.ErrNotFound
	}
	if err != nil {
		return aiworkbench.ErrUnavailable
	}
	if row.BaseRevision == nil || root.AuthoringRevision() != *row.BaseRevision || root.SaveKey != nil || root.State != "COMPLETE" ||
		(root.SkillVersion == skill.GeneralAuthoringVersion) != (row.SkillVersion == skill.GeneralAuthoringVersion) {
		return aiworkbench.ErrProposalStale
	}
	return nil
}

func (s *Store) CheckAuthoringBase(ctx context.Context, scope aiworkbench.Scope, row AuthoringDraft) error {
	return s.checkAuthoringBase(ctx, scope, row, false)
}

func (s *Store) AuthoringDispatch(ctx context.Context, scope aiworkbench.Scope, row AuthoringDraft) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return (&Store{db: tx}).checkAuthoringBase(ctx, scope, row, true)
	})
}

func (s *Store) StartAuthoring(ctx context.Context, scope aiworkbench.Scope, row AuthoringDraft) (AuthoringDraft, error) {
	row.OrganizationID, row.ActorID, row.State = scope.OrganizationID, scope.ActorID, "READY"
	var result AuthoringDraft
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		store := &Store{db: tx}
		if err := store.lockKey(tx, scope, "skill-authoring:"+row.ConversationID+":"+row.Key); err != nil {
			return aiworkbench.ErrUnavailable
		}
		original, err := store.AuthoringDraft(ctx, scope, row.ConversationID, row.Key)
		if err == nil {
			if !original.MatchesAuthoringInput(row) {
				return aiworkbench.ErrIdempotencyConflict
			}
			result = original
			return nil
		}
		if !errors.Is(err, aiworkbench.ErrNotFound) {
			return err
		}
		if err := store.checkAuthoringBase(ctx, scope, row, true); err != nil {
			return err
		}
		var valid bool
		if err := tx.Raw("SELECT clock_timestamp() < ?", row.ExpiresAt).Scan(&valid).Error; err != nil {
			return aiworkbench.ErrUnavailable
		}
		if !valid {
			return aiworkbench.ErrProposalStale
		}
		if err := tx.Create(&row).Error; err != nil {
			return aiworkbench.ErrUnavailable
		}
		result = row
		return nil
	})
	return result, err
}

func (row AuthoringDraft) MatchesAuthoringInput(other AuthoringDraft) bool {
	return row.MemberID == other.MemberID && row.FormatVersion == other.FormatVersion && row.Requirement == other.Requirement &&
		row.SkillVersion == other.SkillVersion && row.SkillDigest == other.SkillDigest && row.InputSnapshot == other.InputSnapshot &&
		sameAuthoringReference(row.RootKey, other.RootKey) && sameAuthoringReference(row.BaseRevision, other.BaseRevision)
}

func sameAuthoringReference(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

// AuthoringConfirmation checks absence only after serializing with confirmation.
func (s *Store) AuthoringConfirmation(ctx context.Context, scope aiworkbench.Scope, conversation, key string, expires time.Time) (AuthoringDraft, error) {
	var row AuthoringDraft
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		store := &Store{db: tx}
		if err := store.lockKey(tx, scope, "skill-authoring:"+conversation+":"+key); err != nil {
			return aiworkbench.ErrUnavailable
		}
		var err error
		row, err = store.AuthoringDraft(ctx, scope, conversation, key)
		if !errors.Is(err, aiworkbench.ErrNotFound) {
			return err
		}
		var expired bool
		if err := tx.Raw("SELECT clock_timestamp() >= ?", expires).Scan(&expired).Error; err != nil {
			return aiworkbench.ErrUnavailable
		}
		if expired {
			return aiworkbench.ErrProposalStale
		}
		return aiworkbench.ErrUnavailable
	})
	return row, err
}

func (s *Store) FreezeAuthoringSave(ctx context.Context, scope aiworkbench.Scope, row AuthoringDraft, saveKey, contentDigest, content string) (AuthoringDraft, error) {
	return s.FreezeAuthoringPackage(ctx, scope, row, row.Key, saveKey, contentDigest, content, 1)
}

func (s *Store) FreezeAuthoringPackage(ctx context.Context, scope aiworkbench.Scope, row AuthoringDraft, revision, saveKey, contentDigest, content string, format int) (AuthoringDraft, error) {
	result := s.authoringQuery(ctx, scope, row.ConversationID).
		Where("key = ? AND member_id = ? AND root_key IS NULL AND state = 'COMPLETE' AND COALESCE(current_revision, key) = ? AND save_key IS NULL", row.Key, row.MemberID, revision).
		Updates(map[string]any{"save_key": saveKey, "save_digest": contentDigest, "save_content": content, "save_format": format})
	if result.Error != nil {
		return row, aiworkbench.ErrUnavailable
	}
	current, err := s.AuthoringDraft(ctx, scope, row.ConversationID, row.Key)
	if err != nil {
		return current, err
	}
	if current.MemberID != row.MemberID || current.AuthoringRevision() != revision || current.SaveKey == nil || current.SaveDigest != contentDigest {
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
