package productsourcing

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/authz"
	"task-processor/internal/imageagent"
	completionstore "task-processor/internal/integration/persistence/listing/completiondraft"
	preparationstore "task-processor/internal/integration/persistence/listing/preparation"
	acquisitionstore "task-processor/internal/integration/persistence/product/acquisition"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	"task-processor/internal/listing/completiondraft"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
)

func TestCollectionPublicationUsesOneActualSourceCatalogTransaction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("supply605"), tcpostgres.WithUsername("supply_owner"), tcpostgres.WithPassword("isolated-supply-test"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	require.NoError(t, db.Exec("CREATE ROLE source_acquisition_runtime LOGIN PASSWORD 'isolated-runtime-test'").Error)
	runtimeURL, err := url.Parse(dsn)
	require.NoError(t, err)
	manifest := acquisitionInitManifest{SchemaVersion: 1, Collections: true}
	manifest.Database.Host = "127.0.0.1"
	manifest.Database.Port, err = strconv.Atoi(runtimeURL.Port())
	require.NoError(t, err)
	manifest.Database.User = runtimeURL.User.Username()
	manifest.Database.Password, _ = runtimeURL.User.Password()
	manifest.Database.Database = "supply605"
	raw, err := json.Marshal(manifest)
	require.NoError(t, err)
	manifestPath := filepath.Join(t.TempDir(), "collection-init.json")
	require.NoError(t, os.WriteFile(manifestPath, raw, 0600))
	require.NoError(t, InitializeAcquisitionDatabase(ctx, manifestPath, "supply605"))
	var initialized int64
	require.NoError(t, db.Raw("SELECT count(*) FROM information_schema.tables WHERE table_schema='public'").Scan(&initialized).Error)
	require.EqualValues(t, 9, initialized)
	require.Error(t, InitializeAcquisitionDatabase(ctx, manifestPath, "supply605"), "a nonempty database cannot be silently repaired")
	runtimeURL.User = url.UserPassword(acquisitionstore.RuntimeRole, "isolated-runtime-test")
	runtimeDB, err := gorm.Open(postgres.Open(runtimeURL.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	runtimePool, err := runtimeDB.DB()
	require.NoError(t, err)
	runtimePool.SetMaxOpenConns(4)
	t.Cleanup(func() { require.NoError(t, runtimePool.Close()) })
	capability := acquisitionstore.RuntimeCapabilities{Collections: true}
	require.NoError(t, acquisitionstore.VerifyRuntimePermissions(ctx, runtimeDB, capability))
	require.NoError(t, db.Exec("REVOKE UPDATE ON product_collection_batches FROM source_acquisition_runtime").Error)
	require.Error(t, acquisitionstore.VerifyRuntimePermissions(ctx, runtimeDB, capability))
	require.NoError(t, acquisitionstore.GrantRuntimePermissions(ctx, db, capability))
	require.NoError(t, acquisitionstore.VerifyRuntimePermissions(ctx, runtimeDB, capability))
	require.Error(t, acquisitionstore.VerifyRuntimePermissions(ctx, runtimeDB), "disabled capability rejects collection grants")
	require.Error(t, acquisitionstore.VerifyRuntimePermissions(ctx, db, capability), "owner cannot serve")
	require.NoError(t, db.Exec("GRANT DELETE ON product_collection_items TO source_acquisition_runtime").Error)
	require.Error(t, acquisitionstore.VerifyRuntimePermissions(ctx, runtimeDB, capability), "extra destructive privilege is refused")
	require.NoError(t, db.Exec("REVOKE DELETE ON product_collection_items FROM source_acquisition_runtime").Error)
	require.NoError(t, acquisitionstore.VerifyRuntimePermissions(ctx, runtimeDB, capability))
	permissions, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	live := liveRolesFunc(func(context.Context, string, string) ([]string, error) { return []string{"listingkit_admin"}, nil })
	authority, err := collection.NewContextAuthorizer(live, permissions)
	require.NoError(t, err)
	repository, err := collectionstore.NewRepository(ctx, runtimeDB, func(tx *gorm.DB) (collectionstore.OwnPublisher, error) { return NewOwnProductWriter(tx, authority) })
	require.NoError(t, err)
	snapshots, err := catalogstore.NewBoundedSnapshotReader(runtimeDB, sourcing.MaxEncodedSnapshotBytes)
	require.NoError(t, err)
	captures, err := NewBrowserAcquisition(ctx, runtimeDB, live, permissions, WithCollectionPublication())
	require.NoError(t, err)
	sources, err := NewPublishedAcquisitionReader(ctx, runtimeDB, live, permissions)
	require.NoError(t, err)
	service, err := collection.NewService(repository, authority, sources)
	require.NoError(t, err)
	service.WithSnapshots(snapshots)
	actor := acquisitionIdentity("supply-org", "supply-actor")

	ownKey := uuid.NewString()
	ownInput := collection.OwnProduct{Title: "原始自有商品", Attributes: map[string]string{"材质": "棉", "适用季节": "夏季"}, Images: []string{"https://images.example.org/original.png"}, Variants: []collection.OwnVariant{{SourceID: "red-s", Title: "红色 / S", SKU: "RED-S", Attributes: map[string]string{"颜色": "红色", "尺码": "S"}, Currency: "CNY", Price: 0, Stock: 0, Images: []string{"https://images.example.org/red.png"}}, {SourceID: "blue-m", Title: "蓝色 / M", SKU: "BLUE-M", Attributes: map[string]string{"颜色": "蓝色", "尺码": "M"}, Currency: "CNY", Price: 12.5, Stock: 8, Images: []string{"https://images.example.org/blue.png"}}}}
	own, err := service.Mutate(actor, ownKey, collection.Mutation{Action: "create_product", Product: &ownInput})
	require.NoError(t, err)
	detail, err := service.ReadItem(actor, own.ItemID)
	require.NoError(t, err)
	require.Equal(t, "原始自有商品", detail.Product.Title)
	require.ElementsMatch(t, []catalog.Attribute{{Name: "材质", Value: "棉"}, {Name: "适用季节", Value: "夏季"}}, detail.Product.Attributes)
	require.Len(t, detail.Product.Variants, 2)
	require.NotNil(t, detail.Product.Variants[0].Price)
	require.Equal(t, "CNY", detail.Product.Variants[0].Price.Currency)
	require.Zero(t, detail.Product.Variants[0].Price.Amount)
	require.Zero(t, detail.Product.Variants[0].Stock)
	require.Equal(t, ownInput.Variants[0].Images[0], detail.Product.Variants[0].Images[0].URL)
	require.Equal(t, ownInput.Variants[1].Images[0], detail.Product.Variants[1].Images[0].URL)
	encodedDetail, err := json.Marshal(detail)
	require.NoError(t, err)
	require.Contains(t, string(encodedDetail), `"amount":0`)
	var evidenceCount, versionCount int64
	require.NoError(t, db.Table("product_source_publications").Count(&evidenceCount).Error)
	require.NoError(t, db.Table("product_snapshot_versions").Count(&versionCount).Error)
	require.EqualValues(t, 1, evidenceCount)
	require.Equal(t, evidenceCount, versionCount)
	replay, err := service.Mutate(actor, ownKey, collection.Mutation{Action: "create_product", Product: &ownInput})
	require.NoError(t, err)
	require.Equal(t, own.ItemID, replay.ItemID)
	changed := ownInput
	changed.Title = "changed payload"
	_, err = service.Mutate(actor, ownKey, collection.Mutation{Action: "create_product", Product: &changed})
	require.ErrorIs(t, err, collection.ErrConflict)

	body, _, _ := browserApplicationFixture(t)
	capture, err := captures.Capture(actor, uuid.NewString(), body)
	require.NoError(t, err)
	scope, err := authority.Authorize(actor, collection.PermissionRead)
	require.NoError(t, err)
	batches, err := repository.ListBatches(actor, scope, collection.Query{Limit: 100})
	require.NoError(t, err)
	require.EqualValues(t, 2, batches.Total)
	var references int64
	require.NoError(t, db.Table("product_collection_items").Where("publication_id=?", capture.Publication.Receipt.PublicationID).Count(&references).Error)
	require.EqualValues(t, 1, references)

	// Fail the new reference write. Publication and Catalog must roll back with it.
	require.NoError(t, db.Exec("CREATE FUNCTION reject_collection() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'fixture reference failure'; END $$").Error)
	require.NoError(t, db.Exec("CREATE TRIGGER reject_collection BEFORE INSERT ON product_collection_items FOR EACH ROW EXECUTE FUNCTION reject_collection()").Error)
	failingKey := uuid.NewString()
	failed, err := captures.Capture(actor, failingKey, body)
	require.Error(t, err)
	require.Nil(t, failed.Publication)
	op, err := captures.core.operations.ByKey(actor, sourcing.PublicationScope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}, failingKey)
	require.NoError(t, err)
	var after int64
	require.NoError(t, db.Table("product_source_publications").Count(&after).Error)
	require.EqualValues(t, 2, after)
	require.NoError(t, db.Exec("DROP TRIGGER reject_collection ON product_collection_items").Error)
	_, err = captures.Verify(actor, failingKey, body)
	require.ErrorIs(t, err, sourcing.ErrAcquisitionUnknown, "verification remains read-only after a rolled-back write")
	verified, err := captures.ByKey(actor, failingKey)
	require.NoError(t, err)
	require.Equal(t, op.ID, verified.Operation.ID)
	require.NotNil(t, verified.Publication)
	require.NoError(t, db.Table("product_collection_items").Where("publication_id=?", verified.Publication.Receipt.PublicationID).Count(&references).Error)
	require.EqualValues(t, 1, references)
	_, err = service.ReadItem(acquisitionIdentity(scope.OrganizationID, "other-actor"), own.ItemID)
	require.ErrorIs(t, err, collection.ErrNotFound)
	t.Run("generated artwork uses the actual Source Collection UoW and original receipt", func(t *testing.T) {
		actor := acquisitionIdentity("artwork-org", "artwork-actor")
		scope, err := authority.Authorize(actor, collection.PermissionManage)
		require.NoError(t, err)
		fixture := completedArtworkFixture(t, scope)
		modes := &artworkModeFixture{}
		receiver := &ArtworkReceiver{Images: fixture, Facts: fixture, Artifacts: fixture, URLs: artworkURLFixture{}, Configuration: modes, Collections: repository, Authorize: authority}
		original := fixture.p
		identity := imageagent.ExecutionIdentity{ScopeProtocol: original.Run.ScopeProtocol, TenantID: scope.OrganizationID, UserID: scope.ActorID, MemberID: scope.MemberID, RunID: original.Run.ID, BusinessTaskID: original.Run.BusinessTaskID}
		input := imageagent.PublishArtworkInput{Identity: identity, PlanRevision: 1, ResultDigest: original.ResultDigest, ActionID: original.PendingCommand.ActionID}
		adopted, err := receiver.PublishArtwork(actor, input)
		require.NoError(t, err)
		source, err := service.ReadItem(actor, adopted.ItemID)
		require.NoError(t, err)
		require.Equal(t, "own", source.Item.Source.Kind)
		require.Equal(t, original.Plan.Artwork.Theme, source.Product.Title)
		require.Len(t, source.Product.Images, 1)
		originGuard := ArtworkOriginGuard{Snapshots: snapshots, Images: fixture, Facts: fixture, Configuration: modes}
		require.NoError(t, originGuard.AuthorizeArtworkSource(actor, scope, adopted.ProductKey, adopted.PublicationID, adopted.SourceVersion))
		foreign := scope
		foreign.MemberID = "foreign-member"
		require.Error(t, originGuard.AuthorizeArtworkSource(actor, foreign, adopted.ProductKey, adopted.PublicationID, adopted.SourceVersion))
		calls := modes.calls
		modes.denied = true
		replay, err := receiver.PublishArtwork(actor, input)
		require.NoError(t, err)
		require.Equal(t, adopted, replay)
		require.Equal(t, calls, modes.calls, "committed receipt precedes fresh new-write gate")
		require.Error(t, originGuard.AuthorizeArtworkSource(actor, scope, adopted.ProductKey, adopted.PublicationID, adopted.SourceVersion))
		calls = modes.calls
		fixture.p.PendingCommand = nil
		fixture.p.Run.Status = imageagent.RunStatusCompleted
		recovered, err := receiver.ReadArtworkReceipt(actor, identity)
		require.NoError(t, err)
		require.Equal(t, adopted, recovered)
		require.Equal(t, calls, modes.calls, "readback has no new write gate")
		identity.TenantID = "another-org"
		_, err = receiver.ReadArtworkReceipt(actor, identity)
		require.Error(t, err)
	})
	// Source writer never installs a schema or repairs omitted collection tables.
	t.Run("own edit appends after derived head and preserves source history", func(t *testing.T) {
		catalogRepo, err := catalogstore.NewRepository(db)
		require.NoError(t, err)
		catalogPublisher, err := catalog.NewPublisher(catalogRepo)
		require.NoError(t, err)
		derived := detail.Product
		derived.Title = "AI derived (not selected)"
		_, err = catalogPublisher.Publish(actor, catalog.PublishRequest{Identity: catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: detail.Item.Source.ProductKey}, PublicationID: uuid.NewString(), Snapshot: derived})
		require.NoError(t, err)
		mutation := collection.Mutation{Action: "revise_product", ItemID: own.ItemID, ExpectedRevision: detail.Item.Revision, ExpectedSourceVersion: detail.Item.Source.Version, ExpectedPublicationID: detail.Item.Source.PublicationID, Product: &changed}
		edited, err := service.Mutate(actor, uuid.NewString(), mutation)
		require.NoError(t, err)
		require.Equal(t, own.ItemID, edited.ItemID)
		current, err := service.ReadItem(actor, own.ItemID)
		require.NoError(t, err)
		require.Equal(t, changed.Title, current.Product.Title)
		require.Equal(t, detail.Item.Source.ProductKey, current.Item.Source.ProductKey)
		require.Equal(t, detail.Item.CreatedAt, current.Item.CreatedAt)
		require.EqualValues(t, 3, current.Item.Source.Version)
		historyReader, err := NewOwnHistoryReader(runtimeDB)
		require.NoError(t, err)
		service.WithOwnHistory(historyReader)
		history, err := service.ListOwnHistory(actor, own.ItemID, 0, 1)
		require.NoError(t, err)
		require.Len(t, history.Versions, 1)
		require.EqualValues(t, 3, history.Versions[0].Version)
		require.Equal(t, "3", history.NextBefore)
		older, err := service.ListOwnHistory(actor, own.ItemID, 3, 20)
		require.NoError(t, err)
		require.Len(t, older.Versions, 1)
		require.EqualValues(t, 1, older.Versions[0].Version)
		require.Equal(t, ownInput.Title, older.Versions[0].Title)
		require.False(t, older.Versions[0].SavedAt.IsZero())
		oldDetail, err := service.ReadOwnVersion(actor, own.ItemID, 1)
		require.NoError(t, err)
		require.Equal(t, detail.Product, oldDetail.Product)
		for _, version := range []uint64{2, 4} {
			_, err = service.ReadOwnVersion(actor, own.ItemID, version)
			require.ErrorIs(t, err, collection.ErrNotFound)
		}
		_, err = service.ListOwnHistory(acquisitionIdentity(scope.OrganizationID, "other-actor"), own.ItemID, 0, 20)
		require.ErrorIs(t, err, collection.ErrNotFound)
		for index, variant := range current.Product.Variants {
			previous := detail.Product.Variants[index]
			require.Equal(t, previous.SourceID, variant.SourceID)
			require.Equal(t, previous.Title, variant.Title)
			require.Equal(t, previous.SKU, variant.SKU)
			require.Equal(t, previous.Attributes, variant.Attributes)
			require.Equal(t, previous.Price, variant.Price)
			require.Equal(t, previous.Stock, variant.Stock)
			for imageIndex, image := range variant.Images {
				require.Equal(t, previous.Images[imageIndex].URL, image.URL)
			}
		}
		original, err := snapshots.GetSnapshot(actor, catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: detail.Item.Source.ProductKey}, detail.Item.Source.Version)
		require.NoError(t, err)
		require.Equal(t, ownInput.Title, original.Snapshot.Title)
		var count int64
		require.NoError(t, db.Table("product_collection_items").Where("organization_id=? AND actor_id=? AND product_key=?", scope.OrganizationID, scope.ActorID, detail.Item.Source.ProductKey).Count(&count).Error)
		require.EqualValues(t, 1, count)
		_, err = service.Mutate(actor, uuid.NewString(), mutation)
		require.ErrorIs(t, err, collection.ErrConflict)
		var evidenceBefore, versionsBefore int64
		require.NoError(t, db.Table("product_source_publications").Count(&evidenceBefore).Error)
		require.NoError(t, db.Table("product_snapshot_versions").Count(&versionsBefore).Error)
		require.NoError(t, db.Exec("CREATE TRIGGER reject_collection_revision BEFORE UPDATE ON product_collection_items FOR EACH ROW EXECUTE FUNCTION reject_collection()").Error)
		mutation.ExpectedRevision, mutation.ExpectedSourceVersion, mutation.ExpectedPublicationID = current.Item.Revision, current.Item.Source.Version, current.Item.Source.PublicationID
		_, err = service.Mutate(actor, uuid.NewString(), mutation)
		require.Error(t, err)
		require.NoError(t, db.Exec("DROP TRIGGER reject_collection_revision ON product_collection_items").Error)
		var evidenceAfter, versionsAfter int64
		require.NoError(t, db.Table("product_source_publications").Count(&evidenceAfter).Error)
		require.NoError(t, db.Table("product_snapshot_versions").Count(&versionsAfter).Error)
		require.Equal(t, evidenceBefore, evidenceAfter)
		require.Equal(t, versionsBefore, versionsAfter)
		afterFailure, err := service.ReadItem(actor, own.ItemID)
		require.NoError(t, err)
		require.Equal(t, current.Item.Source, afterFailure.Item.Source)
	})
	t.Run("single own transfer and completion keep the confirmed source after editing", func(t *testing.T) {
		// Install the existing opt-in owner only in this fresh disposable fixture.
		require.NoError(t, preparationstore.InstallSchema(db))
		require.NoError(t, completionstore.InstallSchema(db))
		require.NoError(t, acquisitionstore.GrantCompletionDraftPermissions(ctx, db))
		t.Cleanup(func() {
			require.NoError(t, db.Exec("DROP TABLE listing_completion_draft_commands,listing_completion_draft_heads,listing_completion_draft_records,listing_preparation_sources,listing_preparations").Error)
		})
		prepAuth, err := preparation.NewContextAuthorizer(live, permissions)
		require.NoError(t, err)
		prepRepo, err := preparationstore.NewRepository(ctx, runtimeDB)
		require.NoError(t, err)
		prepService, err := preparation.NewService(prepRepo, service, prepAuth)
		require.NoError(t, err)
		selector, err := preparation.NewSourceSelector(prepService, service, prepRepo, snapshots)
		require.NoError(t, err)
		draftRepo, err := completionstore.NewRepository(ctx, runtimeDB)
		require.NoError(t, err)
		drafts, err := completiondraft.NewService(selector, service, prepService, prepAuth, draftRepo)
		require.NoError(t, err)
		before, err := service.ReadItem(actor, own.ItemID)
		require.NoError(t, err)
		batch, err := service.ReadBatch(actor, before.Item.BatchID)
		require.NoError(t, err)
		key := uuid.NewString()
		input := preparation.TransferInput{BatchID: batch.ID, ExpectedRevision: batch.Revision, ItemIDs: []string{own.ItemID}}
		transfer, err := prepService.Transfer(actor, key, input)
		require.NoError(t, err)
		require.EqualValues(t, 1, transfer.Preparation.Count)
		selected, err := prepService.ListSources(actor, transfer.Preparation.ID, collection.Query{Limit: 2})
		require.NoError(t, err)
		require.Len(t, selected.Items, 1)
		fixed := selected.Items[0]
		require.Equal(t, before.Item.Source, fixed.Source)
		write, err := drafts.Save(actor, uuid.NewString(), completiondraft.SaveInput{SourceID: fixed.ID, Content: completiondraft.Content{Title: "独立候选草稿", Description: "候选描述"}})
		require.NoError(t, err)
		require.Equal(t, fixed, write.Record.Source)
		next := ownInput
		next.Title = "后来编辑的来源商品"
		_, err = service.Mutate(actor, uuid.NewString(), collection.Mutation{Action: "revise_product", ItemID: own.ItemID, ExpectedRevision: before.Item.Revision, ExpectedSourceVersion: before.Item.Source.Version, ExpectedPublicationID: before.Item.Source.PublicationID, Product: &next})
		require.NoError(t, err)
		_, err = prepService.Transfer(actor, uuid.NewString(), input)
		require.ErrorIs(t, err, collection.ErrConflict, "old confirmation cannot transfer newly edited content")
		replay, err := prepService.Transfer(actor, key, input)
		require.NoError(t, err)
		require.True(t, replay.Replayed)
		require.Equal(t, transfer.Preparation, replay.Preparation)
		read, err := drafts.Read(actor, fixed.ID)
		require.NoError(t, err)
		require.Equal(t, write.Record, read)
		proof, err := selector.Select(actor, fixed.ID)
		require.NoError(t, err)
		_, retained, snapshot, err := proof.Read(actor)
		require.NoError(t, err)
		require.Equal(t, before.Item.Source, retained.Source)
		require.Equal(t, before.Product, snapshot.Snapshot)
		require.Len(t, snapshot.Snapshot.Variants, 2)
		require.Zero(t, snapshot.Snapshot.Variants[0].Price.Amount)
		require.Zero(t, snapshot.Snapshot.Variants[0].Stock)
		_, err = drafts.Read(acquisitionIdentity(scope.OrganizationID, "other-actor"), fixed.ID)
		require.Error(t, err)
	})
	require.NoError(t, db.Exec("DROP TABLE product_collection_operations,product_collection_items,product_collection_batches").Error)
	_, err = captures.Capture(actor, uuid.NewString(), body)
	require.Error(t, err)
	require.False(t, errors.Is(err, catalog.ErrSnapshotNotReady))

}
