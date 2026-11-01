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

func TestTemuNativeProductMaterialUsesCurrentStoreAuthorityAndExplicitPurpose(t *testing.T) {
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
		require.NoError(t, err)
	}
	subject.Purpose = "arbitrary_mutation"
	_, err = repo.ReadProductExecution(ctx, subject, storeID, authorization, now)
	require.ErrorIs(t, err, ErrNotFound)
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

func TestProductExecutionPreservesRetiredSourceAccessSafety(t *testing.T) {
	for _, tc := range []struct {
		name      string
		change    func(*testing.T, *sourceAdmissionSafetyFixture)
		cancelled bool
		want      error
		wantCalls int
	}{
		{name: "active scoped success", wantCalls: 1},
		{name: "foreign organization", change: func(_ *testing.T, f *sourceAdmissionSafetyFixture) {
			f.subject.OrganizationID = "org-foreign"
		}, want: ErrNotFound, wantCalls: 1},
		{name: "nonmatching or missing StoreID", change: func(_ *testing.T, f *sourceAdmissionSafetyFixture) {
			f.storeID = uuid.NewString()
		}, want: ErrNotFound, wantCalls: 1},
		{name: "unsupported platform", change: func(t *testing.T, f *sourceAdmissionSafetyFixture) {
			require.NoError(t, f.db.Model(&workbenchStoreRecord{}).Where("organization_id = ? AND id = ?", "org-a", f.storeID).Update("platform", "amazon").Error)
		}, want: ErrNotFound, wantCalls: 1},
		{name: "disabled", change: func(t *testing.T, f *sourceAdmissionSafetyFixture) {
			require.NoError(t, f.db.Model(&workbenchStoreRecord{}).Where("organization_id = ? AND id = ?", "org-a", f.storeID).Update("record_status", "disabled").Error)
		}, want: ErrNotFound, wantCalls: 1},
		{name: "provisioning", change: func(t *testing.T, f *sourceAdmissionSafetyFixture) {
			require.NoError(t, f.db.Model(&workbenchStoreRecord{}).Where("organization_id = ? AND id = ?", "org-a", f.storeID).Update("record_status", "provisioning").Error)
		}, want: ErrNotFound, wantCalls: 1},
		{name: "deleting", change: func(t *testing.T, f *sourceAdmissionSafetyFixture) {
			require.NoError(t, f.db.Model(&workbenchStoreRecord{}).Where("organization_id = ? AND id = ?", "org-a", f.storeID).Update("record_status", "deleting").Error)
		}, want: ErrNotFound, wantCalls: 1},
		{name: "empty organization", change: func(_ *testing.T, f *sourceAdmissionSafetyFixture) {
			f.subject.OrganizationID = ""
		}, want: ErrNotFound},
		{name: "nil repository receiver", change: func(_ *testing.T, f *sourceAdmissionSafetyFixture) {
			f.repo = nil
		}, want: ErrNotFound},
		{name: "cancelled context", cancelled: true, want: ErrDependencyUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newSourceAdmissionSafetyFixture(t)
			if tc.change != nil {
				tc.change(t, &f)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if tc.cancelled {
				cancel()
			}
			// Exercise the current native reader, not a replacement source gate.
			material, err := f.repo.ReadProductExecution(ctx, f.subject, f.storeID, f.authorization, f.now)
			require.Equal(t, tc.wantCalls, f.authorization.calls)
			if tc.wantCalls > 0 {
				require.Equal(t, f.subject, f.authorization.lastSubject)
			}
			if tc.want != nil {
				require.ErrorIs(t, err, tc.want)
				require.Zero(t, material, "denied or unavailable access must not release Store material")
				return
			}
			require.NoError(t, err)
			require.Equal(t, PlatformShein, material.Platform)
			require.EqualValues(t, 3, material.StoreVersion)
			require.Equal(t, f.subject.OrganizationID, material.Attempt.OrganizationID)
			require.Equal(t, f.storeID, material.Attempt.StoreID)
		})
	}
}

