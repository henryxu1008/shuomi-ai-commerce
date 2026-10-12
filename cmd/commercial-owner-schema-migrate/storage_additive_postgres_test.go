package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/commercial/billing"
	commercialstore "task-processor/internal/integration/persistence/commercialbilling"
	capacitystore "task-processor/internal/integration/persistence/storagecapacity"
	"task-processor/internal/ledger/orgresource"
)

var capacityTables = []string{"storage_capacity_accounts", "storage_capacity_reservations", "storage_capacity_objects", "storage_capacity_bindings", "storage_capacity_lots", "storage_capacity_namespaces"}
var additiveColumns = map[string][]string{
	"commercial_offers":      {"storage_unit_bytes", "storage_duration_seconds"},
	"commercial_quotes":      {"storage_unit_bytes", "storage_duration_seconds"},
	"commercial_order_items": {"storage_unit_bytes", "storage_duration_seconds"},
	"commercial_orders":      {"storage_activation_receipt"},
}

type storageAdditiveFixture struct {
	db       *gorm.DB
	manifest string
}

func newStorageAdditiveFixture(t *testing.T) storageAdditiveFixture {
	t.Helper()
	dsn := os.Getenv("ISSUE573_TEST_DSN")
	if dsn == "" {
		t.Skip("requires isolated ISSUE573_TEST_DSN")
	}
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	root, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	name := "additive656_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = root.Exec("CREATE DATABASE " + name)
	require.NoError(t, err)
	u.Path = "/" + name
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: u.String(), PreferSimpleProtocol: true}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	raw, err := db.DB()
	require.NoError(t, err)
	var existed bool
	require.NoError(t, db.Raw("SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='commercial_owner_runtime')").Scan(&existed).Error)
	if !existed {
		require.NoError(t, db.Exec("CREATE ROLE commercial_owner_runtime NOLOGIN").Error)
	}
	t.Cleanup(func() {
		raw.Close()
		_, _ = root.Exec("DROP DATABASE " + name + " WITH (FORCE)")
		if !existed {
			_, _ = root.Exec("DROP ROLE commercial_owner_runtime")
		}
		root.Close()
	})
	require.NoError(t, commercialstore.AutoMigrate(db))
	store, err := commercialstore.New(db)
	require.NoError(t, err)
	offer := billing.Offer{OfferID: "kept-price", ProductKind: billing.ProductDataRow, ResourceType: orgresource.ResourceDataRow, Currency: billing.CurrencyCNY, UnitPriceMinor: 7, PricingVersion: "owner-kept-price", MinQuantity: 1, MaxQuantity: 100, Status: billing.OfferActive}
	require.NoError(t, store.CreateOffer(context.Background(), offer))
	quote, err := store.CreateQuote(context.Background(), billing.QuoteRequest{OrganizationID: "fixture-org", OfferID: offer.OfferID, Quantity: 3})
	require.NoError(t, err)
	_, err = store.CreatePendingResourceOrder(context.Background(), billing.CreateResourceOrderRequest{OrganizationID: "fixture-org", ActorID: "fixture-actor", QuoteID: quote.QuoteID, IdempotencyKey: "original-order"}, quote)
	require.NoError(t, err)
	for _, table := range capacityTables {
		require.NoError(t, db.Exec("DROP TABLE public."+table).Error)
	}
	for table, columns := range additiveColumns {
		for _, column := range columns {
			require.NoError(t, db.Exec("ALTER TABLE public."+table+" DROP COLUMN "+column).Error)
		}
	}
	require.NoError(t, db.Exec("CREATE TABLE public.retained_unrelated_fact (id text PRIMARY KEY, value text); INSERT INTO public.retained_unrelated_fact VALUES ('kept','original')").Error)
	require.NoError(t, db.Exec("GRANT SELECT ON public.commercial_offers TO commercial_owner_runtime; GRANT SELECT, INSERT ON public.commercial_quotes, public.commercial_order_items TO commercial_owner_runtime; GRANT SELECT, INSERT, UPDATE ON public.commercial_orders TO commercial_owner_runtime").Error)
	port := 5432
	if u.Port() != "" {
		port, err = strconv.Atoi(u.Port())
		require.NoError(t, err)
	}
	password, _ := u.User.Password()
	manifest := filepath.Join(t.TempDir(), "commercial-owner.json")
	body, err := json.Marshal(schemaOwnerManifest{Host: u.Hostname(), Port: port, User: u.User.Username(), Password: password, Database: name, MaxConnections: 2, MaxIdleConnections: 1})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(manifest, body, 0600))
	return storageAdditiveFixture{db: db, manifest: manifest}
}

