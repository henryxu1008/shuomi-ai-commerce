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
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type colorRules struct {
	expires time.Time
	delay   atomic.Bool
}

func (r *colorRules) ReadTargetRules(ctx context.Context, s collection.Scope, store string, in goods.OfficialDraftInput) (storecenter.ProductMerchantBinding, goods.OfficialRuleSnapshot, error) {
	if r.delay.Swap(false) {
		select {
		case <-time.After(5200 * time.Millisecond):
		case <-ctx.Done():
			return storecenter.ProductMerchantBinding{}, goods.OfficialRuleSnapshot{}, ctx.Err()
		}
	}
	m, rule, e := (readyRules{}).ReadTargetRules(ctx, s, store, in)
	m.ServiceExpiresAt = r.expires
	rule.Attributes.MainAttributeStatus = pointer(2)
	rule.Attributes.Attributes = []model.Attribute{{ID: 12, Name: "Color", Type: pointer(1), Show: pointer(1), MainLabel: pointer(1), Mode: pointer(2), Status: pointer(3), MaximumSelections: pointer(1), Options: []model.AttributeOption{{ID: 34, Name: "Black", Show: pointer(1)}, {ID: 35, Name: "White", Show: pointer(1)}}}, {ID: 13, Name: "Size", Type: pointer(1), Show: pointer(1), MainLabel: pointer(2), Mode: pointer(2), Status: pointer(3), Dimension: pointer(3), MaximumSelections: pointer(1), Options: []model.AttributeOption{{ID: 36, Name: "M", Show: pointer(1)}, {ID: 37, Name: "L", Show: pointer(1)}}}}
	flags := map[string]bool{"sku_image_required": false, "spu_image_detail_show": true, "spu_image_detail_required": false, "spu_image_detail_single": true, "spu_image_square_show": false, "spu_image_square_required": false, "skc_image_detail_show": true, "skc_image_detail_required": false, "skc_image_detail_single": true, "skc_image_square_show": false, "skc_image_square_required": false}
	rule.Fill.Pictures = nil
	for _, k := range []string{"sku_image_required", "spu_image_detail_show", "spu_image_detail_required", "spu_image_detail_single", "spu_image_square_show", "spu_image_square_required", "skc_image_detail_show", "skc_image_detail_required", "skc_image_detail_single", "skc_image_square_show", "skc_image_square_required"} {
		rule.Fill.Pictures = append(rule.Fill.Pictures, model.PictureRule{Field: k, Enabled: pointer(flags[k])})
	}
	rule.Linked = nil
	for _, g := range goods.LinkedRuleGroups(in.Product) {
		rule.Linked = append(rule.Linked, model.LinkedRules{GroupID: g.ID, Attributes: []model.LinkedAttributeRule{}})
	}
	return m, rule, e
}

type uncertainColorCommit struct {
	*recordstore.Repository
	once atomic.Bool
}

func (r *uncertainColorCommit) SaveColorSplit(ctx context.Context, p record.ColorSplitPrepared) (record.ColorSplitReceipt, error) {
	v, e := r.Repository.SaveColorSplit(ctx, p)
	if e == nil && r.once.Swap(false) {
		return record.ColorSplitReceipt{}, record.ErrTargetUnknown
	}
	return v, e
}

