package storecenter

import (
	"context"
	"encoding/hex"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrSheinLoginFenced    = errors.New("SHEIN login execution is fenced")
	ErrIdempotencyConflict = errors.New("SHEIN login operation payload conflict")
)

// SheinLoginScope contains current identities only. Generation is issued by Store,
// never by Redis, a process lease, or deployment configuration.
type SheinLoginScope struct {
	OrganizationID    string `json:"organization_id"`
	StoreID           string `json:"store_id"`
	InstallationID    string `json:"installation_id"`
	BindingRevision   string `json:"binding_revision"`
	ConfigFingerprint string `json:"config_fingerprint"`
	Generation        int64  `json:"generation"`
}

func (s SheinLoginScope) Validate() error {
	if _, e := validateOpaqueIdentity("organization", s.OrganizationID, MaxOrganizationIDBytes); e != nil {
		return e
	}
	for _, id := range []string{s.StoreID, s.InstallationID, s.BindingRevision} {
		if _, e := canonicalUUID(id); e != nil {
			return ErrSheinLoginFenced
		}
	}
	b, e := hex.DecodeString(s.ConfigFingerprint)
	if e != nil || len(b) != 32 || s.Generation < 0 {
		return ErrSheinLoginFenced
	}
	return nil
}

type SheinBinding struct {
	SheinLoginScope `gorm:"embedded"`
	State           string `json:"state"`
	ActiveAttemptID string `json:"-"`
	OwnerNonce      string `json:"-"`
	ClosedConfirmed bool   `json:"closed_confirmed"`
}

// The sole optional table is an execution fence, not an account or Store owner.
type sheinLoginBindingRow struct {
	OrganizationID    string `gorm:"primaryKey;size:200"`
	StoreID           string `gorm:"primaryKey;type:char(36)"`
	InstallationID    string `gorm:"type:char(36);not null"`
	BindingRevision   string `gorm:"type:char(36);not null"`
	ConfigFingerprint string `gorm:"size:64;not null"`
	Generation        int64  `gorm:"not null"`
	State             string `gorm:"size:32;not null"`
	ActiveAttemptID   string `gorm:"size:36;not null"`
	OwnerNonce        string `gorm:"size:36;not null"`
	ClosedConfirmed   bool   `gorm:"not null"`
}

func (sheinLoginBindingRow) TableName() string        { return "workbench_store_shein_login_bindings" }
func AutoMigrateSheinLoginForTests(db *gorm.DB) error { return db.AutoMigrate(&sheinLoginBindingRow{}) }
func (r sheinLoginBindingRow) binding() SheinBinding {
	return SheinBinding{SheinLoginScope: SheinLoginScope{r.OrganizationID, r.StoreID, r.InstallationID, r.BindingRevision, r.ConfigFingerprint, r.Generation}, State: r.State, ActiveAttemptID: r.ActiveAttemptID, OwnerNonce: r.OwnerNonce, ClosedConfirmed: r.ClosedConfirmed}
}

type SheinBindingCommand struct {
	Scope                         SheinLoginScope
	ActorID, MemberID, RequestKey string
	ExpectedGeneration            int64
}
type SheinIdentityCommand struct {
	Scope                                    SheinLoginScope
	ActorID, MemberID, AttemptID, OwnerNonce string
	ExpectedStoreVersion                     int64
	MerchantID                               string
}
type SheinIdentityReceipt struct {
	Status       string `json:"status"`
	StoreVersion int64  `json:"store_version"`
}
type SheinLoginRepository struct {
	db         *gorm.DB
	authorizer StoreMemberAuthorizer
}

func (r *SheinLoginRepository) ReadStore(ctx context.Context, s SheinLoginScope, actor, member string) (out StoreSnapshot, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, e := r.authorizedStore(ctx, tx, s, actor, member)
		if e != nil {
			return e
		}
		aggregate, e := rehydrateRecord(row)
		if e != nil {
			return e
		}
		out = aggregate.Snapshot()
		return nil
	})
	return
}

func NewSheinLoginRepository(db *gorm.DB, a StoreMemberAuthorizer) (*SheinLoginRepository, error) {
	if db == nil || a == nil {
		return nil, ErrDependencyUnavailable
	}
	return &SheinLoginRepository{db, a}, nil
}

