package imageagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"task-processor/internal/authidentity"
	"task-processor/internal/shared/aiidentity"
)

type Service struct {
	imageMethods      ImageMethodContext
	imageSets         *ImageSetDependencies
	artworks          *ArtworkDependencies
	organizationScope bool
	repository        Repository
	workflows         WorkflowClient
	catalogs          AuthorizedAssetCatalog
	startGate         TenantStartGate
	imagePolicies     ImagePolicyAvailability
}

// TenantStartGate admits a run before any durable run state or workflow is
// created. It keeps provider governance at the service boundary rather than
// allowing an ineligible tenant to create a run that must fail later.
type TenantStartGate interface {
	AllowTenantStart(context.Context, string) bool
}

// TenantAllowlistStartGate is the provider-neutral admission policy for the
// ProductImage capability. Enabled and the allowlist must both be true for a
// tenant to start an image-agent run.
type TenantAllowlistStartGate struct {
	Enabled          bool
	AllowedTenantIDs []string
}

func (g TenantAllowlistStartGate) AllowTenantStart(_ context.Context, tenantID string) bool {
	if !g.Enabled {
		return false
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return false
	}
	for _, allowed := range g.AllowedTenantIDs {
		if strings.TrimSpace(allowed) == tenantID {
			return true
		}
	}
	return false
}

type ServiceOption func(*Service) error

func WithTenantStartGate(gate TenantStartGate) ServiceOption {
	return func(service *Service) error {
		if gate == nil {
			return fmt.Errorf("image agent tenant start gate is required")
		}
		service.startGate = gate
		return nil
	}
}

func WithImagePolicyAvailability(availability ImagePolicyAvailability) ServiceOption {
	return func(service *Service) error {
		if availability == nil {
			return fmt.Errorf("image agent policy availability is required")
		}
		service.imagePolicies = availability
		return nil
	}
}

func NewService(repository Repository, workflows WorkflowClient, catalogs AuthorizedAssetCatalog, options ...ServiceOption) (*Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("image agent repository is required")
	}
	if workflows == nil {
		return nil, fmt.Errorf("image agent workflow client is required")
	}
	if catalogs == nil {
		return nil, fmt.Errorf("image agent authorized asset catalog is required")
	}
	service := &Service{repository: repository, workflows: workflows, catalogs: catalogs}
	for _, option := range options {
		if option == nil {
			return nil, fmt.Errorf("image agent service option is required")
		}
		if err := option(service); err != nil {
			return nil, err
		}
	}
	return service, nil
}

