package routeconfigstore

import (
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	d "task-processor/internal/aicapability/routeconfig"
	"task-processor/internal/integration/openai"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

//go:embed schema.sql
var SchemaSQL string

func InstallSchema(ctx context.Context, db *gorm.DB) error {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" {
		return d.ErrUnavailable
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := openai.InstallCredentialMetadataProjection(ctx, tx); e != nil {
			return e
		}
		if e := tx.Exec(SchemaSQL).Error; e != nil {
			return e
		}
		return VerifySchema(ctx, tx, true)
	})
}
func VerifySchema(ctx context.Context, db *gorm.DB, imageOptions ...bool) error {
	if len(imageOptions) > 1 {
		return d.ErrUnavailable
	}
	requireOptions := len(imageOptions) == 1 && imageOptions[0]
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" {
		return d.ErrUnavailable
	}
	for _, q := range []string{
		`SELECT revision_id,organization_id,capability,parent_revision,schema_version,payload,payload_digest,actor_id,published_at,idempotency_key,request_fingerprint,rollback_from FROM public.ai_route_config_revisions LIMIT 0`,
		`SELECT organization_id,capability,revision_id FROM public.ai_route_config_current LIMIT 0`,
		`SELECT id,tenant_id,user_id,client_name,base_url,model,api_style,timeout_second,enabled,updated_at,key_present FROM public.ai_client_credential_bindings LIMIT 0`,
	} {
		rows, e := db.WithContext(ctx).Raw(q).Rows()
		if e != nil {
			return d.ErrUnavailable
		}
		_ = rows.Close()
	}
	if requireOptions {
		rows, e := db.WithContext(ctx).Raw(`SELECT organization_id,capability,binding_id,revision_id FROM public.ai_route_config_options LIMIT 0`).Rows()
		if e != nil {
			return d.ErrUnavailable
		}
		_ = rows.Close()
	}
	// A partial/drifted schema must not silently bypass scoped CAS or history
	// identity. Installation remains the only DDL path; startup only inspects.
	for _, constraint := range []struct{ table, name, definition string }{
		{"ai_route_config_revisions", "ai_route_config_revision_scope", "PRIMARY KEY (organization_id, capability, revision_id)"},
		{"ai_route_config_revisions", "ai_route_config_request_scope", "UNIQUE (organization_id, capability, idempotency_key)"},
		{"ai_route_config_current", "ai_route_config_current_scope", "PRIMARY KEY (organization_id, capability)"},
		{"ai_route_config_current", "ai_route_config_current_revision", "FOREIGN KEY (organization_id, capability, revision_id) REFERENCES ai_route_config_revisions(organization_id, capability, revision_id)"},
		{"ai_route_config_options", "ai_route_config_option_scope", "PRIMARY KEY (organization_id, capability, binding_id)"},
		{"ai_route_config_options", "ai_route_config_option_revision", "FOREIGN KEY (organization_id, capability, revision_id) REFERENCES ai_route_config_revisions(organization_id, capability, revision_id)"},
	} {
		if constraint.table == "ai_route_config_options" && !requireOptions {
			continue
		}
		var definition string
		e := db.WithContext(ctx).Raw(`SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid=to_regclass(?) AND conname=? AND convalidated`, "public."+constraint.table, constraint.name).Scan(&definition).Error
		if e != nil || definition != constraint.definition {
			return d.ErrUnavailable
		}
	}
	return nil
}

func VerifyReader(ctx context.Context, db *gorm.DB, imageOptions ...bool) error {
	if e := VerifySchema(ctx, db, imageOptions...); e != nil {
		return e
	}
	var writes bool
	e := db.WithContext(ctx).Raw(`SELECT has_table_privilege(current_user,'public.ai_route_config_revisions','INSERT,UPDATE,DELETE,TRUNCATE') OR has_table_privilege(current_user,'public.ai_route_config_current','INSERT,UPDATE,DELETE,TRUNCATE')`).Scan(&writes).Error
	if len(imageOptions) == 1 && imageOptions[0] && e == nil {
		var extra bool
		e = db.WithContext(ctx).Raw(`SELECT has_any_column_privilege(current_user,'public.ai_route_config_options','INSERT,UPDATE,REFERENCES') OR has_table_privilege(current_user,'public.ai_route_config_options','DELETE,TRUNCATE,TRIGGER')`).Scan(&extra).Error
		writes = writes || extra
	}
	if e != nil || writes {
		return d.ErrUnavailable
	}
	return nil
}

