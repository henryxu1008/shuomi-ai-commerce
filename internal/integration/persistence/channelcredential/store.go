// Package channelcredentialstore persists the single current organization
// connection and immutable command receipts in the original Run database.
package channelcredentialstore

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	credential "task-processor/internal/aicapability/channelcredential"
)

type ConnectionRow struct {
	OrganizationID   string `gorm:"type:varchar(128);primaryKey"`
	ConnectionID     string `gorm:"type:varchar(128);primaryKey"`
	Provider         string `gorm:"type:varchar(128);not null"`
	Channel          string `gorm:"type:varchar(128);not null"`
	Account          string `gorm:"type:varchar(128);not null"`
	Region           string `gorm:"type:varchar(128);not null"`
	BaseURL          string `gorm:"type:varchar(2048);not null"`
	Revision         int64  `gorm:"not null;check:channel_revision_positive,revision > 0"`
	Enabled          bool   `gorm:"not null"`
	ProtectionKeyID  string `gorm:"type:varchar(128);not null"`
	SecretCiphertext string `gorm:"type:varchar(16384);not null"`
}

func (ConnectionRow) TableName() string { return "ai_channel_connections" }

type ReceiptRow struct {
	OrganizationID string `gorm:"type:varchar(128);primaryKey"`
	ActorID        string `gorm:"type:varchar(128);primaryKey"`
	RequestKey     string `gorm:"type:varchar(128);primaryKey"`
	Fingerprint    string `gorm:"type:varchar(128);not null"`
	Payload        []byte `gorm:"type:bytea;not null;check:channel_receipt_payload_bound,octet_length(payload) <= 16384"`
}

func (ReceiptRow) TableName() string { return "ai_channel_connection_receipts" }

type Store struct {
	db                *gorm.DB
	protection        *Protection
	commitTransaction func(*gorm.DB) error
}

func New(db *gorm.DB, protection *Protection) (*Store, error) {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return nil, credential.ErrUnavailable
	}
	return &Store{db: db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}), protection: protection, commitTransaction: func(tx *gorm.DB) error { return tx.Commit().Error }}, nil
}

func metadata(r ConnectionRow) credential.Metadata {
	return (credential.Metadata{OrganizationID: r.OrganizationID, ConnectionID: r.ConnectionID, Provider: r.Provider, Channel: r.Channel, Account: r.Account, Region: r.Region, BaseURL: r.BaseURL, Revision: r.Revision, Enabled: r.Enabled, KeyPresent: r.SecretCiphertext != ""}).WithIdentity()
}

func (s *Store) Metadata(ctx context.Context, org, id string) (credential.Metadata, error) {
	if s == nil || ctx == nil || ctx.Err() != nil || !credential.ValidID(org) || !credential.ValidID(id) {
		return credential.Metadata{}, credential.ErrUnavailable
	}
	var m credential.Metadata
	e := s.db.WithContext(ctx).Table("public.ai_channel_credential_bindings").Where("organization_id=? AND connection_id=?", org, id).Take(&m).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return m, credential.ErrNotFound
	}
	if e != nil {
		return m, credential.ErrUnavailable
	}
	if m.OrganizationID != org || m.ConnectionID != id {
		return credential.Metadata{}, credential.ErrNotFound
	}
	return m.WithIdentity(), nil
}

func (s *Store) Receipt(ctx context.Context, org, actor, key string) (credential.Metadata, error) {
	if s == nil || ctx == nil || ctx.Err() != nil || !credential.ValidID(org) || !credential.ValidID(actor) || !credential.ValidID(key) {
		return credential.Metadata{}, credential.ErrUnavailable
	}
	return receipt(s.db.WithContext(ctx), org, actor, key, "")
}

func receipt(db *gorm.DB, org, actor, key, fingerprint string) (credential.Metadata, error) {
	var r ReceiptRow
	e := db.Where("organization_id=? AND actor_id=? AND request_key=?", org, actor, key).Take(&r).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return credential.Metadata{}, credential.ErrNotFound
	}
	if e != nil {
		return credential.Metadata{}, credential.ErrUnavailable
	}
	if fingerprint != "" && r.Fingerprint != fingerprint {
		return credential.Metadata{}, credential.ErrConflict
	}
	var m credential.Metadata
	if len(r.Payload) > 16384 || json.Unmarshal(r.Payload, &m) != nil || m.OrganizationID != org || m.Revision < 1 || m.Reference == "" {
		return m, credential.ErrUnavailable
	}
	return m, nil
}

