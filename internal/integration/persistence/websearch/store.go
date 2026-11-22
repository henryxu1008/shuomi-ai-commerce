package websearchstore

import (
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	credential "github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/channelcredential"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/websearch"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

//go:embed schema.sql
var schema string

//go:embed grant.sql
var grantRuntime string

// GrantRuntime is an explicit owner operation. Startup only verifies privileges.
func GrantRuntime(ctx context.Context, owner *gorm.DB) error {
	return applyDDL(ctx, owner, grantRuntime)
}

func Install(ctx context.Context, db *gorm.DB) error {
	return applyDDL(ctx, db, schema)
}

func applyDDL(ctx context.Context, db *gorm.DB, script string) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, statement := range strings.Split(script, ";") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if err := tx.Exec(statement).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

type Store struct{ db *gorm.DB }

func New(db *gorm.DB) *Store {
	return &Store{db: db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})}
}

func (s *Store) Check(ctx context.Context) error {
	for _, table := range []string{"ai_web_search_configuration", "ai_web_search_config_receipts", "ai_web_search_requests"} {
		if err := s.db.WithContext(ctx).Exec("SELECT 1 FROM public." + table + " LIMIT 0").Error; err != nil {
			return websearch.ErrUnavailable
		}
	}
	return nil
}

// VerifyRuntime verifies the installed tables and exact existing writer role.
func (s *Store) VerifyRuntime(ctx context.Context) error {
	var ready bool
	err := s.db.WithContext(ctx).Raw(`SELECT current_user = 'model_route_config_writer'
        AND has_table_privilege(current_user, 'public.ai_web_search_configuration', 'SELECT')
        AND has_table_privilege(current_user, 'public.ai_web_search_configuration', 'INSERT')
        AND has_table_privilege(current_user, 'public.ai_web_search_configuration', 'UPDATE')
        AND has_table_privilege(current_user, 'public.ai_web_search_requests', 'SELECT')
        AND has_table_privilege(current_user, 'public.ai_web_search_requests', 'INSERT')
        AND has_table_privilege(current_user, 'public.ai_web_search_requests', 'UPDATE')
        AND has_table_privilege(current_user, 'public.ai_web_search_config_receipts', 'SELECT')
        AND has_table_privilege(current_user, 'public.ai_web_search_config_receipts', 'INSERT')
        AND NOT has_table_privilege(current_user, 'public.ai_web_search_config_receipts', 'UPDATE')
        AND NOT has_table_privilege(current_user, 'public.ai_web_search_requests', 'DELETE')
        AND NOT has_table_privilege(current_user, 'public.ai_web_search_configuration', 'DELETE')`).Row().Scan(&ready)
	if err != nil || !ready {
		return websearch.ErrUnavailable
	}
	return s.Check(ctx)
}

func liveDB(ctx context.Context, tx *gorm.DB, label string) error {
	return tx.WithContext(ctx).Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", "ai-web-search:"+label).Error
}

func storage(err error) error {
	if err == nil || errors.Is(err, websearch.ErrNotFound) || errors.Is(err, websearch.ErrConflict) || errors.Is(err, websearch.ErrQuota) {
		return err
	}
	return websearch.ErrUnavailable
}

func (s *Store) Publication(ctx context.Context) (websearch.Publication, error) {
	var value websearch.Publication
	err := s.readConfig(ctx, credential.PlatformOwner, &value)
	return value, err
}

func (s *Store) Selection(ctx context.Context, org string) (websearch.Selection, error) {
	var value websearch.Selection
	err := s.readConfig(ctx, org, &value)
	return value, err
}

func (s *Store) readConfig(ctx context.Context, owner string, dest any) error {
	var raw []byte
	err := s.db.WithContext(ctx).Raw("SELECT payload FROM public.ai_web_search_configuration WHERE owner = ?", owner).Row().Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return storage(err)
	}
	return storage(json.Unmarshal(raw, dest))
}

func (s *Store) SavePublication(ctx context.Context, actor, key string, input websearch.Publication) (websearch.Publication, error) {
	value := input
	value.Revision++
	err := s.saveConfig(ctx, credential.PlatformOwner, actor, key, input.Revision, input, &value)
	return value, err
}

func (s *Store) SaveSelection(ctx context.Context, org, actor, key string, input websearch.Selection) (websearch.Selection, error) {
	value := input
	value.Revision++
	err := s.saveConfig(ctx, org, actor, key, input.Revision, input, &value)
	return value, err
}

func (s *Store) PublicationReceipt(ctx context.Context, actor, key string) (websearch.Publication, error) {
	var value websearch.Publication
	_, err := configReceipt(s.db.WithContext(ctx), credential.PlatformOwner, actor, key, &value)
	return value, storage(err)
}