// VerifyWriter rejects serving/owner pools and credential table write/read-key
// privileges. The metadata view is the only credential grant needed here.
func VerifyWriter(ctx context.Context, db *gorm.DB, imageOptions ...bool) error {
	if e := VerifySchema(ctx, db, imageOptions...); e != nil {
		return e
	}
	var ok bool
	e := db.WithContext(ctx).Raw(`SELECT NOT (r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolbypassrls OR r.rolreplication)
 AND current_user<>pg_get_userbyid(c.relowner)
 AND has_table_privilege(current_user,'public.ai_route_config_revisions','INSERT')
 AND NOT has_table_privilege(current_user,'public.ai_route_config_revisions','UPDATE')
 AND NOT has_table_privilege(current_user,'public.ai_route_config_revisions','DELETE')
 AND NOT has_table_privilege(current_user,'public.ai_route_config_revisions','TRUNCATE')
 AND has_table_privilege(current_user,'public.ai_route_config_current','INSERT')
 AND has_table_privilege(current_user,'public.ai_route_config_current','UPDATE')
 AND NOT has_table_privilege(current_user,'public.ai_route_config_current','DELETE')
 AND NOT has_table_privilege(current_user,'public.ai_route_config_current','TRUNCATE')
 AND NOT has_column_privilege(current_user,'public.ai_client_credentials','api_key','SELECT')
 AND NOT has_any_column_privilege(current_user,'public.ai_client_credentials','UPDATE')
 AND NOT has_table_privilege(current_user,'public.ai_client_credentials','INSERT')
 AND NOT has_table_privilege(current_user,'public.ai_client_credentials','DELETE')
 AND NOT has_schema_privilege(current_user,'public','CREATE')
 FROM pg_roles r,pg_class c WHERE r.rolname=current_user AND c.oid='public.ai_route_config_revisions'::regclass`).Scan(&ok).Error
	if len(imageOptions) == 1 && imageOptions[0] && e == nil {
		var extra bool
		e = db.WithContext(ctx).Raw(`SELECT has_table_privilege(current_user,'public.ai_route_config_options','INSERT') AND has_table_privilege(current_user,'public.ai_route_config_options','UPDATE') AND NOT has_any_column_privilege(current_user,'public.ai_route_config_options','REFERENCES') AND NOT has_table_privilege(current_user,'public.ai_route_config_options','DELETE,TRUNCATE,TRIGGER')`).Scan(&extra).Error
		ok = ok && extra
	}
	if e != nil || !ok {
		return d.ErrUnavailable
	}
	return nil
}

type Store struct {
	db                *gorm.DB
	commitTransaction func(*gorm.DB) error
}

func New(db *gorm.DB) (*Store, error) {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return nil, d.ErrUnavailable
	}
	return &Store{db: db}, nil
}
func (s *Store) Credential(ctx context.Context, org, id string) (d.Credential, error) {
	var row struct {
		openai.AIClientCredential
		KeyPresent bool
	}
	result := s.db.WithContext(ctx).Table("public.ai_client_credential_bindings").Where("tenant_id=? AND user_id='' AND client_name=?", org, id).Take(&row)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return d.Credential{}, nil
	}
	if result.Error != nil {
		return d.Credential{}, d.ErrUnavailable
	}
	hash := sha256.Sum256([]byte(row.BaseURL))
	return d.Credential{OrganizationID: row.TenantID, ClientName: row.ClientName, Model: row.Model, APIStyle: row.APIStyle, Version: openai.CredentialVersion(row.AIClientCredential), EndpointDigest: hex.EncodeToString(hash[:]), Reference: openai.CredentialReference(row.AIClientCredential), Enabled: row.Enabled, KeyPresent: row.KeyPresent, TimeoutSeconds: row.TimeoutSecond}, nil
}

type record struct {
	RevisionID, OrganizationID, Capability, ParentRevision string
	SchemaVersion                                          int
	Payload                                                string
	PayloadDigest, ActorID                                 string
	PublishedAt                                            time.Time
	IdempotencyKey, RequestFingerprint, RollbackFrom       string
}

const projection = `revision_id,organization_id,capability,parent_revision,schema_version,payload::text AS payload,payload_digest,actor_id,published_at,idempotency_key,request_fingerprint,rollback_from`

