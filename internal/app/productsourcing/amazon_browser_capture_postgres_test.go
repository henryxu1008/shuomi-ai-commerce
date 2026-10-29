package productsourcing

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
	"strings"
	"sync"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	sourcestore "task-processor/internal/integration/persistence/product/sourcing"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
	"testing"
	"time"
)

type amazonTestAuthority struct{ scope collection.Scope }

type amazonDraftRetryAuthority func(context.Context, bool) (collection.Scope, error)

func (f amazonDraftRetryAuthority) Authorize(ctx context.Context, write bool) (collection.Scope, error) {
	return f(ctx, write)
}

func (a amazonTestAuthority) Authorize(context.Context, bool) (collection.Scope, error) {
	return a.scope, nil
}

// A ByKey not-found can precede a delayed original POST. A local draft retry
// must still publish one fixed version whichever request reaches SRC first.
func TestAmazonDraftRetryArrivalOrderAndRevocation(t *testing.T) {
	for _, first := range []string{"original", "retry"} {
		t.Run(first+" arrives first", func(t *testing.T) {
			db := acquisitionDatabase(t)
			require.NoError(t, InstallSchema(db))
			require.NoError(t, collectionstore.InstallSchema(db))
			scope := collection.Scope{OrganizationID: "draft-org", ActorID: "draft-actor", MemberID: "draft-member"}
			entered, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			second := "retry"
			if first == "retry" {
				second = "original"
			}
			type attemptKey struct{}
			auth := amazonDraftRetryAuthority(func(ctx context.Context, write bool) (collection.Scope, error) {
				if write && ctx.Value(attemptKey{}) == second {
					once.Do(func() { close(entered); <-release })
				}
				return scope, nil
			})
			service, err := NewAmazonBrowserCapture(db, auth)
			require.NoError(t, err)
			body, err := os.ReadFile("../../product/sourcing/testdata/amazon-capture-v2.json")
			require.NoError(t, err)
			key := uuid.NewString()
			type result struct {
				capture AmazonCaptureResult
				err     error
			}
			delayed := make(chan result, 1)
			go func() {
				r, e := service.Capture(context.WithValue(context.Background(), attemptKey{}, second), key, body)
				delayed <- result{r, e}
			}()
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("delayed request did not reach authority")
			}
			_, err = service.ByKey(context.Background(), key)
			require.ErrorIs(t, err, sourcing.ErrSourcePublicationNotFound)
			winner, err := service.Capture(context.WithValue(context.Background(), attemptKey{}, first), key, body)
			close(release)
			require.NoError(t, err)
			late := <-delayed
			require.NoError(t, late.err)
			require.True(t, late.capture.Replayed)
			require.Equal(t, winner.CaptureID, late.capture.CaptureID)
			require.Equal(t, winner.ItemID, late.capture.ItemID)
			require.EqualValues(t, 1, late.capture.CatalogVersion)
			for _, table := range []string{"product_source_publications", "product_source_publication_receipts", "product_snapshot_versions", "product_collection_batches", "product_collection_items"} {
				var count int64
				require.NoError(t, db.Table(table).Count(&count).Error)
				require.EqualValues(t, 1, count, table)
			}
			_, err = service.Capture(context.Background(), key, []byte(strings.Replace(string(body), "Cotton bag", "changed body", 1)))
			require.ErrorIs(t, err, sourcing.ErrSourcePublicationConflict)
			denied, _ := NewAmazonBrowserCapture(db, amazonDraftRetryAuthority(func(context.Context, bool) (collection.Scope, error) {
				return collection.Scope{}, sourcing.ErrPublicationForbidden
			}))
			_, err = denied.Capture(context.Background(), key, body)
			require.ErrorIs(t, err, sourcing.ErrPublicationForbidden)
		})
	}
}

