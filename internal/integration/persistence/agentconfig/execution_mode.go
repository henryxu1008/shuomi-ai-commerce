package agentconfigpersistence

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
)

const modeOperation = "set-execution-mode"

type executionModeRow struct {
	OrganizationID, SubjectKind, SubjectID, Mode string
	Revision                                     int64
	CreatedBy, UpdatedBy                         string
	CreatedAt, UpdatedAt                         time.Time
}

func (executionModeRow) TableName() string { return "agent_configuration.execution_modes" }

type modeSnapshotRow struct {
	ID, OrganizationID, ActorID, MemberID, SubjectKind, SubjectID       string
	DefinitionID, DefinitionVersion, ContextKind, ContextID, RequestKey string
	Mode                                                                string
	ModeRevision                                                        int64
	PolicyVersion, Fingerprint, Digest                                  string
	Payload                                                             []byte
	CreatedAt                                                           time.Time
}

func (modeSnapshotRow) TableName() string { return "agent_configuration.execution_mode_snapshots" }

func lockModeSubject(db *gorm.DB, org string, subject agentconfig.ModeSubject) error {
	key, err := json.Marshal(struct {
		Organization string
		Subject      agentconfig.ModeSubject
	}{org, subject})
	if err != nil {
		return agentconfig.ErrInvalid
	}
	return db.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", string(key)).Error
}

func readMode(db *gorm.DB, org string, subject agentconfig.ModeSubject) (agentconfig.ExecutionModeRecord, error) {
	var row executionModeRow
	err := db.Where("organization_id=? AND subject_kind=? AND subject_id=?", org, subject.Kind, subject.ID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return agentconfig.ExecutionModeRecord{Subject: subject, Mode: agent.ModeDefault, Revision: "0"}, nil
	}
	if err != nil {
		return agentconfig.ExecutionModeRecord{}, err
	}
	mode := agent.ExecutionMode(row.Mode)
	if !mode.Valid() || row.Revision <= 0 {
		return agentconfig.ExecutionModeRecord{}, agentconfig.ErrUnavailable
	}
	return agentconfig.ExecutionModeRecord{Subject: subject, Mode: mode, Revision: decimal(row.Revision), UpdatedBy: row.UpdatedBy, UpdatedAt: row.UpdatedAt}, nil
}

func (s *Store) ReadExecutionMode(ctx context.Context, identity agent.ExecutionIdentity, subject agentconfig.ResolvedModeSubject) (agentconfig.ExecutionModeRecord, error) {
	if !identity.Valid() || !subject.Valid() {
		return agentconfig.ExecutionModeRecord{}, agentconfig.ErrInvalid
	}
	return readMode(s.db.WithContext(ctx), identity.OrganizationID, subject.Subject)
}

func modeReceipt(db *gorm.DB, c agentconfig.ModeCommand, fp string) (agentconfig.ModeReceipt, bool, error) {
	var row commandRow
	err := db.Where("organization_id=? AND actor_id=? AND idempotency_key=?", c.Identity.OrganizationID, c.Identity.ActorID, c.Key).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return agentconfig.ModeReceipt{}, false, nil
	}
	if err != nil {
		return agentconfig.ModeReceipt{}, false, err
	}
	if row.Fingerprint != fp || row.Operation != modeOperation || row.AgentID != c.Subject.Subject.ID {
		return agentconfig.ModeReceipt{}, true, agentconfig.ErrConflict
	}
	var result agentconfig.ModeReceipt
	if json.Unmarshal(row.Receipt, &result) != nil || result.CommandID != row.CommandID || result.RequestKey != row.IdempotencyKey || result.Subject != c.Subject.Subject || result.Operation != modeOperation || result.Mode != c.Mode || result.BeforeRevision != decimal(row.BeforeRevision) || result.Revision != decimal(row.AfterRevision) {
		return result, true, agentconfig.ErrUnavailable
	}
	return result, true, nil
}

