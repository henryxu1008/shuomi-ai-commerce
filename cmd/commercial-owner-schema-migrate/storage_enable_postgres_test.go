package main

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/commercial/billing"
	commercialstore "task-processor/internal/integration/persistence/commercialbilling"
	capacitystore "task-processor/internal/integration/persistence/storagecapacity"
)

func TestStorageEnableEmptyCLIOnlyInstallsCapacityGrantAndOriginalOffer(t *testing.T) {
	f := newStorageAdditiveFixture(t)
	require.NoError(t, migrateDisabledStorageOwner(context.Background(), f.db))
	before := storageOriginalState(t, f.db)
	out, err := exec.Command("go", "run", ".", "-config", f.manifest, "-storage-enable-empty-only").CombinedOutput()
	require.NoError(t, err, "%s", out)
	after := storageOriginalState(t, f.db)
	require.Equal(t, before[1:], after[1:], "existing non-offer facts and ACL must not change")
	store, err := commercialstore.New(f.db)
	require.NoError(t, err)
	offer, err := store.ReadOffer(context.Background(), billing.InitialStorageOffer().OfferID)
	require.NoError(t, err)
	require.Equal(t, billing.InitialStorageOffer(), offer)
	for _, table := range capacityTables {
		var read, insert, update, destroy bool
		require.NoError(t, f.db.Raw("SELECT has_table_privilege('commercial_owner_runtime',?,'SELECT'), has_table_privilege('commercial_owner_runtime',?,'INSERT'), has_table_privilege('commercial_owner_runtime',?,'UPDATE'), has_table_privilege('commercial_owner_runtime',?,'DELETE,TRUNCATE')", "public."+table, "public."+table, "public."+table, "public."+table).Row().Scan(&read, &insert, &update, &destroy))
		require.True(t, read && insert && update)
		require.False(t, destroy)
	}
}

func TestStorageEnableEmptyRejectsMissingSchemaOrDurableFactsWithoutChanges(t *testing.T) {
	scenarios := []string{"all_tables_missing", "one_table_missing", "capacity_fact", "capacity_order", "offer_conflict", "offer_failure", "extra_constraint"}
	for table, columns := range additiveColumns {
		for _, column := range columns {
			scenarios = append(scenarios, "missing_column:"+table+":"+column)
		}
	}
	for _, scenario := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			f := newStorageAdditiveFixture(t)
			require.NoError(t, migrateDisabledStorageOwner(context.Background(), f.db))
			switch {
			case scenario == "all_tables_missing":
				for _, table := range capacityTables {
					require.NoError(t, f.db.Exec("DROP TABLE public."+table).Error)
				}
			case scenario == "one_table_missing":
				require.NoError(t, f.db.Exec("DROP TABLE public.storage_capacity_namespaces").Error)
			case strings.HasPrefix(scenario, "missing_column:"):
				parts := strings.Split(scenario, ":")
				require.NoError(t, f.db.Exec("ALTER TABLE public."+parts[1]+" DROP COLUMN "+parts[2]).Error)
			case scenario == "capacity_fact":
				require.NoError(t, f.db.Exec("INSERT INTO public.storage_capacity_accounts(organization_id,occupied,reserved) VALUES ('original-org',0,0)").Error)
			case scenario == "capacity_order":
				require.NoError(t, f.db.Exec("UPDATE public.commercial_orders SET kind=?", billing.OrderStorageCapacityPurchase).Error)
			case scenario == "offer_conflict":
				require.NoError(t, f.db.Exec("UPDATE public.commercial_offers SET offer_id=?", billing.InitialStorageOffer().OfferID).Error)
			case scenario == "offer_failure":
				require.NoError(t, f.db.Exec("CREATE FUNCTION public.reject_storage_offer() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.product_kind='STORAGE_CAPACITY_PACK' THEN RAISE EXCEPTION 'synthetic offer refusal'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_storage_offer BEFORE INSERT ON public.commercial_offers FOR EACH ROW EXECUTE FUNCTION public.reject_storage_offer()").Error)
			case scenario == "extra_constraint":
				require.NoError(t, f.db.Exec("ALTER TABLE public.storage_capacity_accounts ADD CONSTRAINT retained_extra_check CHECK(occupied>=0)").Error)
			}
			before := storageSchemaState(t, f.db)
			original := storageOriginalState(t, f.db)
			require.Error(t, enableEmptyStorageOwner(context.Background(), f.db))
			require.Equal(t, before, storageSchemaState(t, f.db), "refusal must preserve schema, rows and ACL")
			require.Equal(t, original, storageOriginalState(t, f.db))
		})
	}
}

func TestStorageEnableEmptyReplayPreservesOwnerManagedOffer(t *testing.T) {
	f := newStorageAdditiveFixture(t)
	require.NoError(t, migrateDisabledStorageOwner(context.Background(), f.db))
	require.NoError(t, enableEmptyStorageOwner(context.Background(), f.db))
	require.NoError(t, f.db.Exec("UPDATE public.commercial_offers SET unit_price_minor=2700,status='DISABLED',pricing_version='owner-new-price' WHERE offer_id=?", billing.InitialStorageOffer().OfferID).Error)
	before := storageOriginalState(t, f.db)
	capacity := storageSchemaState(t, f.db)
	require.NoError(t, enableEmptyStorageOwner(context.Background(), f.db))
	require.Equal(t, before, storageOriginalState(t, f.db))
	require.Equal(t, capacity, storageSchemaState(t, f.db))
	require.NoError(t, capacitystore.VerifyDisabled(context.Background(), f.db))
}

func TestStorageEnableEmptyCLIRejectsOtherInputsBeforeOpeningConfig(t *testing.T) {
	for _, extra := range [][]string{{"-money-config", ""}, {"-money-config", "not-used.json"}, {"-storage-disabled-additive-only"}, {"-storage-disabled-additive-only=false"}, {"unexpected-position"}} {
		args := append([]string{"run", ".", "-config", "must-not-open.json", "-storage-enable-empty-only"}, extra...)
		out, err := exec.Command("go", args...).CombinedOutput()
		require.Error(t, err)
		require.Contains(t, string(out), "storage-enable-empty-only requires only -config", "%s", out)
	}
}
