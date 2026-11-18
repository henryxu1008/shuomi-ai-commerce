package aiworkbenchpersistence

import (
	"context"

	"gorm.io/gorm"
	"task-processor/internal/aiworkbench"
)

func authoringLayout(db *gorm.DB) (int, error) {
	var count int
	err := db.Raw(`SELECT count(*) FROM pg_attribute
		WHERE attrelid=to_regclass('ai_workbench.skill_authoring_drafts') AND attnum>0 AND NOT attisdropped
		AND attname IN ('format_version','input_snapshot','root_key','base_revision','current_revision','save_format')`).Scan(&count).Error
	if err != nil {
		return 0, err
	}
	switch count {
	case 0:
		return 1, nil
	case 6:
		return 2, nil
	default:
		return 0, aiworkbench.ErrInvalid
	}
}

// InstallSkillAuthoringUpgrade upgrades the original personal authoring table.
// It runs only via the existing schema owner's explicitly targeted initializer.
func InstallSkillAuthoringUpgrade(ctx context.Context, db *gorm.DB, role string) error {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" || role != "ai_workbench_runtime" {
		return aiworkbench.ErrInvalid
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended('ai-workbench-skill-authoring-layout',0))`).Error; err != nil {
			return err
		}
		if !tx.Migrator().HasTable("ai_workbench.skill_authoring_drafts") {
			return aiworkbench.ErrNotFound
		}
		if err := validateBoundedWorkbenchPrivileges(tx, role); err != nil {
			return err
		}
		layout, err := authoringLayout(tx)
		if err != nil || layout == 2 {
			return err
		}
		if err := tx.Exec(`ALTER TABLE ai_workbench.skill_authoring_drafts
			ADD COLUMN format_version integer NOT NULL DEFAULT 1 CHECK (format_version IN (1,2)),
			ADD COLUMN input_snapshot text NOT NULL DEFAULT '' CHECK (octet_length(input_snapshot)<=65536),
			ADD COLUMN root_key uuid,
			ADD COLUMN base_revision uuid,
			ADD COLUMN current_revision uuid,
			ADD COLUMN save_format integer NOT NULL DEFAULT 1 CHECK (save_format IN (1,2)),
			DROP CONSTRAINT skill_authoring_drafts_save_content_check,
			ADD CHECK (octet_length(save_content)<=65536),
			ADD CHECK ((root_key IS NULL)=(base_revision IS NULL)),
			ADD FOREIGN KEY (organization_id,actor_id,conversation_id,root_key)
				REFERENCES ai_workbench.skill_authoring_drafts (organization_id,actor_id,conversation_id,key)`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`GRANT UPDATE (current_revision, save_format)
			ON ai_workbench.skill_authoring_drafts TO ai_workbench_runtime`).Error; err != nil {
			return err
		}
		return validateBoundedWorkbenchPrivileges(tx, role)
	})
}
