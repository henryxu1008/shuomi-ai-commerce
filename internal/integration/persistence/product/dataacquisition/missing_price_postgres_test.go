package dataacquisitionpersistence

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	keystore "task-processor/internal/integration/persistence/dataservice"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
)

func TestUnavailableTerminalReadsAndRejectsWrites(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	c, err := tcpostgres.Run(ctx, "postgres:17-alpine", tcpostgres.WithDatabase("product_acquisition"), tcpostgres.WithUsername("test_owner"), tcpostgres.WithPassword("isolated-only"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, c.Terminate(context.Background())) })
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { pool.Close() })
	for _, install := range []func(*gorm.DB) error{keystore.InstallSchema, catalogstore.AutoMigrate, collectionstore.InstallSchema, InstallSchema} {
		require.NoError(t, install(db))
	}
	current, err := PriceContractState(ctx, db)
	require.NoError(t, err)
	require.True(t, current, "immutable projection must match the real fresh native owner installer")
	scope := collection.Scope{OrganizationID: "retained-org", ActorID: "original-actor", MemberID: "original-member"}
	jobID, itemID, command := uuid.NewString(), uuid.NewString(), uuid.NewString()
	q := dataacquisition.Query{Site: "us", Mode: "asin", ASINs: []string{"B000123456"}, Limit: 1}
	q, err = dataacquisition.NormalizeQuery(q)
	require.NoError(t, err)
	raw, err := json.Marshal(q)
	require.NoError(t, err)
	intent := orgresource.ConsumerChargeIntent{Identity: orgresource.ConsumerChargeIdentity{OrganizationID: scope.OrganizationID, Consumer: orgresource.ConsumerAmazonData, OperationID: itemID}, ActorID: scope.ActorID, MemberID: scope.MemberID, Funding: orgresource.FundingEnterprise, ResourceType: orgresource.ResourceDataRow, Quantity: 1, Fingerprint: collection.Digest("original-unpriced-intent"), BusinessScope: "amazon-data:" + jobID + ":" + itemID}
	iraw, err := json.Marshal(intent)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`INSERT INTO data_acquisition_jobs(organization_id,actor_id,member_id,id,command_key,input_hash,query_json,funding,state,discovered,quota_reserved,day_window,month_window,created_at,deadline) VALUES(?,?,?,?,?,?,?,?, 'FAILED',true,0,CURRENT_DATE,CURRENT_DATE,now(),now()+interval '30 minutes')`, scope.OrganizationID, scope.ActorID, scope.MemberID, jobID, command, collection.Digest("original-input-hash"), string(raw), orgresource.FundingEnterprise).Error)
	require.NoError(t, db.Exec(`INSERT INTO data_acquisition_items(organization_id,actor_id,job_id,id,asin,state,intent_json,terminal_evidence,reason,created_at) VALUES(?,?,?,?,?,'FAILED',?,?,'resource_unavailable',now())`, scope.OrganizationID, scope.ActorID, jobID, itemID, q.ASINs[0], string(iraw), uuid.NewString()).Error)
	fingerprint := func() string {
		var s string
		require.NoError(t, db.Raw(`SELECT jsonb_build_array((SELECT jsonb_agg(to_jsonb(j)) FROM data_acquisition_jobs j),(SELECT jsonb_agg(to_jsonb(i)) FROM data_acquisition_items i),(SELECT count(*) FROM data_service_commands))::text`).Scan(&s).Error)
		return s
	}
	before := fingerprint()
	repo, err := NewRepository(ctx, db, &testAccess{}, func(context.Context, *gorm.DB, dataacquisition.Job, dataacquisition.Item) (collection.Source, error) {
		t.Fatal("must not publish")
		return collection.Source{}, nil
	}, newTestResultReader)
	require.NoError(t, err)
	job, err := repo.Read(ctx, scope, jobID)
	require.NoError(t, err, "missing-price FAILED must remain readable")
	wire, err := json.Marshal(job)
	require.NoError(t, err)
	var view map[string]any
	require.NoError(t, json.Unmarshal(wire, &view))
	require.Equal(t, "UNAVAILABLE_TERMINAL", view["priceStatus"])
	require.Nil(t, view["price"])
	require.Nil(t, view["confirmedFen"])
	require.Nil(t, view["pendingFen"])
	listed, err := repo.List(ctx, scope, 10)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	items, err := repo.Items(ctx, job)
	require.NoError(t, err)
	require.Len(t, items, 1)
	receipt := orgresource.ConsumerChargeReceipt{Intent: intent, ReservationID: uuid.NewString(), State: orgresource.ReservationReserved}
	for name, write := range map[string]func() error{
		"discover":       func() error { _, e := repo.Discover(ctx, job, []string{}); return e },
		"fail discovery": func() error { _, e := repo.FailDiscovery(ctx, job, "deadline"); return e },
		"claim":          func() error { _, e := repo.Claim(ctx, job, itemID); return e },
		"bind":           func() error { _, e := repo.BindReservation(ctx, job, itemID, receipt); return e },
		"fence":          func() error { _, e := repo.Fence(ctx, job, items[0], "deadline"); return e },
		"finish":         func() error { _, e := repo.Finish(ctx, job); return e },
		"record charge":  func() error { return repo.RecordCharge(ctx, job, itemID, receipt) },
		"cancel":         func() error { _, e := repo.Cancel(ctx, scope, jobID, uuid.NewString()); return e },
		"intent":         func() error { _, e := repo.ChargeIntent(ctx, intent.Identity); return e },
		"orphan receipt proof": func() error {
			p, e := repo.ChargeProof(ctx, receipt)
			require.NotEqual(t, orgresource.ConsumerEffectFailed, p.State)
			return e
		},
		"admit replay": func() error {
			_, e := repo.Admit(ctx, dataacquisition.Principal{Scope: scope}, command, q, orgresource.FundingEnterprise, dataacquisition.PriceConfirmation{Version: "current-price", MaximumCostFen: 0})
			return e
		},
	} {
		t.Run(name, func(t *testing.T) { require.Error(t, write()); require.Equal(t, before, fingerprint()) })
	}
	// A missing-price row cannot hide a saved result or pending resource state.
	require.NoError(t, db.Exec(`UPDATE data_acquisition_items SET reservation_id=? WHERE id=?`, uuid.NewString(), itemID).Error)
	_, err = repo.Read(ctx, scope, jobID)
	require.Error(t, err)
}
