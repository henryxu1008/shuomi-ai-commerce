package listingadmin

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

func TestListingStoreToStoreUsesAuditFallbacks(t *testing.T) {
	t.Parallel()

	row := listingStore{
		ID:          1,
		TenantID:    101,
		OwnerUserID: "",
		CreatedBy:   "creator-id",
		Creator:     "creator-name",
		UpdatedBy:   "",
		Updater:     "updater-name",
		Name:        "Demo Store",
		Username:    "demo-user",
		Password:    "secret",
		Platform:    "SHEIN",
		ShopType:    "semi",
		Region:      "US",
	}

	store := row.toStore()
	if store.OwnerUserID != "creator-id" {
		t.Fatalf("ownerUserID = %q, want creator-id fallback", store.OwnerUserID)
	}
	if store.CreatedBy != "creator-id" {
		t.Fatalf("createdBy = %q, want creator-id fallback", store.CreatedBy)
	}
	if store.UpdatedBy != "updater-name" {
		t.Fatalf("updatedBy = %q, want updater-name fallback", store.UpdatedBy)
	}
}

func TestApplyStoreAccessScopeHonorsTenantAndOwnerWithoutDeletedFilter(t *testing.T) {
	t.Parallel()

	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: ":memory:"}, &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&listingStore{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	seedRows := []listingStore{
		{TenantID: 101, OwnerUserID: "user-a", Name: "active", Username: "active", Password: "secret", Platform: "SHEIN", ShopType: "semi", Deleted: 0},
		{TenantID: 101, OwnerUserID: "user-a", Name: "deleted", Username: "deleted", Password: "secret", Platform: "SHEIN", ShopType: "semi", Deleted: 1},
		{TenantID: 101, OwnerUserID: "user-b", Name: "other-owner", Username: "other-owner", Password: "secret", Platform: "SHEIN", ShopType: "semi", Deleted: 0},
	}
	for _, row := range seedRows {
		if err := db.Table("listing_store").Create(&row).Error; err != nil {
			t.Fatalf("seed row: %v", err)
		}
	}

	t.Cleanup(SetOwnerScopeRequiredForTesting(true))

	var rows []listingStore
	err = applyStoreAccessScope(db.Table("listing_store"), StoreQuery{TenantID: 101, OwnerUserID: "user-a"}).
		Where("deleted = 1").
		Find(&rows).Error
	if err != nil {
		t.Fatalf("query rows: %v", err)
	}
	if len(rows) != 1 || rows[0].Name != "deleted" {
		t.Fatalf("rows = %+v, want deleted row for same tenant/owner only", rows)
	}
}

func TestGormStoreRepositoryRejectsDeletedStoreReadsAndSyncWrites(t *testing.T) {
	t.Parallel()
	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: ":memory:"}, &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&listingStore{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	row := listingStore{TenantID: 101, OwnerUserID: "user-a", Name: "deleted", Username: "deleted", Password: "secret", Platform: "SHEIN", ShopType: "semi", StoreID: "original", Remark: "original", Deleted: 1}
	if err := db.Table("listing_store").Create(&row).Error; err != nil {
		t.Fatalf("seed row: %v", err)
	}
	t.Cleanup(SetOwnerScopeRequiredForTesting(true))
	ctx := withRequestIdentity(context.TODO(), "user-a", nil)
	repo := NewGormStoreRepository(db)
	if _, err := repo.GetStore(ctx, 101, row.ID); !errors.Is(err, ErrStoreNotFound) {
		t.Fatalf("GetStore deleted row error = %v, want ErrStoreNotFound", err)
	}
	if _, err := repo.UpdateStoreID(ctx, row.ID, "changed"); !errors.Is(err, ErrStoreNotFound) {
		t.Fatalf("UpdateStoreID deleted row error = %v, want ErrStoreNotFound", err)
	}
	if _, err := repo.UpdateStoreStatus(ctx, 101, row.ID, 1, "changed"); !errors.Is(err, ErrStoreNotFound) {
		t.Fatalf("UpdateStoreStatus deleted row error = %v, want ErrStoreNotFound", err)
	}
	var loaded listingStore
	if err := db.Table("listing_store").Where("id = ?", row.ID).Take(&loaded).Error; err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if loaded.StoreID != "original" || loaded.Status != 0 || loaded.Remark != "original" || loaded.Deleted != 1 {
		t.Fatalf("deleted fixture changed: %+v", loaded)
	}
}