func read(db *gorm.DB, scope d.Scope, condition string, args ...any) (d.Revision, error) {
	var row record
	result := db.Table("public.ai_route_config_revisions").Select(projection).Where("organization_id=? AND capability=?", scope.OrganizationID, scope.Capability).Where(condition, args...).Take(&row)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return d.Revision{}, d.ErrNotFound
	}
	if result.Error != nil {
		return d.Revision{}, d.ErrUnavailable
	}
	return decode(row)
}
func decode(row record) (d.Revision, error) {
	var p d.Payload
	if row.SchemaVersion != 1 || json.Unmarshal([]byte(row.Payload), &p) != nil {
		return d.Revision{}, d.ErrUnavailable
	}
	return d.Revision{ID: row.RevisionID, Scope: d.Scope{OrganizationID: row.OrganizationID, Capability: row.Capability}, Parent: row.ParentRevision, SchemaVersion: row.SchemaVersion, Payload: p, Digest: row.PayloadDigest, ActorID: row.ActorID, PublishedAt: row.PublishedAt.UTC(), Key: row.IdempotencyKey, Fingerprint: row.RequestFingerprint, RollbackFrom: row.RollbackFrom}, nil
}
func (s *Store) Receipt(ctx context.Context, scope d.Scope, key string) (d.Revision, error) {
	return read(s.db.WithContext(ctx), scope, "idempotency_key=?", key)
}
func (s *Store) Revision(ctx context.Context, scope d.Scope, id string) (d.Revision, error) {
	return read(s.db.WithContext(ctx), scope, "revision_id=?", id)
}
func (s *Store) Current(ctx context.Context, scope d.Scope) (d.Revision, error) {
	var id string
	r := s.db.WithContext(ctx).Raw("SELECT revision_id FROM public.ai_route_config_current WHERE organization_id=? AND capability=?", scope.OrganizationID, scope.Capability).Scan(&id)
	if r.Error != nil {
		return d.Revision{}, d.ErrUnavailable
	}
	if r.RowsAffected == 0 {
		return d.Revision{}, d.ErrNotFound
	}
	return s.Revision(ctx, scope, id)
}

