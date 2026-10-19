package productsourcing

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	officialstore "task-processor/internal/integration/persistence/listing/official"
	prepstore "task-processor/internal/integration/persistence/listing/preparation"
	recordstore "task-processor/internal/integration/persistence/listing/record"
	submissionstore "task-processor/internal/integration/persistence/listing/submission"
	acquisitionstore "task-processor/internal/integration/persistence/product/acquisition"
	reviewstore "task-processor/internal/integration/persistence/product/review"
	platformdatabase "task-processor/internal/platform/database"
)

var supplyFeatureBaseTables = []string{
	"product_acquisition_operations", "product_acquisition_charge_intents", "product_source_publications", "product_source_publication_receipts", "product_snapshot_versions", "product_snapshot_heads",
	"product_collection_batches", "product_collection_items", "product_collection_operations", "listing_preparations", "listing_preparation_sources",
	"listing_completion_draft_records", "listing_completion_draft_heads", "listing_completion_draft_commands",
}

var supplyFeatureReviewTables = []string{"product_title_proposals", "product_title_operations"}

func supplyFeatureNewTables() []string {
	tables := append([]string{}, recordstore.Tables...)
	tables = append(tables, prepstore.OperationTables...)
	tables = append(tables, submissionstore.AttemptTable, submissionstore.TargetFenceTable)
	tables = append(tables, officialstore.OfficialTables...)
	return tables
}

// EnableSupplyChain is an explicit owner operation on the confirmed nonempty
// collections/completion installation. It never migrates or repairs that base.
func EnableSupplyChain(ctx context.Context, manifest, confirmedDatabase string) error {
	refused := errors.New("supply chain initialization refused; require a private current owner manifest and exact nonempty Product database confirmation")
	if ctx == nil {
		return refused
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cfg, err := readAcquisitionInitManifest(manifest, confirmedDatabase, refused)
	if err != nil || !cfg.Collections || !cfg.CompletionDrafts || !cfg.SupplyChain || cfg.AcquisitionPlans || cfg.ImageSets || cfg.SupplyMarket || cfg.POD || cfg.DataServices {
		return refused
	}
	d := cfg.Database
	db, err := platformdatabase.OpenExistingWritableContext(ctx, &platformdatabase.Config{Host: d.Host, Port: d.Port, User: d.User, Password: d.Password, Database: d.Database, MaxConnections: 2, MaxIdleConnections: 2, ConnectionMaxLifetime: time.Minute})
	if err != nil {
		return refused
	}
	defer platformdatabase.Close(db)
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(current_database()||':source-acquisition-init',0))").Error != nil || verifySupplyTableOwner(tx, supplyFeatureBaseTables) != nil || prepstore.VerifySourcePageIndex(ctx, tx) != nil {
			return refused
		}
		tables := supplyFeatureNewTables()
		var present int64
		if tx.Raw("SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname IN ?", tables).Scan(&present).Error != nil {
			return refused
		}
		var reviewPresent int64
		if tx.Raw("SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname IN ?", supplyFeatureReviewTables).Scan(&reviewPresent).Error != nil {
			return refused
		}
		switch reviewPresent {
		case 0:
			// Missing Review is valid only for a first Supply installation.
			if present != 0 {
				return refused
			}
		case int64(len(supplyFeatureReviewTables)):
			if verifySupplyTableOwner(tx, supplyFeatureReviewTables) != nil || reviewstore.VerifySchema(ctx, tx) != nil {
				return refused
			}
		default:
			return refused
		}
		switch present {
		case 0:
			if reviewPresent == 0 && reviewstore.InstallSchema(tx) != nil {
				return refused
			}
			if installSupplyExecutionSchema(tx) != nil || verifySupplyTableOwner(tx, tables) != nil || verifySupplyTableOwner(tx, supplyFeatureReviewTables) != nil || acquisitionstore.GrantSupplyChainPermissions(ctx, tx) != nil {
				return refused
			}
		case int64(len(tables)):
			if verifySupplyTableOwner(tx, tables) != nil {
				return refused
			}
		default:
			return refused
		}
		// Check the complete unchanged allow/deny contract before commit. This
		// also admits an existing Review SELECT without revoking/regranting it;
		// any old or new schema/permission drift rolls back all installation.
		capabilities := acquisitionstore.RuntimeCapabilities{Collections: true, CompletionDrafts: true, SupplyChain: true}
		if reviewstore.VerifySchema(ctx, tx) != nil || acquisitionstore.VerifyOwnerRuntimePermissions(ctx, tx, capabilities) != nil {
			return refused
		}
		return nil
	})
	if err != nil {
		return refused
	}
	return nil
}

func verifySupplyTableOwner(db *gorm.DB, tables []string) error {
	var owned int64
	err := db.Raw(`SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
 WHERE n.nspname='public' AND c.relname IN ? AND c.relkind='r' AND NOT c.relrowsecurity AND NOT c.relforcerowsecurity
 AND c.relowner=(SELECT oid FROM pg_roles WHERE rolname=current_user)`, tables).Scan(&owned).Error
	if err != nil || owned != int64(len(tables)) {
		return errors.New("supply installation requires unchanged current owner tables")
	}
	return nil
}
