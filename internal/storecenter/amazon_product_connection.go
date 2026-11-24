package storecenter

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// AmazonProductConnection is a server-only, current authorization snapshot.
// Callers outside Store use the app-layer access port.
type AmazonProductConnection struct {
	Binding    ProductMerchantBinding
	SellerID   string `json:"-"`
	KeyID      string `json:"-"`
	Ciphertext string `json:"-"`
	attemptID  string
}

func (c AmazonProductConnection) CredentialAAD() []byte {
	b := c.Binding
	return AmazonAuthorizationClaim{
		Scope: AmazonBrowserScope{OrganizationID: b.OrganizationID}, StoreID: b.StoreID, AttemptID: c.attemptID,
		Application: AmazonAuthorizationApplication{ID: b.ApplicationID, Revision: b.ApplicationRevision, Beta: b.AmazonBeta},
		Generation:  b.ConnectionRevision, SellerID: c.SellerID,
	}.CredentialAAD()
}

func (r *MemberScopedStoreRepository) ReadAmazonProductConnection(
	ctx context.Context, organizationID, actorID, memberID, storeID string, app AmazonAuthorizationApplication, now time.Time,
) (AmazonProductConnection, error) {
	if ctx == nil || uuid.Validate(storeID) != nil || !validAmazonAuthorizationApp(app) || now.IsZero() {
		return AmazonProductConnection{}, ErrNotFound
	}
	access, err := r.authorize(ctx, organizationID, false)
	if err != nil || access.ActorID != actorID || access.MemberID != memberID {
		return AmazonProductConnection{}, ErrNotFound
	}
	var connection AmazonProductConnection
	err = r.db.WithContext(ctx).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).Transaction(func(tx *gorm.DB) error {
		store, err := amazonAuthorizationStore(tx, access, storeID)
		if err != nil {
			return err
		}
		if store.RecordStatus != string(RecordStatusActive) {
			return ErrNotFound
		}
		var grant amazonAuthorizationGrantRow
		if err := tx.Where("organization_id = ? AND store_id = ?", organizationID, storeID).Take(&grant).Error; err != nil {
			return amazonAuthorizationNotFound(err)
		}
		expiresAt := grant.BoundAt.Add(amazonAuthorizationRenewal)
		if grant.AppID != app.ID || grant.AppRevision != app.Revision || grant.Beta != app.Beta ||
			grant.BoundAt.IsZero() || now.Before(grant.BoundAt) || !now.Before(expiresAt) {
			return ErrNotFound
		}
		binding := ProductMerchantBinding{
			Platform: PlatformAmazon, Site: "amazon-us", OrganizationID: organizationID, StoreID: storeID,
			StoreVersion: store.Version, ConnectionRevision: grant.Generation, AmazonBeta: grant.Beta,
			ApplicationID: grant.AppID, ApplicationRevision: grant.AppRevision, CredentialExpiresAt: expiresAt.UTC(),
		}
		if !binding.ValidAmazonApplication() || grant.SellerID == "" || grant.Ciphertext == "" || grant.KeyID == "" || uuid.Validate(grant.AttemptID) != nil {
			return ErrOfficialConnectionUnavailable
		}
		connection = AmazonProductConnection{Binding: binding, SellerID: grant.SellerID, KeyID: grant.KeyID, Ciphertext: grant.Ciphertext, attemptID: grant.AttemptID}
		return nil
	})
	if err != nil {
		return AmazonProductConnection{}, err
	}
	return connection, nil
}
