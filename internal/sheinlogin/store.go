package sheinlogin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"task-processor/internal/core/config"
	"task-processor/internal/storecenter"
)

var ErrNoUsableCookie = errors.New("SHEIN browser state does not contain a usable cookie")

const cookieTTL = 30 * 24 * time.Hour
const attemptTTL = 30 * 24 * time.Hour
const attemptGroup = "shein-login-workers-v1"

type RedisStore struct {
	client               *redis.Client
	installation, prefix string
}

func NewRedisStoreFromClient(c *redis.Client, installation string) (*RedisStore, error) {
	if c == nil || !validUUID(installation) {
		return nil, ErrInvalidInput
	}
	return &RedisStore{c, installation, "shein-login:v1:" + installation + ":"}, nil
}
func NewRedisStore(c config.RedisConfig, installation string) (*RedisStore, error) {
	if c.Host == "" || c.Port < 1 {
		return nil, ErrInvalidInput
	}
	client := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%d", c.Host, c.Port), Password: c.Password, DB: c.DB, PoolSize: 10, DialTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second})
	s, e := NewRedisStoreFromClient(client, installation)
	if e != nil {
		client.Close()
		return nil, e
	}
	return s, nil
}
func (s *RedisStore) Initialize(ctx context.Context) error {
	marker := s.prefix + "owner"
	expected := "shein-login:v1:" + s.installation
	value, err := s.client.Eval(ctx, `redis.call('SET',KEYS[1],ARGV[1],'NX');return redis.call('GET',KEYS[1])`, []string{marker}, expected).Text()
	if err != nil {
		return ErrUnavailable
	}
	if value != expected {
		return ErrInvalidInput
	}
	return nil
}
func (s *RedisStore) Close() error                   { return s.client.Close() }
func (s *RedisStore) Ready(ctx context.Context) bool { return s.client.Ping(ctx).Err() == nil }
func (s *RedisStore) stream() string                 { return s.prefix + "attempts" }
func (s *RedisStore) attemptKey(id string) string    { return s.prefix + "attempt:" + id }
func (s *RedisStore) scopeKey(scope storecenter.SheinLoginScope) string {
	b, _ := json.Marshal(scope)
	h := sha256.Sum256(b)
	return s.prefix + "scope:" + hex.EncodeToString(h[:])
}
func (s *RedisStore) validScope(scope storecenter.SheinLoginScope) bool {
	return scope.Validate() == nil && scope.Generation > 0 && scope.InstallationID == s.installation
}
func cookieCount(state map[string]any) int {
	if state == nil {
		return 0
	}
	switch c := state["cookies"].(type) {
	case []any:
		return len(c)
	case []map[string]any:
		return len(c)
	}
	return 0
}
func attemptFingerprint(a LoginAttempt) string {
	data, _ := json.Marshal(struct {
		Scope             storecenter.SheinLoginScope
		ID, Actor, Member string
		Force             bool
	}{a.Scope, a.ID, a.ActorID, a.MemberID, a.ForceLogin})
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
func (s *RedisStore) Enqueue(ctx context.Context, a LoginAttempt) (*LoginAttempt, error) {
	if !s.validScope(a.Scope) || !validUUID(a.ID) || !validOpaque(a.ActorID) || !validOpaque(a.MemberID) {
		return nil, ErrInvalidInput
	}
	a.Fingerprint = attemptFingerprint(a)
	a.Status = LoginAttemptQueued
	a.SyncStatus = "not_requested"
	a.MerchantID = ""
	a.ExpectedStoreVersion = 0
	a.OwnerNonce = ""
	a.ErrorCode = ""
	a.CompletedAt = nil
	data, _ := json.Marshal(a)
	if len(data) > 65536 {
		return nil, ErrInvalidInput
	}
	result, e := s.client.Eval(ctx, `
 local old=redis.call('GET',KEYS[1]);if old then return old end
 if redis.call('EXISTS',KEYS[2])==1 then return 'BUSY' end
 redis.call('SET',KEYS[1],ARGV[1],'EX',ARGV[2]);redis.call('SET',KEYS[2],ARGV[3],'EX',ARGV[2])
 redis.call('XADD',KEYS[3],'*','attempt_id',ARGV[3]);return ARGV[1]`, []string{s.attemptKey(a.ID), s.scopeKey(a.Scope) + ":active", s.stream()}, string(data), int(attemptTTL.Seconds()), a.ID).Text()
	if e != nil {
		return nil, ErrUnavailable
	}
	if result == "BUSY" {
		return nil, storecenter.ErrSheinLoginFenced
	}
	var old LoginAttempt
	if DecodeStrict([]byte(result), &old, 65536) != nil || old.Fingerprint != a.Fingerprint || !reflect.DeepEqual(old.Scope, a.Scope) || old.ActorID != a.ActorID || old.MemberID != a.MemberID {
		return nil, storecenter.ErrIdempotencyConflict
	}
	return &old, nil
}
func (s *RedisStore) Load(ctx context.Context, id string) (*LoginAttempt, error) {
	if !validUUID(id) {
		return nil, ErrInvalidInput
	}
	data, e := s.client.Get(ctx, s.attemptKey(id)).Bytes()
	if errors.Is(e, redis.Nil) {
		return nil, ErrInvalidInput
	}
	if e != nil {
		return nil, ErrUnavailable
	}
	var a LoginAttempt
	if DecodeStrict(data, &a, 65536) != nil || !s.validScope(a.Scope) || a.ID != id || a.Fingerprint != attemptFingerprint(a) || !validOpaque(a.ActorID) || !validOpaque(a.MemberID) {
		return nil, ErrInvalidInput
	}
	return &a, nil
}
func validTransition(old, next LoginAttempt, cookie map[string]any) bool {
	if old.ID != next.ID || !reflect.DeepEqual(old.Scope, next.Scope) || old.ActorID != next.ActorID || old.MemberID != next.MemberID || old.Fingerprint != next.Fingerprint || old.ForceLogin != next.ForceLogin || !old.CreatedAt.Equal(next.CreatedAt) {
		return false
	}
	if old.Status.IsActive() {
		if next.Status == LoginAttemptSucceeded && old.Status == LoginAttemptQueued {
			return false
		}
		if next.Status == LoginAttemptWaitingVerifyCode && old.Status == LoginAttemptQueued {
			return false
		}
		if next.Status == LoginAttemptLaunching && old.Status == LoginAttemptWaitingVerifyCode {
			return false
		}
		if next.Status == LoginAttemptQueued && old.Status != LoginAttemptQueued {
			return false
		}
		if next.Status != LoginAttemptQueued && next.Status != LoginAttemptLaunching && next.Status != LoginAttemptWaitingVerifyCode && next.Status != LoginAttemptSucceeded && next.Status != LoginAttemptFailed && next.Status != LoginAttemptCancelled && next.Status != LoginAttemptInterrupted {
			return false
		}
		if next.Status == LoginAttemptSucceeded {
			return cookieCount(cookie) > 0 && next.SyncStatus == "pending" && next.MerchantID == "" && next.ExpectedStoreVersion == 0
		}
		return cookie == nil && next.SyncStatus == "not_requested" && next.MerchantID == "" && next.ExpectedStoreVersion == 0 && (old.OwnerNonce == "" || old.OwnerNonce == next.OwnerNonce)
	}
	if old.Status != LoginAttemptSucceeded || next.Status != old.Status || cookie != nil {
		return false
	}
	if old.SyncStatus != "pending" && old.SyncStatus != "unavailable" {
		return false
	}
	if old.MerchantID != "" && (old.MerchantID != next.MerchantID || old.ExpectedStoreVersion != next.ExpectedStoreVersion) {
		return false
	}
	return next.SyncStatus == "pending" || next.SyncStatus == "applied" || next.SyncStatus == "blocked" || next.SyncStatus == "unavailable"
}
func (s *RedisStore) CompareAndSwap(ctx context.Context, old, next LoginAttempt, cookie map[string]any) error {
	if !s.validScope(old.Scope) || !validTransition(old, next, cookie) {
		return ErrInvalidInput
	}
	oldData, _ := json.Marshal(old)
	data, _ := json.Marshal(next)
	cookieData := ""
	if cookie != nil {
		v, _ := json.Marshal(struct {
			Scope storecenter.SheinLoginScope `json:"scope"`
			State map[string]any              `json:"state"`
		}{old.Scope, cookieOnlyBrowserState(cookie)})
		if len(v) > 1<<20 {
			return ErrInvalidInput
		}
		cookieData = string(v)
	}
	result, e := s.client.Eval(ctx, `
 if redis.call('GET',KEYS[1])~=ARGV[1] then return 0 end
 redis.call('SET',KEYS[1],ARGV[2],'EX',ARGV[3])
 if ARGV[4]~='' then redis.call('SET',KEYS[3],ARGV[4],'EX',ARGV[5]) end
 if ARGV[6]=='terminal' and redis.call('GET',KEYS[2])==ARGV[7] then redis.call('DEL',KEYS[2]);redis.call('DEL',KEYS[4]) end
 return 1`, []string{s.attemptKey(old.ID), s.scopeKey(old.Scope) + ":active", s.scopeKey(old.Scope) + ":cookie", s.attemptKey(old.ID) + ":code"}, string(oldData), string(data), int(attemptTTL.Seconds()), cookieData, int(cookieTTL.Seconds()), map[bool]string{true: "terminal", false: "active"}[!next.Status.IsActive()], old.ID).Int()
	if e != nil {
		return ErrUnavailable
	}
	if result != 1 {
		return storecenter.ErrVersionConflict
	}
	return nil
}
func (s *RedisStore) Cookie(ctx context.Context, scope storecenter.SheinLoginScope) (map[string]any, bool, error) {
	if !s.validScope(scope) {
		return nil, false, ErrInvalidInput
	}
	data, e := s.client.Get(ctx, s.scopeKey(scope)+":cookie").Bytes()
	if errors.Is(e, redis.Nil) {
		return nil, false, nil
	}
	if e != nil {
		return nil, false, ErrUnavailable
	}
	var v struct {
		Scope storecenter.SheinLoginScope `json:"scope"`
		State map[string]any              `json:"state"`
	}
	if DecodeStrict(data, &v, 1<<20) != nil || !reflect.DeepEqual(scope, v.Scope) || cookieCount(v.State) == 0 {
		return nil, false, ErrInvalidInput
	}
	return cookieOnlyBrowserState(v.State), true, nil
}
func (s *RedisStore) Cancel(ctx context.Context, a LoginAttempt) error {
	if !a.Status.IsActive() {
		return nil
	}
	next := a
	next.Status = LoginAttemptCancelled
	now := time.Now().UTC()
	next.CompletedAt = &now
	next.ErrorCode = "LOGIN_CANCELLED"
	return s.CompareAndSwap(ctx, a, next, nil)
}
func (s *RedisStore) SubmitCode(ctx context.Context, a LoginAttempt, code string) error {
	if a.Status != LoginAttemptWaitingVerifyCode || len(code) < 4 || len(code) > 16 || strings.Trim(code, "0123456789") != "" {
		return ErrInvalidInput
	}
	data, _ := json.Marshal(a)
	n, e := s.client.Eval(ctx, `if redis.call('GET',KEYS[1])~=ARGV[1] then return 0 end;redis.call('SET',KEYS[2],ARGV[2],'EX',120);return 1`, []string{s.attemptKey(a.ID), s.attemptKey(a.ID) + ":code"}, string(data), code).Int()
	if e != nil {
		return ErrUnavailable
	}
	if n != 1 {
		return storecenter.ErrVersionConflict
	}
	return nil
}
func (s *RedisStore) ConsumeCode(ctx context.Context, id string) (string, error) {
	v, e := s.client.GetDel(ctx, s.attemptKey(id)+":code").Result()
	if errors.Is(e, redis.Nil) {
		return "", nil
	}
	if e != nil {
		return "", ErrUnavailable
	}
	return v, nil
}
func (s *RedisStore) EnsureGroup(ctx context.Context) error {
	e := s.client.XGroupCreateMkStream(ctx, s.stream(), attemptGroup, "0").Err()
	if e != nil && !strings.Contains(e.Error(), "BUSYGROUP") {
		return ErrUnavailable
	}
	return nil
}
func (s *RedisStore) ReadJobs(ctx context.Context, consumer string) ([]redis.XMessage, error) {
	streams, e := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{Group: attemptGroup, Consumer: consumer, Streams: []string{s.stream(), ">"}, Count: 1, Block: time.Second}).Result()
	if errors.Is(e, redis.Nil) {
		return nil, nil
	}
	if e != nil {
		return nil, ErrUnavailable
	}
	var out []redis.XMessage
	for _, st := range streams {
		out = append(out, st.Messages...)
	}
	return out, nil
}
func (s *RedisStore) Ack(ctx context.Context, id string) error {
	return s.client.XAck(ctx, s.stream(), attemptGroup, id).Err()
}
func (s *RedisStore) Stale(ctx context.Context, consumer string) ([]redis.XMessage, error) {
	v, _, e := s.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{Stream: s.stream(), Group: attemptGroup, Consumer: consumer, Start: "0-0", MinIdle: 45 * time.Second, Count: 10}).Result()
	return v, e
}

// A daily cap is scoped to this installation and exact current binding. It
// never releases Store ownership or authorizes browser work by itself.
func (s *RedisStore) ReserveAutomaticCode(ctx context.Context, scope storecenter.SheinLoginScope) (bool, error) {
	if !s.validScope(scope) {
		return false, ErrInvalidInput
	}
	day := time.Now().UTC().Format("2006-01-02")
	n, e := s.client.Eval(ctx, `local n=tonumber(redis.call('GET',KEYS[1]) or '0');if n>=2 then return 0 end;redis.call('INCR',KEYS[1]);redis.call('EXPIRE',KEYS[1],172800);return 1`, []string{s.scopeKey(scope) + ":automatic-code:" + day}).Int()
	return n == 1, e
}
