package dataservicesapp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	acquisitionstore "task-processor/internal/integration/persistence/product/acquisition"
	"task-processor/internal/product/collection"
)

func TestResearchFixturePublishesThreeWithoutAcquisitionOrChargeRows(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db := publicationTestDB(t, ctx)
	require.NoError(t, acquisitionstore.InstallSchema(db))
	scope := collection.Scope{OrganizationID: "fixture-org", ActorID: "fixture-actor", MemberID: "fixture-member"}
	project := uuid.NewString()
	allowed := true
	fixture, err := NewResearchFixture(ctx, db, scope, project, func(_ context.Context, _ collection.Scope, _ string, _ bool) error {
		if !allowed {
			return collection.ErrForbidden
		}
		return nil
	})
	require.NoError(t, err)
	_, err = fixture.Import(ctx, scope, uuid.NewString())
	require.ErrorIs(t, err, collection.ErrForbidden)
	_, err = fixture.Read(ctx, scope, project)
	require.ErrorIs(t, err, collection.ErrNotFound)
	published := 0
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register("fail-second-fixture-publication", func(tx *gorm.DB) {
		if tx.Statement.Table != "product_source_publications" {
			return
		}
		published++
		if published == 2 {
			tx.AddError(errors.New("isolated second publication failure"))
		}
	}))
	_, err = fixture.Import(ctx, scope, project)
	require.Error(t, err)
	require.NoError(t, db.Callback().Create().Remove("fail-second-fixture-publication"))
	_, err = fixture.Read(ctx, scope, project)
	require.ErrorIs(t, err, collection.ErrNotFound)
	batch, err := fixture.Import(ctx, scope, project)
	require.NoError(t, err)
	require.Len(t, batch.Amazon, 2)
	require.Equal(t, "under sink organizer", batch.Keyword)
	require.Equal(t, batch.Amazon[0].Item.BatchID, batch.Amazon[1].Item.BatchID)
	require.Equal(t, "B000123456", batch.Amazon[0].Original.ID)
	require.Equal(t, "3", batch.Amazon[0].Original.ReferenceVersion)
	require.Equal(t, "sku-adjustable-white-01", batch.Supplier.Original.SKU)
	require.Equal(t, "40–55 cm", batch.Supplier.Original.Width)
	_, err = fixture.Read(ctx, scope, uuid.NewString())
	require.ErrorIs(t, err, collection.ErrForbidden)
	replay, err := fixture.Import(ctx, scope, project)
	require.NoError(t, err)
	require.Equal(t, batch, replay)

	for table, expected := range map[string]int64{
		"product_source_publications": 3, "product_snapshot_versions": 3, "product_collection_items": 3,
		"data_acquisition_jobs": 0, "data_acquisition_items": 0, "product_acquisition_charge_intents": 0,
	} {
		var n int64
		require.NoError(t, db.Table(table).Count(&n).Error)
		require.Equal(t, expected, n, table)
	}
	allowed = false
	_, err = fixture.Read(ctx, scope, project)
	require.ErrorIs(t, err, collection.ErrForbidden)
}