func (s *Store) SelectionReceipt(ctx context.Context, org, actor, key string) (websearch.Selection, error) {
	var value websearch.Selection
	_, err := configReceipt(s.db.WithContext(ctx), org, actor, key, &value)
	return value, storage(err)
}

func configReceipt(tx *gorm.DB, owner, actor, key string, value any) (string, error) {
	var digest string
	var raw []byte
	err := tx.Raw("SELECT input_digest, payload FROM public.ai_web_search_config_receipts WHERE owner = ? AND actor = ? AND request_key = ?", owner, actor, key).Row().Scan(&digest, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return "", websearch.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return digest, json.Unmarshal(raw, value)
}

func (s *Store) saveConfig(ctx context.Context, owner, actor, key string, expected int64, input, value any) error {
	rawInput, err := json.Marshal(input)
	if err != nil {
		return storage(err)
	}
	digestValue := sha256.Sum256(rawInput)
	digest := hex.EncodeToString(digestValue[:])
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := liveDB(ctx, tx, "configuration:"+owner); err != nil {
			return err
		}
		if previous, err := configReceipt(tx, owner, actor, key, value); err == nil {
			if previous != digest {
				return websearch.ErrConflict
			}
			return nil
		} else if !errors.Is(err, websearch.ErrNotFound) {
			return err
		}
		var revision int64
		err := tx.Raw("SELECT revision FROM public.ai_web_search_configuration WHERE owner = ?", owner).Row().Scan(&revision)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if revision != expected {
			return websearch.ErrConflict
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return err
		}
		if err := tx.Exec("INSERT INTO public.ai_web_search_configuration (owner,revision,payload) VALUES (?,?,CAST(? AS jsonb)) ON CONFLICT (owner) DO UPDATE SET revision=EXCLUDED.revision,payload=EXCLUDED.payload",
			owner, expected+1, string(raw)).Error; err != nil {
			return err
		}
		return tx.Exec("INSERT INTO public.ai_web_search_config_receipts (owner,actor,request_key,input_digest,payload) VALUES (?,?,?,?,CAST(? AS jsonb))",
			owner, actor, key, digest, string(raw)).Error
	})
	return storage(err)
}

const scopeWhere = "organization=? AND actor=? AND member=? AND conversation=?"

type record struct {
	websearch.Receipt
	Deadline    time.Time
	ObservedAt  time.Time
	Fingerprint string
}

func scopeArgs(scope websearch.Scope) []any {
	return []any{scope.Organization, scope.Actor, scope.Member, scope.Conversation}
}

func predicate(scope websearch.Scope, key string) (string, []any) {
	return scopeWhere + " AND request_key=?", append(scopeArgs(scope), key)
}

func readRecord(tx *gorm.DB, scope websearch.Scope, key string) (record, error) {
	var r record
	where, args := predicate(scope, key)
	row := tx.Raw("SELECT request_key,query,status,source,started_at,usage,deadline,fingerprint,clock_timestamp() FROM public.ai_web_search_requests WHERE "+where, args...).Row()
	err := row.Scan(&r.Key, &r.Query, &r.Status, &r.Source, &r.StartedAt, &r.Usage, &r.Deadline, &r.Fingerprint, &r.ObservedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return record{}, websearch.ErrNotFound
	}
	return r, err
}

func projected(r websearch.Receipt, due time.Time, now time.Time) websearch.Receipt {
	if !now.Before(due) {
		if r.Status == websearch.Held {
			zero := 0
			r.Status, r.Usage = websearch.NotSent, &zero
		} else if r.Status == websearch.Claimed {
			r.Status = websearch.Unknown
		}
	}
	return r
}

func (s *Store) Find(ctx context.Context, scope websearch.Scope, key, digest string) (websearch.Receipt, error) {
	r, err := readRecord(s.db.WithContext(ctx), scope, key)
	if err != nil {
		return websearch.Receipt{}, storage(err)
	}
	if digest != "" && r.Fingerprint != digest {
		return websearch.Receipt{}, websearch.ErrConflict
	}
	return projected(r.Receipt, r.Deadline, r.ObservedAt), nil
}

