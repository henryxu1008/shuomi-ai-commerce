package completiondraftpersistence_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/authidentity"
	store "task-processor/internal/integration/persistence/listing/completiondraft"
	prepstore "task-processor/internal/integration/persistence/listing/preparation"
	acquisitionstore "task-processor/internal/integration/persistence/product/acquisition"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	"task-processor/internal/listing/completiondraft"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
)

type fixedAuth struct{ scope collection.Scope }

func (a fixedAuth) Authorize(context.Context, string) (collection.Scope, error) { return a.scope, nil }

type unusedSources struct {
	sourcing.PublishedAcquisitionReader
}
type lostCommitPool struct {
	gorm.ConnPool
	sqlDB *sql.DB
}
type lostCommitTx struct{ *sql.Tx }

func (p lostCommitPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	tx, err := p.sqlDB.BeginTx(ctx, opts)
	return &lostCommitTx{tx}, err
}
func (t lostCommitTx) Commit() error {
	if err := t.Tx.Commit(); err != nil {
		return err
	}
	return errors.New("simulated lost COMMIT response")
}

func TestPostgresUnboundDraftAtomicCASReplayAndUnknownRecovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	c, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("completion605"), tcpostgres.WithUsername("completion_owner"), tcpostgres.WithPassword("isolated-completion-test"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, c.Terminate(context.Background())) })
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	require.NoError(t, productsourcing.InstallAcquisitionSchema(db))
	require.NoError(t, collectionstore.InstallSchema(db))
	require.NoError(t, prepstore.InstallSchema(db))
	_, err = store.NewRepository(ctx, db)
	require.ErrorIs(t, err, preparation.ErrUnavailable, "serving never installs schema")
	require.NoError(t, store.InstallSchema(db))
	repo, err := store.NewRepository(ctx, db)
	require.NoError(t, err)
	require.NoError(t, db.Exec("CREATE ROLE source_acquisition_runtime LOGIN PASSWORD 'isolated-completion-runtime'").Error)
	require.NoError(t, acquisitionstore.GrantRuntimePermissions(ctx, db, acquisitionstore.RuntimeCapabilities{Collections: true}))
	var baseline []string
	require.NoError(t, db.Raw("SELECT table_name||':'||privilege_type FROM information_schema.role_table_grants WHERE grantee=? ORDER BY 1", acquisitionstore.RuntimeRole).Scan(&baseline).Error)
	require.NoError(t, acquisitionstore.GrantCompletionDraftPermissions(ctx, db))
	var added []string
	require.NoError(t, db.Raw("SELECT table_name||':'||privilege_type FROM information_schema.role_table_grants WHERE grantee=? ORDER BY 1", acquisitionstore.RuntimeRole).Scan(&added).Error)
	for _, grant := range baseline {
		require.Contains(t, added, grant, "additive installer preserves original privileges")
	}
	require.Len(t, added, len(baseline)+12)
	runtimeURL, err := url.Parse(dsn)
	require.NoError(t, err)
	runtimeURL.User = url.UserPassword(acquisitionstore.RuntimeRole, "isolated-completion-runtime")
	runtimeDB, err := gorm.Open(postgres.Open(runtimeURL.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	runtimePool, err := runtimeDB.DB()
	require.NoError(t, err)
	runtimePool.SetMaxOpenConns(2)
	t.Cleanup(func() { require.NoError(t, runtimePool.Close()) })
	capability := acquisitionstore.RuntimeCapabilities{Collections: true, CompletionDrafts: true}
	require.NoError(t, acquisitionstore.VerifyRuntimePermissions(ctx, runtimeDB, capability))
	require.Error(t, acquisitionstore.VerifyRuntimePermissions(ctx, runtimeDB, acquisitionstore.RuntimeCapabilities{Collections: true}), "disabled capability rejects new grants")
	require.NoError(t, db.Exec("GRANT UPDATE ON listing_completion_draft_records TO source_acquisition_runtime").Error)
	require.Error(t, acquisitionstore.VerifyRuntimePermissions(ctx, runtimeDB, capability), "immutable record UPDATE is forbidden")
	require.NoError(t, db.Exec("REVOKE UPDATE ON listing_completion_draft_records FROM source_acquisition_runtime").Error)
	repo, err = store.NewRepository(ctx, runtimeDB)
	require.NoError(t, err)
	scope := collection.Scope{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a"}
	auth := fixedAuth{scope}
	cr, err := collectionstore.NewRepository(ctx, runtimeDB, func(*gorm.DB) (collectionstore.OwnPublisher, error) { return nil, collection.ErrUnavailable })
	require.NoError(t, err)
	cs, err := collection.NewService(cr, auth, unusedSources{})
	require.NoError(t, err)
	pr, err := prepstore.NewRepository(ctx, runtimeDB)
	require.NoError(t, err)
	ps, err := preparation.NewService(pr, cs, auth)
	require.NoError(t, err)
	batch, item, product, publication := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	raw := json.RawMessage(`{"title":"Original product"}`)
	require.NoError(t, db.Create(&catalogstore.SnapshotVersionRecord{TenantID: scope.OrganizationID, ProductKey: product, Version: 1, PublicationID: publication, PayloadHash: collection.Digest(raw), SnapshotJSON: raw}).Error)
	require.NoError(t, db.Exec("INSERT INTO product_collection_batches(organization_id,actor_id,member_id,id,name,kind,revision,created_at) VALUES(?,?,?,?,?,'manual',1,now())", scope.OrganizationID, scope.ActorID, scope.MemberID, batch, "真实原始批次").Error)
	require.NoError(t, db.Exec("INSERT INTO product_collection_items(organization_id,actor_id,member_id,id,batch_id,product_key,publication_id,original_version,source_kind,source_operation_id,revision,created_at) VALUES(?,?,?,?,?,?,?,1,'own','',1,now())", scope.OrganizationID, scope.ActorID, scope.MemberID, item, batch, product, publication).Error)
	actor := authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute)})
	transfer, err := ps.Transfer(actor, uuid.NewString(), preparation.TransferInput{BatchID: batch, ExpectedRevision: 1})
	require.NoError(t, err)
	sources, err := ps.ListSources(actor, transfer.Preparation.ID, collection.Query{Limit: 100})
	require.NoError(t, err)
	snapshots, err := catalogstore.NewBoundedSnapshotReader(runtimeDB, sourcing.MaxEncodedSnapshotBytes)
	require.NoError(t, err)
	selector, err := preparation.NewSourceSelector(ps, cs, pr, snapshots)
	require.NoError(t, err)
	service, err := completiondraft.NewService(selector, cs, ps, auth, repo)
	require.NoError(t, err)
	key := uuid.NewString()
	runtimePool.SetMaxOpenConns(1) // All source reads share the same bounded serving pool.
	input := completiondraft.SaveInput{SourceID: sources.Items[0].ID, Content: completiondraft.Content{Title: "Candidate R1"}}
	first, err := service.Save(actor, key, input)
	require.NoError(t, err)
	require.EqualValues(t, 1, first.Record.Revision)
	input2 := input
	input2.ExpectedRevision = 1
	input2.Content.Description = "Candidate R2"
	second, err := service.Save(actor, uuid.NewString(), input2)
	require.NoError(t, err)
	require.EqualValues(t, 2, second.Record.Revision)
	replay, err := service.Save(actor, key, input)
	require.NoError(t, err)
	require.True(t, replay.Replayed)
	require.Equal(t, first.Record, replay.Record, "old key returns immutable R1, not the newer head")
	byKey, err := service.ReadCommand(actor, key)
	require.NoError(t, err)
	require.Equal(t, first.Record, byKey.Record)
	input.Content.Title = "different payload"
	_, err = service.Save(actor, key, input)
	require.ErrorIs(t, err, preparation.ErrConflict)
	input2.ExpectedRevision = 2
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := service.Save(actor, uuid.NewString(), input2); results <- e }()
	}
	wg.Wait()
	close(results)
	won, conflicts := 0, 0
	for e := range results {
		if e == nil {
			won++
		} else {
			require.ErrorIs(t, e, preparation.ErrConflict)
			conflicts++
		}
	}
	require.Equal(t, 1, won)
	require.Equal(t, 1, conflicts)
	head, err := service.Read(actor, input2.SourceID)
	require.NoError(t, err)
	require.EqualValues(t, 3, head.Revision)
	page, err := service.List(actor, transfer.Preparation.ID, collection.Query{Limit: 100, Keyword: "Candidate"})
	require.NoError(t, err)
	require.EqualValues(t, 1, page.Total)
	require.Equal(t, head, page.Items[0])
	for _, other := range []collection.Scope{{OrganizationID: "org-b", ActorID: scope.ActorID, MemberID: scope.MemberID}, {OrganizationID: scope.OrganizationID, ActorID: "actor-b", MemberID: scope.MemberID}, {OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: "member-b"}} {
		_, e := repo.Read(ctx, other, input2.SourceID)
		require.ErrorIs(t, e, preparation.ErrNotFound)
		_, e = repo.ReadCommand(ctx, other, key)
		require.ErrorIs(t, e, preparation.ErrNotFound)
	}
	// A failure after record insertion rolls back all three owner tables.
	require.NoError(t, db.Exec(`CREATE FUNCTION reject_completion_command() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test command rejection'; END $$; CREATE TRIGGER reject_completion BEFORE INSERT ON listing_completion_draft_commands FOR EACH ROW EXECUTE FUNCTION reject_completion_command()`).Error)
	input2.ExpectedRevision = 3
	_, err = service.Save(actor, uuid.NewString(), input2)
	require.ErrorIs(t, err, preparation.ErrUnavailable)
	var records int64
	require.NoError(t, db.Table("listing_completion_draft_records").Count(&records).Error)
	require.EqualValues(t, 3, records)
	head, err = service.Read(actor, input2.SourceID)
	require.NoError(t, err)
	require.EqualValues(t, 3, head.Revision)
	require.NoError(t, db.Exec("DROP TRIGGER reject_completion ON listing_completion_draft_commands; DROP FUNCTION reject_completion_command()").Error)
	// A real commit followed by a lost response is UNKNOWN and recoverable by key.
	lostDB := db.Session(&gorm.Session{NewDB: true, Initialized: true})
	lostDB.Statement.ConnPool = lostCommitPool{ConnPool: pool, sqlDB: pool}
	lostRepo, err := store.NewRepository(ctx, lostDB)
	require.NoError(t, err)
	lostService, err := completiondraft.NewService(selector, cs, ps, auth, lostRepo)
	require.NoError(t, err)
	unknownKey := uuid.NewString()
	_, err = lostService.Save(actor, unknownKey, input2)
	require.ErrorIs(t, err, preparation.ErrUnknown)
	recovered, err := service.ReadCommand(actor, unknownKey)
	require.NoError(t, err)
	require.EqualValues(t, 4, recovered.Record.Revision)
	// Candidate text never updates the canonical source snapshot.
	var original string
	require.NoError(t, db.Raw("SELECT snapshot_json->>'title' FROM product_snapshot_versions WHERE product_key=?", product).Scan(&original).Error)
	require.Equal(t, "Original product", original)
	// Large valid descriptions must page within the existing response budget,
	// retaining every remaining source rather than truncating the list.
	content := completiondraft.Content{Title: "large-page-fixture", Description: strings.Repeat("a", 32768)}
	contentJSON, err := json.Marshal(content)
	require.NoError(t, err)
	for i := 0; i < 70; i++ {
		newItem, newSource, recordID, draftID := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
		draftID = collection.StableID(scope.OrganizationID, scope.ActorID, newSource, "completion-draft")
		newProduct, newPublication := uuid.NewString(), uuid.NewString()
		require.NoError(t, db.Create(&catalogstore.SnapshotVersionRecord{TenantID: scope.OrganizationID, ProductKey: newProduct, Version: 1, PublicationID: newPublication, PayloadHash: collection.Digest(raw), SnapshotJSON: raw}).Error)
		require.NoError(t, db.Exec(`INSERT INTO product_collection_items SELECT organization_id,actor_id,member_id,?,batch_id,?,?,original_version,source_kind,source_operation_id,revision,created_at FROM product_collection_items WHERE id=?`, newItem, newProduct, newPublication, item).Error)
		require.NoError(t, db.Exec(`INSERT INTO listing_preparation_sources SELECT organization_id,actor_id,member_id,?,preparation_id,?,collection_revision,?,?,original_version,source_kind,source_operation_id FROM listing_preparation_sources WHERE id=?`, newSource, newItem, newProduct, newPublication, input2.SourceID).Error)
		newValue := sources.Items[0]
		newValue.ID, newValue.CollectionItemID = newSource, newItem
		newValue.Source.ProductKey, newValue.Source.PublicationID = newProduct, newPublication
		sourceJSON, e := json.Marshal(newValue)
		require.NoError(t, e)
		require.NoError(t, db.Exec(`INSERT INTO listing_completion_draft_records VALUES(?,?,?,?,?,?,1,?,?,?,now())`, scope.OrganizationID, scope.ActorID, scope.MemberID, recordID, newSource, draftID, sourceJSON, contentJSON, collection.Digest(content)).Error)
		require.NoError(t, db.Exec(`INSERT INTO listing_completion_draft_heads VALUES(?,?,?,?,?,?,1)`, scope.OrganizationID, scope.ActorID, scope.MemberID, newSource, recordID, draftID).Error)
	}
	largePage, err := repo.List(ctx, scope, transfer.Preparation.ID, collection.Query{Limit: 100, Keyword: "large-page-fixture"})
	require.NoError(t, err)
	require.EqualValues(t, 70, largePage.Total)
	require.NotEmpty(t, largePage.NextCursor)
	require.Less(t, len(largePage.Items), 70)
	encoded, err := json.Marshal(largePage)
	require.NoError(t, err)
	require.LessOrEqual(t, len(encoded), 2<<20)
	rest, err := repo.List(ctx, scope, transfer.Preparation.ID, collection.Query{Limit: 100, Keyword: "large-page-fixture", After: largePage.NextCursor})
	require.NoError(t, err)
	require.Empty(t, rest.NextCursor)
	seen := map[string]bool{}
	for _, v := range append(largePage.Items, rest.Items...) {
		require.False(t, seen[v.Source.ID], "keyset pages do not overlap")
		seen[v.Source.ID] = true
	}
	require.Len(t, seen, 70)
}
