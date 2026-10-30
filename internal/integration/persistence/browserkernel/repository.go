package kernelpersistence

import (
	"context"
	_ "embed"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	k "task-processor/internal/browserkernel"
)

//go:embed schema.sql
var schema string

const table = "browser_kernel.releases"

type Store struct{ db *gorm.DB }

func New(db *gorm.DB) (*Store, error) {
	if db == nil || db.Dialector.Name() != "postgres" {
		return nil, k.ErrUnavailable
	}
	return &Store{db}, nil
}

// InstallSchema is exclusively a fresh schema-owner operation, never startup.
func InstallSchema(db *gorm.DB) error {
	if _, e := New(db); e != nil {
		return e
	}
	var name string
	if e := db.Raw("SELECT current_database()").Scan(&name).Error; e != nil || name != "browser_kernel" {
		return k.ErrUnavailable
	}
	return db.Exec(schema).Error
}
func GrantRuntime(db *gorm.DB, role string) error {
	if role != "browser_kernel_runtime" {
		return k.ErrInvalid
	}
	if _, e := New(db); e != nil {
		return e
	}
	return db.Exec(`GRANT USAGE ON SCHEMA browser_kernel TO browser_kernel_runtime;
 GRANT SELECT, INSERT ON browser_kernel.releases TO browser_kernel_runtime;
 GRANT UPDATE(recommended,withdrawn_at,withdrawn_by,recommendation_updated_at,recommendation_updated_by) ON browser_kernel.releases TO browser_kernel_runtime`).Error
}

