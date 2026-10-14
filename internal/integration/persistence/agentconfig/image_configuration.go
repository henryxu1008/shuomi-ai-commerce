package agentconfigpersistence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
)

type visualConfigurationProfile struct {
	agentID, version, context string
	artwork                   bool
}

var productImageProfile = visualConfigurationProfile{agentconfig.ImageAgentID, agentconfig.ImageAgentVersion, agentconfig.ImageSnapshotContext, false}
var artworkProfile = visualConfigurationProfile{agentconfig.ArtworkAgentID, agentconfig.ArtworkAgentVersion, agentconfig.ArtworkSnapshotContext, true}

type visualConfigurationSnapshot struct {
	agentconfig.ImageConfigurationSnapshot
	ArtworkParameters agentconfig.ArtworkTemplate `json:"-"`
}

func validImageStart(c agentconfig.ImageStartCommand) bool {
	return validVisualStart(c, productImageProfile)
}
func validVisualStart(c agentconfig.ImageStartCommand, profile visualConfigurationProfile) bool {
	if !scopeOK(c.Scope) || !agent.ValidID(c.MemberID) || !agentconfig.UUID(c.RequestKey) || !agentconfig.UUID(c.RunID) || !agent.ValidID(c.ContextID) || (!profile.artwork && c.TargetPlatform != "product" && !agentconfig.Platform(c.TargetPlatform) || profile.artwork && (c.TargetPlatform != "sds" || c.SourceDigest != c.InputDigest || c.HardLimits.Images != 1)) || !agentconfig.ImageDigest(c.SourceDigest) || !agentconfig.ImageDigest(c.InputDigest) || !c.HardLimits.Valid() {
		return false
	}
	if c.Template != nil {
		if !agentconfig.UUID(c.Template.TemplateID) {
			return false
		}
		if _, err := version(c.Template.Revision); err != nil {
			return false
		}
	}
	return true
}

func imageSnapshotView(tx *gorm.DB, row snapshotRow) (agentconfig.ImageConfigurationSnapshot, error) {
	value, err := visualSnapshotView(tx, row, productImageProfile)
	return value.ImageConfigurationSnapshot, err
}
func visualParameters(t agentconfig.Template, profile visualConfigurationProfile) ([]byte, error) {
	if profile.artwork {
		if t.Artwork == nil || t.Image != nil || t.Content != nil || t.TargetPlatform != "sds" {
			return nil, agentconfig.ErrInvalid
		}
		return json.Marshal(t.Artwork)
	}
	if t.Image == nil || t.Artwork != nil {
		return nil, agentconfig.ErrInvalid
	}
	return json.Marshal(t.Image)
}
func visualSnapshotView(tx *gorm.DB, row snapshotRow, profile visualConfigurationProfile) (visualConfigurationSnapshot, error) {
	var value visualConfigurationSnapshot
	if len(row.Payload) > 8192 || row.AgentID != profile.agentID || row.ContextKind != profile.context || row.Digest != digest(row.Payload) || json.Unmarshal(row.Payload, &value) != nil {
		return value, agentconfig.ErrUnavailable
	}
	if row.ID != value.ID || row.OrganizationID != value.Scope.OrganizationID || row.ActorID != value.Scope.ActorID || row.ContextID != value.ContextID || row.RequestKey != value.RequestKey || row.AgentVersion != profile.version || value.AgentVersion != row.AgentVersion || value.Epoch != decimal(row.ActivationEpoch) || value.AgentRevision != decimal(row.AgentRevision) || text(row.TemplateID) != value.Template.TemplateID || row.TemplateRevision == nil || decimal(*row.TemplateRevision) != value.Template.Revision {
		return value, agentconfig.ErrUnavailable
	}
	if !validVisualStart(agentconfig.ImageStartCommand{Scope: value.Scope, MemberID: value.MemberID, RequestKey: value.RequestKey, ContextID: value.ContextID, RunID: value.RunID, TargetPlatform: value.TargetPlatform, SourceDigest: value.SourceDigest, InputDigest: value.InputDigest, Template: &value.Template, HardLimits: value.HardLimits}, profile) {
		return value, agentconfig.ErrUnavailable
	}
	t, err := findTemplate(tx, value.Scope.OrganizationID, profile.agentID, value.Template.TemplateID, false)
	if err != nil {
		return value, err
	}
	parameters, err := templateView(tx, t, *row.TemplateRevision)
	if err != nil || (parameters.TargetPlatform != "product" && parameters.TargetPlatform != value.TargetPlatform) {
		return value, agentconfig.ErrUnavailable
	}
	raw, err := visualParameters(parameters, profile)
	if err != nil || digest(raw) != value.ParametersDigest {
		return value, agentconfig.ErrUnavailable
	}
	value.Digest = row.Digest
	if profile.artwork {
		value.ArtworkParameters = *parameters.Artwork
	} else {
		value.Parameters = agentconfig.CloneSetTemplate(*parameters.Image)
	}
	return value, nil
}