func (s *Service) Start(ctx context.Context, input StartRunInput) error {
	identity, err := s.executionIdentity(ctx)
	if err != nil {
		return err
	}
	if input.Plan.Set != nil || input.Plan.Artwork != nil {
		return fmt.Errorf("%w: image sets require PrepareImageSet and ConfirmImagePlan", ErrCommandBlocked)
	}
	if input.Mode != RunModeManual {
		return fmt.Errorf("%w: image agent start mode must be manual", ErrValidation)
	}
	input.RunID = strings.TrimSpace(input.RunID)
	input.BusinessTaskID = strings.TrimSpace(input.BusinessTaskID)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if err := ValidateImagePolicyContext(input.TargetPlatform, input.ImagePolicyContext); err != nil {
		return err
	}
	if err := ValidateMaxConcurrentSlots(input.MaxConcurrentSlots); err != nil {
		return err
	}
	input.MaxConcurrentSlots = NormalizeMaxConcurrentSlots(input.MaxConcurrentSlots)
	if input.RunID == "" || input.BusinessTaskID == "" || input.IdempotencyKey == "" {
		return fmt.Errorf("%w: image agent run ID, business task ID, and idempotency key are required", ErrValidation)
	}
	if err := validatePersistedIdempotencyKey("run", input.IdempotencyKey); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := ValidateArtifactKeyIdentifier(input.RunID); err != nil {
		return fmt.Errorf("%w: image agent run ID cannot be used in a durable artifact key", err)
	}
	if _, err := input.Budget.Policy(); err != nil {
		return fmt.Errorf("%w: validate image agent budget: %v", ErrValidation, err)
	}
	identity.BusinessTaskID = input.BusinessTaskID
	input.Plan.CreatedBy = identity.UserID
	if err := ValidatePlan(input.Plan); err != nil {
		return fmt.Errorf("%w: validate image agent plan: %v", ErrValidation, err)
	}
	scope := RunScope{TenantID: identity.TenantID, OwnerUserID: identity.UserID, RunID: input.RunID}
	if existing, getErr := s.repository.GetProjection(ctx, scope); getErr == nil {
		if existing.Run.BusinessTaskID != input.BusinessTaskID || existing.Run.TargetPlatform != input.TargetPlatform || existing.Run.ImagePolicyContext != input.ImagePolicyContext || existing.Run.IdempotencyKey != input.IdempotencyKey || existing.Run.Budget != input.Budget || existing.Run.MaxConcurrentSlots != input.MaxConcurrentSlots || !reflect.DeepEqual(existing.Plan, input.Plan) {
			return ErrRevisionConflict
		}
		if existing.Run.Status == RunStatusCompleted || existing.Run.Status == RunStatusCancelled {
			return nil
		}
		return s.startExistingProjection(ctx, existing, identity)
	} else if !errors.Is(getErr, ErrRunNotFound) {
		return getErr
	}
	if !s.tenantStartAllowed(ctx, identity.TenantID) {
		return fmt.Errorf("%w: image agent provider is unavailable for tenant", ErrCommandBlocked)
	}
	if err := s.validateImagePolicyAvailable(ctx, input.TargetPlatform, input.ImagePolicyContext); err != nil {
		return err
	}
	if err := ValidateInitialSubmittedPlan(input.Plan); err != nil {
		return fmt.Errorf("%w: validate image agent plan: %v", ErrValidation, err)
	}
	primarySourceAssetID := ""
	if len(input.Plan.SourceAssetIDs) == 1 {
		primarySourceAssetID = strings.TrimSpace(input.Plan.SourceAssetIDs[0])
	}
	catalog, err := s.catalogs.Resolve(ctx, AssetCatalogScope{TenantID: identity.TenantID, OwnerUserID: identity.UserID, BusinessTaskID: input.BusinessTaskID, RunID: input.RunID, TargetPlatform: input.TargetPlatform, PrimarySourceAssetID: primarySourceAssetID, StyleReferenceIDs: append([]string(nil), input.Plan.StyleReferenceIDs...)})
	if err != nil {
		return fmt.Errorf("%w: resolve authorized image assets: %v", ErrValidation, err)
	}
	catalog, err = NormalizeAssetCatalog(catalog)
	if err != nil {
		return fmt.Errorf("%w: normalize authorized image assets: %v", ErrValidation, err)
	}
	if err := ValidateSubmittedPlanAgainstCatalog(input.Plan, catalog); err != nil {
		return fmt.Errorf("%w: validate authorized image assets: %v", ErrValidation, err)
	}
	run := Run{
		ScopeProtocol: identity.ScopeProtocol,
		ID:            input.RunID, BusinessTaskID: input.BusinessTaskID, TargetPlatform: input.TargetPlatform,
		ImagePolicyContext: input.ImagePolicyContext,
		TenantID:           identity.TenantID, UserID: identity.UserID, MemberID: identity.MemberID,
		Mode: RunModeManual, IdempotencyKey: input.IdempotencyKey,
		Status: RunStatusPlanning, CurrentNode: "plan", Version: 1, ActivePlanRevision: input.Plan.Revision,
		Budget: input.Budget, MaxConcurrentSlots: input.MaxConcurrentSlots,
	}
	projection, err := s.repository.InitializeRun(ctx, ProjectionInitialization{
		Scope: scope, Run: run, Plan: input.Plan, Catalog: catalog,
		Snapshot: RunProjection{Run: run, Plan: input.Plan}, CommitID: "start:" + input.IdempotencyKey,
		EventType: "run.initialized", EventPayload: json.RawMessage(`{}`),
	})
	if err != nil {
		return fmt.Errorf("initialize image agent run: %w", err)
	}
	identity, err = s.identityForRun(identity, projection.Run)
	if err != nil {
		return err
	}
	return s.workflows.StartManual(ctx, WorkflowStart{
		Run: projection.Run, Plan: projection.Plan, Identity: identity,
		MaxConcurrentSlots: projection.Run.MaxConcurrentSlots,
		AssetCatalog:       projection.AssetCatalog,
	})
}