func (s *Store) Optional(ctx context.Context, scope d.Scope, binding string) (d.Revision, error) {
	var id string
	r := s.db.WithContext(ctx).Raw("SELECT revision_id FROM public.ai_route_config_options WHERE organization_id=? AND capability=? AND binding_id=?", scope.OrganizationID, scope.Capability, binding).Scan(&id)
	if r.Error != nil {
		return d.Revision{}, d.ErrUnavailable
	}
	if r.RowsAffected == 0 {
		return d.Revision{}, d.ErrNotFound
	}
	return s.Revision(ctx, scope, id)
}
func (s *Store) History(ctx context.Context, scope d.Scope, cursor string, limit int) ([]d.Revision, error) {
	if !scope.Valid() || limit < 1 || limit > 50 || cursor != "" && !d.ValidFact(cursor, 128) {
		return nil, d.ErrInvalid
	}
	q := s.db.WithContext(ctx).Table("public.ai_route_config_revisions").Select(projection).Where("organization_id=? AND capability=?", scope.OrganizationID, scope.Capability)
	if cursor != "" {
		anchor, e := s.Revision(ctx, scope, cursor)
		if e != nil {
			return nil, e
		}
		q = q.Where("(published_at,revision_id)<(?,?)", anchor.PublishedAt, anchor.ID)
	}
	var rows []record
	if q.Order("published_at DESC,revision_id DESC").Limit(limit).Find(&rows).Error != nil {
		return nil, d.ErrUnavailable
	}
	out := []d.Revision{}
	for _, row := range rows {
		r, e := decode(row)
		if e != nil {
			return nil, e
		}
		out = append(out, r)
	}
	return out, nil
}
func (s *Store) Publish(ctx context.Context, cmd d.Command, build func(d.CredentialReader, string) (d.Payload, error)) (d.Revision, error) {
	if ctx == nil || ctx.Err() != nil || !cmd.Scope.Valid() || !d.ValidFact(cmd.Key, 128) || !d.ValidFact(cmd.ActorID, 128) || !d.ValidFact(cmd.Fingerprint, 128) || build == nil {
		return d.Revision{}, d.ErrInvalid
	}
	tx := s.db.WithContext(ctx).Begin(&sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if tx.Error != nil {
		return d.Revision{}, d.ErrUnavailable
	}
	defer tx.Rollback()
	prior, e := read(tx, cmd.Scope, "idempotency_key=?", cmd.Key)
	if e == nil {
		if prior.Fingerprint != cmd.Fingerprint {
			return d.Revision{}, d.ErrConflict
		}
		return prior, nil
	}
	if !errors.Is(e, d.ErrNotFound) {
		return d.Revision{}, e
	}
	id := "route-" + uuid.NewString()
	payload, e := build(&Store{db: tx}, id)
	if e != nil {
		return d.Revision{}, e
	}
	if payload.Optional != cmd.Input.Optional || cmd.Input.Optional && (cmd.Scope.Capability != d.ImageCapability || payload.BindingID != cmd.Input.BindingID) {
		return d.Revision{}, d.ErrInvalid
	}
	var current string
	var res *gorm.DB
	if cmd.Input.Optional {
		res = tx.Raw("SELECT revision_id FROM public.ai_route_config_options WHERE organization_id=? AND capability=? AND binding_id=? FOR UPDATE", cmd.Scope.OrganizationID, cmd.Scope.Capability, cmd.Input.BindingID).Scan(&current)
	} else {
		res = tx.Raw("SELECT revision_id FROM public.ai_route_config_current WHERE organization_id=? AND capability=? FOR UPDATE", cmd.Scope.OrganizationID, cmd.Scope.Capability).Scan(&current)
	}
	if res.Error != nil {
		return d.Revision{}, d.ErrUnavailable
	}
	if current == "" {
		current = d.NoRevision
	}
	if current != cmd.Input.ExpectedRevision {
		// A duplicate may have committed while this transaction waited for
		// current. Recover its receipt before classifying a stale CAS.
		return s.conflict(ctx, tx, cmd, nil)
	}
	raw, e := json.Marshal(payload)
	if e != nil {
		return d.Revision{}, d.ErrInvalid
	}
	digest := sha256.Sum256(raw)
	now := time.Now().UTC().Truncate(time.Microsecond)
	r := d.Revision{ID: id, Scope: cmd.Scope, Parent: current, SchemaVersion: 1, Payload: payload, Digest: hex.EncodeToString(digest[:]), ActorID: cmd.ActorID, PublishedAt: now, Key: cmd.Key, Fingerprint: cmd.Fingerprint, RollbackFrom: cmd.Input.RollbackFrom}
	e = tx.Exec(`INSERT INTO public.ai_route_config_revisions(revision_id,organization_id,capability,parent_revision,schema_version,payload,payload_digest,actor_id,published_at,idempotency_key,request_fingerprint,rollback_from) VALUES(?,?,?,?,1,?::jsonb,?,?,?,?,?,?)`, id, cmd.Scope.OrganizationID, cmd.Scope.Capability, current, string(raw), r.Digest, cmd.ActorID, now, cmd.Key, cmd.Fingerprint, cmd.Input.RollbackFrom).Error
	if e != nil {
		return s.conflict(ctx, tx, cmd, e)
	}
	if cmd.Input.Optional {
		if current == d.NoRevision {
			res = tx.Exec(`INSERT INTO public.ai_route_config_options(organization_id,capability,binding_id,revision_id) VALUES(?,?,?,?) ON CONFLICT DO NOTHING`, cmd.Scope.OrganizationID, cmd.Scope.Capability, cmd.Input.BindingID, id)
		} else {
			res = tx.Exec(`UPDATE public.ai_route_config_options SET revision_id=? WHERE organization_id=? AND capability=? AND binding_id=? AND revision_id=?`, id, cmd.Scope.OrganizationID, cmd.Scope.Capability, cmd.Input.BindingID, current)
		}
	} else if current == d.NoRevision {
		res = tx.Exec(`INSERT INTO public.ai_route_config_current(organization_id,capability,revision_id) VALUES(?,?,?) ON CONFLICT DO NOTHING`, cmd.Scope.OrganizationID, cmd.Scope.Capability, id)
	} else {
		res = tx.Exec(`UPDATE public.ai_route_config_current SET revision_id=? WHERE organization_id=? AND capability=? AND revision_id=?`, id, cmd.Scope.OrganizationID, cmd.Scope.Capability, current)
	}
	if res.Error != nil || res.RowsAffected != 1 {
		return s.conflict(ctx, tx, cmd, res.Error)
	}
	commit := s.commitTransaction
	if commit == nil {
		commit = func(tx *gorm.DB) error { return tx.Commit().Error }
	}
	if commit(tx) != nil {
		return d.Revision{}, d.ErrOutcomeUnknown
	}
	return r, nil
}
func (s *Store) conflict(ctx context.Context, tx *gorm.DB, cmd d.Command, cause error) (d.Revision, error) {
	_ = tx.Rollback().Error
	var pg *pgconn.PgError
	if cause != nil && (!errors.As(cause, &pg) || pg.Code != "23505") {
		return d.Revision{}, d.ErrUnavailable
	}
	prior, e := s.Receipt(ctx, cmd.Scope, cmd.Key)
	if e == nil && prior.Fingerprint == cmd.Fingerprint {
		return prior, nil
	}
	if e != nil && !errors.Is(e, d.ErrNotFound) {
		return d.Revision{}, d.ErrOutcomeUnknown
	}
	return d.Revision{}, d.ErrConflict
}

// GrantSQL is consumed only by explicit schema installation; never startup.
func GrantSQL(role string, writer bool) string {
	identifier := pgx.Identifier{role}.Sanitize()
	q := fmt.Sprintf("GRANT SELECT ON public.ai_route_config_revisions,public.ai_route_config_current,public.ai_route_config_options,public.ai_client_credential_bindings TO %s;", identifier)
	if writer {
		q += fmt.Sprintf("GRANT INSERT ON public.ai_route_config_revisions TO %s;GRANT INSERT,UPDATE ON public.ai_route_config_current,public.ai_route_config_options TO %s;", identifier, identifier)
	}
	return q
}

var _ d.Repository = (*Store)(nil)
