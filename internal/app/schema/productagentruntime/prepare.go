// Package productagentruntime composes fixed native schema preparation.
// It owns no business data, runtime or migration registry.
package productagentruntime

import (
	"context"
	"errors"
	"time"

	agentstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/agent"
	channelstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/channelcredential"
	routestore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/routeconfig"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Result struct {
	Target   string   `json:"target"`
	Outcome  string   `json:"outcome"`
	Database string   `json:"database"`
	Tables   []string `json:"tables"`
	Roles    []string `json:"roles"`
}

func PrepareManagedModelChannels(ctx context.Context, db *gorm.DB) (Result, error) {
	return prepare(ctx, db, "managed-model-channels", "product_agent_owner", func(tx *gorm.DB, result *Result) error {
		if err := routestore.VerifyManagedChannelPrerequisites(ctx, tx); err != nil {
			return err
		}
		if err := routestore.InstallManagedChannels(ctx, tx); err != nil {
			return err
		}
		result.Outcome = "CREATED"
		result.Tables = []string{"ai_channel_registrations", "ai_enterprise_model_bindings", "ai_enterprise_model_binding_receipts"}
		result.Roles = []string{"product_agent_runtime", "title_credential_writer", "model_route_config_writer"}
		return nil
	})
}

func PrepareOwnedGRSaiProductImage(ctx context.Context, db *gorm.DB) (Result, error) {
	return prepareOwnedProductImage(ctx, db, "owned-grsai-product-image", channelstore.EnableGRSaiImageRegistration)
}

func PrepareOwnedToAPIsProductImage(ctx context.Context, db *gorm.DB) (Result, error) {
	return prepareOwnedProductImage(ctx, db, "owned-toapis-product-image", channelstore.EnableToAPIsImageRegistration)
}

func prepareOwnedProductImage(ctx context.Context, db *gorm.DB, target string, enable func(context.Context, *gorm.DB) (bool, error)) (Result, error) {
	return prepare(ctx, db, target, "product_agent_owner", func(tx *gorm.DB, result *Result) error {
		if !routestore.ManagedChannelSchemaInstalled(ctx, tx) {
			return errors.New("managed channels schema is unavailable")
		}
		updated, err := enable(ctx, tx)
		if err != nil {
			return err
		}
		result.Outcome = "VERIFIED"
		if updated {
			result.Outcome = "UPDATED"
		}
		result.Tables = []string{"ai_channel_registrations"}
		return nil
	})
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
		// The official bootstrap uses LOGIN owners for schema commands. Keep
		// that capability intact; serving and writer roles are checked separately.
		if err := tx.Raw(`SELECT NOT (rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls) FROM pg_catalog.pg_roles WHERE rolname=?`, owner).Scan(&safeOwner).Error; err != nil {
			return err
		}
		if !safeOwner {
			return errors.New("fixed schema owner must be a bounded role")
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

// PrepareRun creates only an entirely absent current four-table group. An
// existing group must already match schema and grants; it is never repaired.
func PrepareRun(ctx context.Context, db *gorm.DB) (Result, error) {
	return prepare(ctx, db, "run", "product_agent_owner", func(tx *gorm.DB, result *Result) error {
		result.Roles = []string{"product_agent_runtime", "title_credential_writer", "ai_channel_secret_reader"}
		for _, role := range result.Roles {
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
		if _, err = verifyGrants(tx, grants[:1]); err != nil {
			return err
		}
		if present == 0 {
			if err = agentstore.EnsureRuntimeSchema(tx.Statement.Context, tx); err != nil {
				return err
			}
			if err = channelstore.EnsureSchema(tx.Statement.Context, tx); err != nil {
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
		if err = channelstore.VerifyMetadataProjection(tx.Statement.Context, tx); err != nil {
			return err
		}
		ready, err := verifyGrants(tx, grants)
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
		ready, err = verifyGrants(tx, grants)
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
		ready, err := verifyGrants(tx, grants)
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
		ready, err = verifyGrants(tx, grants)
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