func TestAmazonCaptureRollbackPartialStateAndUnknown(t *testing.T) {
	db := acquisitionDatabase(t)
	require.NoError(t, InstallSchema(db))
	require.NoError(t, collectionstore.InstallSchema(db))
	scope := collection.Scope{OrganizationID: "amazon-org", ActorID: "amazon-actor", MemberID: "amazon-member"}
	auth := amazonTestAuthority{scope}
	service, err := NewAmazonBrowserCapture(db, auth)
	require.NoError(t, err)
	body, err := os.ReadFile("../../product/sourcing/testdata/amazon-capture-v2.json")
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, db.Exec(`CREATE FUNCTION fail_amazon_item() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'fixture failure'; END; $$`).Error)
	require.NoError(t, db.Exec(`CREATE TRIGGER fail_amazon_item BEFORE INSERT ON product_collection_items FOR EACH ROW EXECUTE FUNCTION fail_amazon_item()`).Error)
	_, err = service.Capture(ctx, uuid.NewString(), body)
	require.Error(t, err)
	for _, table := range []string{"product_source_publications", "product_source_publication_receipts", "product_snapshot_versions", "product_collection_batches", "product_collection_items"} {
		var n int64
		require.NoError(t, db.Table(table).Count(&n).Error)
		require.Zero(t, n, table)
	}
	require.NoError(t, db.Exec(`DROP TRIGGER fail_amazon_item ON product_collection_items`).Error)
	// Reuse existing real-driver COMMIT-response-loss injection; no new runner.
	var name string
	require.NoError(t, db.Raw("SELECT current_database()").Scan(&name).Error)
	fault := &acquisitionAckDriver{target: "publication"}
	driverName := "amazon905_" + uuid.NewString()
	sql.Register(driverName, fault)
	pool, err := sql.Open(driverName, os.Getenv("ISSUE398_TEST_DSN")+" dbname="+name)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })
	faultDB, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	lossy, err := NewAmazonBrowserCapture(faultDB, auth)
	require.NoError(t, err)
	key := uuid.NewString()
	_, err = lossy.Capture(ctx, key, body)
	require.ErrorIs(t, err, sourcing.ErrSourcePublicationOutcomeUnknown)
	require.True(t, fault.lost.Load())
	recovered, err := service.ByKey(ctx, key)
	require.NoError(t, err)
	require.EqualValues(t, 1, recovered.CatalogVersion)
	// A held original lock cannot become an authoritative not-found.
	pendingKey := uuid.NewString()
	tx := db.Begin()
	require.NoError(t, sourcestore.LockPublication(ctx, tx, scope.OrganizationID, amazonCaptureID(scope, pendingKey)))
	short, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	_, err = service.ByKey(short, pendingKey)
	require.ErrorIs(t, err, sourcing.ErrSourcePublicationOutcomeUnknown)
	require.NoError(t, tx.Rollback().Error)
	_, err = service.ByKey(ctx, pendingKey)
	require.ErrorIs(t, err, sourcing.ErrSourcePublicationNotFound)
	// Catalog+Collection without SRC must never be repaired by replay.
	require.NoError(t, db.Exec("DELETE FROM product_source_publication_receipts WHERE publication_id=?", recovered.PublicationID).Error)
	_, err = service.Capture(ctx, key, body)
	require.ErrorIs(t, err, sourcing.ErrSourcePublicationStateInvalid)
}
func TestAmazonCaptureAtomicReplayAndExactHistory(t *testing.T) {
	db := acquisitionDatabase(t)
	require.NoError(t, InstallSchema(db))
	require.NoError(t, collectionstore.InstallSchema(db))
	scope := collection.Scope{OrganizationID: "amazon-org", ActorID: "amazon-actor", MemberID: "amazon-member"}
	service, err := NewAmazonBrowserCapture(db, amazonTestAuthority{scope})
	require.NoError(t, err)
	body, err := os.ReadFile("../../product/sourcing/testdata/amazon-capture-v2.json")
	require.NoError(t, err)
	key := uuid.NewString()
	ctx := context.Background()
	first, err := service.Capture(ctx, key, body)
	require.NoError(t, err)
	replay, err := service.Capture(ctx, key, body)
	require.NoError(t, err)
	require.Equal(t, first.CatalogVersion, replay.CatalogVersion)
	require.True(t, replay.Replayed)
	_, err = service.Capture(ctx, key, []byte(strings.Replace(string(body), "Cotton bag", "Different", 1)))
	require.ErrorIs(t, err, sourcing.ErrSourcePublicationConflict)
	second, err := service.Capture(ctx, uuid.NewString(), body)
	require.NoError(t, err)
	old, err := service.ByKey(ctx, key)
	require.NoError(t, err)
	require.Equal(t, first.CatalogVersion, old.CatalogVersion)
	require.NoError(t, db.Exec("UPDATE product_collection_items SET batch_id=? WHERE id=?", second.BatchID, first.ItemID).Error)
	moved, err := service.Capture(ctx, key, body)
	require.NoError(t, err)
	require.Equal(t, first.BatchID, moved.BatchID)
	require.Equal(t, second.BatchID, moved.CurrentBatchID)
	readonly, err := NewAmazonBrowserCapture(browserReadOnlyDatabase(t, db), amazonTestAuthority{scope})
	require.NoError(t, err)
	verified, err := readonly.Verify(ctx, key, body)
	require.NoError(t, err)
	require.Equal(t, first.CatalogVersion, verified.CatalogVersion)
	published, err := readonly.Published(ctx, first.CaptureID)
	require.NoError(t, err)
	require.Equal(t, first.CatalogVersion, published.CatalogVersion)
	require.NoError(t, db.Exec("UPDATE product_collection_items SET archived_at=now() WHERE id=?", first.ItemID).Error)
	archived, err := service.Capture(ctx, key, body)
	require.NoError(t, err)
	require.True(t, archived.Archived)
	other, _ := NewAmazonBrowserCapture(db, amazonTestAuthority{collection.Scope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: "new-member"}})
	_, err = other.ByKey(ctx, key)
	require.ErrorIs(t, err, sourcing.ErrPublicationForbidden)
	var count int64
	require.NoError(t, db.Table("product_source_publications").Count(&count).Error)
	require.EqualValues(t, 2, count)
	var wg sync.WaitGroup
	concurrent := uuid.NewString()
	for range 4 {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := service.Capture(ctx, concurrent, body); require.NoError(t, err) }()
	}
	wg.Wait()
	require.NoError(t, db.Table("product_source_publications").Count(&count).Error)
	require.EqualValues(t, 3, count)
}