func (s *Store) PrepareImageConfiguration(ctx context.Context, c agentconfig.ImageStartCommand) (agentconfig.ImageConfigurationSnapshot, error) {
	value, err := s.prepareVisualConfiguration(ctx, c, productImageProfile)
	return value.ImageConfigurationSnapshot, err
}
func (s *Store) prepareVisualConfiguration(ctx context.Context, c agentconfig.ImageStartCommand, profile visualConfigurationProfile) (visualConfigurationSnapshot, error) {
	modeRevision, modeErr := agentconfig.ParseModeRevision(c.ExpectedModeRevision)
	if modeErr != nil {
		return visualConfigurationSnapshot{}, agentconfig.ErrInvalid
	}
	if !validVisualStart(c, profile) {
		return visualConfigurationSnapshot{}, agentconfig.ErrInvalid
	}
	// Deployment ceilings are frozen output, not a retry input that rewrites it.
	public := c
	public.HardLimits = agentconfig.ImageRunLimits{}
	fingerprint, err := hash(public)
	if err != nil {
		return visualConfigurationSnapshot{}, err
	}
	var result visualConfigurationSnapshot
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		a, agentErr := findAgent(tx, c.Scope.OrganizationID, profile.agentID, true)
		var old snapshotRow
		err := tx.Where("organization_id=? AND actor_id=? AND context_kind=? AND context_id=? AND request_key=?", c.Scope.OrganizationID, c.Scope.ActorID, profile.context, c.ContextID, c.RequestKey).Take(&old).Error
		if err == nil {
			if old.Fingerprint != fingerprint {
				return agentconfig.ErrConflict
			}
			result, err = visualSnapshotView(tx, old, profile)
			return err
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if errors.Is(agentErr, agentconfig.ErrNotFound) || agentErr == nil && a.Activation != "ENABLED" {
			return agentconfig.ErrNotEnabled
		}
		if agentErr != nil {
			return agentErr
		}
		ref := c.Template
		if ref == nil && a.DefaultTemplateID != nil && a.DefaultTemplateRevision != nil {
			ref = &agentconfig.TemplateRef{TemplateID: *a.DefaultTemplateID, Revision: decimal(*a.DefaultTemplateRevision)}
		}
		if ref == nil {
			return agentconfig.ErrNotFound
		}
		t, err := findTemplate(tx, c.Scope.OrganizationID, profile.agentID, ref.TemplateID, true)
		if err != nil {
			return err
		}
		if t.Lifecycle != "ACTIVE" {
			return agentconfig.ErrArchived
		}
		revision, err := version(ref.Revision)
		if err != nil {
			return err
		}
		parameters, err := templateView(tx, t, revision)
		if err != nil {
			return err
		}
		if parameters.TargetPlatform != "product" && parameters.TargetPlatform != c.TargetPlatform {
			return agentconfig.ErrInvalid
		}
		parameterBytes, err := visualParameters(parameters, profile)
		if err != nil {
			return err
		}
		result.ImageConfigurationSnapshot = agentconfig.ImageConfigurationSnapshot{ID: uuid.NewString(), Scope: c.Scope, MemberID: c.MemberID, RequestKey: c.RequestKey, ContextID: c.ContextID, RunID: c.RunID, TargetPlatform: c.TargetPlatform, AgentVersion: profile.version, Epoch: decimal(a.ActivationEpoch), AgentRevision: decimal(a.Revision), SourceDigest: c.SourceDigest, InputDigest: c.InputDigest, ParametersDigest: digest(parameterBytes), Template: *ref, HardLimits: c.HardLimits}
		mode, err := prepareModeInTransaction(tx, agentconfig.ModeStartCommand{Identity: agent.ExecutionIdentity{OrganizationID: c.Scope.OrganizationID, ActorID: c.Scope.ActorID, MemberID: c.MemberID}, Subject: agentconfig.ResolvedModeSubject{Subject: agentconfig.ModeSubject{Kind: agentconfig.RegisteredAgentSubject, ID: profile.agentID}, Definition: commercetool.AgentRef{ID: profile.agentID, Version: profile.version}}, Binding: agentconfig.ModeBinding{ContextKind: profile.context, ContextID: c.ContextID, RequestKey: c.RequestKey, IntentDigest: fingerprint}, ExpectedRevision: modeRevision})
		if err != nil {
			return err
		}
		result.ExpectedModeRevision, result.ModeSnapshotRef = c.ExpectedModeRevision, mode.Ref()
		raw, err := json.Marshal(result)
		if err != nil || len(raw) > 8192 {
			return agentconfig.ErrInvalid
		}
		row := snapshotRow{ID: result.ID, OrganizationID: c.Scope.OrganizationID, ActorID: c.Scope.ActorID, ContextKind: profile.context, ContextID: c.ContextID, RequestKey: c.RequestKey, AgentID: profile.agentID, AgentVersion: profile.version, Fingerprint: fingerprint, Payload: raw, Digest: digest(raw), ActivationEpoch: a.ActivationEpoch, AgentRevision: a.Revision, TemplateID: &ref.TemplateID, TemplateRevision: &revision, CreatedAt: time.Now().UTC()}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		result.Digest = row.Digest
		if profile.artwork {
			result.ArtworkParameters = *parameters.Artwork
		} else {
			result.Parameters = agentconfig.CloneSetTemplate(*parameters.Image)
		}
		return nil
	})
	return result, err
}