// All grant mutations already lock Store before the grant. Holding that Store
// lock fences grant revocation without giving this worker UPDATE on grant rows.
func (r *SheinLoginRepository) authorizedStore(ctx context.Context, tx *gorm.DB, s SheinLoginScope, actor, member string) (workbenchStoreRecord, error) {
	if e := s.Validate(); e != nil {
		return workbenchStoreRecord{}, e
	}
	a, e := (&MemberScopedStoreRepository{db: tx, access: r.authorizer}).authorize(ctx, s.OrganizationID, true)
	if e != nil || a.ActorID != actor || a.MemberID != member {
		return workbenchStoreRecord{}, ErrNotFound
	}
	var row workbenchStoreRecord
	e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND id=? AND deleted_at IS NULL", s.OrganizationID, s.StoreID).Take(&row).Error
	if e != nil {
		return row, ErrNotFound
	}
	if row.Platform != "shein" || row.RecordStatus != "active" {
		return row, ErrSheinLoginFenced
	}
	if !a.Administrator {
		var n int64
		e = tx.Model(&storeMemberGrantRow{}).Where("organization_id=? AND store_id=? AND member_id=? AND active=?", s.OrganizationID, s.StoreID, member, true).Count(&n).Error
		if e != nil || n != 1 {
			return row, ErrNotFound
		}
	}
	return row, nil
}
func bindingRow(tx *gorm.DB, s SheinLoginScope) (sheinLoginBindingRow, error) {
	var row sheinLoginBindingRow
	e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND store_id=?", s.OrganizationID, s.StoreID).Take(&row).Error
	return row, e
}
func matchesBinding(row sheinLoginBindingRow, s SheinLoginScope) bool {
	return row.InstallationID == s.InstallationID && row.BindingRevision == s.BindingRevision && row.ConfigFingerprint == s.ConfigFingerprint && row.Generation == s.Generation
}
func (r *SheinLoginRepository) Activate(ctx context.Context, c SheinBindingCommand) (SheinBinding, error) {
	return r.control(ctx, c, false)
}
func (r *SheinLoginRepository) Quiesce(ctx context.Context, c SheinBindingCommand) (SheinBinding, error) {
	return r.control(ctx, c, true)
}
func (r *SheinLoginRepository) control(ctx context.Context, c SheinBindingCommand, quiesce bool) (out SheinBinding, err error) {
	if _, e := canonicalUUID(c.RequestKey); e != nil {
		return out, e
	}
	action := AuditActionSheinActivate
	if quiesce {
		action = AuditActionSheinQuiesce
	}
	fp := hashTuple("shein-control-v1", c.Scope.OrganizationID, c.Scope.StoreID, c.Scope.InstallationID, c.Scope.BindingRevision, c.Scope.ConfigFingerprint, c.ActorID, c.MemberID, fmtInt(c.ExpectedGeneration), fmtInt(c.Scope.Generation), string(action))
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		st, e := r.authorizedStore(ctx, tx, c.Scope, c.ActorID, c.MemberID)
		if e != nil {
			return e
		}
		audit := &GormAuditRepository{db: tx}
		old, e := audit.Get(ctx, c.Scope.OrganizationID, c.RequestKey, action)
		if e == nil {
			if old.PayloadFingerprint != fp || old.StoreID != c.Scope.StoreID || old.ActorSubject != c.ActorID {
				return ErrIdempotencyConflict
			}
			out = SheinBinding{SheinLoginScope: c.Scope, State: "active", ClosedConfirmed: true}
			out.Generation = old.StoreVersion
			if quiesce {
				out.State = "quiescing"
			}
			return nil
		}
		if !errors.Is(e, ErrNotFound) {
			return e
		}
		row, e := bindingRow(tx, c.Scope)
		if errors.Is(e, gorm.ErrRecordNotFound) && !quiesce {
			if c.ExpectedGeneration != 0 || c.Scope.Generation != 0 {
				return ErrSheinLoginFenced
			}
			row = sheinLoginBindingRow{OrganizationID: c.Scope.OrganizationID, StoreID: c.Scope.StoreID, InstallationID: c.Scope.InstallationID, BindingRevision: c.Scope.BindingRevision, ConfigFingerprint: c.Scope.ConfigFingerprint, Generation: 1, State: "active", ClosedConfirmed: true}
			e = tx.Create(&row).Error
		} else if e == nil {
			if quiesce {
				if !matchesBinding(row, c.Scope) {
					return ErrSheinLoginFenced
				}
				row.State = "quiescing"
				if row.ActiveAttemptID == "" && row.ClosedConfirmed {
					row.State = "quiesced"
				}
			} else {
				if row.State != "quiesced" || !row.ClosedConfirmed || row.ActiveAttemptID != "" || row.OwnerNonce != "" || c.ExpectedGeneration != row.Generation || row.Generation == math.MaxInt64 {
					return ErrSheinLoginFenced
				}
				row.InstallationID = c.Scope.InstallationID
				row.BindingRevision = c.Scope.BindingRevision
				row.ConfigFingerprint = c.Scope.ConfigFingerprint
				row.Generation++
				row.State = "active"
			}
			e = tx.Model(&sheinLoginBindingRow{}).Where("organization_id=? AND store_id=?", row.OrganizationID, row.StoreID).Select("installation_id", "binding_revision", "config_fingerprint", "generation", "state", "closed_confirmed").Updates(&row).Error
		}
		if e != nil {
			return e
		}
		event := newAuditEvent(c.Scope.OrganizationID, c.Scope.StoreID, c.RequestKey, action, AuditOutcomeSucceeded, c.ActorID, nil, RecordStatus(st.RecordStatus), RecordStatus(st.RecordStatus), AuditFailureNone, time.Now().UTC())
		event.PayloadFingerprint = fp
		event.StoreVersion = row.Generation
		if _, _, e = audit.Record(ctx, event); e != nil {
			return e
		}
		out = row.binding()
		return nil
	})
	return
}
func fmtInt(n int64) string { return strconv.FormatInt(n, 10) }