// RestartFailed exposes the same-request recovery path using only immutable,
// owner-scoped inputs already persisted with the run.
func (s *Service) RestartFailed(ctx context.Context, runID string) error {
	identity, err := s.executionIdentity(ctx)
	if err != nil {
		return err
	}
	scope := RunScope{TenantID: identity.TenantID, OwnerUserID: identity.UserID, RunID: strings.TrimSpace(runID)}
	projection, err := s.repository.GetProjection(ctx, scope)
	if err != nil {
		return err
	}
	if projection.Run.Status != RunStatusFailed {
		return fmt.Errorf("%w: only a failed image agent run can be restarted", ErrCommandBlocked)
	}
	identity.BusinessTaskID = projection.Run.BusinessTaskID
	if projection.Plan.Set != nil {
		// Restore the original whole-run admission. Current new-start gates must
		// not replace its budget or deadline. Live source checks belong to the
		// actual dispatch, allowing the original workflow to close denials.
		identity, err = s.identityForRun(identity, projection.Run)
		if err != nil {
			return err
		}
		if s.imageSets == nil || projection.PendingCommand != nil {
			return ErrCommandBlocked
		}
		if err := ValidateImageSetAdmission(projection.Run, projection.Plan); err != nil {
			return err
		}
		if _, err := ImageSetClosedEffectsDigest(projection.Plan, projection.Slots, projection.RecoverableEffects); err == nil {
			return ErrCommandBlocked
		}
		return s.workflows.StartManual(ctx, WorkflowStart{Run: projection.Run, Plan: projection.Plan, Identity: identity, MaxConcurrentSlots: projection.Run.MaxConcurrentSlots, AssetCatalog: projection.AssetCatalog})
	}
	return s.startExistingProjection(ctx, projection, identity)
}

func (s *Service) startExistingProjection(ctx context.Context, projection RunProjection, identity ExecutionIdentity) error {
	identity, err := s.identityForRun(identity, projection.Run)
	if err != nil {
		return err
	}
	// The workflow is already durable (Temporal USE_EXISTING conflict policy
	// keyed by run ID); StartManual on an existing projection is a re-dispatch
	// of the same workflow, not a new paid execution.
	if !s.tenantStartAllowed(ctx, identity.TenantID) {
		return fmt.Errorf("%w: image agent provider is unavailable for tenant", ErrCommandBlocked)
	}
	if err := s.validateImagePolicyAvailable(ctx, projection.Run.TargetPlatform, projection.Run.ImagePolicyContext); err != nil {
		return err
	}
	return s.workflows.StartManual(ctx, WorkflowStart{
		Run: projection.Run, Plan: projection.Plan, Identity: identity,
		MaxConcurrentSlots: projection.Run.MaxConcurrentSlots, AssetCatalog: projection.AssetCatalog,
	})
}

func (s *Service) tenantStartAllowed(ctx context.Context, tenantID string) bool {
	return s == nil || s.startGate == nil || s.startGate.AllowTenantStart(ctx, tenantID)
}

func (s *Service) validateImagePolicyAvailable(ctx context.Context, marketplace string, policy ImagePolicyContext) error {
	if s == nil || s.imagePolicies == nil {
		return nil
	}
	if err := s.imagePolicies.ValidateAvailable(ctx, marketplace, policy); err != nil {
		return fmt.Errorf("%w: image policy is unavailable: %v", ErrCommandBlocked, err)
	}
	return nil
}

func (s *Service) Get(ctx context.Context, runID string) (RunProjection, error) {
	identity, err := s.executionIdentity(ctx)
	if err != nil {
		return RunProjection{}, err
	}
	scope := RunScope{TenantID: identity.TenantID, OwnerUserID: identity.UserID, RunID: strings.TrimSpace(runID)}
	projection, err := s.repository.GetProjection(ctx, scope)
	if err != nil {
		return RunProjection{}, err
	}
	if s.organizationScope {
		if _, err := s.identityForRun(identity, projection.Run); err != nil {
			return RunProjection{}, err
		}
	}
	return projection, nil
}

