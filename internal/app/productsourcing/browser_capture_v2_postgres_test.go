package productsourcing

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/authz"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
)

func browserApplicationV2Fixture(t *testing.T) []byte {
	body, _, _ := browserApplicationFixture(t)
	var p map[string]any
	require.NoError(t, json.Unmarshal(body, &p))
	p["captureVersion"] = 2
	e := p["evidence"].(map[string]any)
	e["schemaVersion"], e["parserVersion"] = 2, "1688-browser-dom/v3"
	blue := e["variants"].([]any)[0].(map[string]any)
	blue["images"] = []any{map[string]any{"url": "https://cbu01.alicdn.com/blue.jpg", "role": "source"}}
	red := map[string]any{"sourceID": "red", "sku": nil, "title": nil, "attributes": []any{map[string]any{"name": "颜色", "value": "红色"}}, "price": nil, "images": []any{map[string]any{"url": "https://cbu01.alicdn.com/red.jpg", "role": "source"}, map[string]any{"url": "https://cbu01.alicdn.com/blue.jpg", "role": "source"}}}
	e["variants"] = []any{blue, red}
	raw, err := json.Marshal(p)
	require.NoError(t, err)
	return raw
}

func TestBrowserCaptureV2PostgresExactImagesRecoveryAndCrossVersionConflict(t *testing.T) {
	db := acquisitionDatabase(t)
	require.NoError(t, InstallAcquisitionSchema(db))
	require.NoError(t, collectionstore.InstallSchema(db))
	permissions, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	live := liveRolesFunc(func(context.Context, string, string) ([]string, error) { return []string{"listingkit_admin"}, nil })
	service, err := NewBrowserAcquisition(context.Background(), db, live, permissions, WithCollectionPublication())
	require.NoError(t, err)
	ctx := acquisitionIdentity("browser-org", "browser-actor")
	key := uuid.NewString()
	body := browserApplicationV2Fixture(t)
	first, err := service.Capture(ctx, key, body)
	require.NoError(t, err)
	require.Equal(t, "v2", first.Publication.Receipt.Producer.Version)
	require.Equal(t, "https://cbu01.alicdn.com/blue.jpg", first.Publication.Snapshot.Variants[0].Images[0].URL)
	require.Equal(t, "https://cbu01.alicdn.com/red.jpg", first.Publication.Snapshot.Variants[1].Images[0].URL)
	old, _, _ := browserApplicationFixture(t)
	_, err = service.Capture(ctx, key, old)
	require.ErrorIs(t, err, sourcing.ErrAcquisitionConflict)
	_, err = service.Verify(ctx, key, old)
	require.ErrorIs(t, err, sourcing.ErrAcquisitionConflict)
	changed := []byte(strings.Replace(string(body), "red.jpg", "changed.jpg", 1))
	_, err = service.Verify(ctx, key, changed)
	require.ErrorIs(t, err, sourcing.ErrAcquisitionConflict)
	second, err := service.Capture(ctx, uuid.NewString(), changed)
	require.NoError(t, err)
	require.EqualValues(t, 2, second.Publication.Receipt.CatalogVersion)
	restarted, err := NewBrowserAcquisition(context.Background(), db, live, permissions, WithCollectionPublication())
	require.NoError(t, err)
	for _, read := range []func() (sourcing.AcquisitionResult, error){
		func() (sourcing.AcquisitionResult, error) { return restarted.Verify(ctx, key, body) },
		func() (sourcing.AcquisitionResult, error) { return restarted.ByKey(ctx, key) },
		func() (sourcing.AcquisitionResult, error) { return restarted.Read(ctx, first.Operation.ID) },
	} {
		result, err := read()
		require.NoError(t, err)
		require.Equal(t, first.Publication, result.Publication)
	}
	snapshots, err := catalogstore.NewBoundedSnapshotReader(db, sourcing.MaxEncodedSnapshotBytes)
	require.NoError(t, err)
	exact, err := snapshots.GetSnapshot(ctx, catalog.SnapshotIdentity{TenantID: "browser-org", ProductKey: first.Publication.Receipt.ProductKey}, first.Publication.Receipt.CatalogVersion)
	require.NoError(t, err)
	require.Equal(t, "https://cbu01.alicdn.com/red.jpg", exact.Snapshot.Variants[1].Images[0].URL)
	authority, err := collection.NewContextAuthorizer(live, permissions)
	require.NoError(t, err)
	repo, err := collectionstore.NewRepository(ctx, db, func(tx *gorm.DB) (collectionstore.OwnPublisher, error) { return NewOwnProductWriter(tx, authority) })
	require.NoError(t, err)
	sources, err := NewPublishedAcquisitionReader(ctx, db, live, permissions)
	require.NoError(t, err)
	collections, err := collection.NewService(repo, authority, sources)
	require.NoError(t, err)
	collections.WithSnapshots(snapshots)
	var ref struct{ ID string }
	require.NoError(t, db.Table("product_collection_items").Select("id").Where("publication_id=?", first.Publication.Receipt.PublicationID).Scan(&ref).Error)
	detail, err := collections.ReadItem(ctx, ref.ID)
	require.NoError(t, err)
	require.Equal(t, "https://cbu01.alicdn.com/red.jpg", detail.Product.Variants[1].Images[0].URL)
	var charges int64
	require.NoError(t, db.Table("product_acquisition_charge_intents").Count(&charges).Error)
	require.Zero(t, charges)
}
