package preparationpersistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/authidentity"
	recordstore "task-processor/internal/integration/persistence/listing/record"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
)

type uploadPlanInputsFixture struct{ preparation.UploadPlanInputs }

// Same real-commit/lost-ack pattern as the existing completiondraft tests.
type uploadPlanLostCommitPool struct {
	gorm.ConnPool
	db *sql.DB
}
type uploadPlanLostCommitTx struct{ *sql.Tx }

func (p uploadPlanLostCommitPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	tx, err := p.db.BeginTx(ctx, opts)
	return &uploadPlanLostCommitTx{tx}, err
}
func (tx uploadPlanLostCommitTx) Commit() error {
	if err := tx.Tx.Commit(); err != nil {
		return err
	}
	return errors.New("isolated lost COMMIT acknowledgement")
}

func (uploadPlanInputsFixture) ReadUploadPlanBinding(context.Context, collection.Scope, preparation.UploadPlanInput) (string, error) {
	return collection.Digest("current-store-binding"), nil
}
func (uploadPlanInputsFixture) ReadUploadPlanFacts(_ context.Context, _ preparation.UploadPlan, _ preparation.UploadPlanItem) (preparation.UploadPlanFacts, error) {
	return preparation.UploadPlanFacts{}, nil
}
func (uploadPlanInputsFixture) AuthorizeUploadPlanStep(context.Context, preparation.UploadPlan, preparation.OperationInput) error {
	return nil
}

