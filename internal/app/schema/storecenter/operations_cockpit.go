package storecenter

import (
	"context"
	"errors"
	cockpit "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/operationscockpit"
	"gorm.io/gorm"
)

// InstallOperationsCockpit provisions a fresh empty Store instance explicitly.
// Runtime startup verifies the schema; it never installs or alters it.
func InstallOperationsCockpit(ctx context.Context, records *gorm.DB) error {
	if ctx == nil || records == nil {
		return errors.New("Cockpit schema-owner unavailable")
	}
	return records.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Table("public.workbench_stores").Count(&count).Error; err != nil || count != 0 {
			return errors.New("cockpit initialization requires a fresh empty Store instance")
		}
		if err := cockpit.Install(ctx, tx); err != nil {
			return err
		}
		return cockpit.GrantRuntime(ctx, tx, "store_center_runtime")
	})
}