func storageOriginalState(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var out []string
	for _, table := range []string{"commercial_offers", "commercial_quotes", "commercial_orders", "commercial_order_items", "retained_unrelated_fact"} {
		var value string
		require.NoError(t, db.Raw("SELECT coalesce(jsonb_agg(to_jsonb(t) - ARRAY['storage_unit_bytes','storage_duration_seconds','storage_activation_receipt'] ORDER BY to_jsonb(t)::text), '[]')::text FROM public."+table+" t").Scan(&value).Error)
		out = append(out, value)
	}
	var relations, roles, databaseACL, originalColumns, constraints, indexes string
	require.NoError(t, db.Raw("SELECT coalesce(jsonb_agg(jsonb_build_array(c.relname,c.relkind,c.relacl) ORDER BY c.relname), '[]')::text FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname NOT LIKE 'storage_capacity_%' AND c.relname NOT LIKE 'idx_storage_capacity_%' AND c.relname <> 'storage_write_identity'").Scan(&relations).Error)
	require.NoError(t, db.Raw("SELECT to_jsonb(r)::text FROM pg_roles r WHERE rolname='commercial_owner_runtime'").Scan(&roles).Error)
	require.NoError(t, db.Raw("SELECT coalesce(datacl::text,'') FROM pg_database WHERE datname=current_database()").Scan(&databaseACL).Error)
	require.NoError(t, db.Raw("SELECT coalesce(jsonb_agg(jsonb_build_array(table_name,column_name,ordinal_position,data_type,character_maximum_length,is_nullable,column_default) ORDER BY table_name,ordinal_position),'[]')::text FROM information_schema.columns WHERE table_schema='public' AND table_name NOT LIKE 'storage_capacity_%' AND column_name NOT IN ('storage_unit_bytes','storage_duration_seconds','storage_activation_receipt')").Scan(&originalColumns).Error)
	require.NoError(t, db.Raw("SELECT coalesce(jsonb_agg(jsonb_build_array(t.relname,c.conname,c.contype,pg_get_constraintdef(c.oid),c.convalidated,c.condeferrable,c.condeferred) ORDER BY t.relname,c.conname),'[]')::text FROM pg_constraint c JOIN pg_class t ON t.oid=c.conrelid JOIN pg_namespace n ON n.oid=t.relnamespace WHERE n.nspname='public' AND t.relname NOT LIKE 'storage_capacity_%'").Scan(&constraints).Error)
	require.NoError(t, db.Raw("SELECT coalesce(jsonb_agg(jsonb_build_array(t.relname,x.relname,pg_get_indexdef(i.indexrelid),i.indisunique,i.indisvalid,i.indisready) ORDER BY t.relname,x.relname),'[]')::text FROM pg_index i JOIN pg_class t ON t.oid=i.indrelid JOIN pg_class x ON x.oid=i.indexrelid JOIN pg_namespace n ON n.oid=t.relnamespace WHERE n.nspname='public' AND t.relname NOT LIKE 'storage_capacity_%'").Scan(&indexes).Error)
	return append(out, relations, roles, databaseACL, originalColumns, constraints, indexes)
}

func runStorageAdditiveCLI(t *testing.T, f storageAdditiveFixture, extra ...string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	args := append([]string{"run", ".", "-config", f.manifest, "-storage-disabled-additive-only"}, extra...)
	cmd := exec.CommandContext(ctx, "go", args...)
	wire, err := cmd.CombinedOutput()
	return string(wire), err
}