func (s *Store) ExecuteMode(ctx context.Context, c agentconfig.ModeCommand, eligibility ...func(context.Context) error) (agentconfig.ModeReceipt, error) {
	if c.Validate() != nil || len(eligibility) > 1 {
		return agentconfig.ModeReceipt{}, agentconfig.ErrInvalid
	}
	fp, err := hash(c)
	if err != nil {
		return agentconfig.ModeReceipt{}, err
	}
	if result, found, err := modeReceipt(s.db.WithContext(ctx), c, fp); found || err != nil {
		return result, err
	}
	if len(eligibility) == 1 && eligibility[0] != nil {
		if err := eligibility[0](ctx); err != nil {
			if result, found, readErr := modeReceipt(s.db.WithContext(ctx), c, fp); found || readErr != nil {
				return result, readErr
			}
			return agentconfig.ModeReceipt{}, err
		}
	}
	var result agentconfig.ModeReceipt
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockModeSubject(tx, c.Identity.OrganizationID, c.Subject.Subject); err != nil {
			return err
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		claim := commandRow{OrganizationID: c.Identity.OrganizationID, ActorID: c.Identity.ActorID, IdempotencyKey: c.Key, CommandID: uuid.NewString(), Operation: modeOperation, AgentID: c.Subject.Subject.ID, Fingerprint: fp, Receipt: []byte("{}"), CommittedAt: now}
		insert := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&claim)
		if insert.Error != nil {
			return insert.Error
		}
		if insert.RowsAffected == 0 {
			var found bool
			var err error
			result, found, err = modeReceipt(tx, c, fp)
			if err != nil {
				return err
			}
			if !found {
				return agentconfig.ErrUnavailable
			}
			return nil
		}
		current, err := readMode(tx, c.Identity.OrganizationID, c.Subject.Subject)
		if err != nil {
			return err
		}
		revision, err := agentconfig.ParseModeRevision(current.Revision)
		if err != nil {
			return err
		}
		if revision != c.ExpectedRevision {
			return agentconfig.ErrRevision
		}
		noop := revision > 0 && current.Mode == c.Mode
		if revision == math.MaxInt64 && !noop {
			return agentconfig.ErrChanged
		}
		after := revision
		if !noop {
			after++
		}
		if revision == 0 {
			row := executionModeRow{OrganizationID: c.Identity.OrganizationID, SubjectKind: string(c.Subject.Subject.Kind), SubjectID: c.Subject.Subject.ID, Mode: string(c.Mode), Revision: int64(after), CreatedBy: c.Identity.ActorID, UpdatedBy: c.Identity.ActorID, CreatedAt: now, UpdatedAt: now}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		} else if !noop {
			update := tx.Model(&executionModeRow{}).Where("organization_id=? AND subject_kind=? AND subject_id=? AND revision=?", c.Identity.OrganizationID, c.Subject.Subject.Kind, c.Subject.Subject.ID, revision).Updates(map[string]any{"mode": string(c.Mode), "revision": after, "updated_by": c.Identity.ActorID, "updated_at": now})
			if update.Error != nil {
				return update.Error
			}
			if update.RowsAffected != 1 {
				return agentconfig.ErrRevision
			}
		}
		result = agentconfig.ModeReceipt{CommandID: claim.CommandID, RequestKey: claim.IdempotencyKey, Operation: modeOperation, Subject: c.Subject.Subject, Mode: c.Mode, BeforeRevision: decimal(int64(revision)), Revision: decimal(int64(after)), Noop: noop, CommittedAt: now}
		result.UpdatedBy, result.UpdatedAt = c.Identity.ActorID, now
		if noop {
			// Command commit time describes this save attempt. The configuration
			// row did not change, so its audit belongs to the original updater.
			result.UpdatedBy, result.UpdatedAt = current.UpdatedBy, current.UpdatedAt.UTC()
		}
		raw, err := json.Marshal(result)
		if err != nil || len(raw) > 4096 {
			return agentconfig.ErrInvalid
		}
		return tx.Model(&commandRow{}).Where("command_id=?", claim.CommandID).Updates(map[string]any{"before_revision": revision, "after_revision": after, "receipt": raw}).Error
	})
	if err != nil {
		return agentconfig.ModeReceipt{}, err
	}
	return result, nil
}

func modeSnapshotView(row modeSnapshotRow) (agentconfig.ModeSnapshot, error) {
	var snapshot agentconfig.ModeSnapshot
	if len(row.Payload) > 8192 || digest(row.Payload) != row.Digest || json.Unmarshal(row.Payload, &snapshot) != nil {
		return snapshot, agentconfig.ErrUnavailable
	}
	snapshot.Digest = row.Digest
	if !snapshot.Ref().Valid() || !snapshot.Identity.Valid() || !snapshot.Subject.Valid() || !snapshot.Binding.Valid() || !snapshot.Mode.Valid() || snapshot.ID != row.ID || snapshot.Identity.OrganizationID != row.OrganizationID || snapshot.Identity.ActorID != row.ActorID || snapshot.Identity.MemberID != row.MemberID || string(snapshot.Subject.Subject.Kind) != row.SubjectKind || snapshot.Subject.Subject.ID != row.SubjectID || snapshot.Subject.Definition.ID != row.DefinitionID || snapshot.Subject.Definition.Version != row.DefinitionVersion || snapshot.Binding.ContextKind != row.ContextKind || snapshot.Binding.ContextID != row.ContextID || snapshot.Binding.RequestKey != row.RequestKey || string(snapshot.Mode) != row.Mode || snapshot.Revision != decimal(row.ModeRevision) || snapshot.PolicyVersion != row.PolicyVersion || snapshot.PolicyVersion != agent.ExecutionModePolicyVersion || !snapshot.CreatedAt.Equal(row.CreatedAt) {
		return snapshot, agentconfig.ErrUnavailable
	}
	return snapshot, nil
}

