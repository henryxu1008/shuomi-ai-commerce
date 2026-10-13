package storeobservations

import (
	"context"
	_ "embed"
	"errors"
	"gorm.io/gorm"
	"regexp"
	o "task-processor/internal/storecenter/observations"
	"time"
)

//go:embed schema.sql
var schemaSQL string
var roleName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)
var tables = map[string]bool{"commands": false, "syncs": true, "records": true, "heads": true}

func Install(ctx context.Context, db *gorm.DB) error {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" {
		return o.ErrUnavailable
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var retired bool
		if e := tx.Raw("SELECT to_regnamespace('shein_observations') IS NOT NULL").Scan(&retired).Error; e != nil {
			return e
		}
		if retired {
			return errors.New("retired SHEIN observation schema requires a fresh installation")
		}
		var exists bool
		if e := tx.Raw("SELECT to_regnamespace('store_observations') IS NOT NULL").Scan(&exists).Error; e != nil {
			return e
		}
		if exists {
			return errors.New("observations schema already exists; no migration is admitted")
		}
		return tx.Exec(schemaSQL).Error
	})
}

// Explicit fresh-instance provisioning only. The common Store installer and
// enabled preflight are owned by the runtime Writer, not a serving constructor.
func GrantRuntime(ctx context.Context, db *gorm.DB, role string) error {
	if !roleName.MatchString(role) {
		return o.ErrInvalid
	}
	if e := db.WithContext(ctx).Exec("GRANT USAGE ON SCHEMA store_observations TO " + role).Error; e != nil {
		return e
	}
	for table, mutable := range tables {
		privileges := "SELECT,INSERT"
		if mutable {
			privileges += ",UPDATE"
		}
		if e := db.WithContext(ctx).Exec("GRANT " + privileges + " ON store_observations." + table + " TO " + role).Error; e != nil {
			return e
		}
	}
	return nil
}
func VerifyRuntime(ctx context.Context, db *gorm.DB) error {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" {
		return o.ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var safe bool
	e := db.WithContext(ctx).Raw(`SELECT NOT(rolsuper OR rolcreaterole OR rolcreatedb OR rolreplication OR rolbypassrls) AND current_schemas(false)=ARRAY['public']::name[] AND has_schema_privilege(current_user,'store_observations','USAGE') AND NOT has_schema_privilege(current_user,'store_observations','CREATE') FROM pg_roles WHERE rolname=current_user`).Scan(&safe).Error
	if e != nil || !safe {
		return o.ErrUnavailable
	}
	for table, mutable := range tables {
		var valid bool
		e := db.WithContext(ctx).Raw(`SELECT has_table_privilege(current_user,?,'SELECT') AND has_table_privilege(current_user,?,'INSERT') AND has_table_privilege(current_user,?,'UPDATE')=? AND NOT has_table_privilege(current_user,?,'DELETE,TRUNCATE,REFERENCES,TRIGGER') AND NOT pg_has_role(current_user,(SELECT relowner FROM pg_class WHERE oid=?::regclass),'MEMBER')`, "store_observations."+table, "store_observations."+table, "store_observations."+table, mutable, "store_observations."+table, "store_observations."+table).Scan(&valid).Error
		if e != nil || !valid {
			return o.ErrUnavailable
		}
	}
	var count int64
	if e := db.WithContext(ctx).Raw(`SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='store_observations' AND c.relkind IN('r','p')`).Scan(&count).Error; e != nil || count != 4 {
		return o.ErrUnavailable
	}
	var variant bool
	if e := db.WithContext(ctx).Raw(`SELECT to_regnamespace('shein_observations') IS NULL AND EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='store_observations.records'::regclass AND conname='store_observation_record_variant' AND contype='c' AND convalidated AND pg_get_constraintdef(oid) LIKE '%temuProduct%')`).Scan(&variant).Error; e != nil || !variant {
		return o.ErrUnavailable
	}
	return nil
}
func NewRepository(ctx context.Context, db *gorm.DB) (*Repository, error) {
	if e := VerifyRuntime(ctx, db); e != nil {
		return nil, e
	}
	return &Repository{db: db}, nil
}

// Preparation must not attach a credential-free worker to retained work.
func (r *Repository) VerifyEmpty(ctx context.Context) error {
	if r == nil || r.db == nil || ctx == nil {
		return o.ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var exists bool
	err := r.db.WithContext(ctx).Raw(`SELECT EXISTS(SELECT 1 FROM store_observations.commands) OR EXISTS(SELECT 1 FROM store_observations.syncs) OR EXISTS(SELECT 1 FROM store_observations.records) OR EXISTS(SELECT 1 FROM store_observations.heads)`).Scan(&exists).Error
	if err != nil || exists {
		return o.ErrUnavailable
	}
	return nil
}
