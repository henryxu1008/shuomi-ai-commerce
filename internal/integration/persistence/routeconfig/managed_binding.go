package routeconfigstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
	credential "task-processor/internal/aicapability/channelcredential"
	d "task-processor/internal/aicapability/routeconfig"
	channelstore "task-processor/internal/integration/persistence/channelcredential"
)

const managedBindingsTable = "public.ai_enterprise_model_bindings"
const managedBindingReceiptsTable = "public.ai_enterprise_model_binding_receipts"

func (s *Store) LockManagedSelection(ctx context.Context, org, id string) error {
	if _, ok := s.db.Statement.ConnPool.(gorm.TxCommitter); !ok {
		return d.ErrUnavailable
	}
	b, err := s.ManagedBinding(ctx, org, id)
	if err != nil {
		return err
	}
	if s.db.WithContext(ctx).Exec("SET LOCAL lock_timeout='5s'").Error != nil ||
		s.db.WithContext(ctx).Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", "channel-current:"+org+":"+b.ConnectionID).Error != nil {
		return d.ErrUnavailable
	}
	return nil
}

func (s *Store) ManagedBinding(ctx context.Context, org, id string) (d.ManagedBinding, error) {
	var row d.ManagedBinding
	err := s.db.WithContext(ctx).Table(managedBindingsTable).Where("organization_id=? AND binding_id=?", org, id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return row, d.ErrNotFound
	}
	if err != nil {
		return row, d.ErrUnavailable
	}
	return row, nil
}

func (s *Store) ManagedBindings(ctx context.Context, org string) ([]d.ManagedBinding, error) {
	rows := []d.ManagedBinding{}
	err := s.db.WithContext(ctx).Table(managedBindingsTable).Where("organization_id=?", org).
		Order("binding_id").Limit(129).Find(&rows).Error
	if err != nil || len(rows) > 128 {
		return nil, d.ErrUnavailable
	}
	return rows, nil
}

type managedBindingReceipt struct {
	OrganizationID string
	ActorID        string
	RequestKey     string
	Fingerprint    string
	Payload        []byte
}

func (s *Store) ManagedBindingReceipt(ctx context.Context, org, actor, key string) (d.ManagedBinding, error) {
	return readManagedBindingReceipt(s.db.WithContext(ctx), org, actor, key, "")
}

func readManagedBindingReceipt(db *gorm.DB, org, actor, key, fingerprint string) (d.ManagedBinding, error) {
	var result d.ManagedBinding
	var receipt managedBindingReceipt
	err := db.Table(managedBindingReceiptsTable).Where("organization_id=? AND actor_id=? AND request_key=?", org, actor, key).Take(&receipt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return result, d.ErrNotFound
	}
	if err != nil {
		return result, d.ErrUnavailable
	}
	if fingerprint != "" && receipt.Fingerprint != fingerprint {
		return result, d.ErrConflict
	}
	if json.Unmarshal(receipt.Payload, &result) != nil || result.OrganizationID != org ||
		!credential.IsRegisteredConnection(result.ConnectionID) || !d.IsManagedBindingID(result.BindingID) {
		return d.ManagedBinding{}, d.ErrUnavailable
	}
	return result, nil
}