// VerifySchema reads catalog/ACL facts with the serving identity. It does not
// repair drift, grant permissions, install a schema, or fall back to memory.
func VerifySchema(ctx context.Context, db *gorm.DB) error {
	if _, e := New(db); e != nil {
		return e
	}
	db = db.WithContext(ctx)
	checks := []string{
		`SELECT current_database()='browser_kernel' AND current_user='browser_kernel_runtime' AND NOT (rolsuper OR rolcreaterole OR rolcreatedb OR rolreplication OR rolbypassrls) FROM pg_roles WHERE rolname=current_user`,
		`SELECT NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=(SELECT oid FROM pg_roles WHERE rolname=current_user))`,
		`SELECT NOT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname NOT IN ('pg_catalog','information_schema') AND nspname NOT LIKE 'pg_toast%' AND (nspowner=(SELECT oid FROM pg_roles WHERE rolname=current_user) OR has_schema_privilege(current_user,oid,'CREATE')))`,
		`SELECT EXISTS(SELECT 1 FROM pg_class WHERE oid=to_regclass('browser_kernel.releases') AND relkind='r' AND NOT relrowsecurity AND NOT relforcerowsecurity) AND has_schema_privilege(current_user,'browser_kernel','USAGE') AND has_table_privilege(current_user,'browser_kernel.releases','SELECT') AND has_table_privilege(current_user,'browser_kernel.releases','INSERT') AND NOT has_table_privilege(current_user,'browser_kernel.releases','UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')`,
		`SELECT NOT EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname NOT IN ('pg_catalog','information_schema') AND n.nspname NOT LIKE 'pg_toast%' AND c.relkind IN ('r','p','v','m','f','S') AND (c.relowner=(SELECT oid FROM pg_roles WHERE rolname=current_user) OR ((n.nspname<>'browser_kernel' OR c.relname<>'releases') AND (has_table_privilege(current_user,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') OR has_any_column_privilege(current_user,c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))))`,
		`SELECT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass('browser_kernel.releases') AND contype='p' AND convalidated AND conkey=ARRAY[(SELECT attnum FROM pg_attribute WHERE attrelid=to_regclass('browser_kernel.releases') AND attname='release_id')])`,
		`SELECT NOT EXISTS(SELECT 1 FROM pg_attribute a WHERE attrelid=to_regclass('browser_kernel.releases') AND attnum>0 AND NOT attisdropped AND (has_column_privilege(current_user,a.attrelid,a.attnum,'REFERENCES') OR has_column_privilege(current_user,a.attrelid,a.attnum,'UPDATE')<>(a.attname IN ('recommended','withdrawn_at','withdrawn_by','recommendation_updated_at','recommendation_updated_by'))))`,
		`SELECT NOT EXISTS(SELECT 1 FROM (SELECT x.* FROM pg_class c CROSS JOIN LATERAL aclexplode(c.relacl) x WHERE c.oid=to_regclass('browser_kernel.releases') UNION ALL SELECT x.* FROM pg_attribute a CROSS JOIN LATERAL aclexplode(a.attacl) x WHERE a.attrelid=to_regclass('browser_kernel.releases') UNION ALL SELECT x.* FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) x WHERE n.nspname='browser_kernel') grants WHERE is_grantable AND grantee IN (0,(SELECT oid FROM pg_roles WHERE rolname=current_user)))`,
	}
	for _, q := range checks {
		var valid bool
		if e := db.Raw(q).Scan(&valid).Error; e != nil || !valid {
			return k.ErrUnavailable
		}
	}
	// Exact physical columns keep required NOT NULL/UUID/integer facts explicit.
	expected := map[string]string{"release_id": "uuid", "version": "text", "os": "text", "arch": "text", "min_client_version": "text", "max_client_version_exclusive": "text", "archive_format": "text", "size": "bigint", "sha256": "text", "download_url": "text", "recommended": "boolean", "published_at": "timestamp with time zone", "published_by": "character varying(128)", "withdrawn_at": "timestamp with time zone", "withdrawn_by": "character varying(128)", "recommendation_updated_at": "timestamp with time zone", "recommendation_updated_by": "character varying(128)"}
	var columns []struct {
		Name, Type string
		Required   bool
	}
	if e := db.Raw(`SELECT attname AS name,format_type(atttypid,atttypmod) AS type,attnotnull AS required FROM pg_attribute WHERE attrelid=to_regclass('browser_kernel.releases') AND attnum>0 AND NOT attisdropped`).Scan(&columns).Error; e != nil || len(columns) != len(expected) {
		return k.ErrUnavailable
	}
	for _, c := range columns {
		optional := c.Name == "withdrawn_at" || c.Name == "withdrawn_by" || c.Name == "recommendation_updated_at" || c.Name == "recommendation_updated_by"
		if expected[c.Name] != c.Type || c.Required == optional {
			return k.ErrUnavailable
		}
	}
	return nil
}
func storageError(e error) error {
	if e == nil {
		return nil
	}
	for _, v := range []error{k.ErrNotFound, k.ErrConflict, k.ErrWithdrawn} {
		if errors.Is(e, v) {
			return v
		}
	}
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return k.ErrNotFound
	}
	return k.ErrUnavailable
}
func (s *Store) Candidates(ctx context.Context, os, arch, after string, limit int) ([]k.Release, error) {
	if limit < 1 || limit > 101 {
		return nil, k.ErrInvalid
	}
	q := s.db.WithContext(ctx).Table(table).Where("os=? AND arch=? AND withdrawn_at IS NULL", os, arch)
	if after != "" {
		q = q.Where("release_id>?::uuid", after)
	}
	var rows []k.Release
	e := q.Order("release_id ASC").Limit(limit).Find(&rows).Error
	return rows, storageError(e)
}
func (s *Store) Find(ctx context.Context, id string) (k.Release, error) {
	var r k.Release
	e := s.db.WithContext(ctx).Table(table).Where("release_id=?::uuid", id).Take(&r).Error
	return r, storageError(e)
}
func (s *Store) Publish(ctx context.Context, id string, p k.Publish, actor string) (k.Release, error) {
	var out k.Release
	e := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		e := tx.Exec(`INSERT INTO browser_kernel.releases(release_id,version,os,arch,min_client_version,max_client_version_exclusive,archive_format,size,sha256,download_url,published_by) VALUES(?::uuid,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(release_id) DO NOTHING`, id, p.Version, p.OS, p.Arch, p.MinClientVersion, p.MaxClientVersionExclusive, p.ArchiveFormat, p.Size, p.SHA256, p.DownloadURL, actor).Error
		if e != nil {
			return e
		}
		out, e = (&Store{tx}).Find(ctx, id)
		if e != nil {
			return e
		}
		if out.Publish != p {
			return k.ErrConflict
		}
		return nil
	})
	return out, storageError(e)
}
func (s *Store) mutate(ctx context.Context, id, actor string, recommend *bool) (k.Release, error) {
	var out k.Release
	e := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Table(table).Clauses(clause.Locking{Strength: "UPDATE"}).Where("release_id=?::uuid", id).Take(&out).Error; e != nil {
			return e
		}
		values := map[string]any{}
		if recommend == nil {
			if out.WithdrawnAt != nil {
				return nil
			}
			values = map[string]any{"withdrawn_at": gorm.Expr("clock_timestamp()"), "withdrawn_by": actor, "recommended": false}
		} else {
			if out.WithdrawnAt != nil {
				return k.ErrWithdrawn
			}
			if out.Recommended == *recommend {
				return nil
			}
			values = map[string]any{"recommended": *recommend, "recommendation_updated_at": gorm.Expr("clock_timestamp()"), "recommendation_updated_by": actor}
		}
		if e := tx.Table(table).Where("release_id=?::uuid", id).UpdateColumns(values).Error; e != nil {
			return e
		}
		var e error
		out, e = (&Store{tx}).Find(ctx, id)
		return e
	})
	return out, storageError(e)
}
func (s *Store) Withdraw(ctx context.Context, id, actor string) (k.Release, error) {
	return s.mutate(ctx, id, actor, nil)
}
func (s *Store) Recommend(ctx context.Context, id string, value bool, actor string) (k.Release, error) {
	return s.mutate(ctx, id, actor, &value)
}
