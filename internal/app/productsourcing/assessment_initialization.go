package productsourcing

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	researchstore "task-processor/internal/integration/persistence/product/research"
	platformdatabase "task-processor/internal/platform/database"
)

// EnableResearchAssessments adds the approved table to a confirmed current
// Research installation. It never initializes, migrates or repairs base facts.
func EnableResearchAssessments(ctx context.Context, manifest, confirmedDatabase string) error {
	refused := errors.New("assessment installation refused; require private acquisition_owner manifest and exact current Research database confirmation")
	if ctx == nil {
		return refused
	}
	cfg, err := readAcquisitionInitManifest(manifest, confirmedDatabase, refused)
	if err != nil || !cfg.Collections || !cfg.SupplierReview || cfg.Database.User != "acquisition_owner" {
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
		if tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(current_database()||':source-acquisition-init',0))").Error != nil {
			return refused
		}
		var exact, runtimeSafe bool
		if tx.Raw(`SELECT current_database()=? AND current_user='acquisition_owner'
 AND d.datdba=r.oid AND NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
 FROM pg_database d JOIN pg_roles r ON r.rolname=current_user WHERE d.datname=current_database()`, confirmedDatabase).Scan(&exact).Error != nil || !exact {
			return refused
		}
		if tx.Raw(`SELECT NOT(rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls)
 AND NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=r.oid)
 FROM pg_roles r WHERE rolname=?`, researchstore.RuntimeRole).Scan(&runtimeSafe).Error != nil || !runtimeSafe {
			return refused
		}
		base := append(append([]string{}, researchstore.Tables...), "research_supplier_reviews")
		if verifyListingTableOwner(tx, base) != nil || researchstore.VerifySchema(ctx, tx) != nil || researchstore.VerifySupplierReviewOwnerSchema(ctx, tx) != nil {
			return refused
		}
		var present bool
		if tx.Raw("SELECT to_regclass('public.research_assessments') IS NOT NULL").Scan(&present).Error != nil {
			return refused
		}
		if !present {
			if researchstore.InstallAssessmentSchema(tx) != nil {
				return refused
			}
		}
		return researchstore.VerifyAssessmentOwnerInstallation(ctx, tx)
	})
	if err != nil {
		return refused
	}
	return nil
}