func (s *Store) Recent(ctx context.Context, scope websearch.Scope) ([]websearch.Receipt, error) {
	rows, err := s.db.WithContext(ctx).Raw("SELECT request_key,query,status,source,started_at,usage,deadline,clock_timestamp() FROM public.ai_web_search_requests WHERE "+scopeWhere+
		" ORDER BY started_at DESC, request_key DESC LIMIT 20", scopeArgs(scope)...).Rows()
	if err != nil {
		return nil, storage(err)
	}
	defer rows.Close()
	results := []websearch.Receipt{}
	for rows.Next() {
		var r record
		if err := rows.Scan(&r.Key, &r.Query, &r.Status, &r.Source, &r.StartedAt, &r.Usage, &r.Deadline, &r.ObservedAt); err != nil {
			return nil, storage(err)
		}
		results = append(results, projected(r.Receipt, r.Deadline, r.ObservedAt))
	}
	return results, storage(rows.Err())
}

func (s *Store) Hold(ctx context.Context, a websearch.Attempt) (websearch.Receipt, bool, time.Time, error) {
	var result websearch.Receipt
	var deadline time.Time
	var fresh bool
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := liveDB(ctx, tx, "request:"+a.Scope.Organization+":"+a.Scope.Member+":"+a.Key); err != nil {
			return err
		}
		if prior, err := readRecord(tx, a.Scope, a.Key); err == nil {
			if prior.Fingerprint != a.Fingerprint {
				return websearch.ErrConflict
			}
			result, deadline = projected(prior.Receipt, prior.Deadline, prior.ObservedAt), prior.Deadline
			return nil
		} else if !errors.Is(err, websearch.ErrNotFound) {
			return err
		}
		if err := liveDB(ctx, tx, "budget:"+a.Route.Owner); err != nil {
			return err
		}
		var now time.Time
		if err := tx.Raw("SELECT clock_timestamp()").Row().Scan(&now); err != nil {
			return err
		}
		now = now.UTC()
		day := now.Truncate(24 * time.Hour)
		deadline = now.Add(10 * time.Second)
		if end := day.Add(24 * time.Hour); end.Before(deadline) {
			deadline = end
		}
		if deadline.Sub(now) < time.Millisecond {
			return websearch.ErrQuota
		}
		if err := tx.Exec("UPDATE public.ai_web_search_requests SET status='not_sent', usage=0 WHERE connection_owner=? AND status='held' AND deadline <= clock_timestamp()", a.Route.Owner).Error; err != nil {
			return err
		}
		var total, tenant int
		err := tx.Raw("SELECT count(*), count(*) FILTER (WHERE organization=?) FROM public.ai_web_search_requests WHERE connection_owner=? AND day_start=? AND status<>'not_sent'",
			a.Scope.Organization, a.Route.Owner, day).Row().Scan(&total, &tenant)
		if err != nil {
			return err
		}
		if total >= a.Route.TotalLimit || tenant >= a.Route.TenantLimit {
			return websearch.ErrQuota
		}
		result = websearch.Receipt{Key: a.Key, Query: a.Query, Status: websearch.Held, Source: a.Route.Source, StartedAt: now}
		args := append(scopeArgs(a.Scope), a.Key, a.Query, a.Fingerprint, a.Route.Source, a.Route.Owner, a.Route.Preview, a.Route.Credential.Reference, a.Route.PublicationRevision, a.Route.ChoiceRevision, a.Route.MaxUSDMicros, day, now, deadline)
		if err := tx.Exec("INSERT INTO public.ai_web_search_requests (organization,actor,member,conversation,request_key,query,fingerprint,source,connection_owner,preview,credential_reference,publication_revision,choice_revision,max_usd_micros,day_start,started_at,deadline,status) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,'held')", args...).Error; err != nil {
			return err
		}
		fresh = true
		return nil
	})
	return result, fresh && err == nil, deadline, storage(err)
}

func (s *Store) Claim(ctx context.Context, a websearch.Attempt) (bool, error) {
	where, args := predicate(a.Scope, a.Key)
	args = append(args, a.Fingerprint)
	update := s.db.WithContext(ctx).Exec("UPDATE public.ai_web_search_requests SET status='claimed' WHERE "+where+" AND fingerprint=? AND status='held' AND clock_timestamp()<deadline", args...)
	return update.RowsAffected == 1, storage(update.Error)
}

func (s *Store) Finish(ctx context.Context, a websearch.Attempt, state string, usage *int) (websearch.Receipt, error) {
	where, args := predicate(a.Scope, a.Key)
	args = append([]any{state, usage}, append(args, a.Fingerprint)...)
	allowed := "status IN ('claimed','unknown')"
	if state == websearch.NotSent {
		allowed = "status IN ('held','claimed')"
	}
	err := s.db.WithContext(ctx).Exec("UPDATE public.ai_web_search_requests SET status=?,usage=? WHERE "+where+" AND fingerprint=? AND "+allowed, args...).Error
	if err != nil {
		return websearch.Receipt{}, storage(err)
	}
	return s.Find(ctx, a.Scope, a.Key, a.Fingerprint)
}