func TestDisabledStorageAdditiveCLIKeepsOriginalFactsAndACL(t *testing.T) {
	f := newStorageAdditiveFixture(t)
	require.NoError(t, f.db.Exec("ALTER TABLE public.commercial_offers ADD CONSTRAINT kept_price_check CHECK (unit_price_minor >= 0); ALTER TABLE public.commercial_quotes ADD CONSTRAINT kept_quantity_check CHECK (resource_quantity > 0); CREATE INDEX kept_price_lookup ON public.commercial_offers (unit_price_minor)").Error)
	before := storageOriginalState(t, f.db)
	wire, err := runStorageAdditiveCLI(t, f)
	require.NoError(t, err, wire)
	require.Equal(t, before, storageOriginalState(t, f.db), "only approved storage schema and SELECT may change")
	for _, table := range capacityTables {
		var n int64
		require.NoError(t, f.db.Table("public."+table).Count(&n).Error)
		require.Zero(t, n)
		var read, write bool
		require.NoError(t, f.db.Raw("SELECT has_table_privilege('commercial_owner_runtime',?,'SELECT'), has_table_privilege('commercial_owner_runtime',?,'INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')", "public."+table, "public."+table).Row().Scan(&read, &write))
		require.True(t, read)
		require.False(t, write)
		var columnWrite bool
		require.NoError(t, f.db.Raw("SELECT has_any_column_privilege('commercial_owner_runtime',?,'INSERT,UPDATE,REFERENCES')", "public."+table).Scan(&columnWrite).Error)
		require.False(t, columnWrite)
	}
	for table, columns := range additiveColumns {
		for _, column := range columns {
			require.True(t, f.db.Migrator().HasColumn(table, column))
		}
	}
	require.NoError(t, f.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE commercial_owner_runtime").Error; err != nil {
			return err
		}
		store, err := commercialstore.New(tx)
		if err != nil {
			return err
		}
		quote, err := store.CreateQuote(context.Background(), billing.QuoteRequest{OrganizationID: "fixture-org", OfferID: "kept-price", Quantity: 2})
		if err != nil {
			return err
		}
		require.EqualValues(t, 14, quote.TotalMinor)
		_, err = store.CreatePendingResourceOrder(context.Background(), billing.CreateResourceOrderRequest{OrganizationID: "fixture-org", ActorID: "fixture-actor", QuoteID: quote.QuoteID, IdempotencyKey: "after-schema"}, quote)
		return err
	}), "ordinary quote/order writes must work with the preserved runtime grants")
	before = storageOriginalState(t, f.db)
	wire, err = runStorageAdditiveCLI(t, f)
	require.NoError(t, err, wire)
	require.Equal(t, before, storageOriginalState(t, f.db), "replay must not reinstall catalog or other schema")
	var available bool
	require.NoError(t, f.db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec("SET LOCAL ROLE commercial_owner_runtime").Error; e != nil {
			return e
		}
		available = commercialstore.VerifyStorageCapacityDisabled(context.Background(), tx) == nil
		return nil
	}))
	require.True(t, available)
}

func storageSchemaState(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var out []string
	for _, table := range capacityTables {
		if !db.Migrator().HasTable(table) {
			out = append(out, "absent:"+table)
			continue
		}
		var rows, columns, acl, columnACL, constraints, indexes string
		require.NoError(t, db.Raw("SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text),'[]')::text FROM public."+table+" t").Scan(&rows).Error)
		require.NoError(t, db.Raw("SELECT jsonb_agg(jsonb_build_array(column_name,data_type,character_maximum_length,is_nullable,column_default) ORDER BY ordinal_position)::text FROM information_schema.columns WHERE table_schema='public' AND table_name=?", table).Scan(&columns).Error)
		require.NoError(t, db.Raw("SELECT coalesce(relacl::text,'') FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND relname=?", table).Scan(&acl).Error)
		require.NoError(t, db.Raw("SELECT coalesce(jsonb_agg(jsonb_build_array(a.attname,a.attacl) ORDER BY a.attnum),'[]')::text FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname=? AND a.attnum>0 AND NOT a.attisdropped", table).Scan(&columnACL).Error)
		require.NoError(t, db.Raw("SELECT coalesce(jsonb_agg(jsonb_build_array(c.conname,c.contype,pg_get_constraintdef(c.oid),c.convalidated,c.condeferrable,c.condeferred) ORDER BY c.conname),'[]')::text FROM pg_constraint c JOIN pg_class t ON t.oid=c.conrelid JOIN pg_namespace n ON n.oid=t.relnamespace WHERE n.nspname='public' AND t.relname=?", table).Scan(&constraints).Error)
		require.NoError(t, db.Raw("SELECT coalesce(jsonb_agg(jsonb_build_array(x.relname,pg_get_indexdef(i.indexrelid),i.indisunique,i.indisvalid,i.indisready) ORDER BY x.relname),'[]')::text FROM pg_index i JOIN pg_class t ON t.oid=i.indrelid JOIN pg_class x ON x.oid=i.indexrelid JOIN pg_namespace n ON n.oid=t.relnamespace WHERE n.nspname='public' AND t.relname=?", table).Scan(&indexes).Error)
		out = append(out, table, rows, columns, acl, columnACL, constraints, indexes)
	}
	for table, columns := range additiveColumns {
		for _, column := range columns {
			var definition string
			require.NoError(t, db.Raw("SELECT coalesce((SELECT jsonb_build_array(data_type,character_maximum_length,is_nullable,column_default)::text FROM information_schema.columns WHERE table_schema='public' AND table_name=? AND column_name=?),'absent')", table, column).Scan(&definition).Error)
			out = append(out, table+"."+column+":"+definition)
		}
	}
	// Map iteration must not determine the observable pre/post comparison.
	slices.Sort(out)
	return out
}