func (s *Store) LoadImageConfiguration(ctx context.Context, scope agent.Scope, ref agent.ConfigurationSnapshotRef) (agentconfig.ImageConfigurationSnapshot, error) {
	value, err := s.loadVisualConfiguration(ctx, scope, ref, productImageProfile)
	return value.ImageConfigurationSnapshot, err
}
func (s *Store) loadVisualConfiguration(ctx context.Context, scope agent.Scope, ref agent.ConfigurationSnapshotRef, profile visualConfigurationProfile) (visualConfigurationSnapshot, error) {
	if !scopeOK(scope) || ref.Kind != agentconfig.SnapshotKind || !agentconfig.UUID(ref.ID) || !agentconfig.ImageDigest(ref.Digest) {
		return visualConfigurationSnapshot{}, agentconfig.ErrInvalid
	}
	var row snapshotRow
	if err := s.db.WithContext(ctx).Where("id=? AND organization_id=? AND actor_id=?", ref.ID, scope.OrganizationID, scope.ActorID).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return visualConfigurationSnapshot{}, agentconfig.ErrNotFound
		}
		return visualConfigurationSnapshot{}, err
	}
	if row.Digest != ref.Digest {
		return visualConfigurationSnapshot{}, agentconfig.ErrConflict
	}
	return visualSnapshotView(s.db.WithContext(ctx), row, profile)
}

type imageAdmissionRow struct {
	SnapshotID, OrganizationID, ActorID, MemberID, RunID, ActionID, ID, Fingerprint, Digest string
	Payload                                                                                 []byte
	AdmittedAt, Deadline                                                                    time.Time
}