func (s *Store) Resolve(ctx context.Context, org, id, reference string) (credential.Secret, error) {
	if s == nil || s.protection == nil || ctx == nil || ctx.Err() != nil || !credential.ValidID(org) || !credential.ValidID(id) {
		return credential.Secret{}, credential.ErrUnavailable
	}
	// One SELECT supplies both metadata and secret at one MVCC observation.
	var r ConnectionRow
	e := s.db.WithContext(ctx).Where("organization_id=? AND connection_id=?", org, id).Take(&r).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return credential.Secret{}, credential.ErrNotFound
	}
	if e != nil {
		return credential.Secret{}, credential.ErrUnavailable
	}
	if r.OrganizationID != org || r.ConnectionID != id {
		return credential.Secret{}, credential.ErrNotFound
	}
	m := metadata(r)
	if reference == "" || reference != m.Reference || !r.Enabled || !m.KeyPresent {
		return credential.Secret{}, credential.ErrChanged
	}
	plain, e := s.protection.cipher.Open(r.ProtectionKeyID, r.SecretCiphertext, associatedData(m))
	if e != nil || len(plain) == 0 || len(plain) > 4096 {
		return credential.Secret{}, credential.ErrUnavailable
	}
	return credential.Secret{Metadata: m, APIKey: string(plain)}, nil
}

func (s *Store) Publish(ctx context.Context, c credential.Command) (credential.Metadata, error) {
	if c.Validate() != nil {
		return credential.Metadata{}, credential.ErrInvalid
	}
	if s == nil || s.protection == nil || ctx == nil || ctx.Err() != nil {
		return credential.Metadata{}, credential.ErrUnavailable
	}
	fingerprint, e := s.protection.Fingerprint(c)
	if e != nil {
		return credential.Metadata{}, e
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return credential.Metadata{}, credential.ErrUnavailable
	}
	defer tx.Rollback()
	if tx.Exec("SET LOCAL lock_timeout='5s'").Error != nil || tx.Exec("SET LOCAL statement_timeout='25s'").Error != nil {
		return credential.Metadata{}, credential.ErrUnavailable
	}
	// Reuse the original publication pattern: bounded transaction-scoped keys,
	// receipt first, then the exact current row. No network call under a lock.
	for _, lock := range []string{"channel-command:" + c.OrganizationID + ":" + c.ActorID + ":" + c.Key, "channel-current:" + c.OrganizationID + ":" + c.ConnectionID} {
		if tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", lock).Error != nil {
			return credential.Metadata{}, credential.ErrUnavailable
		}
	}
	prior, e := receipt(tx, c.OrganizationID, c.ActorID, c.Key, fingerprint)
	if e == nil {
		return prior, nil
	}
	if !errors.Is(e, credential.ErrNotFound) {
		return credential.Metadata{}, e
	}
	var current ConnectionRow
	e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND connection_id=?", c.OrganizationID, c.ConnectionID).Take(&current).Error
	if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
		return credential.Metadata{}, credential.ErrUnavailable
	}
	exists := e == nil
	if exists && (current.Revision != c.ExpectedRevision || current.Revision == math.MaxInt64) || !exists && c.ExpectedRevision != 0 {
		return credential.Metadata{}, credential.ErrConflict
	}
	if exists && (current.Provider != c.Provider || current.Channel != c.Channel || current.Account != c.Account || current.Region != c.Region || current.BaseURL != c.BaseURL) {
		return credential.Metadata{}, credential.ErrConflict
	}
	secret := c.Secret
	if !c.Enabled {
		if !exists {
			return credential.Metadata{}, credential.ErrNotFound
		}
		plain, e := s.protection.cipher.Open(current.ProtectionKeyID, current.SecretCiphertext, associatedData(metadata(current)))
		if e != nil {
			return credential.Metadata{}, credential.ErrUnavailable
		}
		secret = string(plain)
	}
	r := ConnectionRow{OrganizationID: c.OrganizationID, ConnectionID: c.ConnectionID, Provider: c.Provider, Channel: c.Channel, Account: c.Account, Region: c.Region, BaseURL: c.BaseURL, Revision: c.ExpectedRevision + 1, Enabled: c.Enabled, ProtectionKeyID: s.protection.cipher.KeyID()}
	m := metadata(r)
	m.KeyPresent = true
	r.SecretCiphertext, e = s.protection.cipher.Seal([]byte(secret), associatedData(m))
	if e != nil {
		return credential.Metadata{}, credential.ErrUnavailable
	}
	if exists {
		result := tx.Model(&ConnectionRow{}).Where("organization_id=? AND connection_id=? AND revision=?", c.OrganizationID, c.ConnectionID, c.ExpectedRevision).Select("revision", "enabled", "protection_key_id", "secret_ciphertext").Updates(&r)
		e = result.Error
		if e == nil && result.RowsAffected != 1 {
			return credential.Metadata{}, credential.ErrConflict
		}
	} else {
		e = tx.Create(&r).Error
	}
	if e != nil {
		return credential.Metadata{}, credential.ErrUnavailable
	}
	payload, e := json.Marshal(m)
	if e != nil {
		return credential.Metadata{}, credential.ErrUnavailable
	}
	if tx.Create(&ReceiptRow{OrganizationID: c.OrganizationID, ActorID: c.ActorID, RequestKey: c.Key, Fingerprint: fingerprint, Payload: payload}).Error != nil {
		return credential.Metadata{}, credential.ErrUnavailable
	}
	if s.commitTransaction(tx) != nil {
		return credential.Metadata{}, credential.ErrOutcomeUnknown
	}
	return m, nil
}
