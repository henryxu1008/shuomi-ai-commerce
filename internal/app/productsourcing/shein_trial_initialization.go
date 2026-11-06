package productsourcing

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	acquisitionstore "task-processor/internal/integration/persistence/product/acquisition"
	jobstore "task-processor/internal/integration/persistence/product/dataacquisition"
	platformdatabase "task-processor/internal/platform/database"
)

// EnableSHEINUSTrialLedger is explicit owner maintenance for the existing
// current Product schema. It does not repair that schema or open the trial.
func EnableSHEINUSTrialLedger(ctx context.Context, manifest, confirmedDatabase string) error {
	refused := errors.New("SHEIN trial ledger initialization refused; require the private Product owner manifest and current product_acquisition base")
	if ctx == nil || confirmedDatabase != "product_acquisition" {
		return refused
	}
	cfg, err := readAcquisitionInitManifest(manifest, confirmedDatabase, refused)
	if err != nil || !cfg.Collections || !cfg.DataServices || cfg.Database.User == acquisitionstore.DataServicesRuntimeRole {
		return refused
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	d := cfg.Database
	db, err := platformdatabase.OpenExistingWritableContext(ctx, &platformdatabase.Config{Host: d.Host, Port: d.Port, User: d.User, Password: d.Password, Database: d.Database, MaxConnections: 2, MaxIdleConnections: 2, ConnectionMaxLifetime: time.Minute})
	if err != nil {
		return refused
	}
	defer platformdatabase.Close(db)
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(current_database()||':source-acquisition-init',0))").Error != nil ||
			verifyListingTableOwner(tx, jobstore.Tables) != nil || acquisitionstore.VerifyDataServicesOwnerPermissions(ctx, tx, false) != nil {
			return refused
		}
		var exactTarget bool
		if tx.Raw("SELECT current_database()=? AND to_regclass('public.data_acquisition_shein_us_trials') IS NULL", confirmedDatabase).Scan(&exactTarget).Error != nil || !exactTarget {
			return refused
		}
		if jobstore.InstallSHEINUSTrialLedger(ctx, tx) != nil || acquisitionstore.GrantSHEINTrialLedgerPermissions(ctx, tx) != nil {
			return refused
		}
		return acquisitionstore.VerifyDataServicesOwnerPermissions(ctx, tx, true)
	})
	if err != nil {
		return refused
	}
	return nil
}