func TestPostgresColorSplitAtomicReplayMembershipAndLocalOnly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	pg, e := tcpostgres.Run(ctx, "postgres:17.2-alpine", tcpostgres.WithDatabase("color_split"), tcpostgres.WithUsername("color_owner"), tcpostgres.WithPassword("isolated-color-fixture"), tcpostgres.BasicWaitStrategies())
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
	snapshot := catalog.ProductSnapshot{Title: "Explicit synthetic black white T-shirt", Variants: []catalog.Variant{{SKU: "BLACK-M", Stock: pointer(0)}, {SKU: "BLACK-L"}, {SKU: "WHITE-M"}}}
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
	draft := goods.OfficialDraftInput{Product: model.PublishProduct{CategoryID: 123, ProductTypeID: 456, BrandCode: "brand-a", SourceSystem: "OpenAPI", SuitFlag: "0", IsSPUPic: true, Names: []model.LanguageContent{{Language: "en", Name: snapshot.Title}}, Attributes: []model.AttributeValue{}, SKCs: []model.ProductSKC{}}, Images: []goods.OfficialImageSlot{{Group: "spu", AssetID: "main", Sort: 1, Type: 1}, {Group: "sku", SKC: 1, SKU: 0, AssetID: "detail", Sort: 1, Type: 1}}}
	for i, c := range []int64{34, 35} {
		skc := model.ProductSKC{SupplierCode: []string{"BLACK", "WHITE"}[i], SaleAttribute: model.AttributeValue{AttributeID: 12, AttributeValueID: pointer(c)}}
		for j, code := range [][]string{{"BLACK-M", "BLACK-L"}, {"WHITE-M"}}[i] {
			sku := model.ProductSKU{SupplierSKU: code, Length: "10", Width: "8", Height: "2", Weight: pointer(float64(100)), MallState: 1, Prices: []model.ProductPrice{{Currency: "USD", BasePrice: 12.5, SubSite: "shein-us"}}, Stock: []model.ProductStock{}, SaleAttributes: []model.AttributeValue{{AttributeID: 13, AttributeValueID: pointer(int64(36 + j))}}}
			if i == 0 && j == 0 {
				sku.Stock = []model.ProductStock{{Quantity: 0}}
			}
			skc.SKUs = append(skc.SKUs, sku)
		}
		draft.Product.SKCs = append(draft.Product.SKCs, skc)
	}
	makePlan := func() (record.ColorSplitPlan, record.TargetRecord) {
		base, e := service.Create(actor, uuid.NewString(), record.TargetInput{SourceID: source.ID, StoreID: uuid.NewString(), EffectiveVersion: 1, Draft: draft})
		require.NoError(t, e)
		// A parent created by the existing candidate adoption must not propagate its one-use binding.
		imported, e := service.PreviewSheinImport(actor, record.SheinImportRequest{SourceID: source.ID, BaseRecordID: base.Record.ID, CompletionDraft: record.CompletionDraftSelection{RecordID: candidate.ID, Revision: 1}, Selection: record.SheinImportSelection{Title: true}})
		require.NoError(t, e)
		base, e = service.Create(actor, uuid.NewString(), imported.Input)
		require.NoError(t, e)
		require.NotNil(t, base.Record.Input.SheinImport)
		p, e := service.PreviewColorSplit(actor, record.ColorSplitRequest{SourceID: source.ID, BaseRecordID: base.Record.ID, CompletionDraft: record.CompletionDraftSelection{RecordID: candidate.ID, Revision: 1}, Selection: record.ColorSplitSelection{ColorAttributeID: 12, SKUs: []record.ColorSKUMapping{{SourceIndex: 0, SKCIndex: 0, SKUIndex: 0}, {SourceIndex: 1, SKCIndex: 0, SKUIndex: 1}, {SourceIndex: 2, SKCIndex: 1, SKUIndex: 0}}, SharedImages: []record.ColorSharedImage{{SlotIndex: 0, ColorValueIDs: []int64{34, 35}}}}})
		require.NoError(t, e)
		return p, base.Record
	}
	plan, base := makePlan()
	key := uuid.NewString()
	// An actual PostgreSQL failure on the second child must not leak the first child/head.
	require.NoError(t, db.Exec(`CREATE FUNCTION reject_white_child() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.purpose='shein_color_local' AND NEW.body_json->'input'->'colorPartition'->>'colorValueId'='35' THEN RAISE EXCEPTION 'controlled second child failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_white_child BEFORE INSERT ON listing_target_records FOR EACH ROW EXECUTE FUNCTION reject_white_child()`).Error)
	_, e = service.CreateColorSplit(actor, key, plan)
	require.Error(t, e)
	var count int64
	require.NoError(t, db.Table("listing_target_records").Count(&count).Error)
	require.EqualValues(t, 2, count)
	require.NoError(t, db.Table("listing_target_split_commands").Count(&count).Error)
	require.Zero(t, count)
	require.NoError(t, db.Exec("DROP TRIGGER reject_white_child ON listing_target_records; DROP FUNCTION reject_white_child()").Error)
	saved, e := service.CreateColorSplit(actor, key, plan)
	require.NoError(t, e)
	require.Len(t, saved.Records, 2)
	replay, e := service.CreateColorSplit(actor, key, plan)
	require.NoError(t, e)
	require.True(t, replay.Replayed)
	require.Equal(t, saved.Records, replay.Records)
	_, e = service.CreateColorSplit(actor, uuid.NewString(), plan)
	require.ErrorIs(t, e, record.ErrSplitExists)
	original, e := repo.ReadTargetHead(actor, scope, base.TargetID)
	require.NoError(t, e)
	require.Equal(t, base, original)
	require.Equal(t, snapshot.Title, candidate.Content.Title)
	require.Len(t, saved.Records[0].Input.Draft.Product.SKCs[0].SKUs, 2)
	require.Len(t, saved.Records[1].Input.Draft.Product.SKCs[0].SKUs, 1)
	require.Len(t, saved.Records[0].Input.Draft.Images, 1)
	require.Len(t, saved.Records[1].Input.Draft.Images, 2)
	require.Equal(t, 0, saved.Records[1].Input.Draft.Images[1].SKC)
	for _, v := range saved.Records {
		require.False(t, record.PublicationReady(v))
		require.Nil(t, v.Input.SheinImport)
		require.True(t, v.Input.ColorPartition.LocalOnly)
	}
	edit := saved.Records[0].Input
	edit.ExpectedRevision = 1
	edit.Draft.Product.Names = []model.LanguageContent{{Language: "en", Name: "Edited black only"}}
	updated, e := service.Create(actor, uuid.NewString(), edit)
	require.NoError(t, e)
	require.EqualValues(t, 2, updated.Record.Revision)
	current, e := service.ReadColorSplitFamily(actor, source.ID, saved.FamilyID)
	require.NoError(t, e)
	require.EqualValues(t, 2, current.Records[0].Revision)
	require.EqualValues(t, 1, current.Records[1].Revision)
	old, e := service.ReadColorSplitCommand(actor, key)
	require.NoError(t, e)
	require.EqualValues(t, 1, old.Records[0].Revision)
	edit.Draft.Product.SKCs[0].SKUs[0].SupplierSKU = "FORGED"
	_, e = service.Create(actor, uuid.NewString(), edit)
	require.Error(t, e)
	// Composite FK must reject mixing a valid record from another store/family.
	otherPlan, _ := makePlan()
	other, e := service.CreateColorSplit(actor, uuid.NewString(), otherPlan)
	require.NoError(t, e)
	require.Error(t, db.Exec("UPDATE listing_target_split_commands SET second_record_id=?,second_target_id=? WHERE command_key=?", other.Records[1].ID, other.Records[1].TargetID, key).Error)
	for _, foreign := range []collection.Scope{{"foreign", scope.ActorID, scope.MemberID}, {scope.OrganizationID, scope.ActorID, "new-member"}} {
		_, e = repo.ReadColorSplitCommand(actor, foreign, key)
		require.ErrorIs(t, e, record.ErrNotFound)
	}
	// A direct attempt to downgrade a child to a standard identity must fail.
	downgraded := saved.Records[0].Input
	downgraded.ColorPartition = nil
	downgraded.ExpectedRevision = 2
	_, e = service.Create(actor, uuid.NewString(), downgraded)
	require.Error(t, e)
	require.Error(t, db.Exec("UPDATE listing_preparation_targets SET member_id='foreign' WHERE id=?", saved.Records[0].TargetID).Error)
	require.Error(t, db.Exec("UPDATE listing_target_split_commands SET second_record_id=first_record_id,second_target_id=first_target_id WHERE command_key=?", key).Error)
	// Lost COMMIT/HTTP acknowledgement uses the original key, never a second create.
	uncertainPlan, _ := makePlan()
	uncertainRepo := &uncertainColorCommit{Repository: repo}
	uncertainRepo.once.Store(true)
	uncertainService, e := record.NewTargetService(record.TargetDependencies{Sources: selector, Products: record.OriginalTargetProduct{}, Assets: readyAssets{}, Rules: rules, Records: uncertainRepo, Images: recordTestImages{}, Authorizer: auth, CompletionVersions: versions})
	require.NoError(t, e)
	uncertainKey := uuid.NewString()
	_, e = uncertainService.CreateColorSplit(actor, uncertainKey, uncertainPlan)
	require.ErrorIs(t, e, record.ErrTargetUnknown)
	found, e := uncertainService.ReadColorSplitCommand(actor, uncertainKey)
	require.NoError(t, e)
	again, e := uncertainService.CreateColorSplit(actor, uncertainKey, uncertainPlan)
	require.NoError(t, e)
	require.Equal(t, found.Records, again.Records)
	require.True(t, again.Replayed)
	// Parallel same-key delivery returns exactly the same pair; fresh keys collide on family.
	concurrent, _ := makePlan()
	sharedKey := uuid.NewString()
	var wg sync.WaitGroup
	errs := make([]error, 2)
	pairs := make([]record.ColorSplitReceipt, 2)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			pairs[i], errs[i] = service.CreateColorSplit(actor, sharedKey, concurrent)
		}(i)
	}
	wg.Wait()
	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	require.Equal(t, pairs[0].Records, pairs[1].Records)
	concurrent, _ = makePlan()
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = service.CreateColorSplit(actor, uuid.NewString(), concurrent)
		}(i)
	}
	wg.Wait()
	require.True(t, (errs[0] == nil) != (errs[1] == nil))
	if errs[0] != nil {
		require.ErrorIs(t, errs[0], record.ErrSplitExists)
	} else {
		require.ErrorIs(t, errs[1], record.ErrSplitExists)
	}
	// Preparation exceeding the original five-second proof still succeeds after a fresh Select.
	slow, _ := makePlan()
	rules.delay.Store(true)
	_, e = service.CreateColorSplit(actor, uuid.NewString(), slow)
	require.NoError(t, e)
	expired, expiredBase := makePlan()
	lock := db.Begin()
	require.NoError(t, lock.Error)
	lockKey := collection.Digest([]string{scope.OrganizationID, scope.ActorID, "supply-target", expiredBase.TargetID})
	require.NoError(t, lock.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", lockKey).Error)
	beforeExpired := int64(0)
	require.NoError(t, db.Table("listing_target_records").Count(&beforeExpired).Error)
	outcome := make(chan error, 1)
	go func() { _, err := service.CreateColorSplit(actor, uuid.NewString(), expired); outcome <- err }()
	time.Sleep(5300 * time.Millisecond)
	require.NoError(t, lock.Rollback().Error)
	require.Error(t, <-outcome)
	require.NoError(t, db.Table("listing_target_records").Count(&count).Error)
	require.Equal(t, beforeExpired, count)
	stored, e := reader.GetSnapshot(actor, catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: productKey}, 1)
	require.NoError(t, e)
	require.Equal(t, snapshot, stored.Snapshot)
	require.Equal(t, candidate, versions.value)

	// Serving verification must reject a schema missing the durable purpose constraint.
	require.NoError(t, db.Exec("ALTER TABLE listing_target_records DROP CONSTRAINT listing_target_records_purpose").Error)
	require.ErrorIs(t, recordstore.VerifySchema(ctx, db), record.ErrUnavailable)

}