func TestDisabledStorageAdditiveCLIRejectsUnsafeExistingStateWithoutChanges(t *testing.T) {
	for _, scenario := range []string{"capacity fact", "capacity purchase", "partial tables", "changed capacity shape", "changed capacity default", "partial capacity index", "changed commercial column", "inherited write permission", "column write permission"} {
		t.Run(scenario, func(t *testing.T) {
			f := newStorageAdditiveFixture(t)
			switch scenario {
			case "capacity fact":
				require.NoError(t, capacitystore.AutoMigrate(f.db))
				require.NoError(t, f.db.Exec("INSERT INTO storage_capacity_accounts (organization_id,occupied,reserved) VALUES ('retained-capacity',0,0)").Error)
			case "capacity purchase":
				require.NoError(t, f.db.Exec("UPDATE commercial_orders SET kind=?,status=?", billing.OrderStorageCapacityPurchase, billing.OrderCancelled).Error)
			case "partial tables":
				require.NoError(t, f.db.Exec("CREATE TABLE storage_capacity_accounts (organization_id varchar(128) PRIMARY KEY, occupied bigint, reserved bigint)").Error)
			case "changed capacity shape":
				require.NoError(t, capacitystore.AutoMigrate(f.db))
				require.NoError(t, f.db.Exec("ALTER TABLE storage_capacity_namespaces ALTER COLUMN prefix TYPE varchar(64)").Error)
			case "changed capacity default":
				require.NoError(t, capacitystore.AutoMigrate(f.db))
				require.NoError(t, f.db.Exec("ALTER TABLE storage_capacity_namespaces ALTER COLUMN prefix SET DEFAULT 'unexpected'").Error)
			case "partial capacity index":
				require.NoError(t, capacitystore.AutoMigrate(f.db))
				require.NoError(t, f.db.Exec("DROP INDEX storage_write_identity; CREATE UNIQUE INDEX storage_write_identity ON storage_capacity_reservations (organization_id, owner, operation_id) WHERE organization_id <> 'excluded'").Error)
			case "changed commercial column":
				require.NoError(t, f.db.Exec("ALTER TABLE commercial_offers ADD COLUMN storage_unit_bytes text NOT NULL DEFAULT 'unchanged'").Error)
			case "inherited write permission":
				require.NoError(t, f.db.Exec("ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT INSERT ON TABLES TO commercial_owner_runtime").Error)
			case "column write permission":
				require.NoError(t, capacitystore.AutoMigrate(f.db))
				require.NoError(t, f.db.Exec("GRANT INSERT (organization_id) ON public.storage_capacity_accounts TO commercial_owner_runtime").Error)
			}
			before, schemaBefore := storageOriginalState(t, f.db), storageSchemaState(t, f.db)
			var defaultsBefore, defaultsAfter string
			require.NoError(t, f.db.Raw("SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.oid),'[]')::text FROM pg_default_acl d").Scan(&defaultsBefore).Error)
			wire, err := runStorageAdditiveCLI(t, f)
			require.Error(t, err, wire)
			require.Equal(t, before, storageOriginalState(t, f.db))
			require.Equal(t, schemaBefore, storageSchemaState(t, f.db), "DDL and SELECT must roll back together")
			require.NoError(t, f.db.Raw("SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.oid),'[]')::text FROM pg_default_acl d").Scan(&defaultsAfter).Error)
			require.Equal(t, defaultsBefore, defaultsAfter)
		})
	}
}

