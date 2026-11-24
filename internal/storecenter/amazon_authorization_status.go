package storecenter

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const amazonAuthorizationRenewal = 365 * 24 * time.Hour

// ReadAmazonAuthorizationStatus reports which authorization is saved. It does
// not test Amazon live or read the seller ID or credential envelope.
func (r *MemberScopedStoreRepository) ReadAmazonAuthorizationStatus(
	ctx context.Context, organizationID, storeID string, app AmazonAuthorizationApplication, now time.Time,
) (AmazonAuthorizationStatus, error) {
	if uuid.Validate(storeID) != nil || !validAmazonAuthorizationApp(app) || now.IsZero() {
		return "", ErrNotFound
	}
	access, err := r.authorize(ctx, organizationID, false)
	if err != nil {
		return "", err
	}

	var status AmazonAuthorizationStatus
	err = r.db.WithContext(ctx).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).Transaction(func(tx *gorm.DB) error {
		if _, err := amazonAuthorizationStore(tx, access, storeID); err != nil {
			return err
		}
		var grant struct {
			AppID, AppRevision string
			Beta               bool
			BoundAt            time.Time
		}
		err := tx.Model(&amazonAuthorizationGrantRow{}).Select("app_id", "app_revision", "beta", "bound_at").
			Where("organization_id = ? AND store_id = ?", organizationID, storeID).Take(&grant).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			status = AmazonAuthorizationStatusNotConnected
		case err != nil:
			return err
		case grant.AppID != app.ID || grant.AppRevision != app.Revision || grant.Beta != app.Beta ||
			grant.BoundAt.IsZero() || now.Before(grant.BoundAt) || !now.Before(grant.BoundAt.Add(amazonAuthorizationRenewal)):
			status = AmazonAuthorizationStatusReconnect
		default:
			status = AmazonAuthorizationStatusSaved
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return status, nil
}
