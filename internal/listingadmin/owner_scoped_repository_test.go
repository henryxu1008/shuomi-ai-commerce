package listingadmin

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

func newOwnerScopedSQLiteDB(t *testing.T, models ...any) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: ":memory:"}, &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate owner-scoped model: %v", err)
	}
	return db
}

func assertOwnerlessWriteRejected(t *testing.T, db *gorm.DB, table string, err error) {
	t.Helper()
	if !errors.Is(err, ErrOwnerUserIDRequired) {
		t.Fatalf("write error = %v, want ErrOwnerUserIDRequired", err)
	}
	var count int64
	if err := db.Table(table).Count(&count).Error; err != nil {
		t.Fatalf("count %s rows: %v", table, err)
	}
	if count != 0 {
		t.Fatalf("%s rows = %d, want 0", table, count)
	}
}

func TestCreateStoreRejectsOwnerlessWrite(t *testing.T) {
	db := newOwnerScopedSQLiteDB(t, &listingStore{})
	_, err := NewGormStoreRepository(db).CreateStore(context.Background(), &Store{
		TenantID: 101,
		StoreID:  "store-1",
		Name:     "Store",
	})
	assertOwnerlessWriteRejected(t, db, "listing_store", err)
}

func TestCreateStorePersistsTrustedInternalOwner(t *testing.T) {
	db := newOwnerScopedSQLiteDB(t, &listingStore{})
	created, err := NewGormStoreRepository(db).CreateStore(WithOwnerUserID(context.Background(), " internal-sub "), &Store{
		TenantID: 101,
		StoreID:  "store-1",
		Name:     "Store",
	})
	if err != nil {
		t.Fatalf("CreateStore() error = %v", err)
	}
	if created.OwnerUserID != "internal-sub" {
		t.Fatalf("created owner = %q, want internal-sub", created.OwnerUserID)
	}
}

func TestCreateGenerationTopicPolicyRejectsOwnerlessWrite(t *testing.T) {
	db := newOwnerScopedSQLiteDB(t, &listingGenerationTopicPolicy{})
	_, err := NewGormGenerationTopicPolicyRepository(db).CreateGenerationTopicPolicy(context.Background(), &GenerationTopicPolicy{
		TenantID: 101,
		Platform: "SHEIN",
		TopicKey: "title",
	})
	assertOwnerlessWriteRejected(t, db, "listing_generation_topic_policy", err)
}

func TestCreateGenerationTopicOverrideRejectsOwnerlessWrite(t *testing.T) {
	db := newOwnerScopedSQLiteDB(t, &listingGenerationTopicOverride{})
	_, err := NewGormGenerationTopicOverrideRepository(db).CreateGenerationTopicOverride(context.Background(), &GenerationTopicOverride{
		TenantID: 101,
		Platform: "SHEIN",
		TopicKey: "title",
	})
	assertOwnerlessWriteRejected(t, db, "listing_generation_topic_override", err)
}

func TestCreateSensitiveWordRejectsOwnerlessWrite(t *testing.T) {
	db := newOwnerScopedSQLiteDB(t, &listingSensitiveWord{})
	_, err := NewGormSensitiveWordRepository(db).CreateSensitiveWord(context.Background(), &SensitiveWord{
		TenantID: 101,
		Word:     "blocked",
		Language: "en",
	})
	assertOwnerlessWriteRejected(t, db, "listing_sensitive_word", err)
}