func (imageAdmissionRow) TableName() string { return "agent_configuration.image_run_admissions" }

func imageAdmissionView(row imageAdmissionRow) (agentconfig.ImageRunAdmissionReceipt, error) {
	var receipt agentconfig.ImageRunAdmissionReceipt
	if len(row.Payload) > 8192 || digest(row.Payload) != row.Digest || json.Unmarshal(row.Payload, &receipt) != nil {
		return receipt, agentconfig.ErrUnavailable
	}
	c := receipt.Command
	fingerprint, err := hash(c)
	if err != nil || fingerprint != row.Fingerprint || receipt.ID != row.ID || c.Snapshot.ID != row.SnapshotID || c.Scope.OrganizationID != row.OrganizationID || c.Scope.ActorID != row.ActorID || c.MemberID != row.MemberID || c.RunID != row.RunID || c.ConfirmActionID != row.ActionID || !receipt.AdmittedAt.Equal(row.AdmittedAt) || !receipt.Deadline.Equal(row.Deadline) || !c.Limits.Valid() || receipt.Deadline.Sub(receipt.AdmittedAt) != time.Duration(c.Limits.ElapsedSeconds)*time.Second {
		return receipt, agentconfig.ErrUnavailable
	}
	receipt.Digest = row.Digest
	return receipt, nil
}

func validImageAdmission(c agentconfig.ImageRunAdmissionCommand) bool {
	return scopeOK(c.Scope) && c.Snapshot.Kind == agentconfig.SnapshotKind && agentconfig.UUID(c.Snapshot.ID) && agentconfig.ImageDigest(c.Snapshot.Digest) && agent.ValidID(c.MemberID) && agentconfig.UUID(c.RunID) && agentconfig.UUID(c.ConfirmActionID) && agentconfig.ImageDigest(c.SourceDigest) && agentconfig.ImageDigest(c.InputDigest) && agentconfig.ImageDigest(c.PlanDigest) && agentconfig.ImageDigest(c.QuoteDigest) && c.Limits.Valid()
}