func TestGormStoreRepositoryGetStorePlatformAdminBypassesOwnerScope(t *testing.T) {
	t.Parallel()

	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: ":memory:"}, &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&listingStore{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	row := listingStore{
		TenantID:    101,
		OwnerUserID: "user-b",
		Name:        "Owned by B",
		Username:    "owner-b",
		Password:    "secret",
		Platform:    "SHEIN",
		ShopType:    "semi",
		Deleted:     0,
	}
	if err := db.Table("listing_store").Create(&row).Error; err != nil {
		t.Fatalf("seed row: %v", err)
	}
	t.Cleanup(SetOwnerScopeRequiredForTesting(true))

	repo := NewGormStoreRepository(db)
	regularCtx := withRequestIdentity(context.Background(), "user-a", nil)
	if _, err := repo.GetStore(regularCtx, 101, row.ID); !errors.Is(err, ErrStoreNotFound) {
		t.Fatalf("regular user err = %v, want ErrStoreNotFound", err)
	}

	adminCtx := withRequestIdentity(context.Background(), "platform-admin", []string{"platform_admin"})
	store, err := repo.GetStore(adminCtx, 101, row.ID)
	if err != nil {
		t.Fatalf("platform admin GetStore error = %v", err)
	}
	if store.ID != row.ID || store.OwnerUserID != "user-b" {
		t.Fatalf("store = %+v, want owner user-b row", store)
	}
}

func TestGormStoreRepositoryGetStoreAllowsLegacyUnownedStoreRead(t *testing.T) {
	t.Parallel()

	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: ":memory:"}, &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&listingStore{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	legacyRow := listingStore{
		TenantID: 101,
		Name:     "Legacy Store",
		Username: "legacy",
		Password: "secret",
		Platform: "SHEIN",
		ShopType: "semi",
		Deleted:  0,
		Creator:  "179",
	}
	otherOwnerRow := listingStore{
		TenantID:    101,
		OwnerUserID: "user-b",
		Name:        "Owned by B",
		Username:    "owner-b",
		Password:    "secret",
		Platform:    "SHEIN",
		ShopType:    "semi",
		Deleted:     0,
	}
	for _, row := range []*listingStore{&legacyRow, &otherOwnerRow} {
		if err := db.Table("listing_store").Create(row).Error; err != nil {
			t.Fatalf("seed row: %v", err)
		}
	}
	t.Cleanup(SetOwnerScopeRequiredForTesting(true))

	repo := NewGormStoreRepository(db)
	userCtx := withRequestIdentity(context.Background(), "user-a", nil)
	store, err := repo.GetStore(userCtx, 101, legacyRow.ID)
	if err != nil {
		t.Fatalf("GetStore legacy unowned row error = %v", err)
	}
	if store.ID != legacyRow.ID || store.OwnerUserID != "179" {
		t.Fatalf("store = %+v, want legacy row with creator fallback", store)
	}
	if _, err := repo.GetStore(userCtx, 101, otherOwnerRow.ID); !errors.Is(err, ErrStoreNotFound) {
		t.Fatalf("other owner err = %v, want ErrStoreNotFound", err)
	}
}