func TestPostgresUploadPlanWholeSelectionAtomicContinueAndReplay(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("upload_plan_1079"), tcpostgres.WithUsername("upload_plan_owner"), tcpostgres.WithPassword("isolated-upload-plan-test"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(8)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	require.NoError(t, catalogstore.AutoMigrate(db))
	require.NoError(t, collectionstore.InstallSchema(db))
	require.NoError(t, InstallSchema(db))
	require.NoError(t, recordstore.InstallSchema(db))
	require.NoError(t, InstallOperationSchema(db))
	require.NoError(t, InstallUploadPlanSchema(db))
	scope := collection.Scope{OrganizationID: "plan-org", ActorID: "plan-actor", MemberID: "plan-member"}
	actor := authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Hour)})
	batchID, prepID := uuid.NewString(), uuid.NewString()
	require.NoError(t, db.Exec("INSERT INTO product_collection_batches(organization_id,actor_id,member_id,id,name,kind,revision,created_at) VALUES(?,?,?,?,?,'manual',1,now())", scope.OrganizationID, scope.ActorID, scope.MemberID, batchID, "complete-plan-selection").Error)
	in := preparation.TransferInput{BatchID: batchID, ExpectedRevision: 1}
	raw, err := json.Marshal(in)
	require.NoError(t, err)
	require.NoError(t, db.Create(&preparationRow{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, ID: prepID, CommandKey: uuid.NewString(), InputHash: collection.Digest(in), InputJSON: raw, SourceBatchID: batchID, SourceRevision: 1, Name: "complete plan", ItemCount: 1005, Revision: 1, CreatedAt: time.Now()}).Error)
	for i := 0; i < 1005; i++ {
		product, itemID, sourceID, publication := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
		snapshot := []byte(`{"title":"isolated product"}`)
		require.NoError(t, db.Create(&catalogstore.SnapshotVersionRecord{TenantID: scope.OrganizationID, ProductKey: product, Version: 1, PublicationID: publication, PayloadHash: collection.Digest(i), SnapshotJSON: snapshot}).Error)
		require.NoError(t, db.Exec("INSERT INTO product_collection_items(organization_id,actor_id,member_id,id,batch_id,product_key,publication_id,original_version,source_kind,source_operation_id,revision,created_at) VALUES(?,?,?,?,?,?,?,1,'own','',1,now())", scope.OrganizationID, scope.ActorID, scope.MemberID, itemID, batchID, product, publication).Error)
		require.NoError(t, db.Create(&sourceRow{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, ID: sourceID, PreparationID: prepID, CollectionItemID: itemID, CollectionRevision: 1, ProductKey: product, PublicationID: publication, OriginalVersion: 1, SourceKind: "own"}).Error)
	}
	repo, err := NewUploadPlanRepository(ctx, db)
	require.NoError(t, err)
	service, err := preparation.NewUploadPlanService(fixedAuth{scope}, repo, uploadPlanInputsFixture{})
	require.NoError(t, err)
	config := preparation.UploadPlanInput{PreparationID: prepID, ExpectedRevision: 1, StoreID: uuid.NewString(), Rule: "standard", PublicationMode: "platform"}
	key := uuid.NewString()
	created, err := service.Create(actor, key, config)
	require.NoError(t, err)
	require.EqualValues(t, 1005, created.Plan.Count)
	again, err := service.Create(actor, key, config)
	require.NoError(t, err)
	require.True(t, again.Replayed)
	require.Equal(t, created.Plan, again.Plan)
	preview, err := service.Preview(actor, created.Plan.ID, preparation.UploadPlanContinueInput{ExpectedRevision: 1})
	require.NoError(t, err)
	require.EqualValues(t, 1005, preview.Count)
	require.EqualValues(t, 1005, preview.Actionable)
	require.Len(t, preview.Items, 100, "display rows are bounded while the full selection is confirmed")
	command := preparation.UploadPlanContinueInput{ExpectedRevision: 1, PreviewHash: preview.Hash}
	commandKey := uuid.NewString()
	advanced, err := service.Continue(actor, created.Plan.ID, commandKey, command)
	require.NoError(t, err)
	require.NotEmpty(t, advanced.OperationIDs)
	var total int64
	require.NoError(t, db.Model(&operationItemRow{}).Where("operation_id IN ?", advanced.OperationIDs).Count(&total).Error)
	require.EqualValues(t, 1005, total, "a whole batch never truncates to explicit-array limit")
	replayed, err := service.Continue(actor, created.Plan.ID, commandKey, command)
	require.NoError(t, err)
	require.True(t, replayed.Replayed)
	require.Equal(t, advanced.OperationIDs, replayed.OperationIDs)
	changed := command
	changed.PreviewHash = collection.Digest("different approval")
	_, err = service.Continue(actor, created.Plan.ID, commandKey, changed)
	require.ErrorIs(t, err, preparation.ErrConflict)
	otherScope := scope
	otherScope.ActorID = "other-actor"
	_, err = repo.ReadUploadPlan(ctx, otherScope, created.Plan.ID)
	require.ErrorIs(t, err, preparation.ErrNotFound)

	selected := preview.Items[0].Item.SourceID
	t.Run("runtime role keeps immutable sources read only", func(t *testing.T) {
		// Match the serving role's relevant grants, not the schema owner's
		// superuser privileges used to seed this isolated database.
		require.NoError(t, db.Exec("CREATE ROLE upload_plan_serving NOLOGIN").Error)
		require.NoError(t, db.Exec(`GRANT USAGE ON SCHEMA public TO upload_plan_serving;
GRANT SELECT,INSERT ON listing_preparation_sources TO upload_plan_serving;
GRANT SELECT,INSERT,UPDATE ON listing_preparations,listing_preparation_targets,listing_preparation_operations,listing_preparation_operation_items,listing_preparation_upload_plans,listing_preparation_upload_plan_items TO upload_plan_serving`).Error)
		conn, err := pool.Conn(ctx)
		require.NoError(t, err)
		defer conn.Close()
		_, err = conn.ExecContext(ctx, "SET ROLE upload_plan_serving")
		require.NoError(t, err)
		defer conn.ExecContext(context.Background(), "RESET ROLE")
		serving, err := gorm.Open(postgres.New(postgres.Config{Conn: conn}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, err)
		var mayUpdate, mayDelete bool
		require.NoError(t, serving.Raw("SELECT has_table_privilege(current_user,'listing_preparation_sources','UPDATE')").Scan(&mayUpdate).Error)
		require.NoError(t, serving.Raw("SELECT has_table_privilege(current_user,'listing_preparation_sources','DELETE')").Scan(&mayDelete).Error)
		require.False(t, mayUpdate)
		require.False(t, mayDelete)
		t.Run("create and continue original plan", func(t *testing.T) {
			runtimeRepo, err := NewUploadPlanRepository(ctx, serving)
			require.NoError(t, err)
			runtimeService, err := preparation.NewUploadPlanService(fixedAuth{scope}, runtimeRepo, uploadPlanInputsFixture{})
			require.NoError(t, err)
			subset := config
			subset.SourceIDs = []string{selected}
			created, err := runtimeService.Create(actor, uuid.NewString(), subset)
			require.NoError(t, err)
			checked, err := runtimeService.Preview(actor, created.Plan.ID, preparation.UploadPlanContinueInput{ExpectedRevision: 1})
			require.NoError(t, err)
			command := preparation.UploadPlanContinueInput{ExpectedRevision: 1, PreviewHash: checked.Hash}
			key := uuid.NewString()
			continued, err := runtimeService.Continue(actor, created.Plan.ID, key, command)
			require.NoError(t, err)
			require.Equal(t, created.Plan.ID, continued.PlanID)
			require.Len(t, continued.OperationIDs, 1)
			again, err := runtimeService.Continue(actor, created.Plan.ID, key, command)
			require.NoError(t, err)
			require.True(t, again.Replayed)
			require.Equal(t, continued.OperationIDs, again.OperationIDs)
		})
		t.Run("prepare direct operation", func(t *testing.T) {
			err := serving.Transaction(func(tx *gorm.DB) error {
				out, err := prepareOperationTx(ctx, tx, scope, uuid.NewString(), preparation.OperationInput{PreparationID: prepID, ExpectedRevision: 1, StoreID: config.StoreID, Action: preparation.OperationAdapt, SourceIDs: []string{selected}})
				if err == nil {
					require.EqualValues(t, 1, out.Operation.Count)
				}
				return err
			})
			require.NoError(t, err)
		})
	})
	t.Run("different keys at one revision bind only one operation", func(t *testing.T) {
		subset := config
		subset.SourceIDs = []string{selected}
		p, err := service.Create(actor, uuid.NewString(), subset)
		require.NoError(t, err)
		v, err := service.Preview(actor, p.Plan.ID, preparation.UploadPlanContinueInput{ExpectedRevision: 1})
		require.NoError(t, err)
		request := preparation.UploadPlanContinueInput{ExpectedRevision: 1, PreviewHash: v.Hash}
		var wg sync.WaitGroup
		results := make(chan error, 2)
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, e := service.Continue(actor, p.Plan.ID, uuid.NewString(), request)
				results <- e
			}()
		}
		wg.Wait()
		close(results)
		success, conflict := 0, 0
		for e := range results {
			if e == nil {
				success++
			} else if errors.Is(e, preparation.ErrConflict) {
				conflict++
			} else {
				t.Fatal(e)
			}
		}
		require.Equal(t, 1, success)
		require.Equal(t, 1, conflict)
	})

	t.Run("late local failure rolls back child and plan revision", func(t *testing.T) {
		subset := config
		subset.SourceIDs = []string{selected}
		p, err := service.Create(actor, uuid.NewString(), subset)
		require.NoError(t, err)
		v, err := service.Preview(actor, p.Plan.ID, preparation.UploadPlanContinueInput{ExpectedRevision: 1})
		require.NoError(t, err)
		require.NoError(t, db.Callback().Update().Before("gorm:update").Register("plan-late-failure", func(tx *gorm.DB) {
			if tx.Statement.Table == "listing_preparation_upload_plan_items" {
				tx.AddError(errors.New("isolated late failure"))
			}
		}))
		commandKey := uuid.NewString()
		_, err = service.Continue(actor, p.Plan.ID, commandKey, preparation.UploadPlanContinueInput{ExpectedRevision: 1, PreviewHash: v.Hash})
		require.Error(t, err)
		require.NoError(t, db.Callback().Update().Remove("plan-late-failure"))
		current, err := service.Read(actor, p.Plan.ID)
		require.NoError(t, err)
		require.EqualValues(t, 1, current.Revision)
		var children int64
		require.NoError(t, db.Model(&operationRow{}).Where("input_json->>'planId'=?", p.Plan.ID).Count(&children).Error)
		require.Zero(t, children)
		_, err = service.Continue(actor, p.Plan.ID, commandKey, preparation.UploadPlanContinueInput{ExpectedRevision: 1, PreviewHash: v.Hash})
		require.NoError(t, err)
	})
	t.Run("lost commit acknowledgement replays original child", func(t *testing.T) {
		subset := config
		subset.SourceIDs = []string{selected}
		p, err := service.Create(actor, uuid.NewString(), subset)
		require.NoError(t, err)
		v, err := service.Preview(actor, p.Plan.ID, preparation.UploadPlanContinueInput{ExpectedRevision: 1})
		require.NoError(t, err)
		lostDB := db.Session(&gorm.Session{NewDB: true, Initialized: true})
		lostDB.Statement.ConnPool = uploadPlanLostCommitPool{ConnPool: pool, db: pool}
		lostRepo, err := NewUploadPlanRepository(ctx, lostDB)
		require.NoError(t, err)
		lostService, err := preparation.NewUploadPlanService(fixedAuth{scope}, lostRepo, uploadPlanInputsFixture{})
		require.NoError(t, err)
		key := uuid.NewString()
		command := preparation.UploadPlanContinueInput{ExpectedRevision: 1, PreviewHash: v.Hash}
		_, err = lostService.Continue(actor, p.Plan.ID, key, command)
		require.ErrorIs(t, err, preparation.ErrUnknown)
		recovered, err := service.Continue(actor, p.Plan.ID, key, command)
		require.NoError(t, err)
		require.True(t, recovered.Replayed)
		require.Len(t, recovered.OperationIDs, 1)
		var total int64
		require.NoError(t, db.Model(&operationRow{}).Where("input_json->>'planId'=?", p.Plan.ID).Count(&total).Error)
		require.EqualValues(t, 1, total)
	})
	t.Run("stop is durable and stale continuation cannot bind later work", func(t *testing.T) {
		subset := config
		subset.SourceIDs = []string{selected}
		p, err := service.Create(actor, uuid.NewString(), subset)
		require.NoError(t, err)
		v, err := service.Preview(actor, p.Plan.ID, preparation.UploadPlanContinueInput{ExpectedRevision: 1})
		require.NoError(t, err)
		stopped, err := service.Cancel(actor, p.Plan.ID, 1)
		require.NoError(t, err)
		require.True(t, stopped.Cancelled)
		require.EqualValues(t, 2, stopped.Revision)
		_, err = service.Continue(actor, p.Plan.ID, uuid.NewString(), preparation.UploadPlanContinueInput{ExpectedRevision: 1, PreviewHash: v.Hash})
		require.ErrorIs(t, err, preparation.ErrConflict)
		again, err := service.Cancel(actor, p.Plan.ID, 1)
		require.NoError(t, err)
		require.Equal(t, stopped, again)
		var total int64
		require.NoError(t, db.Model(&operationRow{}).Where("input_json->>'planId'=?", p.Plan.ID).Count(&total).Error)
		require.Zero(t, total)
	})

	t.Run("synthetic evidence cannot start direct or planned execution", func(t *testing.T) {
		product := "research-fixture-v1-" + uuid.NewString()
		require.NoError(t, db.Create(&catalogstore.SnapshotVersionRecord{TenantID: scope.OrganizationID, ProductKey: product, Version: 1, PublicationID: uuid.NewString(), PayloadHash: collection.Digest(product), SnapshotJSON: []byte(`{"title":"合成原件"}`)}).Error)
		require.NoError(t, db.Model(&sourceRow{}).Where("id=?", selected).Update("product_key", product).Error)

		operations, err := NewOperationRepository(ctx, db)
		require.NoError(t, err)

		for _, ids := range [][]string{nil, {selected}} {
			input := config.Operation(preparation.OperationAdapt)
			input.SourceIDs = ids
			key := uuid.NewString()
			err = db.WithContext(actor).Transaction(func(tx *gorm.DB) error {
				_, err := prepareOperationTx(actor, tx, scope, key, input)
				return err
			})
			require.ErrorIs(t, err, preparation.ErrForbidden)
			_, err = operations.FindOperation(actor, scope, key, collection.Digest(input))
			require.ErrorIs(t, err, preparation.ErrNotFound)
		}

		subset := config
		subset.SourceIDs = []string{selected}
		plan, err := service.Create(actor, uuid.NewString(), subset)
		require.NoError(t, err)
		v, err := service.Preview(actor, plan.Plan.ID, preparation.UploadPlanContinueInput{ExpectedRevision: 1})
		require.NoError(t, err)
		_, err = service.Continue(actor, plan.Plan.ID, uuid.NewString(), preparation.UploadPlanContinueInput{ExpectedRevision: 1, PreviewHash: v.Hash})
		require.ErrorIs(t, err, preparation.ErrForbidden)
		stored, err := service.Read(actor, plan.Plan.ID)
		require.NoError(t, err)
		require.EqualValues(t, 1, stored.Revision)
	})
}