func (s *Store) WriteManagedBinding(ctx context.Context, cmd d.ManagedBindingCommand, validateTemplate func(string) error) (d.ManagedBinding, error) {
	var zero d.ManagedBinding
	attach := cmd.Input.Action == "attach"
	if ctx == nil || ctx.Err() != nil || !credential.ValidID(cmd.OrganizationID) ||
		!credential.ValidID(cmd.ActorID) || !credential.ValidID(cmd.RequestKey) ||
		!credential.IsRegisteredConnection(cmd.ConnectionID) || cmd.Input.ExpectedRevision < 1 ||
		attach && (cmd.Input.BindingID != "" || validateTemplate == nil) ||
		!attach && (cmd.Input.Action != "disable" || !d.IsManagedBindingID(cmd.Input.BindingID)) {
		return zero, d.ErrInvalid
	}
	raw, _ := json.Marshal(cmd)
	hash := sha256.Sum256(raw)
	fingerprint := "managed-binding-v1:" + hex.EncodeToString(hash[:])
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return zero, d.ErrUnavailable
	}
	defer tx.Rollback()
	if tx.Exec("SET LOCAL lock_timeout='5s'").Error != nil || tx.Exec("SET LOCAL statement_timeout='25s'").Error != nil {
		return zero, d.ErrUnavailable
	}
	// The connection lock is shared with original Key writes. The definition
	// writer reads the metadata view and never locks or reads a secret row.
	for _, lock := range []string{
		"model-binding-command:" + cmd.OrganizationID + ":" + cmd.ActorID + ":" + cmd.RequestKey,
		"model-binding-org:" + cmd.OrganizationID,
		"channel-current:" + cmd.OrganizationID + ":" + cmd.ConnectionID,
	} {
		if tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", lock).Error != nil {
			return zero, d.ErrUnavailable
		}
	}
	prior, err := readManagedBindingReceipt(tx, cmd.OrganizationID, cmd.ActorID, cmd.RequestKey, fingerprint)
	if !errors.Is(err, d.ErrNotFound) {
		return prior, err
	}
	channels, err := channelstore.New(tx, nil)
	if err != nil {
		return zero, d.ErrUnavailable
	}
	current, err := channels.Metadata(ctx, cmd.OrganizationID, cmd.ConnectionID)
	if errors.Is(err, credential.ErrNotFound) {
		return zero, d.ErrNotFound
	}
	if err != nil {
		return zero, d.ErrUnavailable
	}
	if current.Revision != cmd.Input.ExpectedRevision || attach && (!current.Enabled || !current.KeyPresent) {
		return zero, d.ErrConflict
	}
	local := &Store{db: tx}
	var result d.ManagedBinding
	if attach {
		r, err := channels.Registration(ctx, cmd.OrganizationID, cmd.ConnectionID)
		if err != nil {
			return zero, d.ErrUnavailable
		}
		if err = validateTemplate(r.Template); err != nil {
			return zero, err
		}
		existing, err := local.ManagedBindings(ctx, cmd.OrganizationID)
		if err != nil {
			return zero, err
		}
		for _, row := range existing {
			if row.Enabled && row.ConnectionID == cmd.ConnectionID {
				return zero, d.ErrConflict
			}
		}
		if len(existing) >= 128 {
			return zero, d.ErrNeedsConfiguration
		}
		result = d.ManagedBinding{OrganizationID: cmd.OrganizationID, BindingID: cmd.BindingID(),
			ConnectionID: cmd.ConnectionID, Template: r.Template, Enabled: true}
		err = tx.Table(managedBindingsTable).Create(&result).Error
		if err != nil {
			return zero, d.ErrUnavailable
		}
	} else {
		result, err = local.ManagedBinding(ctx, cmd.OrganizationID, cmd.BindingID())
		if err != nil {
			return zero, err
		}
		if result.ConnectionID != cmd.ConnectionID {
			return zero, d.ErrNotFound
		}
		if !result.Enabled {
			return zero, d.ErrConflict
		}
		write := tx.Table(managedBindingsTable).Where("organization_id=? AND binding_id=? AND enabled", result.OrganizationID, result.BindingID).Update("enabled", false)
		if write.Error != nil || write.RowsAffected != 1 {
			return zero, d.ErrUnavailable
		}
		result.Enabled = false
	}
	payload, _ := json.Marshal(result)
	receipt := managedBindingReceipt{OrganizationID: cmd.OrganizationID, ActorID: cmd.ActorID, RequestKey: cmd.RequestKey, Fingerprint: fingerprint, Payload: payload}
	if tx.Table(managedBindingReceiptsTable).Create(&receipt).Error != nil {
		return zero, d.ErrUnavailable
	}
	commit := s.commitTransaction
	if commit == nil {
		commit = func(tx *gorm.DB) error { return tx.Commit().Error }
	}
	if commit(tx) != nil {
		return zero, d.ErrOutcomeUnknown
	}
	return result, nil
}