func TestProductExecutionRechecksStoreStateOnNextRead(t *testing.T) {
	f := newSourceAdmissionSafetyFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	material, err := f.repo.ReadProductExecution(ctx, f.subject, f.storeID, f.authorization, f.now)
	require.NoError(t, err)
	require.Equal(t, f.storeID, material.Attempt.StoreID)
	require.Equal(t, 1, f.authorization.calls)
	require.NoError(t, f.db.Model(&workbenchStoreRecord{}).Where("organization_id = ? AND id = ?", f.subject.OrganizationID, f.storeID).Update("record_status", "disabled").Error)
	material, err = f.repo.ReadProductExecution(ctx, f.subject, f.storeID, f.authorization, f.now)
	require.ErrorIs(t, err, ErrNotFound)
	require.Zero(t, material)
	require.Equal(t, 2, f.authorization.calls, "each native read rechecks current original authority")
	require.Equal(t, f.subject, f.authorization.lastSubject)
}

type sourceAdmissionSafetyAuthorization struct {
	value       ProductExecutionAuthorization
	calls       int
	lastSubject ProductExecutionSubject
}

func (a *sourceAdmissionSafetyAuthorization) AuthorizeProductExecution(_ context.Context, subject ProductExecutionSubject) (ProductExecutionAuthorization, error) {
	a.calls++
	a.lastSubject = subject
	return a.value, nil
}

type sourceAdmissionSafetyFixture struct {
	db            *gorm.DB
	repo          *MemberScopedStoreRepository
	authorization *sourceAdmissionSafetyAuthorization
	subject       ProductExecutionSubject
	storeID       string
	now           time.Time
}

// The fixture creates only current native facts in a fresh in-memory database.
// Eligibility and failure classification remain entirely in the native reader.
func newSourceAdmissionSafetyFixture(t *testing.T) sourceAdmissionSafetyFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, db.AutoMigrate(&workbenchStoreRecord{}, &storeMemberGrantRow{}, &officialConnectionRow{}, &officialAttemptRow{}))
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	start, end := now.Add(-time.Hour), now.Add(time.Hour)
	storeID, attemptID := uuid.NewString(), uuid.NewString()
	active := "active"
	require.NoError(t, db.Create(&workbenchStoreRecord{ID: storeID, OrganizationID: "org-a", Platform: "shein", RecordStatus: "active", ServiceStatus: &active, ServiceStartedAt: &start, ServiceExpiresAt: &end, ConnectionRef: attemptID, Version: 3}).Error)
	require.NoError(t, db.Create(&storeMemberGrantRow{OrganizationID: "org-a", StoreID: storeID, MemberID: "member-a", Active: true, Version: 1, UpdatedBy: "actor-a", UpdatedAt: now}).Error)
	require.NoError(t, db.Create(&officialConnectionRow{OrganizationID: "org-a", StoreID: storeID, AttemptID: attemptID, Version: 2, Status: "connected"}).Error)
	require.NoError(t, db.Create(&officialAttemptRow{OrganizationID: "org-a", StoreID: storeID, AttemptID: attemptID, ActorID: "actor-a", MemberID: "member-a", AppID: "app-a", AppVersion: "v1", ConnectionVersion: 2, State: "verified", KeyID: "key-a", Ciphertext: "sealed"}).Error)
	subject := ProductExecutionSubject{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a", Purpose: ProductPurposePublish}
	authorization := &sourceAdmissionSafetyAuthorization{value: ProductExecutionAuthorization{Access: StoreMemberAccess{OrganizationID: subject.OrganizationID, ActorID: subject.ActorID, MemberID: subject.MemberID}, Allowed: true}}
	return sourceAdmissionSafetyFixture{db: db, repo: &MemberScopedStoreRepository{db: db}, authorization: authorization, subject: subject, storeID: storeID, now: now}
}