func (s *Service) ReplacePlan(ctx context.Context, runID string, expectedRevision int64, plan Plan, actionID string) error {
	if plan.Artwork != nil {
		return ErrCommandBlocked
	}
	identity, err := s.commandIdentity(ctx, runID, expectedRevision, actionID)
	if err != nil {
		return err
	}
	plan.CreatedBy = identity.UserID
	if err := ValidateReplacementSubmittedPlan(expectedRevision, plan); err != nil {
		return fmt.Errorf("%w: validate replacement plan: %v", ErrValidation, err)
	}
	projection, err := s.repository.GetProjection(ctx, RunScope{TenantID: identity.TenantID, OwnerUserID: identity.UserID, RunID: strings.TrimSpace(runID)})
	if err != nil {
		return err
	}
	if projection.Run.Status == RunStatusBlocked && !BlockAllowsAction(projection.Run.Block, ActionEditPlan) {
		return fmt.Errorf("%w: the current block permits cancellation only", ErrCommandBlocked)
	}
	catalog, err := s.repository.GetAssetCatalog(ctx, RunScope{TenantID: identity.TenantID, OwnerUserID: identity.UserID, RunID: strings.TrimSpace(runID)})
	if err != nil {
		return err
	}
	if err := ValidateSubmittedPlanAgainstCatalog(plan, catalog); err != nil {
		return fmt.Errorf("%w: validate authorized image assets: %v", ErrValidation, err)
	}
	return s.workflows.ReplacePlan(ctx, ReplacePlanCommand{
		RunID: strings.TrimSpace(runID), ExpectedRevision: expectedRevision, Plan: plan,
		ActorID: identity.UserID, ActionID: strings.TrimSpace(actionID), Identity: identity,
	})
}

func (s *Service) RetrySlot(ctx context.Context, runID, slotID string, planRevision int64, actionID string) error {
	identity, err := s.commandIdentity(ctx, runID, planRevision, actionID)
	if err != nil {
		return err
	}
	slotID = strings.TrimSpace(slotID)
	if slotID == "" {
		return fmt.Errorf("%w: retry slot ID is required", ErrValidation)
	}
	if err := ValidateArtifactKeyIdentifier(slotID); err != nil {
		return fmt.Errorf("%w: retry slot ID cannot be used in a durable artifact key", err)
	}
	return s.workflows.RetrySlot(ctx, RetrySlotCommand{RunID: strings.TrimSpace(runID), SlotID: slotID, PlanRevision: planRevision, ActorID: identity.UserID, ActionID: strings.TrimSpace(actionID), Identity: identity})
}

func (s *Service) RecoverEffect(ctx context.Context, runID, slotID string, attempt int, planRevision int64, actionID string) error {
	identity, err := s.commandIdentity(ctx, runID, planRevision, actionID)
	if err != nil {
		return err
	}
	slotID = strings.TrimSpace(slotID)
	if slotID == "" || attempt <= 0 {
		return fmt.Errorf("%w: recover slot ID and attempt are required", ErrValidation)
	}
	if err := ValidateArtifactKeyIdentifier(slotID); err != nil {
		return fmt.Errorf("%w: recover slot ID cannot be used in a durable artifact key", err)
	}
	runID = strings.TrimSpace(runID)
	projection, err := s.repository.GetProjection(ctx, RunScope{TenantID: identity.TenantID, OwnerUserID: identity.UserID, RunID: runID})
	if err != nil {
		return err
	}
	if projection.Run.Status != RunStatusBlocked {
		return fmt.Errorf("%w: only the current blocked effect can be recovered", ErrCommandBlocked)
	}
	if projection.Plan.Revision != planRevision {
		return fmt.Errorf("%w: recover effect plan revision is stale", ErrRevisionConflict)
	}
	if _, _, ok := FindRecoverableEffect(projection, slotID, attempt); !ok {
		return fmt.Errorf("%w: only external-effect recovery blocks support re-drive", ErrCommandBlocked)
	}
	return s.workflows.RecoverEffect(ctx, RecoverEffectCommand{
		RunID: runID, PlanRevision: planRevision, SlotID: slotID, Attempt: attempt,
		ActionID: strings.TrimSpace(actionID), Identity: identity, Projection: projection,
	})
}

