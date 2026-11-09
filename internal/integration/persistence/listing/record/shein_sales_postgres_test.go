package recordpersistence_test

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"sync"
	"sync/atomic"
	"task-processor/internal/authidentity"
	preparationstore "task-processor/internal/integration/persistence/listing/preparation"
	recordstore "task-processor/internal/integration/persistence/listing/record"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	"task-processor/internal/listing/completiondraft"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/shein/goods"
	model "task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"testing"
	"time"
)

func TestPostgresSalesMappingPreservesSingleTargetAndCommandProtocol(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	pg, e := tcpostgres.Run(ctx, "postgres:17.2-alpine", tcpostgres.WithDatabase("sales_mapping"), tcpostgres.WithUsername("color_owner"), tcpostgres.WithPassword("isolated-color-fixture"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, e)
	t.Cleanup(func() { require.NoError(t, pg.Terminate(context.Background())) })
	dsn, e := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, e)
	db, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	pool, e := db.DB()
	require.NoError(t, e)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	for _, install := range []func(*gorm.DB) error{catalogstore.AutoMigrate, collectionstore.InstallSchema, preparationstore.InstallSchema, recordstore.InstallSchema} {
		require.NoError(t, install(db))
	}
	repo, e := recordstore.NewRepository(ctx, db)
	require.NoError(t, e)
	scope := collection.Scope{OrganizationID: "color-org", ActorID: "actor", MemberID: "member"}
	auth := fixedAuth{scope}
	collRepo, e := collectionstore.NewRepository(ctx, db, func(*gorm.DB) (collectionstore.OwnPublisher, error) { return nil, collection.ErrUnavailable })
	require.NoError(t, e)
	coll, e := collection.NewService(collRepo, auth, unusedSources{})
	require.NoError(t, e)
	prepRepo, e := preparationstore.NewRepository(ctx, db)
	require.NoError(t, e)
	prep, e := preparation.NewService(prepRepo, coll, auth)
	require.NoError(t, e)
	batch, item, productKey, pub := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	snapshot := catalog.ProductSnapshot{Title: "Explicit synthetic black white T-shirt", Variants: []catalog.Variant{{SourceID: "black-m", SKU: "BLACK-M", Stock: pointer(0), Attributes: []catalog.Attribute{{Name: "color", Value: "black"}, {Name: "size", Value: "M"}}}, {SourceID: "black-l", SKU: "BLACK-L", Attributes: []catalog.Attribute{{Name: "color", Value: "black"}, {Name: "size", Value: "L"}}}, {SourceID: "white-m", SKU: "WHITE-M", Attributes: []catalog.Attribute{{Name: "color", Value: "white"}, {Name: "size", Value: "M"}}}}}
	raw, e := json.Marshal(snapshot)
	require.NoError(t, e)
	require.NoError(t, db.Create(&catalogstore.SnapshotVersionRecord{TenantID: scope.OrganizationID, ProductKey: productKey, Version: 1, PublicationID: pub, PayloadHash: collection.Digest(snapshot), SnapshotJSON: raw}).Error)
	require.NoError(t, db.Exec("INSERT INTO product_collection_batches(organization_id,actor_id,member_id,id,name,kind,revision,created_at) VALUES(?,?,?,?,?,'manual',1,now())", scope.OrganizationID, scope.ActorID, scope.MemberID, batch, "Synthetic color split").Error)
	require.NoError(t, db.Exec("INSERT INTO product_collection_items(organization_id,actor_id,member_id,id,batch_id,product_key,publication_id,original_version,source_kind,source_operation_id,revision,created_at) VALUES(?,?,?,?,?,?,?,1,'own','',1,now())", scope.OrganizationID, scope.ActorID, scope.MemberID, item, batch, productKey, pub).Error)
	actor := authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(3 * time.Minute)})
	transfer, e := prep.Transfer(actor, uuid.NewString(), preparation.TransferInput{BatchID: batch, ExpectedRevision: 1})
	require.NoError(t, e)
	page, e := prep.ListSources(actor, transfer.Preparation.ID, collection.Query{Limit: 10})
	require.NoError(t, e)
	source := page.Items[0]
	reader, e := catalogstore.NewBoundedSnapshotReader(db, record.MaxPayloadBytes)
	require.NoError(t, e)
	selector, e := preparation.NewSourceSelector(prep, coll, prepRepo, reader)
	require.NoError(t, e)
	candidate := completiondraft.Record{ID: uuid.NewString(), Source: source, Revision: 1, Content: completiondraft.Content{Title: snapshot.Title, Structured: &completiondraft.StructuredContent{Attributes: []completiondraft.CandidateAttribute{}, Variants: []completiondraft.CandidateVariant{{SourceIndex: pointer(0), Stock: pointer(int64(0))}, {SourceIndex: pointer(1)}, {SourceIndex: pointer(2)}}}}}
	versions := importCandidateVersion{scope: scope, value: candidate}
	rules := &colorRules{expires: time.Now().Add(time.Hour)}
	service, e := record.NewTargetService(record.TargetDependencies{Sources: selector, Products: record.OriginalTargetProduct{}, Assets: readyAssets{}, Rules: rules, Records: repo, Images: recordTestImages{}, Authorizer: auth, CompletionVersions: versions})
	require.NoError(t, e)
	draft := goods.OfficialDraftInput{Product: model.PublishProduct{CategoryID: 123, ProductTypeID: 456, Names: []model.LanguageContent{{Language: "en", Name: snapshot.Title}}, SKCs: []model.ProductSKC{{SKUs: []model.ProductSKU{{SupplierSKU: "BLACK-M", Stock: []model.ProductStock{{Quantity: 0}}, Prices: []model.ProductPrice{{BasePrice: 12.5, Currency: "USD", SubSite: "shein-us"}}}, {SupplierSKU: "BLACK-L"}, {SupplierSKU: "WHITE-M"}}}}}, Images: []goods.OfficialImageSlot{{Group: "spu", AssetID: "main", Type: 1, Sort: 1}, {Group: "sku", SKU: 2, AssetID: "detail", Type: 1, Sort: 1}}}
	base, e := service.Create(actor, uuid.NewString(), record.TargetInput{SourceID: source.ID, StoreID: uuid.NewString(), EffectiveVersion: 1, Draft: draft})
	require.NoError(t, e)
	selection := record.SheinSalesSelection{Dimensions: []record.SheinSalesDimension{{SourceName: "color", AttributeID: 12, Main: true}, {SourceName: "size", AttributeID: 13}}, Values: []record.SheinSalesValue{{SourceName: "color", SourceValue: "black", OptionID: 34}, {SourceName: "color", SourceValue: "white", OptionID: 35}, {SourceName: "size", SourceValue: "M", OptionID: 36}, {SourceName: "size", SourceValue: "L", OptionID: 37}}, Rows: []record.SheinSalesRow{{SourceIndex: 0, SKCIndex: 0, SKUIndex: 0}, {SourceIndex: 1, SKCIndex: 0, SKUIndex: 1}, {SourceIndex: 2, SKCIndex: 0, SKUIndex: 2}}, Images: []record.SheinSalesImage{{SlotIndex: 0}, {SlotIndex: 1}}}
	preview, e := service.PreviewSheinSalesMapping(actor, record.SheinSalesRequest{SourceID: source.ID, BaseRecordID: base.Record.ID, CompletionDraft: record.CompletionDraftSelection{RecordID: candidate.ID, Revision: 1}, Selection: selection})
	require.NoError(t, e)
	require.Empty(t, preview.Issues)
	require.NotNil(t, preview.Input)
	beforeSource, beforeCandidate := collection.Digest(snapshot), collection.Digest(candidate)
	// A real SQL failure cannot leave a record, head or command behind.
	require.NoError(t, db.Exec(`CREATE FUNCTION reject_sales_mapping() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.body_json->'input'->'sheinSalesMapping' IS NOT NULL THEN RAISE EXCEPTION 'controlled mapping failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_sales_mapping BEFORE INSERT ON listing_target_records FOR EACH ROW EXECUTE FUNCTION reject_sales_mapping()`).Error)
	failKey := uuid.NewString()
	_, e = service.Create(actor, failKey, *preview.Input)
	require.Error(t, e)
	head, e := service.ReadHead(actor, base.Record.TargetID)
	require.NoError(t, e)
	require.Equal(t, base.Record, head)
	_, e = service.ReadCommand(actor, failKey)
	require.ErrorIs(t, e, record.ErrNotFound)
	require.NoError(t, db.Exec("DROP TRIGGER reject_sales_mapping ON listing_target_records; DROP FUNCTION reject_sales_mapping()").Error)
	var wg sync.WaitGroup
	var success atomic.Int64
	var winner string
	var mu sync.Mutex
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key := uuid.NewString()
			_, err := service.Create(actor, key, *preview.Input)
			if err == nil {
				success.Add(1)
				mu.Lock()
				winner = key
				mu.Unlock()
			} else {
				require.ErrorIs(t, err, record.ErrConflict)
			}
		}()
	}
	wg.Wait()
	require.EqualValues(t, 1, success.Load())
	saved, e := service.ReadCommand(actor, winner)
	require.NoError(t, e)
	require.Equal(t, base.Record.TargetID, saved.Record.TargetID)
	require.EqualValues(t, 2, saved.Record.Revision)
	require.Len(t, saved.Record.Input.Draft.Product.SKCs, 2)
	require.Len(t, saved.Record.Input.Draft.Product.SKCs[0].SKUs, 2)
	require.Len(t, saved.Record.Input.Draft.Product.SKCs[1].SKUs, 1)
	require.Zero(t, saved.Record.Input.Draft.Product.SKCs[0].SKUs[0].Stock[0].Quantity)
	require.Empty(t, saved.Record.Input.Draft.Product.SKCs[0].SKUs[1].Stock)
	require.Empty(t, saved.Record.Input.Draft.Product.SKCs[1].SKUs[0].Stock)
	require.Equal(t, 1, saved.Record.Input.Draft.Images[1].SKC)
	require.Zero(t, saved.Record.Input.Draft.Images[1].SKU)
	require.False(t, saved.Record.Result.ReadyForUpload)
	require.Empty(t, saved.Record.Result.SubmissionPayload)
	replay, e := service.Create(actor, winner, *preview.Input)
	require.NoError(t, e)
	require.True(t, replay.Replayed)
	require.Equal(t, saved.Record, replay.Record)
	changed := *preview.Input
	changed.ExpectedRevision++
	_, e = service.Create(actor, winner, changed)
	require.ErrorIs(t, e, record.ErrConflict)
	// Proof minted before waiting cannot be used after its original expiry.
	next, e := service.PreviewSheinSalesMapping(actor, record.SheinSalesRequest{SourceID: source.ID, BaseRecordID: saved.Record.ID, CompletionDraft: record.CompletionDraftSelection{RecordID: candidate.ID, Revision: 1}, Selection: record.SheinSalesSelection{Dimensions: selection.Dimensions, Values: selection.Values, Rows: []record.SheinSalesRow{{SourceIndex: 0, SKCIndex: 0, SKUIndex: 0}, {SourceIndex: 1, SKCIndex: 0, SKUIndex: 1}, {SourceIndex: 2, SKCIndex: 1, SKUIndex: 0}}, Images: selection.Images}})
	require.NoError(t, e)
	require.NotNil(t, next.Input)
	delayed := &salesDelayedRepository{TargetRepository: repo}
	delayed.delay = true
	delayedService, e := record.NewTargetService(record.TargetDependencies{Sources: selector, Products: record.OriginalTargetProduct{}, Assets: readyAssets{}, Rules: rules, Records: delayed, Images: recordTestImages{}, Authorizer: auth, CompletionVersions: versions})
	require.NoError(t, e)
	expiredKey := uuid.NewString()
	_, e = delayedService.Create(actor, expiredKey, *next.Input)
	require.Error(t, e)
	_, e = service.ReadCommand(actor, expiredKey)
	require.ErrorIs(t, e, record.ErrNotFound)
	// A lost acknowledgement is reconciled using the original key and full input.
	delayed.delay = false
	delayed.unknown = true
	unknownKey := uuid.NewString()
	_, e = delayedService.Create(actor, unknownKey, *next.Input)
	require.ErrorIs(t, e, record.ErrTargetUnknown)
	found, e := delayedService.ReadCommand(actor, unknownKey)
	require.NoError(t, e)
	replayed, e := delayedService.Create(actor, unknownKey, *next.Input)
	require.NoError(t, e)
	require.Equal(t, found.Record, replayed.Record)
	var count int64
	require.NoError(t, db.Table("listing_preparation_targets").Count(&count).Error)
	require.EqualValues(t, 1, count)
	require.NoError(t, db.Table("listing_target_split_commands").Count(&count).Error)
	require.Zero(t, count)
	original, e := reader.GetSnapshot(actor, catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: productKey}, 1)
	require.NoError(t, e)
	require.Equal(t, beforeSource, collection.Digest(original.Snapshot))
	require.Equal(t, beforeCandidate, collection.Digest(candidate))
}

type salesDelayedRepository struct {
	record.TargetRepository
	delay   bool
	unknown bool
}

func (r *salesDelayedRepository) SaveTarget(ctx context.Context, p record.TargetPrepared) (record.TargetReceipt, error) {
	if r.delay {
		time.Sleep(5200 * time.Millisecond)
	}
	saved, e := r.TargetRepository.SaveTarget(ctx, p)
	if e == nil && r.unknown {
		r.unknown = false
		return record.TargetReceipt{}, record.ErrTargetUnknown
	}
	return saved, e
}
