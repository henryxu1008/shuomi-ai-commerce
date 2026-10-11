package storecenter

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"testing"
	"time"
)

type productExecutionAccess struct {
	value ProductExecutionAuthorization
	err   error
}

func TestTemuNativeProductMaterialUsesCurrentStoreAuthorityAndOnlyRulePurpose(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, db.AutoMigrate(&workbenchStoreRecord{}, &storeMemberGrantRow{}, &officialConnectionRow{}, &officialAttemptRow{}))
	now := time.Now().UTC()
	start, end := now.Add(-time.Hour), now.Add(time.Hour)
	storeID, attemptID := uuid.NewString(), uuid.NewString()
	active := "active"
	facts := OfficialVerificationFacts{Platform: PlatformTemu, IdentityPartition: PartitionCN, Namespace: MerchantSupplierNamespace, MerchantID: "pa-mall-1", Mode: MerchantSemiManaged, ExpiresAt: now.Add(time.Minute), Scopes: []string{"bg.goods.cats.get"}, ContractRevision: "temu-identity-E08-E10-E11-E26-v1", VerifiedAt: now}
	raw, err := json.Marshal(facts)
	require.NoError(t, err)
	require.NoError(t, db.Create(&workbenchStoreRecord{ID: storeID, OrganizationID: "org-a", Platform: "temu", RecordStatus: "active", ServiceStatus: &active, ServiceStartedAt: &start, ServiceExpiresAt: &end, ConnectionRef: attemptID, Version: 3}).Error)
	require.NoError(t, db.Create(&storeMemberGrantRow{OrganizationID: "org-a", StoreID: storeID, MemberID: "member-a", Active: true, Version: 1, UpdatedBy: "actor-a", UpdatedAt: now}).Error)
	require.NoError(t, db.Create(&officialConnectionRow{OrganizationID: "org-a", StoreID: storeID, AttemptID: attemptID, Version: 2, Status: "connected"}).Error)
	require.NoError(t, db.Create(&officialAttemptRow{OrganizationID: "org-a", StoreID: storeID, AttemptID: attemptID, ActorID: "actor-a", MemberID: "member-a", AppID: "app-a", AppVersion: "v1:temu:CN:manual_token:6:r1", Platform: "temu", IdentityPartition: "CN", AuthorizationMethod: "manual_token", ConnectionVersion: 2, State: "verified", KeyID: "key-a", Ciphertext: "sealed", VerificationJSON: string(raw)}).Error)
	repo := &MemberScopedStoreRepository{db: db}
	authorization := &productExecutionAccess{value: ProductExecutionAuthorization{Access: StoreMemberAccess{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a"}, Allowed: true}}
	subject := ProductExecutionSubject{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a", Purpose: ProductPurposeRules}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	material, err := repo.ReadProductExecution(ctx, subject, storeID, authorization, now)
	require.NoError(t, err)
	require.Equal(t, PlatformTemu, material.Platform)
	require.Len(t, material.Connection.MerchantIdentityHash, 64)
	for _, purpose := range []string{ProductPurposePublish, ProductPurposeImage} {
		subject.Purpose = purpose
		_, err = repo.ReadProductExecution(ctx, subject, storeID, authorization, now)
		require.ErrorIs(t, err, ErrNotFound, "a metadata capability cannot grant an unimplemented mutation")
	}
	subject.Purpose = ProductPurposeRules
	_, err = repo.ReadProductExecution(ctx, subject, storeID, authorization, now.Add(time.Minute))
	require.ErrorIs(t, err, ErrNotFound)
	authorization.value.Access.MemberID = "rejoined-member"
	_, err = repo.ReadProductExecution(ctx, subject, storeID, authorization, now)
	require.ErrorIs(t, err, ErrNotFound)
	authorization.value.Access.MemberID = "member-a"
	require.NoError(t, db.Model(&storeMemberGrantRow{}).Where("organization_id = ? AND store_id = ?", "org-a", storeID).Update("active", false).Error)
	_, err = repo.ReadProductExecution(ctx, subject, storeID, authorization, now)
	require.ErrorIs(t, err, ErrNotFound)
}

func (a *productExecutionAccess) AuthorizeProductExecution(context.Context, ProductExecutionSubject) (ProductExecutionAuthorization, error) {
	return a.value, a.err
}
func TestProductExecutionRequiresCurrentOriginalMemberGrantServiceAndConnection(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, db.AutoMigrate(&workbenchStoreRecord{}, &storeMemberGrantRow{}, &officialConnectionRow{}, &officialAttemptRow{}))
	now := time.Now().UTC()
	start, end := now.Add(-time.Hour), now.Add(time.Hour)
	id, attemptID := uuid.NewString(), uuid.NewString()
	active := "active"
	require.NoError(t, db.Create(&workbenchStoreRecord{ID: id, OrganizationID: "org-a", Platform: "shein", RecordStatus: "active", ServiceStatus: &active, ServiceStartedAt: &start, ServiceExpiresAt: &end, ConnectionRef: attemptID, Version: 3}).Error)
	require.NoError(t, db.Create(&storeMemberGrantRow{OrganizationID: "org-a", StoreID: id, MemberID: "member-a", Active: true, Version: 1, UpdatedBy: "actor-a", UpdatedAt: now}).Error)
	require.NoError(t, db.Create(&officialConnectionRow{OrganizationID: "org-a", StoreID: id, AttemptID: attemptID, Version: 2, Status: "connected"}).Error)
	require.NoError(t, db.Create(&officialAttemptRow{OrganizationID: "org-a", StoreID: id, AttemptID: attemptID, ActorID: "actor-a", MemberID: "member-a", AppID: "app-a", AppVersion: "v1", ConnectionVersion: 2, State: "verified", KeyID: "key-a", Ciphertext: "sealed"}).Error)
	repo := &MemberScopedStoreRepository{db: db}
	authorization := &productExecutionAccess{value: ProductExecutionAuthorization{Access: StoreMemberAccess{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a"}, Allowed: true}}
	subject := ProductExecutionSubject{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a", Purpose: ProductPurposePublish}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	material, err := repo.ReadProductExecution(ctx, subject, id, authorization, now)
	require.NoError(t, err)
	require.EqualValues(t, 3, material.StoreVersion)
	require.Equal(t, "sealed", material.Attempt.Ciphertext)
	authorization.err = ErrDependencyUnavailable
	_, err = repo.ReadProductExecution(ctx, subject, id, authorization, now)
	require.ErrorIs(t, err, ErrDependencyUnavailable)
	authorization.err = nil
	authorization.value.Allowed = false
	_, err = repo.ReadProductExecution(ctx, subject, id, authorization, now)
	require.ErrorIs(t, err, ErrNotFound)
	authorization.value.Allowed = true
	authorization.value.Access.MemberID = "rejoined-member"
	_, err = repo.ReadProductExecution(ctx, subject, id, authorization, now)
	require.ErrorIs(t, err, ErrNotFound)
	authorization.value.Access.MemberID = "member-a"
	require.NoError(t, db.Model(&storeMemberGrantRow{}).Where("organization_id = ? AND store_id = ?", "org-a", id).Update("active", false).Error)
	_, err = repo.ReadProductExecution(ctx, subject, id, authorization, now)
	require.ErrorIs(t, err, ErrNotFound)
	authorization.value.Access.Administrator = true
	_, err = repo.ReadProductExecution(ctx, subject, id, authorization, now)
	require.NoError(t, err, "current administrator follows existing Store grant policy")
	_, err = repo.ReadProductExecution(ctx, subject, id, authorization, end)
	require.ErrorIs(t, err, ErrNotFound)
	require.NoError(t, db.Model(&officialConnectionRow{}).Where("organization_id = ?", "org-a").Update("status", "expired").Error)
	_, err = repo.ReadProductExecution(ctx, subject, id, authorization, now)
	require.ErrorIs(t, err, ErrNotFound)
	require.NoError(t, db.Migrator().DropTable(&officialConnectionRow{}))
	_, err = repo.ReadProductExecution(ctx, subject, id, authorization, now)
	require.ErrorIs(t, err, ErrDependencyUnavailable, "database failure is not authoritative Store revocation")
}