func (s *Service) ApproveResults(ctx context.Context, runID string, planRevision int64, resultDigest, actionID string) error {
	identity, err := s.commandIdentity(ctx, runID, planRevision, actionID)
	if err != nil {
		return err
	}
	if resultDigest == "" || resultDigest != strings.TrimSpace(resultDigest) {
		return ErrCommandBlocked
	}
	return s.workflows.ApproveResults(ctx, ApproveResultsCommand{RunID: strings.TrimSpace(runID), PlanRevision: planRevision, ResultDigest: resultDigest, ActorID: identity.UserID, ActionID: strings.TrimSpace(actionID), Identity: identity})
}

func (s *Service) Cancel(ctx context.Context, runID string, planRevision int64, actionID string) error {
	identity, err := s.commandIdentity(ctx, runID, planRevision, actionID)
	if err != nil {
		return err
	}
	scope := RunScope{TenantID: identity.TenantID, OwnerUserID: identity.UserID, RunID: runID}
	current, err := s.repository.GetProjection(ctx, scope)
	if err != nil {
		return err
	}
	if (current.Plan.Set != nil || current.Plan.Artwork != nil) && current.Run.ImageAdmission == nil {
		if _, err = s.identityForRun(identity, current.Run); err != nil {
			return err
		}
		if current.Plan.Revision != planRevision {
			return ErrRevisionConflict
		}
		if current.Run.Status == RunStatusCancelled {
			return nil
		}
		if current.Run.Status != RunStatusAwaitingPlanApproval {
			return ErrCommandBlocked
		}
		next := current
		next.Run.Status, next.Run.CurrentNode, next.Run.Version = RunStatusCancelled, "cancel", current.Run.Version+1
		_, err = s.repository.CommitProjection(ctx, ProjectionCommit{Scope: scope, CommitID: "cancel-prepared:" + actionID, ExpectedProjectionVersion: current.ProjectionVersion, ExpectedRunVersion: current.Run.Version, Snapshot: next, RunMutation: &RunMutation{Status: next.Run.Status, CurrentNode: next.Run.CurrentNode, ActivePlanRevision: planRevision}, EventType: "run.cancelled", EventPayload: json.RawMessage(`{}`)})
		return err
	}
	return s.workflows.Cancel(ctx, CancelRunCommand{RunID: strings.TrimSpace(runID), PlanRevision: planRevision, ActorID: identity.UserID, ActionID: strings.TrimSpace(actionID), Identity: identity})
}

func (s *Service) Resume(ctx context.Context, runID, actionID string) (CommandAcknowledgement, error) {
	identity, err := s.executionIdentity(ctx)
	if err != nil {
		return CommandAcknowledgement{}, err
	}
	runID = strings.TrimSpace(runID)
	if runID == "" || ValidateActionID(actionID) != nil {
		return CommandAcknowledgement{}, fmt.Errorf("%w: run ID and action ID are required", ErrValidation)
	}
	projection, err := s.repository.GetProjection(ctx, RunScope{TenantID: identity.TenantID, OwnerUserID: identity.UserID, RunID: runID})
	if err != nil {
		return CommandAcknowledgement{}, err
	}
	if s.organizationScope {
		identity, err = s.identityForRun(identity, projection.Run)
		if err != nil {
			return CommandAcknowledgement{}, err
		}
	}
	return s.workflows.Resume(ctx, ResumeCommand{RunID: runID, ActorID: identity.UserID, ActionID: actionID, Identity: identity})
}

func (s *Service) ListEvents(ctx context.Context, runID string, afterCursor int64, limit int) ([]RunEvent, error) {
	identity, err := s.executionIdentity(ctx)
	if err != nil {
		return nil, err
	}
	if afterCursor < 0 || limit <= 0 {
		return nil, fmt.Errorf("%w: event cursor and limit are invalid", ErrValidation)
	}
	scope := RunScope{TenantID: identity.TenantID, OwnerUserID: identity.UserID, RunID: strings.TrimSpace(runID)}
	if s.organizationScope {
		projection, err := s.repository.GetProjection(ctx, scope)
		if err != nil {
			return nil, err
		}
		if _, err := s.identityForRun(identity, projection.Run); err != nil {
			return nil, err
		}
	}
	events, err := s.repository.ListEvents(ctx, scope, afterCursor, limit)
	if err != nil {
		return nil, err
	}
	last := afterCursor
	for _, event := range events {
		if event.TenantID != scope.TenantID || event.OwnerUserID != scope.OwnerUserID || event.RunID != scope.RunID || event.Cursor <= last || event.ProjectionVersion != event.Cursor {
			return nil, ErrRevisionConflict
		}
		last = event.Cursor
	}
	return events, nil
}

