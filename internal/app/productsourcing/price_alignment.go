package productsourcing

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	privateconfig "task-processor/internal/core/config"
	acquisitionstore "task-processor/internal/integration/persistence/product/acquisition"
	jobstore "task-processor/internal/integration/persistence/product/dataacquisition"
	platformdatabase "task-processor/internal/platform/database"
)

// AlignAcquisitionPriceSnapshot is explicit native owner maintenance, never a
// serving startup initializer. Unknown commit outcomes are checked by reentry.
func AlignAcquisitionPriceSnapshot(ctx context.Context, manifest, database string) error {
	refused := errors.New("price snapshot alignment refused; require private native owner manifest and exact product_acquisition confirmation")
	if ctx == nil || database != "product_acquisition" {
		return refused
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := privateconfig.VerifyPrivateFiles(ctx, []string{manifest}); err != nil {
		return fmt.Errorf("price snapshot alignment refused: %w", err)
	}
	cfg, err := readAcquisitionInitManifest(manifest, database, refused)
	if err != nil || !cfg.Collections || !cfg.DataServices || cfg.Database.User != "acquisition_owner" {
		return refused
	}
	d := cfg.Database
	db, err := platformdatabase.OpenExistingWritableContext(ctx, &platformdatabase.Config{Host: d.Host, Port: d.Port, User: d.User, Password: d.Password, Database: d.Database, MaxConnections: 1, MaxIdleConnections: 1, ConnectionMaxLifetime: time.Minute})
	if err != nil {
		return refused
	}
	defer platformdatabase.Close(db)
	return alignAcquisitionPriceDB(ctx, db, database, cfg)
}

func alignAcquisitionPriceDB(ctx context.Context, db *gorm.DB, database string, cfg acquisitionInitManifest) error {
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, ok := tx.Statement.ConnPool.(*sql.Tx); !ok {
			return errors.New("pinned native Product transaction required")
		}
		if err := tx.Exec(`SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='25s'; SELECT pg_advisory_xact_lock(hashtextextended(current_database()||':source-acquisition-init',0)); LOCK TABLE public.data_acquisition_jobs,public.data_acquisition_items IN ACCESS EXCLUSIVE MODE`).Error; err != nil {
			return err
		}
		if err := verifyPriceAlignmentBoundary(ctx, tx, database, cfg); err != nil {
			return err
		}
		before, err := pricePreservation(tx)
		if err != nil {
			return err
		}
		if err = jobstore.AlignPriceContractTx(ctx, tx); err != nil {
			return err
		}
		if err = verifyPriceAlignmentBoundary(ctx, tx, database, cfg); err != nil {
			return err
		}
		after, err := pricePreservation(tx)
		if err != nil {
			return err
		}
		if before != after {
			return errors.New("Product price alignment preservation differs")
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := verifyPriceAlignmentBoundary(ctx, tx, database, cfg); err != nil {
			return err
		}
		return jobstore.VerifySchema(ctx, tx)
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
}

func verifyPriceAlignmentBoundary(ctx context.Context, tx *gorm.DB, database string, cfg acquisitionInitManifest) error {
	var safe bool
	if err := tx.Raw(`SELECT current_database()=? AND current_schema()='public' AND current_user='acquisition_owner' AND session_user=current_user AND (SELECT pg_get_userbyid(datdba)='acquisition_owner' FROM pg_database WHERE datname=current_database()) AND EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolcanlogin AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls) AND NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member IN(SELECT oid FROM pg_roles WHERE rolname IN('acquisition_owner','data_services_runtime')) OR roleid IN(SELECT oid FROM pg_roles WHERE rolname IN('acquisition_owner','data_services_runtime')))`, database).Scan(&safe).Error; err != nil || !safe {
		return errors.New("native Product database/owner/role boundary differs")
	}
	if err := verifyListingTableOwner(tx, jobstore.Tables); err != nil {
		return err
	}
	if err := verifyPriceACL(tx); err != nil {
		return err
	}
	capabilities := acquisitionstore.RuntimeCapabilities{OfflineRecords: cfg.OfflineRecords, Collections: cfg.Collections, CompletionDrafts: cfg.CompletionDrafts, AcquisitionPlans: cfg.AcquisitionPlans, SupplyChain: cfg.SupplyChain, SupplyMarket: cfg.SupplyMarket, POD: cfg.POD, ImageSets: cfg.ImageSets}
	if err := acquisitionstore.VerifyOwnerRuntimePermissions(ctx, tx, capabilities); err != nil {
		return err
	}
	return acquisitionstore.VerifyPriceAlignmentPermissions(ctx, tx)
}

func verifyPriceACL(tx *gorm.DB) error {
	var safe bool
	err := tx.Raw(`SELECT NOT EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(c.relacl) a WHERE c.oid IN('public.data_acquisition_jobs'::regclass,'public.data_acquisition_items'::regclass) AND (pg_get_userbyid(a.grantee) NOT IN('acquisition_owner','data_services_runtime') OR pg_get_userbyid(a.grantor)<>'acquisition_owner' OR (pg_get_userbyid(a.grantee)='data_services_runtime' AND (a.is_grantable OR a.privilege_type NOT IN('SELECT','INSERT','UPDATE')))))
 AND NOT EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid IN('public.data_acquisition_jobs'::regclass,'public.data_acquisition_items'::regclass) AND attacl IS NOT NULL)
 AND NOT EXISTS(SELECT 1 FROM pg_default_acl WHERE defaclrole IN(SELECT oid FROM pg_roles WHERE rolname IN('acquisition_owner','data_services_runtime')) OR defaclnamespace='public'::regnamespace)
 AND NOT EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) a WHERE n.nspname='public' AND (pg_get_userbyid(a.grantee) NOT IN('pg_database_owner','acquisition_owner','data_services_runtime','source_acquisition_runtime','product_research_runtime','product_review_runtime') OR pg_get_userbyid(a.grantor) NOT IN('pg_database_owner','acquisition_owner') OR (pg_get_userbyid(a.grantee) NOT IN('pg_database_owner','acquisition_owner') AND (a.is_grantable OR a.privilege_type<>'USAGE'))))
 AND NOT EXISTS(SELECT 1 FROM pg_database d CROSS JOIN LATERAL aclexplode(d.datacl) a WHERE d.datname=current_database() AND (pg_get_userbyid(a.grantee) NOT IN('acquisition_owner','data_services_runtime','source_acquisition_runtime','product_research_runtime','product_review_runtime') OR (pg_get_userbyid(a.grantee)<>'acquisition_owner' AND (a.is_grantable OR a.privilege_type<>'CONNECT'))))`).Scan(&safe).Error
	if err != nil {
		return err
	}
	if !safe {
		return errors.New("Product price alignment refuses ACL drift")
	}
	return nil
}

func pricePreservation(tx *gorm.DB) (string, error) {
	var raw string
	err := tx.Raw(`SELECT jsonb_build_object(
 'jobs',(SELECT jsonb_agg(to_jsonb(j)-'price_snapshot' ORDER BY organization_id,actor_id,id) FROM public.data_acquisition_jobs j),
 'items',(SELECT jsonb_agg(to_jsonb(i) ORDER BY organization_id,actor_id,job_id,id) FROM public.data_acquisition_items i),
 'relations',(SELECT jsonb_agg(jsonb_build_array(oid,relname,relowner,relacl) ORDER BY oid) FROM pg_class WHERE relnamespace='public'::regnamespace),
 'schemas',(SELECT jsonb_agg(jsonb_build_array(oid,nspname,nspowner,nspacl) ORDER BY oid) FROM pg_namespace WHERE nspname='public'),
 'database',(SELECT jsonb_build_array(oid,datname,datdba,datacl) FROM pg_database WHERE datname=current_database()),
 'columns',(SELECT jsonb_agg(jsonb_build_array(attrelid,attnum,attname,attacl) ORDER BY attrelid,attnum) FROM pg_attribute WHERE attrelid IN(SELECT oid FROM pg_class WHERE relnamespace='public'::regnamespace) AND attnum>0 AND NOT attisdropped AND attname<>'price_snapshot'),
 'constraints',(SELECT jsonb_agg(to_jsonb(k) ORDER BY oid) FROM pg_constraint k WHERE connamespace='public'::regnamespace AND conname<>'data_acquisition_price_snapshot'),
 'defaults',(SELECT jsonb_agg(to_jsonb(d) ORDER BY oid) FROM pg_default_acl d),
 'roles',(SELECT jsonb_agg(to_jsonb(r) ORDER BY oid) FROM pg_roles r),
 'members',(SELECT jsonb_agg(to_jsonb(m) ORDER BY oid) FROM pg_auth_members m)
)::text`).Scan(&raw).Error
	return raw, err
}
