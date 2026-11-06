package acquisition

import (
	"context"

	"gorm.io/gorm"
	"task-processor/internal/dataservice"
)

const sheinTrialAdmittedPrivileges = `,('data_acquisition_shein_us_trials','SELECT'),('data_acquisition_shein_us_trials','INSERT'),('data_acquisition_shein_us_trials','UPDATE'))`

const sheinTrialRoleAccess = `WITH subject AS (SELECT oid FROM pg_roles WHERE rolname=?)
SELECT rel IS NOT NULL AS present, COALESCE((SELECT has_table_privilege(oid,rel,'SELECT')
 OR has_any_column_privilege(oid,rel,'SELECT,INSERT,UPDATE,REFERENCES') FROM subject), false) AS granted
FROM to_regclass('public.data_acquisition_shein_us_trials') AS installed(rel)`

// GrantSHEINTrialLedgerPermissions is limited to the new ledger. The caller
// owns the existing Product initialization transaction and advisory lock.
func GrantSHEINTrialLedgerPermissions(ctx context.Context, tx *gorm.DB) error {
	if ctx == nil || tx == nil || tx.Dialector.Name() != "postgres" {
		return dataservice.ErrUnavailable
	}
	for _, statement := range []string{
		"GRANT SELECT ON public.data_acquisition_shein_us_trials TO data_services_runtime",
		"GRANT INSERT (approval_id,binding,claimed_at), UPDATE (request_id,credits) ON public.data_acquisition_shein_us_trials TO data_services_runtime",
	} {
		if tx.WithContext(ctx).Exec(statement).Error != nil {
			return dataservice.ErrUnavailable
		}
	}
	return nil
}

func verifySHEINTrialLedgerACL(ctx context.Context, db *gorm.DB, granted bool) error {
	var exact bool
	tableGrants, columnGrants := 0, 0
	if granted {
		tableGrants, columnGrants = 1, 5
	}
	err := db.WithContext(ctx).Raw(`WITH ledger AS (
 SELECT c.*, r.oid AS role_oid FROM pg_class c
 JOIN pg_namespace n ON n.oid=c.relnamespace CROSS JOIN pg_roles r
 WHERE n.nspname='public' AND c.relname='data_acquisition_shein_us_trials' AND r.rolname=?
), columns AS (
 SELECT a.*,l.relowner,l.role_oid FROM pg_attribute a JOIN ledger l ON l.oid=a.attrelid
 WHERE a.attnum>0 AND NOT a.attisdropped
), column_acl AS (
 SELECT c.attname,c.relowner,c.role_oid,p.* FROM columns c CROSS JOIN LATERAL aclexplode(c.attacl) p
)
SELECT l.relkind='r' AND l.relpersistence='p' AND NOT l.relrowsecurity AND NOT l.relforcerowsecurity AND l.relowner<>l.role_oid
 AND (SELECT array_agg(attname::text ORDER BY attnum) FROM columns)=ARRAY['approval_id','binding','claimed_at','request_id','credits']
 AND (SELECT array_agg(format_type(atttypid,atttypmod) ORDER BY attnum) FROM columns)=ARRAY['uuid','character(64)','timestamp with time zone','character varying(128)','integer']
 AND (SELECT array_agg(attnotnull ORDER BY attnum) FROM columns)=ARRAY[true,true,true,false,false]
 AND EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid=l.oid AND c.contype='p' AND c.conkey=ARRAY[1]::smallint[] AND c.convalidated AND NOT c.condeferrable)
 AND EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid=l.oid AND c.contype='u' AND c.conkey=ARRAY[4]::smallint[] AND c.convalidated AND NOT c.condeferrable)
 AND (SELECT count(*) FROM pg_constraint c WHERE c.conrelid=l.oid AND c.contype='c' AND c.convalidated
      AND c.conname IN ('shein_trial_binding','shein_trial_request_id','shein_trial_credits'))=3
 AND (SELECT count(*) FROM aclexplode(l.relacl) p WHERE p.grantee=l.role_oid)=?
 AND NOT EXISTS(SELECT 1 FROM aclexplode(l.relacl) p WHERE p.grantee<>l.relowner
      AND (p.grantee<>l.role_oid OR p.privilege_type<>'SELECT' OR p.is_grantable))
 AND (SELECT count(*) FROM column_acl p WHERE p.grantee=l.role_oid)=?
 AND (SELECT count(DISTINCT (p.attname,p.privilege_type)) FROM column_acl p WHERE p.grantee=l.role_oid)=?
 AND NOT EXISTS(SELECT 1 FROM column_acl p WHERE p.grantee<>p.relowner AND (
      p.grantee<>p.role_oid OR p.is_grantable OR NOT (
        (p.privilege_type='INSERT' AND p.attname IN ('approval_id','binding','claimed_at')) OR
        (p.privilege_type='UPDATE' AND p.attname IN ('request_id','credits')))))
FROM ledger l`, DataServicesRuntimeRole, tableGrants, columnGrants, columnGrants).Scan(&exact).Error
	if err != nil || !exact {
		return dataservice.ErrUnavailable
	}
	return nil
}