func (s *Service) commandIdentity(ctx context.Context, runID string, revision int64, actionID string) (ExecutionIdentity, error) {
	identity, err := s.executionIdentity(ctx)
	if err != nil {
		return ExecutionIdentity{}, err
	}
	runID = strings.TrimSpace(runID)
	if runID == "" || ValidateActionID(actionID) != nil {
		return ExecutionIdentity{}, fmt.Errorf("%w: run ID and action ID are required", ErrValidation)
	}
	if revision <= 0 || revision > MaxJSONSafePlanRevision {
		return ExecutionIdentity{}, fmt.Errorf("%w: positive JSON-safe plan revision is required", ErrValidation)
	}
	projection, err := s.repository.GetProjection(ctx, RunScope{TenantID: identity.TenantID, OwnerUserID: identity.UserID, RunID: runID})
	if err != nil {
		return ExecutionIdentity{}, err
	}
	return s.identityForRun(identity, projection.Run)
}

func verifiedExecutionIdentity(ctx context.Context) (ExecutionIdentity, error) {
	identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok {
		return ExecutionIdentity{}, ErrIdentityRequired
	}
	ai := aiidentity.FromContext(ctx)
	return ExecutionIdentity{TenantID: identity.TenantID, UserID: identity.UserID, MemberID: identity.EffectiveMemberID, TraceID: ai.TraceID}, nil
}

func cloneBlock(block *Block) *Block {
	if block == nil {
		return nil
	}
	cloned := *block
	return &cloned
}

func clonePendingCommand(receipt *PendingCommandReceipt) *PendingCommandReceipt {
	if receipt == nil {
		return nil
	}
	cloned := *receipt
	return &cloned
}

const OrganizationScopeProtocol = "image-agent.organization.v1"

func (s *Service) identityForRun(identity ExecutionIdentity, run Run) (ExecutionIdentity, error) {
	identity.BusinessTaskID = run.BusinessTaskID
	if s.organizationScope {
		if run.ScopeProtocol != OrganizationScopeProtocol || run.TenantID != identity.TenantID || run.UserID != identity.UserID || run.MemberID == "" || run.MemberID != identity.MemberID {
			return ExecutionIdentity{}, ErrIdentityRequired
		}
		identity.RunID = run.ID
		if err := ValidateOrganizationExecution(identity, run.ID); err != nil {
			return ExecutionIdentity{}, err
		}
	}
	return identity, nil
}

// ExecutionAuthorizer checks current permission for the immutable run owner.
// Implementations must query current grants, never replay captured roles.
type ExecutionAuthorizer interface {
	AuthorizeExecution(context.Context, ExecutionIdentity) error
}

func WithOrganizationScope() ServiceOption {
	return func(service *Service) error { service.organizationScope = true; return nil }
}

func (s *Service) executionIdentity(ctx context.Context) (ExecutionIdentity, error) {
	identity, err := verifiedExecutionIdentity(ctx)
	if err != nil || !s.organizationScope {
		return identity, err
	}
	verified, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || verified.EffectiveOrganizationID == "" || verified.TenantID != verified.EffectiveOrganizationID {
		return ExecutionIdentity{}, ErrIdentityRequired
	}
	identity.ScopeProtocol = OrganizationScopeProtocol
	return identity, nil
}

func ValidateOrganizationExecution(identity ExecutionIdentity, runID string) error {
	if identity.ScopeProtocol != OrganizationScopeProtocol || identity.RunID != runID ||
		runID == "" || identity.TenantID == "" || identity.UserID == "" || identity.BusinessTaskID == "" {
		return ErrIdentityRequired
	}
	for _, value := range []string{identity.TenantID, identity.UserID, runID, identity.BusinessTaskID} {
		if value != strings.TrimSpace(value) {
			return ErrIdentityRequired
		}
	}
	return nil
}
