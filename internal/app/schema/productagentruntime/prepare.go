// Package productagentruntime composes only the two fixed native preparation
// transactions. It owns no business data, runtime or migration registry.
package productagentruntime

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/integration/openai"
	agentstore "task-processor/internal/integration/persistence/agent"
)

type Result struct {
	Target   string   `json:"target"`
	Outcome  string   `json:"outcome"`
	Database string   `json:"database"`
	Tables   []string `json:"tables"`
	Roles    []string `json:"roles"`
}

func prepare(ctx context.Context, db *gorm.DB, target, owner string, run func(*gorm.DB, *Result) error) (Result, error) {
	if ctx == nil || db == nil || db.Statement == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return Result{}, errors.New("preparation requires PostgreSQL and a context")
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	result := Result{Target: target}
	err := db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var actual struct{ Name, Owner string }
		if err := tx.Raw(`SELECT current_database() AS name,pg_catalog.pg_get_userbyid(datdba) AS owner FROM pg_catalog.pg_database WHERE datname=current_database()`).Scan(&actual).Error; err != nil {
			return err
		}
		if actual.Owner != owner {
			return errors.New("database owner differs from fixed target")
		}
		var safeOwner bool
		if err := tx.Raw(`SELECT NOT (rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls) FROM pg_catalog.pg_roles WHERE rolname=?`, owner).Scan(&safeOwner).Error; err != nil {
			return err
		}
		if !safeOwner {
			return errors.New("fixed schema owner must be a bounded NOLOGIN role")
		}
		result.Database = actual.Name
		if err := tx.Exec("SET LOCAL ROLE " + quote(owner)).Error; err != nil {
			return errors.New("schema owner role unavailable")
		}
		if err := tx.Exec("SET LOCAL search_path=public,pg_catalog").Error; err != nil {
			return err
		}
		if err := tx.Exec("SET LOCAL lock_timeout='5s'").Error; err != nil {
			return err
		}
		if err := tx.Exec("SET LOCAL statement_timeout='40s'").Error; err != nil {
			return err
		}
		return run(tx, &result)
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

// PrepareRun creates only an entirely absent current three-table group. An
// existing group must already match schema and grants; it is never repaired.
func PrepareRun(ctx context.Context, db *gorm.DB) (Result, error) {
	return prepare(ctx, db, "run", "product_agent_owner", func(tx *gorm.DB, result *Result) error {
		result.Roles = []string{"product_agent_runtime", "title_credential_writer"}
		for _, role := range []string{"product_agent_runtime", "title_credential_writer"} {
			if err := verifyRole(tx, role); err != nil {
				return err
			}
		}
		ledger, err := invocationContract(tx)
		if err != nil {
			return err
		}
		if err = verifyRelation(tx, ledger); err != nil {
			return err
		}
		specs, err := runContracts(tx)
		if err != nil {
			return err
		}
		present := 0
		for _, spec := range specs {
			result.Tables = append(result.Tables, spec.Name)
			var exists bool
			if err = tx.Raw("SELECT pg_catalog.to_regclass(?) IS NOT NULL", "public."+spec.Name).Scan(&exists).Error; err != nil {
				return err
			}
			if exists {
				present++
			}
		}
		if present != 0 && present != len(specs) {
			return errors.New("runtime target group is partial; no changes made")
		}
		result.Tables = append(result.Tables, ledger.Name)
		grants := runGrants(ledger)
		if _, err = verifyGrants(tx, grants[:1], false); err != nil {
			return err
		}
		if present == 0 {
			if err = agentstore.EnsureRuntimeSchema(tx.Statement.Context, tx); err != nil {
				return err
			}
			if err = openai.EnsureCredentialSchema(tx.Statement.Context, tx); err != nil {
				return err
			}
		}
		for _, spec := range specs {
			if err = verifyRelation(tx, spec); err != nil {
				return err
			}
			if err = verifyPrivateACL(tx, spec.Name); err != nil {
				return err
			}
		}
		if err = verifyCredentialSequence(tx); err != nil {
			return err
		}
		ready, err := verifyGrants(tx, grants, true)
		if err != nil {
			return err
		}
		if present != 0 {
			if !ready {
				return errors.New("existing runtime group grants differ; no repair performed")
			}
			result.Outcome = "ALREADY_READY"
			return nil
		}
		if err = applyGrants(tx, grants); err != nil {
			return err
		}
		if err = tx.Exec("GRANT USAGE ON SEQUENCE public.ai_client_credentials_id_seq TO title_credential_writer").Error; err != nil {
			return err
		}
		ready, err = verifyGrants(tx, grants, true)
		if err != nil {
			return err
		}
		if !ready {
			return errors.New("runtime bounded grants incomplete")
		}
		result.Outcome = "CREATED"
		return nil
	})
}

// PrepareReview qualifies the six existing definitions and a fresh role. An
// exact complete grant set is a read-only no-op; partial sets are not repaired.
func PrepareReview(ctx context.Context, db *gorm.DB) (Result, error) {
	return prepare(ctx, db, "review", "acquisition_owner", func(tx *gorm.DB, result *Result) error {
		result.Roles = []string{"product_review_runtime"}
		if err := verifyRole(tx, "product_review_runtime"); err != nil {
			return err
		}
		specs, err := reviewContracts(tx)
		if err != nil {
			return err
		}
		for _, spec := range specs {
			if err = verifyRelation(tx, spec); err != nil {
				return err
			}
			result.Tables = append(result.Tables, spec.Name)
		}
		grants := reviewGrants()
		ready, err := verifyGrants(tx, grants, false)
		if err != nil {
			return err
		}
		if ready {
			result.Outcome = "ALREADY_READY"
			return nil
		}
		var any bool
		if err = tx.Raw(`SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(c.relacl,pg_catalog.acldefault('r',c.relowner))) a WHERE n.nspname='public' AND c.relname IN ? AND a.grantee=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='product_review_runtime')) OR EXISTS(SELECT 1 FROM pg_catalog.pg_attribute c CROSS JOIN LATERAL pg_catalog.aclexplode(c.attacl) a WHERE c.attrelid IN (SELECT oid FROM pg_catalog.pg_class WHERE relnamespace='public'::regnamespace AND relname IN ?) AND a.grantee=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='product_review_runtime'))`, result.Tables, result.Tables).Scan(&any).Error; err != nil {
			return err
		}
		if any {
			return errors.New("review target grants are partial; no repair performed")
		}
		if err = applyGrants(tx, grants); err != nil {
			return err
		}
		ready, err = verifyGrants(tx, grants, false)
		if err != nil {
			return err
		}
		if !ready {
			return errors.New("review bounded grants incomplete")
		}
		result.Outcome = "GRANTED"
		return nil
	})
}