func (s *Store) AdmitImageRun(ctx context.Context, c agentconfig.ImageRunAdmissionCommand, ceiling agentconfig.ImageRunLimits) (agentconfig.ImageRunAdmissionReceipt, error) {
	return s.admitVisualRun(ctx, c, ceiling, productImageProfile)
}
func (s *Store) admitVisualRun(ctx context.Context, c agentconfig.ImageRunAdmissionCommand, ceiling agentconfig.ImageRunLimits, profile visualConfigurationProfile) (agentconfig.ImageRunAdmissionReceipt, error) {
	if !validImageAdmission(c) {
		return agentconfig.ImageRunAdmissionReceipt{}, agentconfig.ErrInvalid
	}
	snapshot, err := s.loadVisualConfiguration(ctx, c.Scope, c.Snapshot, profile)
	if err != nil {
		return agentconfig.ImageRunAdmissionReceipt{}, err
	}
	if snapshot.MemberID != c.MemberID || snapshot.RunID != c.RunID || snapshot.SourceDigest != c.SourceDigest || snapshot.InputDigest != c.InputDigest {
		return agentconfig.ImageRunAdmissionReceipt{}, agentconfig.ErrConflict
	}
	mode, err := s.LoadModeSnapshot(ctx, agent.ExecutionIdentity{OrganizationID: c.Scope.OrganizationID, ActorID: c.Scope.ActorID, MemberID: c.MemberID}, snapshot.ModeSnapshotRef)
	if err != nil {
		return agentconfig.ImageRunAdmissionReceipt{}, err
	}
	if mode.Subject.Subject.Kind != agentconfig.RegisteredAgentSubject || mode.Subject.Subject.ID != profile.agentID || mode.Subject.Definition.Version != snapshot.AgentVersion || mode.Binding.ContextKind != profile.context || mode.Binding.ContextID != snapshot.ContextID || mode.Binding.RequestKey != snapshot.RequestKey {
		return agentconfig.ImageRunAdmissionReceipt{}, agentconfig.ErrConflict
	}
	fingerprint, err := hash(c)
	if err != nil {
		return agentconfig.ImageRunAdmissionReceipt{}, err
	}
	var result agentconfig.ImageRunAdmissionReceipt
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// This is the same lock held by Execute(enable/disable). The committed
		// write-once receipt is the whole-run admission point across ImageDB.
		a, agentErr := findAgent(tx, c.Scope.OrganizationID, profile.agentID, true)
		var rows []imageAdmissionRow
		if err := tx.Where("organization_id=? AND (snapshot_id=? OR run_id=? OR (actor_id=? AND action_id=?))", c.Scope.OrganizationID, c.Snapshot.ID, c.RunID, c.Scope.ActorID, c.ConfirmActionID).Limit(2).Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) > 0 {
			if len(rows) != 1 || rows[0].SnapshotID != c.Snapshot.ID || rows[0].ActorID != c.Scope.ActorID || rows[0].Fingerprint != fingerprint {
				return agentconfig.ErrConflict
			}
			var err error
			result, err = imageAdmissionView(rows[0])
			return err
		}
		if errors.Is(agentErr, agentconfig.ErrNotFound) || agentErr == nil && a.Activation != "ENABLED" {
			return agentconfig.ErrNotEnabled
		}
		if agentErr != nil {
			return agentErr
		}
		if snapshot.Epoch != decimal(a.ActivationEpoch) || !c.Limits.Within(snapshot.HardLimits) || !c.Limits.Within(ceiling) {
			return agentconfig.ErrChanged
		}
		if err := checkModeInTransaction(tx, mode); err != nil {
			return err
		}
		if mode.Mode == agent.ModeReadOnly {
			return agentconfig.ErrForbidden
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		result = agentconfig.ImageRunAdmissionReceipt{ID: uuid.NewString(), Command: c, AdmittedAt: now, Deadline: now.Add(time.Duration(c.Limits.ElapsedSeconds) * time.Second)}
		raw, err := json.Marshal(result)
		if err != nil || len(raw) > 8192 {
			return agentconfig.ErrInvalid
		}
		row := imageAdmissionRow{SnapshotID: c.Snapshot.ID, OrganizationID: c.Scope.OrganizationID, ActorID: c.Scope.ActorID, MemberID: c.MemberID, RunID: c.RunID, ActionID: c.ConfirmActionID, ID: result.ID, Fingerprint: fingerprint, Digest: digest(raw), Payload: raw, AdmittedAt: result.AdmittedAt, Deadline: result.Deadline}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		result.Digest = row.Digest
		return nil
	})
	return result, err
}

func (s *Store) ReadImageRunAdmission(ctx context.Context, scope agent.Scope, ref agent.ConfigurationSnapshotRef) (agentconfig.ImageRunAdmissionReceipt, error) {
	return s.readVisualAdmission(ctx, scope, ref, productImageProfile)
}
func (s *Store) readVisualAdmission(ctx context.Context, scope agent.Scope, ref agent.ConfigurationSnapshotRef, profile visualConfigurationProfile) (agentconfig.ImageRunAdmissionReceipt, error) {
	if _, err := s.loadVisualConfiguration(ctx, scope, ref, profile); err != nil {
		return agentconfig.ImageRunAdmissionReceipt{}, err
	}
	var row imageAdmissionRow
	if err := s.db.WithContext(ctx).Where("snapshot_id=? AND organization_id=? AND actor_id=?", ref.ID, scope.OrganizationID, scope.ActorID).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return agentconfig.ImageRunAdmissionReceipt{}, agentconfig.ErrNotFound
		}
		return agentconfig.ImageRunAdmissionReceipt{}, err
	}
	return imageAdmissionView(row)
}

var _ agentconfig.ImageConfigurationRepository = (*Store)(nil)