func TestDisabledStorageAdditiveCLIRejectsUnexpectedConstraints(t *testing.T) {
	assertRejected := func(t *testing.T, f storageAdditiveFixture) {
		t.Helper()
		before, schemaBefore := storageOriginalState(t, f.db), storageSchemaState(t, f.db)
		wire, err := runStorageAdditiveCLI(t, f)
		require.Error(t, err, wire)
		require.Equal(t, before, storageOriginalState(t, f.db))
		require.Equal(t, schemaBefore, storageSchemaState(t, f.db), "constraints, DDL and SELECT must remain unchanged on refusal")
	}
	for _, table := range capacityTables {
		t.Run(table, func(t *testing.T) {
			f := newStorageAdditiveFixture(t)
			require.NoError(t, capacitystore.AutoMigrate(f.db))
			require.NoError(t, f.db.Exec("ALTER TABLE public."+table+" ADD CONSTRAINT changed_capacity_check CHECK (false)").Error)
			assertRejected(t, f)
		})
	}
	for table, columns := range additiveColumns {
		for _, column := range columns {
			t.Run(table+"."+column, func(t *testing.T) {
				f := newStorageAdditiveFixture(t)
				if column == "storage_activation_receipt" {
					require.NoError(t, f.db.Exec("ALTER TABLE public."+table+" ADD COLUMN "+column+" varchar(128) NOT NULL DEFAULT ''; ALTER TABLE public."+table+" ADD CONSTRAINT changed_storage_field_check CHECK (storage_activation_receipt <> '') NOT VALID").Error)
				} else {
					require.NoError(t, f.db.Exec("ALTER TABLE public."+table+" ADD COLUMN "+column+" bigint NOT NULL DEFAULT 0; ALTER TABLE public."+table+" ADD CONSTRAINT changed_storage_field_check CHECK ("+column+" <> 0) NOT VALID").Error)
				}
				assertRejected(t, f)
			})
		}
	}
	for _, kind := range []string{"unique", "foreign key", "expression exclusion"} {
		t.Run(kind, func(t *testing.T) {
			f := newStorageAdditiveFixture(t)
			require.NoError(t, f.db.Exec("ALTER TABLE public.commercial_quotes ADD COLUMN storage_unit_bytes bigint NOT NULL DEFAULT 0").Error)
			switch kind {
			case "unique":
				require.NoError(t, f.db.Exec("ALTER TABLE public.commercial_quotes ADD CONSTRAINT changed_storage_unique UNIQUE (storage_unit_bytes)").Error)
			case "foreign key":
				require.NoError(t, f.db.Exec("CREATE TABLE public.retained_field_reference (bytes bigint PRIMARY KEY); INSERT INTO public.retained_field_reference VALUES (0); ALTER TABLE public.commercial_quotes ADD CONSTRAINT changed_storage_fk FOREIGN KEY (storage_unit_bytes) REFERENCES public.retained_field_reference(bytes)").Error)
			case "expression exclusion":
				require.NoError(t, f.db.Exec("ALTER TABLE public.commercial_quotes ADD CONSTRAINT changed_storage_exclusion EXCLUDE USING btree ((storage_unit_bytes + 0) WITH =)").Error)
			}
			assertRejected(t, f)
		})
	}
}

func TestDisabledStorageAdditiveCLIRejectsMoneyConfigBeforeInstallation(t *testing.T) {
	f := newStorageAdditiveFixture(t)
	before, schemaBefore := storageOriginalState(t, f.db), storageSchemaState(t, f.db)
	for _, moneyConfig := range []string{filepath.Join(t.TempDir(), "must-not-open-money.json"), ""} {
		wire, err := runStorageAdditiveCLI(t, f, "-money-config", moneyConfig)
		require.Error(t, err)
		require.Contains(t, wire, "-money-config")
		require.Equal(t, before, storageOriginalState(t, f.db))
		require.Equal(t, schemaBefore, storageSchemaState(t, f.db))
	}
}

func TestDisabledStorageAdditiveCLIRejectsUnexpectedIndexes(t *testing.T) {
	assertRejected := func(t *testing.T, f storageAdditiveFixture) {
		t.Helper()
		before, schemaBefore := storageOriginalState(t, f.db), storageSchemaState(t, f.db)
		wire, err := runStorageAdditiveCLI(t, f)
		require.Error(t, err, wire)
		require.Equal(t, before, storageOriginalState(t, f.db))
		require.Equal(t, schemaBefore, storageSchemaState(t, f.db), "indexes, DDL and SELECT must remain unchanged on refusal")
	}
	for _, table := range capacityTables {
		t.Run(table, func(t *testing.T) {
			f := newStorageAdditiveFixture(t)
			require.NoError(t, capacitystore.AutoMigrate(f.db))
			require.NoError(t, f.db.Exec("CREATE UNIQUE INDEX changed_capacity_standalone ON public."+table+" ((true))").Error)
			assertRejected(t, f)
		})
	}
	for table, columns := range additiveColumns {
		for _, column := range columns {
			t.Run(table+"."+column, func(t *testing.T) {
				f := newStorageAdditiveFixture(t)
				if column == "storage_activation_receipt" {
					require.NoError(t, f.db.Exec("ALTER TABLE public."+table+" ADD COLUMN "+column+" varchar(128) NOT NULL DEFAULT ''; CREATE UNIQUE INDEX changed_field_standalone ON public."+table+" ((storage_activation_receipt || ''))").Error)
				} else {
					require.NoError(t, f.db.Exec("ALTER TABLE public."+table+" ADD COLUMN "+column+" bigint NOT NULL DEFAULT 0; CREATE UNIQUE INDEX changed_field_standalone ON public."+table+" (("+column+" + 0))").Error)
				}
				assertRejected(t, f)
			})
		}
	}
}