func (r *SheinLoginRepository) ReadBinding(ctx context.Context, s SheinLoginScope, actor, member string) (out SheinBinding, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, e := r.authorizedStore(ctx, tx, s, actor, member); e != nil {
			return e
		}
		row, e := bindingRow(tx, s)
		if e != nil || row.InstallationID != s.InstallationID || row.BindingRevision != s.BindingRevision || row.ConfigFingerprint != s.ConfigFingerprint {
			return ErrSheinLoginFenced
		}
		out = row.binding()
		return nil
	})
	return
}
func (r *SheinLoginRepository) Claim(ctx context.Context, s SheinLoginScope, actor, member, attempt, nonce string) error {
	for _, v := range []string{attempt, nonce} {
		if _, e := canonicalUUID(v); e != nil {
			return e
		}
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, e := r.authorizedStore(ctx, tx, s, actor, member); e != nil {
			return e
		}
		row, e := bindingRow(tx, s)
		if e != nil || !matchesBinding(row, s) || row.State != "active" || row.ActiveAttemptID != "" || !row.ClosedConfirmed {
			return ErrSheinLoginFenced
		}
		return tx.Model(&row).Updates(map[string]any{"active_attempt_id": attempt, "owner_nonce": nonce, "closed_confirmed": false}).Error
	})
}
func (r *SheinLoginRepository) CheckOwner(ctx context.Context, s SheinLoginScope, actor, member, attempt, nonce string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, e := r.authorizedStore(ctx, tx, s, actor, member); e != nil {
			return e
		}
		row, e := bindingRow(tx, s)
		if e != nil || !matchesBinding(row, s) || row.State != "active" || row.ActiveAttemptID != attempt || row.OwnerNonce != nonce || row.ClosedConfirmed {
			return ErrSheinLoginFenced
		}
		return nil
	})
}

// ConfirmClosed is called only by the exact executor after synchronous closure
// of its browser and HTTP work. It deliberately has no TTL or force path.
func (r *SheinLoginRepository) ConfirmClosed(ctx context.Context, s SheinLoginScope, attempt, nonce string) error {
	if e := s.Validate(); e != nil {
		return e
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Same Store-first order; revocation must not prevent an owner closing itself.
		if e := lockMemberStore(tx, s.OrganizationID, s.StoreID); e != nil {
			return e
		}
		row, e := bindingRow(tx, s)
		if e != nil || !matchesBinding(row, s) || row.ActiveAttemptID != attempt || row.OwnerNonce != nonce || nonce == "" {
			return ErrSheinLoginFenced
		}
		state := row.State
		if state == "quiescing" {
			state = "quiesced"
		}
		return tx.Model(&row).Updates(map[string]any{"active_attempt_id": "", "owner_nonce": "", "closed_confirmed": true, "state": state}).Error
	})
}