func TestGormStoreRepositoryListStoresExcludesDeletedAndForeignScopes(t *testing.T) {
	t.Parallel()
	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: ":memory:"}, &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&listingStore{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, row := range []listingStore{
		{TenantID: 101, OwnerUserID: "user-a", Name: "deleted-a", Username: "deleted-a", Password: "secret", Platform: "SHEIN", ShopType: "semi", Deleted: 1},
		{TenantID: 101, OwnerUserID: "user-b", Name: "deleted-b", Username: "deleted-b", Password: "secret", Platform: "SHEIN", ShopType: "semi", Deleted: 1},
		{TenantID: 101, OwnerUserID: "user-a", Name: "active-a", Username: "active-a", Password: "secret", Platform: "SHEIN", ShopType: "semi", Deleted: 0},
		{TenantID: 202, OwnerUserID: "user-a", Name: "foreign-tenant", Username: "foreign-tenant", Password: "secret", Platform: "SHEIN", ShopType: "semi", Deleted: 0},
		{TenantID: 101, OwnerUserID: "user-b", Name: "foreign-owner", Username: "foreign-owner", Password: "secret", Platform: "SHEIN", ShopType: "semi", Deleted: 0},
	} {
		if err := db.Table("listing_store").Create(&row).Error; err != nil {
			t.Fatalf("seed row: %v", err)
		}
	}
	t.Cleanup(SetOwnerScopeRequiredForTesting(true))
	ctx := withRequestIdentity(context.TODO(), "user-a", nil)
	page, err := NewGormStoreRepository(db).ListStores(ctx, StoreQuery{TenantID: 101, OwnerUserID: "user-a"})
	if err != nil {
		t.Fatalf("ListStores: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Name != "active-a" {
		t.Fatalf("page = %+v, want only active row for same tenant/owner", page)
	}
}

func TestApplyStoreQuerySupportsSharedReadAccess(t *testing.T) {
	t.Parallel()

	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: ":memory:"}, &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&listingStore{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, row := range []listingStore{
		{TenantID: 101, OwnerUserID: "user-a", Name: "owned", Username: "owned", Password: "secret", Platform: "SHEIN", ShopType: "semi", Deleted: 0, Status: 0},
		{TenantID: 101, Name: "shared-null", Username: "shared-null", Password: "secret", Platform: "SHEIN", ShopType: "semi", Deleted: 0, Status: 0},
		{TenantID: 101, OwnerUserID: "", Name: "shared-empty", Username: "shared-empty", Password: "secret", Platform: "SHEIN", ShopType: "semi", Deleted: 0, Status: 0},
		{TenantID: 101, OwnerUserID: "user-b", Name: "other-owner", Username: "other-owner", Password: "secret", Platform: "SHEIN", ShopType: "semi", Deleted: 0, Status: 0},
	} {
		if err := db.Table("listing_store").Create(&row).Error; err != nil {
			t.Fatalf("seed row: %v", err)
		}
	}

	t.Cleanup(SetOwnerScopeRequiredForTesting(true))
	ctx := withRequestIdentity(context.Background(), "user-a", nil)
	page, err := NewGormStoreRepository(db).ListStores(ctx, StoreQuery{
		TenantID:   101,
		Platform:   "SHEIN",
		ReadAccess: true,
	})
	if err != nil {
		t.Fatalf("ListStores: %v", err)
	}
	if len(page.Items) != 3 || page.Total != 3 {
		t.Fatalf("page = %+v, want owned plus shared stores", page)
	}
	expected := map[string]bool{"owned": true, "shared-null": true, "shared-empty": true}
	for _, row := range page.Items {
		if !expected[row.Name] {
			t.Fatalf("unexpected shared-read row: %+v", row)
		}
		delete(expected, row.Name)
	}
	if len(expected) != 0 {
		t.Fatalf("missing shared-read rows: %v", expected)
	}
}

func TestApplyStoreQueryMatchesPlatformCaseInsensitively(t *testing.T) {
	t.Parallel()

	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: ":memory:"}, &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&listingStore{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	row := listingStore{
		TenantID: 101,
		Name:     "lowercase shein",
		Username: "demo",
		Password: "secret",
		Platform: "shein",
		ShopType: "semi",
		Deleted:  0,
		Status:   0,
	}
	if err := db.Table("listing_store").Create(&row).Error; err != nil {
		t.Fatalf("seed row: %v", err)
	}

	page, err := NewGormStoreRepository(db).ListStores(context.Background(), StoreQuery{
		TenantID:   101,
		Platform:   "SHEIN",
		ReadAccess: true,
	})
	if err != nil {
		t.Fatalf("ListStores: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Platform != "shein" {
		t.Fatalf("page = %+v, want lowercase shein store", page)
	}
}
