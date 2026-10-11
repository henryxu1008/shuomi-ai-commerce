// Package acquisitionplan owns recovery metadata in the Product database.
// Product, Collection and Preparation remain the owners of business facts.
package acquisitionplan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"gorm.io/gorm"
	sigjson "sigs.k8s.io/json"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
)

type Repository struct{ db *gorm.DB }

const plans = "public.product_acquisition_plans"
const items = "public.product_acquisition_plan_items"
const rules = "public.product_acquisition_rule_templates"
const receipts = "public.product_acquisition_rule_commands"
const maxProjection = 2 << 20

// InstallSchema is called only by an explicit schema-owner command.
func InstallSchema(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "postgres" {
		return sourcing.ErrAcquisitionUnavailable
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, sql := range []string{
			`CREATE TABLE IF NOT EXISTS public.product_acquisition_plans (
    organization_id varchar(128) NOT NULL,actor_id varchar(128) NOT NULL,member_id varchar(128) NOT NULL,
    id uuid NOT NULL,command_key uuid NOT NULL,input_json bytea NOT NULL,input_hash varchar(64) NOT NULL,
    revision bigint NOT NULL,fence bigint NOT NULL DEFAULT 0,lease_until timestamptz NOT NULL DEFAULT '1970-01-01 00:00:00+00',
    result_json bytea NOT NULL,result_hash varchar(64) NOT NULL,transfer_key uuid,transfer_input bytea,created_at timestamptz NOT NULL,
    PRIMARY KEY(organization_id,actor_id,id),UNIQUE(organization_id,actor_id,command_key),
    CONSTRAINT acq_plan_bounds CHECK(revision>0 AND fence>=0 AND octet_length(input_json) BETWEEN 1 AND 65536 AND octet_length(result_json) BETWEEN 1 AND 2097152 AND length(input_hash)=64 AND length(result_hash)=64),
    CONSTRAINT acq_plan_transfer CHECK((transfer_key IS NULL AND transfer_input IS NULL) OR (transfer_key IS NOT NULL AND octet_length(transfer_input) BETWEEN 1 AND 65536)))`,
			`CREATE TABLE IF NOT EXISTS public.product_acquisition_plan_items (
    organization_id varchar(128) NOT NULL,actor_id varchar(128) NOT NULL,plan_id uuid NOT NULL,ordinal integer NOT NULL,
    source_url varchar(128) NOT NULL,child_key uuid NOT NULL,item_json bytea NOT NULL,item_hash varchar(64) NOT NULL,
    PRIMARY KEY(organization_id,actor_id,plan_id,ordinal),UNIQUE(organization_id,actor_id,plan_id,source_url),
    FOREIGN KEY(organization_id,actor_id,plan_id) REFERENCES public.product_acquisition_plans(organization_id,actor_id,id),
    CONSTRAINT acq_plan_item_bounds CHECK(ordinal BETWEEN 0 AND 199 AND octet_length(item_json) BETWEEN 1 AND 65536 AND length(item_hash)=64))`,
			`CREATE TABLE IF NOT EXISTS public.product_acquisition_rule_templates (
    organization_id varchar(128) NOT NULL,actor_id varchar(128) NOT NULL,member_id varchar(128) NOT NULL,id uuid NOT NULL,
    revision bigint NOT NULL,archived boolean NOT NULL,body bytea NOT NULL,body_hash varchar(64) NOT NULL,
    PRIMARY KEY(organization_id,actor_id,id),CONSTRAINT acq_rule_bounds CHECK(revision>0 AND octet_length(body) BETWEEN 1 AND 4096 AND length(body_hash)=64))`,
			`CREATE TABLE IF NOT EXISTS public.product_acquisition_rule_commands (
    organization_id varchar(128) NOT NULL,actor_id varchar(128) NOT NULL,member_id varchar(128) NOT NULL,command_key uuid NOT NULL,
    input_hash varchar(64) NOT NULL,body bytea NOT NULL,body_hash varchar(64) NOT NULL,
    PRIMARY KEY(organization_id,actor_id,command_key),CONSTRAINT acq_rule_command_bounds CHECK(octet_length(body) BETWEEN 1 AND 4096 AND length(input_hash)=64 AND length(body_hash)=64))`,
		} {
			if err := tx.Exec(sql).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func NewRepository(ctx context.Context, db *gorm.DB) (*Repository, error) {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" {
		return nil, sourcing.ErrAcquisitionUnavailable
	}
	checks := []struct {
		table, columns string
		constraints    int64
	}{
		{plans, "organization_id,actor_id,member_id,id,command_key,input_json,input_hash,revision,fence,lease_until,result_json,result_hash,transfer_key,transfer_input,created_at", 4},
		{items, "organization_id,actor_id,plan_id,ordinal,source_url,child_key,item_json,item_hash", 4},
		{rules, "organization_id,actor_id,member_id,id,revision,archived,body,body_hash", 2},
		{receipts, "organization_id,actor_id,member_id,command_key,input_hash,body,body_hash", 2},
	}
	for _, c := range checks {
		var result []map[string]any
		if db.WithContext(ctx).Raw("SELECT "+c.columns+" FROM "+c.table+" LIMIT 0").Scan(&result).Error != nil {
			return nil, sourcing.ErrAcquisitionUnavailable
		}
		var n int64
		if db.WithContext(ctx).Raw(`SELECT count(*) FROM pg_constraint WHERE conrelid=?::regclass AND convalidated AND NOT condeferrable`, c.table).Scan(&n).Error != nil || n != c.constraints {
			return nil, sourcing.ErrAcquisitionUnavailable
		}
		var regular bool
		if db.WithContext(ctx).Raw(`SELECT relkind='r' AND NOT relrowsecurity AND NOT relforcerowsecurity FROM pg_class WHERE oid=?::regclass`, c.table).Scan(&regular).Error != nil || !regular {
			return nil, sourcing.ErrAcquisitionUnavailable
		}
	}
	var scoped bool
	if db.WithContext(ctx).Raw(`WITH expected(table_name,kind,keys) AS (VALUES
 ('product_acquisition_plans','p',ARRAY['organization_id','actor_id','id']),
 ('product_acquisition_plans','u',ARRAY['organization_id','actor_id','command_key']),
 ('product_acquisition_plan_items','p',ARRAY['organization_id','actor_id','plan_id','ordinal']),
 ('product_acquisition_plan_items','u',ARRAY['organization_id','actor_id','plan_id','source_url']),
 ('product_acquisition_rule_templates','p',ARRAY['organization_id','actor_id','id']),
 ('product_acquisition_rule_commands','p',ARRAY['organization_id','actor_id','command_key']))
 SELECT NOT EXISTS(SELECT 1 FROM expected e WHERE NOT EXISTS(SELECT 1 FROM pg_constraint c
 WHERE c.conrelid=to_regclass('public.'||e.table_name) AND c.contype::text=e.kind AND c.convalidated AND NOT c.condeferrable
 AND ARRAY(SELECT a.attname::text FROM unnest(c.conkey) WITH ORDINALITY k(num,pos)
 JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=k.num ORDER BY k.pos)=e.keys))
 AND EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid='public.product_acquisition_plan_items'::regclass
 AND c.contype='f' AND c.confrelid='public.product_acquisition_plans'::regclass AND c.convalidated AND NOT c.condeferrable
 AND c.confupdtype='a' AND c.confdeltype='a'
 AND ARRAY(SELECT a.attname::text FROM unnest(c.conkey) WITH ORDINALITY k(num,pos)
 JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=k.num ORDER BY k.pos)=ARRAY['organization_id','actor_id','plan_id']
 AND ARRAY(SELECT a.attname::text FROM unnest(c.confkey) WITH ORDINALITY k(num,pos)
 JOIN pg_attribute a ON a.attrelid=c.confrelid AND a.attnum=k.num ORDER BY k.pos)=ARRAY['organization_id','actor_id','id'])`).Scan(&scoped).Error != nil || !scoped {
		return nil, sourcing.ErrAcquisitionUnavailable
	}
	return &Repository{db}, nil
}
func digest(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func decode(raw []byte, hash string, target any, limit int) error {
	if len(raw) == 0 || len(raw) > limit || digest(raw) != hash {
		return sourcing.ErrAcquisitionUnavailable
	}
	strict, e := sigjson.UnmarshalStrict(raw, target, sigjson.DisallowDuplicateFields, sigjson.DisallowUnknownFields)
	if e != nil || len(strict) > 0 {
		return sourcing.ErrAcquisitionUnavailable
	}
	return nil
}
func validScope(s sourcing.AcquisitionPlanScope) bool {
	return (collection.Scope{OrganizationID: s.OrganizationID, ActorID: s.ActorID, MemberID: s.MemberID}).Validate() == nil
}
func (r *Repository) transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return sourcing.ErrAcquisitionUnavailable
	}
	defer tx.Rollback()
	if e := fn(tx); e != nil {
		return e
	}
	if tx.Commit().Error != nil {
		return sourcing.ErrAcquisitionUnknown
	}
	return nil
}
func lock(tx *gorm.DB, scope sourcing.AcquisitionPlanScope) error {
	if !validScope(scope) {
		return sourcing.ErrInvalidAcquisition
	}
	return tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", "acquisition-plan:"+scope.OrganizationID+":"+scope.ActorID).Error
}

type planRecord struct {
	OrganizationID, ActorID, MemberID, ID, CommandKey string
	InputJSON, ResultJSON, TransferInput              []byte
	InputHash, ResultHash                             string
	TransferKey                                       *string
	Revision, Fence                                   int64
	LeaseUntil, CreatedAt                             time.Time
}
type itemRecord struct {
	Ordinal             int
	SourceURL, ChildKey string
	ItemJSON            []byte
	ItemHash            string
}
type storedItem struct {
	Item               sourcing.AcquisitionPlanItem `json:"item"`
	CollectionBatchID  string                       `json:"collectionBatchId"`
	CollectionRevision int64                        `json:"collectionRevision"`
}

func read(tx *gorm.DB, scope sourcing.AcquisitionPlanScope, column, id string) (sourcing.AcquisitionPlan, error) {
	var p sourcing.AcquisitionPlan
	if !validScope(scope) || !collection.ValidID(id) {
		return p, sourcing.ErrInvalidAcquisition
	}
	var row planRecord
	q := tx.Raw("SELECT * FROM "+plans+" WHERE organization_id=? AND actor_id=? AND member_id=? AND "+column+"=?", scope.OrganizationID, scope.ActorID, scope.MemberID, id).Scan(&row)
	if q.Error != nil {
		return p, sourcing.ErrAcquisitionUnavailable
	}
	if q.RowsAffected != 1 {
		return p, sourcing.ErrAcquisitionNotFound
	}
	if e := decode(row.ResultJSON, row.ResultHash, &p, maxProjection); e != nil {
		return p, e
	}
	var input sourcing.AcquisitionPlanInput
	if e := decode(row.InputJSON, row.InputHash, &input, sourcing.MaxPlanBodyBytes); e != nil {
		return p, e
	}
	if !reflect.DeepEqual(input, p.Input) || p.ID != row.ID || p.Revision != row.Revision || len(p.Items) != 0 {
		return p, sourcing.ErrAcquisitionUnavailable
	}
	p.Scope = scope
	p.Key = row.CommandKey
	p.Fence = row.Fence
	p.LeaseUntil = row.LeaseUntil
	p.CreatedAt = row.CreatedAt
	p.TransferInputJSON = row.TransferInput
	if row.TransferKey != nil {
		p.TransferKey = *row.TransferKey
	}
	var members []itemRecord
	if tx.Raw("SELECT ordinal,source_url,child_key,item_json,item_hash FROM "+items+" WHERE organization_id=? AND actor_id=? AND plan_id=? ORDER BY ordinal LIMIT 201", scope.OrganizationID, scope.ActorID, p.ID).Scan(&members).Error != nil || len(members) > sourcing.MaxPlanProducts {
		return p, sourcing.ErrAcquisitionUnavailable
	}
	p.Items = []sourcing.AcquisitionPlanItem{}
	for i, row := range members {
		var item storedItem
		if row.Ordinal != i || decode(row.ItemJSON, row.ItemHash, &item, 65536) != nil || item.Item.Source != row.SourceURL || !collection.ValidID(row.ChildKey) {
			return p, sourcing.ErrAcquisitionUnavailable
		}
		item.Item.ChildKey = row.ChildKey
		item.Item.CollectionBatchID = item.CollectionBatchID
		item.Item.CollectionRevision = item.CollectionRevision
		p.Items = append(p.Items, item.Item)
	}
	raw, _ := json.Marshal(p)
	if len(raw) > maxProjection {
		return p, sourcing.ErrAcquisitionUnavailable
	}
	return p, nil
}
func (r *Repository) Read(ctx context.Context, s sourcing.AcquisitionPlanScope, id string) (sourcing.AcquisitionPlan, error) {
	return read(r.db.WithContext(ctx), s, "id", id)
}
func (r *Repository) ByKey(ctx context.Context, s sourcing.AcquisitionPlanScope, key string) (sourcing.AcquisitionPlan, error) {
	return read(r.db.WithContext(ctx), s, "command_key", key)
}
func (r *Repository) List(ctx context.Context, s sourcing.AcquisitionPlanScope, after string) ([]sourcing.AcquisitionPlan, error) {
	if !validScope(s) || (after != "" && !collection.ValidID(after)) {
		return nil, sourcing.ErrInvalidAcquisition
	}
	var ids []string
	query := "SELECT id FROM " + plans + " WHERE organization_id=? AND actor_id=? AND member_id=?"
	args := []any{s.OrganizationID, s.ActorID, s.MemberID}
	if after != "" {
		query += " AND id>?::uuid"
		args = append(args, after)
	}
	if r.db.WithContext(ctx).Raw(query+" ORDER BY id LIMIT 100", args...).Scan(&ids).Error != nil {
		return nil, sourcing.ErrAcquisitionUnavailable
	}
	result := []sourcing.AcquisitionPlan{}
	bytes := 2
	for _, id := range ids {
		p, e := r.Read(ctx, s, id)
		if e != nil {
			return nil, e
		}
		raw, err := json.Marshal(p)
		if err != nil {
			return nil, sourcing.ErrAcquisitionUnavailable
		}
		if bytes+len(raw)+1 > maxProjection-1024 {
			break
		}
		result = append(result, p)
		bytes += len(raw) + 1
	}
	// A list must obey the same response budget as one projection.
	raw, _ := json.Marshal(result)
	if len(raw) > maxProjection {
		return nil, sourcing.ErrSourcePublicationTooLarge
	}
	return result, nil
}
func projection(p sourcing.AcquisitionPlan) ([]byte, error) {
	if len(p.Items) > sourcing.MaxPlanProducts || p.Revision < 1 || p.Input.Rule.Validate() != nil {
		return nil, sourcing.ErrInvalidAcquisition
	}
	p.Items = []sourcing.AcquisitionPlanItem{}
	raw, e := json.Marshal(p)
	if e != nil || len(raw) > maxProjection {
		return nil, sourcing.ErrSourcePublicationTooLarge
	}
	return raw, nil
}
func writeItems(tx *gorm.DB, p sourcing.AcquisitionPlan) error {
	for i, item := range p.Items {
		canonical, e := sourcing.Canonical1688Source(item.Source)
		if e != nil || canonical.URL != item.Source || !collection.ValidID(item.ChildKey) {
			return sourcing.ErrInvalidAcquisition
		}
		raw, e := json.Marshal(storedItem{item, item.CollectionBatchID, item.CollectionRevision})
		if e != nil || len(raw) > 65536 {
			return sourcing.ErrSourcePublicationTooLarge
		}
		result := tx.Exec("INSERT INTO "+items+" (organization_id,actor_id,plan_id,ordinal,source_url,child_key,item_json,item_hash) VALUES (?,?,?,?,?,?,?,?) ON CONFLICT(organization_id,actor_id,plan_id,ordinal) DO UPDATE SET item_json=EXCLUDED.item_json,item_hash=EXCLUDED.item_hash WHERE "+items+".source_url=EXCLUDED.source_url AND "+items+".child_key=EXCLUDED.child_key", p.Scope.OrganizationID, p.Scope.ActorID, p.ID, i, item.Source, item.ChildKey, raw, digest(raw))
		if result.Error != nil {
			return sourcing.ErrAcquisitionUnavailable
		}
		if result.RowsAffected != 1 {
			return sourcing.ErrAcquisitionConflict
		}
	}
	return nil
}
func (r *Repository) Create(ctx context.Context, p sourcing.AcquisitionPlan, check sourcing.PlanAuthorityCheck) (sourcing.AcquisitionPlan, error) {
	if check == nil || !validScope(p.Scope) || !collection.ValidID(p.ID) || !collection.ValidID(p.Key) || p.Revision != 1 {
		return p, sourcing.ErrInvalidAcquisition
	}
	input, _ := json.Marshal(p.Input)
	raw, e := projection(p)
	if e != nil || len(input) > sourcing.MaxPlanBodyBytes {
		return p, sourcing.ErrInvalidAcquisition
	}
	e = r.transaction(ctx, func(tx *gorm.DB) error {
		if e := lock(tx, p.Scope); e != nil {
			return sourcing.ErrAcquisitionUnavailable
		}
		existing, e := read(tx, p.Scope, "command_key", p.Key)
		if e == nil {
			if !reflect.DeepEqual(existing.Input, p.Input) {
				return sourcing.ErrAcquisitionConflict
			}
			p = existing
			return check(ctx)
		}
		if !errors.Is(e, sourcing.ErrAcquisitionNotFound) {
			return e
		}
		var count int64
		if tx.Raw("SELECT count(*) FROM "+plans+" WHERE organization_id=? AND actor_id=?", p.Scope.OrganizationID, p.Scope.ActorID).Scan(&count).Error != nil {
			return sourcing.ErrAcquisitionUnavailable
		}
		if count >= sourcing.MaxAcquisitionOperations {
			return sourcing.ErrAcquisitionCapacity
		}
		if e := tx.Exec("INSERT INTO "+plans+" (organization_id,actor_id,member_id,id,command_key,input_json,input_hash,revision,result_json,result_hash,created_at) VALUES (?,?,?,?,?,?,?,1,?,?,?)", p.Scope.OrganizationID, p.Scope.ActorID, p.Scope.MemberID, p.ID, p.Key, input, digest(input), raw, digest(raw), p.CreatedAt).Error; e != nil {
			return sourcing.ErrAcquisitionConflict
		}
		if e := writeItems(tx, p); e != nil {
			return e
		}
		return check(ctx)
	})
	return p, e
}
func (r *Repository) Claim(ctx context.Context, s sourcing.AcquisitionPlanScope, id string, expected int64, check sourcing.PlanAuthorityCheck) (p sourcing.AcquisitionPlan, err error) {
	if check == nil || expected < 1 || !validScope(s) || !collection.ValidID(id) {
		return p, sourcing.ErrInvalidAcquisition
	}
	err = r.transaction(ctx, func(tx *gorm.DB) error {
		q := tx.Exec("UPDATE "+plans+" SET fence=fence+1,lease_until=clock_timestamp()+interval '60 seconds' WHERE organization_id=? AND actor_id=? AND member_id=? AND id=? AND revision=? AND lease_until<=clock_timestamp() AND fence<9223372036854775807", s.OrganizationID, s.ActorID, s.MemberID, id, expected)
		if q.Error != nil {
			return sourcing.ErrAcquisitionUnavailable
		}
		if q.RowsAffected != 1 {
			return sourcing.ErrAcquisitionConflict
		}
		var e error
		p, e = read(tx, s, "id", id)
		if e != nil {
			return e
		}
		return check(ctx)
	})
	return
}
func (r *Repository) Save(ctx context.Context, p sourcing.AcquisitionPlan, check sourcing.PlanAuthorityCheck) (sourcing.AcquisitionPlan, error) {
	if check == nil || !validScope(p.Scope) || !collection.ValidID(p.ID) || p.Fence < 1 {
		return p, sourcing.ErrInvalidAcquisition
	}
	err := r.transaction(ctx, func(tx *gorm.DB) error {
		old, e := read(tx, p.Scope, "id", p.ID)
		if e != nil {
			return e
		}
		if old.Fence != p.Fence || old.Revision != p.Revision || old.Key != p.Key || !reflect.DeepEqual(old.Input, p.Input) || len(p.Items) < len(old.Items) || old.TransferKey != "" && (old.TransferKey != p.TransferKey || !reflect.DeepEqual(old.TransferInputJSON, p.TransferInputJSON)) {
			return sourcing.ErrAcquisitionConflict
		}
		next := p
		next.Revision++
		raw, e := projection(next)
		if e != nil {
			return e
		}
		var transferKey any
		var transferInput any
		if p.TransferKey != "" {
			transferKey = p.TransferKey
			transferInput = p.TransferInputJSON
		}
		q := tx.Exec("UPDATE "+plans+" SET revision=?,result_json=?,result_hash=?,transfer_key=?,transfer_input=?,lease_until='1970-01-01 00:00:00+00' WHERE organization_id=? AND actor_id=? AND member_id=? AND id=? AND revision=? AND fence=? AND lease_until>clock_timestamp()", next.Revision, raw, digest(raw), transferKey, transferInput, p.Scope.OrganizationID, p.Scope.ActorID, p.Scope.MemberID, p.ID, p.Revision, p.Fence)
		if q.Error != nil {
			return sourcing.ErrAcquisitionUnavailable
		}
		if q.RowsAffected != 1 {
			return sourcing.ErrAcquisitionConflict
		}
		if e := writeItems(tx, next); e != nil {
			return e
		}
		if e := check(ctx); e != nil {
			return e
		}
		p = next
		return nil
	})
	return p, err
}

type ruleRecord struct {
	ID       string
	Revision int64
	Archived bool
	Body     []byte
	BodyHash string
}

func readRule(tx *gorm.DB, s sourcing.AcquisitionPlanScope, id string) (rule sourcing.AcquisitionRuleTemplate, e error) {
	var row ruleRecord
	q := tx.Raw("SELECT id,revision,archived,body,body_hash FROM "+rules+" WHERE organization_id=? AND actor_id=? AND member_id=? AND id=?", s.OrganizationID, s.ActorID, s.MemberID, id).Scan(&row)
	if q.Error != nil {
		return rule, sourcing.ErrAcquisitionUnavailable
	}
	if q.RowsAffected != 1 {
		return rule, sourcing.ErrAcquisitionNotFound
	}
	if decode(row.Body, row.BodyHash, &rule, 4096) != nil || rule.ID != row.ID || rule.Revision != row.Revision || rule.Archived != row.Archived || rule.Rule.Validate() != nil {
		return rule, sourcing.ErrAcquisitionUnavailable
	}
	return
}
func (r *Repository) ReadRule(ctx context.Context, s sourcing.AcquisitionPlanScope, id string) (sourcing.AcquisitionRuleTemplate, error) {
	if !validScope(s) || !collection.ValidID(id) {
		return sourcing.AcquisitionRuleTemplate{}, sourcing.ErrInvalidAcquisition
	}
	return readRule(r.db.WithContext(ctx), s, id)
}
func (r *Repository) ListRules(ctx context.Context, s sourcing.AcquisitionPlanScope) ([]sourcing.AcquisitionRuleTemplate, error) {
	if !validScope(s) {
		return nil, sourcing.ErrInvalidAcquisition
	}
	var ids []string
	if r.db.WithContext(ctx).Raw("SELECT id FROM "+rules+" WHERE organization_id=? AND actor_id=? AND member_id=? AND NOT archived ORDER BY id LIMIT 100", s.OrganizationID, s.ActorID, s.MemberID).Scan(&ids).Error != nil {
		return nil, sourcing.ErrAcquisitionUnavailable
	}
	result := []sourcing.AcquisitionRuleTemplate{}
	for _, id := range ids {
		rule, e := r.ReadRule(ctx, s, id)
		if e != nil {
			return nil, e
		}
		result = append(result, rule)
	}
	return result, nil
}
func (r *Repository) MutateRule(ctx context.Context, s sourcing.AcquisitionPlanScope, key string, input sourcing.RuleTemplateMutation, check sourcing.PlanAuthorityCheck) (rule sourcing.AcquisitionRuleTemplate, err error) {
	if !validScope(s) || !collection.ValidID(key) || check == nil {
		return rule, sourcing.ErrInvalidAcquisition
	}
	encoded, _ := json.Marshal(input)
	hash := digest(encoded)
	err = r.transaction(ctx, func(tx *gorm.DB) error {
		if lock(tx, s) != nil {
			return sourcing.ErrAcquisitionUnavailable
		}
		var receipt struct {
			InputHash string
			Body      []byte
			BodyHash  string
		}
		q := tx.Raw("SELECT input_hash,body,body_hash FROM "+receipts+" WHERE organization_id=? AND actor_id=? AND member_id=? AND command_key=?", s.OrganizationID, s.ActorID, s.MemberID, key).Scan(&receipt)
		if q.Error != nil {
			return sourcing.ErrAcquisitionUnavailable
		}
		if q.RowsAffected == 1 {
			if receipt.InputHash != hash {
				return sourcing.ErrAcquisitionConflict
			}
			if decode(receipt.Body, receipt.BodyHash, &rule, 4096) != nil {
				return sourcing.ErrAcquisitionUnavailable
			}
			return check(ctx)
		}
		if input.Action == "create" {
			var count int64
			if tx.Raw("SELECT count(*) FROM "+rules+" WHERE organization_id=? AND actor_id=? AND NOT archived", s.OrganizationID, s.ActorID).Scan(&count).Error != nil {
				return sourcing.ErrAcquisitionUnavailable
			}
			if count >= 100 {
				return sourcing.ErrAcquisitionCapacity
			}
			rule = sourcing.AcquisitionRuleTemplate{ID: collection.StableID(s.OrganizationID, s.ActorID, "acquisition-rule", key), Name: input.Name, Rule: input.Rule, Revision: 1, CreatedAt: time.Now().UTC()}
		} else {
			var e error
			rule, e = readRule(tx, s, input.ID)
			if e != nil {
				return e
			}
			if rule.Archived || rule.Revision != input.ExpectedRevision {
				return sourcing.ErrAcquisitionConflict
			}
			rule.Revision++
			if input.Action == "archive" {
				rule.Archived = true
			} else if input.Action == "update" {
				rule.Name = input.Name
				rule.Rule = input.Rule
			} else {
				return sourcing.ErrInvalidAcquisition
			}
		}
		body, _ := json.Marshal(rule)
		if input.Action == "create" {
			q = tx.Exec("INSERT INTO "+rules+" (organization_id,actor_id,member_id,id,revision,archived,body,body_hash) VALUES (?,?,?,?,?,?,?,?)", s.OrganizationID, s.ActorID, s.MemberID, rule.ID, rule.Revision, rule.Archived, body, digest(body))
		} else {
			q = tx.Exec("UPDATE "+rules+" SET revision=?,archived=?,body=?,body_hash=? WHERE organization_id=? AND actor_id=? AND member_id=? AND id=? AND revision=?", rule.Revision, rule.Archived, body, digest(body), s.OrganizationID, s.ActorID, s.MemberID, rule.ID, input.ExpectedRevision)
		}
		if q.Error != nil {
			return sourcing.ErrAcquisitionUnavailable
		}
		if q.RowsAffected != 1 {
			return sourcing.ErrAcquisitionConflict
		}
		if tx.Exec("INSERT INTO "+receipts+" (organization_id,actor_id,member_id,command_key,input_hash,body,body_hash) VALUES (?,?,?,?,?,?,?)", s.OrganizationID, s.ActorID, s.MemberID, key, hash, body, digest(body)).Error != nil {
			return sourcing.ErrAcquisitionConflict
		}
		return check(ctx)
	})
	return
}
