package listingadmin

import (
	"fmt"

	"gorm.io/gorm"
)

func ensureOwnerAuditColumns(db *gorm.DB, table string) error {
	if db == nil {
		return fmt.Errorf("database is not configured")
	}
	columnDefinitions := map[string]string{
		"owner_user_id": "varchar(128)",
		"created_by":    "varchar(128)",
		"updated_by":    "varchar(128)",
	}
	for column, definition := range columnDefinitions {
		hasColumn := db.Migrator().HasColumn(table, column)
		if !hasColumn {
			statement := fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN "%s" %s`, table, column, definition)
			if err := db.Exec(statement).Error; err != nil {
				return err
			}
		}
	}
	statements := []string{
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS "idx_%s_owner_user_id" ON "%s" (owner_user_id)`, table, table),
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return ensureOwnerUserIDConstraint(db, table)
}