func (s *Store) PrepareMode(ctx context.Context, c agentconfig.ModeStartCommand) (agentconfig.ModeSnapshot, error) {
	var result agentconfig.ModeSnapshot
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = prepareModeInTransaction(tx, c)
		return err
	})
	return result, err
}

func prepareModeInTransaction(tx *gorm.DB, c agentconfig.ModeStartCommand) (agentconfig.ModeSnapshot, error) {
	if c.Validate() != nil {
		return agentconfig.ModeSnapshot{}, agentconfig.ErrInvalid
	}
	fp, err := hash(c)
	if err != nil {
		return agentconfig.ModeSnapshot{}, err
	}
	var result agentconfig.ModeSnapshot
	err = func() error {
		if err := lockModeSubject(tx, c.Identity.OrganizationID, c.Subject.Subject); err != nil {
			return err
		}
		var row modeSnapshotRow
		err := tx.Where("organization_id=? AND actor_id=? AND subject_kind=? AND subject_id=? AND context_kind=? AND context_id=? AND request_key=?", c.Identity.OrganizationID, c.Identity.ActorID, c.Subject.Subject.Kind, c.Subject.Subject.ID, c.Binding.ContextKind, c.Binding.ContextID, c.Binding.RequestKey).Take(&row).Error
		if err == nil {
			if row.Fingerprint != fp {
				return agentconfig.ErrConflict
			}
			result, err = modeSnapshotView(row)
			return err
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		current, err := readMode(tx, c.Identity.OrganizationID, c.Subject.Subject)
		if err != nil {
			return err
		}
		revision, err := agentconfig.ParseModeRevision(current.Revision)
		if err != nil {
			return err
		}
		if revision != c.ExpectedRevision {
			return agentconfig.ErrChanged
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		result = agentconfig.ModeSnapshot{ID: uuid.NewString(), Identity: c.Identity, Subject: c.Subject, Binding: c.Binding, Mode: current.Mode, Revision: current.Revision, PolicyVersion: agent.ExecutionModePolicyVersion, CreatedAt: now}
		raw, err := json.Marshal(result)
		if err != nil || len(raw) > 8192 {
			return agentconfig.ErrInvalid
		}
		result.Digest = digest(raw)
		row = modeSnapshotRow{ID: result.ID, OrganizationID: c.Identity.OrganizationID, ActorID: c.Identity.ActorID, MemberID: c.Identity.MemberID, SubjectKind: string(c.Subject.Subject.Kind), SubjectID: c.Subject.Subject.ID, DefinitionID: c.Subject.Definition.ID, DefinitionVersion: c.Subject.Definition.Version, ContextKind: c.Binding.ContextKind, ContextID: c.Binding.ContextID, RequestKey: c.Binding.RequestKey, Mode: string(result.Mode), ModeRevision: int64(revision), PolicyVersion: result.PolicyVersion, Fingerprint: fp, Digest: result.Digest, Payload: raw, CreatedAt: now}
		return tx.Create(&row).Error
	}()
	if err != nil {
		return agentconfig.ModeSnapshot{}, err
	}
	return result, nil
}

func (s *Store) LoadModeSnapshot(ctx context.Context, identity agent.ExecutionIdentity, ref agent.ModeSnapshotRef) (agentconfig.ModeSnapshot, error) {
	if !identity.Valid() || !ref.Valid() {
		return agentconfig.ModeSnapshot{}, agentconfig.ErrInvalid
	}
	var row modeSnapshotRow
	err := s.db.WithContext(ctx).Where("id=? AND digest=? AND organization_id=? AND actor_id=? AND member_id=?", ref.ID, ref.Digest, identity.OrganizationID, identity.ActorID, identity.MemberID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return agentconfig.ModeSnapshot{}, agentconfig.ErrNotFound
	}
	if err != nil {
		return agentconfig.ModeSnapshot{}, err
	}
	return modeSnapshotView(row)
}

func checkModeInTransaction(tx *gorm.DB, snapshot agentconfig.ModeSnapshot) error {
	if err := lockModeSubject(tx, snapshot.Identity.OrganizationID, snapshot.Subject.Subject); err != nil {
		return err
	}
	current, err := readMode(tx, snapshot.Identity.OrganizationID, snapshot.Subject.Subject)
	if err != nil {
		return err
	}
	if current.Revision != snapshot.Revision || current.Mode != snapshot.Mode {
		return agentconfig.ErrChanged
	}
	return nil
}

func (s *Store) CheckModeSnapshot(ctx context.Context, identity agent.ExecutionIdentity, ref agent.ModeSnapshotRef) (agentconfig.ModeSnapshot, error) {
	snapshot, err := s.LoadModeSnapshot(ctx, identity, ref)
	if err != nil {
		return snapshot, err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return checkModeInTransaction(tx, snapshot) })
	if err != nil {
		return agentconfig.ModeSnapshot{}, err
	}
	return snapshot, nil
}

var _ agentconfig.ModeRepository = (*Store)(nil)