func (r *SheinLoginRepository) RecordSheinLoginIdentity(ctx context.Context, c SheinIdentityCommand) (out SheinIdentityReceipt, err error) {
	if e := c.Scope.Validate(); e != nil {
		return out, e
	}
	if _, e := canonicalUUID(c.AttemptID); e != nil {
		return out, e
	}
	merchant, e := validateNormalizedUserValue("merchant", c.MerchantID, MaxExternalStoreIDCodePoints, true)
	if e != nil || c.ExpectedStoreVersion <= 0 {
		return out, ErrSheinLoginFenced
	}
	fp := hashTuple("shein-identity-v1", c.Scope.OrganizationID, c.Scope.StoreID, c.Scope.InstallationID, c.Scope.BindingRevision, c.Scope.ConfigFingerprint, fmtInt(c.Scope.Generation), c.ActorID, c.MemberID, c.AttemptID, fmtInt(c.ExpectedStoreVersion), merchant)
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		st, e := r.authorizedStore(ctx, tx, c.Scope, c.ActorID, c.MemberID)
		if e != nil {
			return e
		}
		audit := &GormAuditRepository{db: tx}
		old, e := audit.Get(ctx, c.Scope.OrganizationID, c.AttemptID, AuditActionSheinIdentitySync)
		if e == nil {
			if old.PayloadFingerprint != fp || old.StoreID != c.Scope.StoreID || old.ActorSubject != c.ActorID {
				return ErrIdempotencyConflict
			}
			out = SheinIdentityReceipt{"applied", old.StoreVersion}
			if old.Outcome != AuditOutcomeSucceeded {
				out.Status = "blocked"
			}
			return nil
		}
		if !errors.Is(e, ErrNotFound) {
			return e
		}
		row, e := bindingRow(tx, c.Scope)
		if e != nil || !matchesBinding(row, c.Scope) || row.State != "active" || row.ActiveAttemptID != c.AttemptID || row.OwnerNonce != c.OwnerNonce || row.ClosedConfirmed {
			return ErrSheinLoginFenced
		}
		out = SheinIdentityReceipt{"blocked", st.Version}
		if st.Version < math.MaxInt64 && st.Version == c.ExpectedStoreVersion && (st.ExternalStoreID == "" || st.ExternalStoreID == merchant) {
			if st.ExternalStoreID == merchant {
				out.Status = "applied"
			} else {
				if e = tx.SavePoint("shein_identity_update").Error; e != nil {
					return e
				}
				aggregate, e := rehydrateRecord(st)
				if e != nil {
					return e
				}
				snap := aggregate.Snapshot()
				snap.ExternalStoreID = merchant
				res := tx.Model(&workbenchStoreRecord{}).Where("organization_id=? AND id=? AND version=?", c.Scope.OrganizationID, c.Scope.StoreID, st.Version).Updates(map[string]any{"external_store_id": merchant, "identity_key": identityKey(snap), "version": st.Version + 1, "updated_by": c.ActorID, "updated_at": time.Now().UTC()})
				if res.Error != nil {
					if e = tx.RollbackTo("shein_identity_update").Error; e != nil {
						return e
					}
					if !isSheinUniqueConflict(res.Error) {
						return res.Error
					}
				} else if res.RowsAffected == 1 {
					out = SheinIdentityReceipt{"applied", st.Version + 1}
				}
			}
		}
		outcome, failure := AuditOutcomeSucceeded, AuditFailureNone
		if out.Status == "blocked" {
			outcome = AuditOutcomeFailed
			failure = AuditFailureAlreadyExists
		}
		event := newAuditEvent(c.Scope.OrganizationID, c.Scope.StoreID, c.AttemptID, AuditActionSheinIdentitySync, outcome, c.ActorID, []string{"external_store_id"}, RecordStatus(st.RecordStatus), RecordStatus(st.RecordStatus), failure, time.Now().UTC())
		event.PayloadFingerprint = fp
		event.StoreVersion = out.StoreVersion
		_, _, e = audit.Record(ctx, event)
		return e
	})
	return
}
func isSheinUniqueConflict(err error) bool {
	var e interface{ SQLState() string }
	return errors.As(err, &e) && e.SQLState() == "23505" || errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "UNIQUE constraint failed")
}
