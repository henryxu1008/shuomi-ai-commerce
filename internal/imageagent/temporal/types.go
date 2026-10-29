package temporal

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"task-processor/internal/agentconfig"
	"task-processor/internal/imageagent"
	"task-processor/internal/imageagent/objectstore"
	productasset "task-processor/internal/product/asset"
)

const (
	// V3WorkflowExecutionTimeout bounds new manual runs within the documented
	// 30-day lifecycle, leaving a seven-day reconciliation allowance before the
	// 45-day staging retention can expire.
	V3WorkflowExecutionTimeout         = 30 * 24 * time.Hour
	V3LifecycleDeadlineSafetyMargin    = 24 * time.Hour
	V3MinimumStagingLifecycleRetention = 45 * 24 * time.Hour
)

const (
	TaskQueue                             = "image-agent-manual"
	TaskQueueV3                           = "image-agent-manual-v3"
	TaskQueueV3Canary                     = "image-agent-manual-v3-canary"
	activityPersistSlotResultV3           = "imageagent.persist_slot_result.v3"
	activityPersistImageSetSlotResult     = "imageagent.persist_image_set_slot_result.v1"
	activityPersistRunState               = "imageagent.persist_run_state.v2"
	activityPersistWorkflowFailure        = "imageagent.persist_workflow_failure.v1"
	activityPersistWorkflowFailureV2      = "imageagent.persist_workflow_failure.v2"
	activityPersistPlanRevision           = "imageagent.persist_plan_revision.v2"
	activityPersistPendingCommand         = "imageagent.persist_pending_command.v2"
	activityExecuteSlotV3                 = "imageagent.execute_slot.v3"
	activityReviewStagedSlotV3            = "imageagent.review_staged_slot.v3"
	activityStartEffectRecoveryV3         = "imageagent.start_effect_recovery.v3"
	activityRecoverEffectV3               = "imageagent.recover_effect.v3"
	activityPersistRecoveryBlockedV3      = "imageagent.persist_recovery_blocked.v3"
	activityReconcileEffectRecoveryV3     = "imageagent.reconcile_effect_recovery.v3"
	activityPublishApprovedV3             = "imageagent.publish_approved.v3"
	activityPublishApprovedImageSet       = "imageagent.publish_approved_image_set.v1"
	activityPrepareImageSetPolicyAdoption = "imageagent.prepare_policy_adoption.v1"
	workflowNameCompatibilityCanary       = "ImageAgentCompatibilityCanaryWorkflow"
	signalApproveResults                  = "approve_results"
	signalEffectRecoveryCompleted         = "effect_recovery_completed"
	signalRetrySlot                       = "retry_slot"
	signalReplacePlan                     = "replace_plan"
	signalCancel                          = "cancel"
	updateResumeCommand                   = "resume_command"
	updateErrorRevisionConflict           = "imageagent_revision_conflict"
	updateErrorCommandBlocked             = "imageagent_command_blocked"
	updateErrorRunNotFound                = "imageagent_run_not_found"
	defaultMaxConcurrentSlots             = 4
	QueryWorkflowProjection               = "image_agent_projection"
	EffectRecoveryWorkflowName            = "ImageAgentEffectRecoveryWorkflow"
	effectRecoveryBlockedCode             = imageagent.SlotRecoveryBlockedCode
)

type WorkerWireMode string

const (
	WorkerWireModeV3 WorkerWireMode = "v3"
)

func (mode WorkerWireMode) DefaultTaskQueue() (string, error) {
	if mode == WorkerWireModeOrganization {
		return OrganizationTaskQueue, nil
	}
	if mode == "" {
		return "", fmt.Errorf("image agent temporal wire mode is required")
	}
	return "", fmt.Errorf("unsupported image agent temporal wire mode %q; organization-v1 is required", mode)
}

type WorkflowInput struct {
	ImageAdmission      *agentconfig.ImageRunAdmissionReceipt `json:",omitempty"`
	RunID               string
	TargetPlatform      string                         `json:",omitempty"`
	ImagePolicyContext  *imageagent.ImagePolicyContext `json:",omitempty"`
	Mode                imageagent.RunMode
	Identity            imageagent.ExecutionIdentity
	Plan                imageagent.Plan
	MaxConcurrentSlots  int
	WaitForCommands     bool
	AssetCatalog        imageagent.AssetCatalog
	BudgetPolicy        imageagent.BudgetPolicy
	StartedAt           time.Time
	DeadlineAt          time.Time
	LifecycleDeadlineAt time.Time
	BudgetAuthorization bool
	// externalEffectFinalization is runtime-only versioned behavior selected by
	// the parent workflow before it launches v3 slot children.
	externalEffectFinalization bool
	// enforceIngressPlanPolicy is runtime-only versioned behavior for new
	// workflow histories. It is never serialized from clients.
	enforceIngressPlanPolicy bool
	// projectionExecutionID is runtime-only state populated deterministically
	// from Temporal's workflow execution RunID, never serialized from clients.
	projectionExecutionID string
}

type WorkflowResult struct {
	Status             imageagent.RunStatus
	Block              *imageagent.Block
	Plan               imageagent.Plan
	Slots              []imageagent.SlotProjection
	RecoverableEffects []imageagent.RecoverableEffect
	CompletedSlotIDs   []string
	ResultDigest       string
	PendingCommand     *imageagent.PendingCommandReceipt
	CommandIngress     imageagent.CommandIngress
}

type slotChildIdentity struct {
	RunID        string
	Identity     imageagent.ExecutionIdentity
	PlanRevision int64
	Slot         imageagent.Slot
	Attempt      int
	AssetCatalog imageagent.AssetCatalog
}

type SlotWorkflowResult struct {
	Closure   *imageagent.ImageSlotClosure `json:",omitempty"`
	Execution imageagent.SlotExecutionResult
	Status    imageagent.SlotStatus
	ErrorCode string
	// EffectPhase is current v3 evidence used by the versioned parent
	// cancellation gate.
	EffectPhase imageagent.SlotEffectV3Phase
}

// SlotWorkflowV3Input is the current durable execution child input.
type SlotWorkflowV3Input struct {
	Artwork            *imageagent.ArtworkPlan  `json:",omitempty"`
	ImageSet           *imageagent.ImageSetPlan `json:",omitempty"`
	RunID              string
	TargetPlatform     string                         `json:",omitempty"`
	ImagePolicyContext *imageagent.ImagePolicyContext `json:",omitempty"`
	Identity           imageagent.ExecutionIdentity
	PlanRevision       int64
	Slot               imageagent.Slot
	Attempt            int
	AssetCatalog       imageagent.AssetCatalog
	// ExecuteActivityName is supplied by the Task 6 wire-selection gate. Task 4
	// deliberately defines no production v3 activity name or registration.
	ExecuteActivityName        string
	BudgetAuthorization        bool
	BudgetPolicy               imageagent.BudgetPolicy
	DeadlineAt                 time.Time
	LifecycleDeadlineAt        time.Time
	ExternalEffectFinalization bool
}

type SlotWorkflowV3Result struct {
	Published imageagent.SlotEffectV3PublishedResult
	Status    imageagent.SlotStatus
	ErrorCode string
	// EffectPhase proves the durable external-effect phase observed by the child.
	// SlotStatusBlocked alone is deliberately not terminalization evidence.
	EffectPhase imageagent.SlotEffectV3Phase
}

type EffectRecoveryWorkflowInput struct {
	Artwork            *imageagent.ArtworkPlan  `json:",omitempty"`
	ImageSet           *imageagent.ImageSetPlan `json:",omitempty"`
	RunID              string
	TargetPlatform     string                         `json:",omitempty"`
	ImagePolicyContext *imageagent.ImagePolicyContext `json:",omitempty"`
	Identity           imageagent.ExecutionIdentity
	PlanRevision       int64
	Slot               imageagent.Slot
	Attempt            int
	// ActionID is empty for automatic recovery and historical workflow inputs.
	// Explicit redrives use it to scope a restartable workflow execution.
	ActionID     string
	AssetCatalog imageagent.AssetCatalog
}

type EffectRecoveryOutcome string

const (
	EffectRecoveryOutcomePublished          EffectRecoveryOutcome = "published"
	EffectRecoveryOutcomeProviderUnknown    EffectRecoveryOutcome = "provider_unknown"
	EffectRecoveryOutcomeStagingUnknown     EffectRecoveryOutcome = "staging_unknown"
	EffectRecoveryOutcomePublicationUnknown EffectRecoveryOutcome = "publication_unknown"
	EffectRecoveryOutcomeReviewRequired     EffectRecoveryOutcome = "review_required"
	EffectRecoveryOutcomeRecoveryBlocked    EffectRecoveryOutcome = "recovery_blocked"
)

type EffectRecoveryResult struct {
	Closure         *imageagent.ImageSlotClosure     `json:",omitempty"`
	GenerationProof *imageagent.ImageGenerationProof `json:",omitempty"`
	Outcome         EffectRecoveryOutcome
	Published       imageagent.SlotEffectV3PublishedResult
	EffectPhase     imageagent.SlotEffectV3Phase
	BlockedCode     string
}

// EffectRecoveryCompletedSignal is emitted after the recovery workflow has
// durably reconciled its external effect. The parent uses the identity tuple
// to ignore stale or duplicate completions while refreshing its in-memory
// projection and pending cancellation intent.
type EffectRecoveryCompletedSignal struct {
	RunID        string
	PlanRevision int64
	SlotID       string
	Attempt      int
	Result       EffectRecoveryResult
}

// ExecuteSlotV3ActivityInput is additive until Task 6 selects the v3 wire.
// Keep ExecuteSlotActivityInput frozen for imageagent.execute_slot.v2 replay.
type ExecuteSlotV3ActivityInput struct {
	Artwork                    *imageagent.ArtworkPlan  `json:",omitempty"`
	ImageSet                   *imageagent.ImageSetPlan `json:",omitempty"`
	RunID                      string
	TargetPlatform             string                         `json:",omitempty"`
	ImagePolicyContext         *imageagent.ImagePolicyContext `json:",omitempty"`
	Identity                   imageagent.ExecutionIdentity
	PlanRevision               int64
	Slot                       imageagent.Slot
	Attempt                    int
	IdempotencyKey             string
	ReviewActionID             string `json:",omitempty"`
	AssetCatalog               imageagent.AssetCatalog
	ExternalEffectFinalization bool
	BudgetAuthorization        bool
	BudgetPolicy               imageagent.BudgetPolicy
	DeadlineAt                 time.Time
	LifecycleDeadlineAt        time.Time
}

// DurableArtifactStore is the production/recovery boundary around deterministic
// staging and publication objects. Generated bytes are preserved as one
// immutable recovery bundle before the staging phase is persisted, so a fresh
// activity can rehydrate a partially uploaded manifest without regeneration.
type DurableArtifactStore interface {
	imageagent.DurableAssetPublicURLResolver
	PrepareSlotArtifacts(objectstore.PrepareSlotArtifactsInput) (objectstore.PreparedSlotArtifacts, error)
	PreserveSlotArtifacts(context.Context, imageagent.SlotExternalEffectIdentity, objectstore.PreparedSlotArtifacts) error
	RecoverSlotArtifacts(context.Context, imageagent.SlotExternalEffectIdentity, imageagent.StagingManifest) (objectstore.PreparedSlotArtifacts, error)
	EnsureStaged(context.Context, objectstore.PreparedSlotArtifacts) error
	Finalize(context.Context, imageagent.StagingManifest) (imageagent.FinalManifest, error)
	FinalizeWithProgress(context.Context, imageagent.StagingManifest, func(context.Context, int) error) (imageagent.FinalManifest, error)
}

// PersistSlotResultV3ActivityInput is an additive durable-result contract. It
// is deliberately not registered under imageagent.persist_slot_result.v2.
type PersistSlotResultV3ActivityInput struct {
	RunID        string
	Identity     imageagent.ExecutionIdentity
	PlanRevision int64
	Result       SlotWorkflowV3Result
	AttemptKey   string
}

type PersistRunStateActivityInput struct {
	RunID        string
	Identity     imageagent.ExecutionIdentity
	PlanRevision int64
	Projection   WorkflowResult
	CurrentNode  string
	CommitID     string
}

type PersistWorkflowFailureActivityInput struct {
	RunID          string
	Identity       imageagent.ExecutionIdentity
	FailureCode    string
	FailureMessage string
}

type PersistWorkflowFailureV2ActivityInput struct {
	RunID          string
	Identity       imageagent.ExecutionIdentity
	FailureCode    string
	FailureMessage string
	CommitID       string
}

type PersistPlanRevisionActivityInput struct {
	RunID            string
	Identity         imageagent.ExecutionIdentity
	ExpectedRevision int64
	Plan             imageagent.Plan
}

type PersistPendingCommandActivityInput struct {
	RunID          string
	Identity       imageagent.ExecutionIdentity
	Receipt        *imageagent.PendingCommandReceipt
	CommitID       string
	CommandIngress imageagent.CommandIngress
}

// Legacy activity payloads are frozen wire contracts. Their handlers never
// write owner-scoped v2 state; they fail with an explicit migration error.

// PublishApprovedV3ActivityInput is a distinct durable-approval wire payload.
// Task 6 owns its Temporal registration and workflow routing.
type PublishApprovedV3ActivityInput struct {
	RunID             string
	Identity          imageagent.ExecutionIdentity
	PlanRevision      int64
	CandidateAssetIDs []string
	IdempotencyKey    string
}

type ApproveResultsSignal struct {
	Selection    *productasset.ImageSetCommand `json:",omitempty"`
	RunID        string
	PlanRevision int64
	ResultDigest string
	ActorID      string
	ActionID     string
}

type PublishImageSetActivityInput struct {
	RunID        string
	Identity     imageagent.ExecutionIdentity
	PlanRevision int64
	ResultDigest string
	Selection    productasset.ImageSetCommand
}

type RetrySlotSignal struct {
	RunID        string
	PlanRevision int64
	SlotID       string
	ActorID      string
	ActionID     string
}

type ReplacePlanSignal struct {
	RunID            string
	ExpectedRevision int64
	Plan             imageagent.Plan
	ActorID          string
	ActionID         string
}

type CancelSignal struct {
	RunID        string
	PlanRevision int64
	ActorID      string
	ActionID     string
}

type ResumeCommandInput struct {
	RunID    string
	ActorID  string
	ActionID string
}

type CommandAcknowledgement = imageagent.CommandAcknowledgement

func slotAttemptKey(planRevision int64, slot imageagent.Slot, attempt int) string {
	return imageagent.SlotAttemptKey(planRevision, slot, attempt)
}

func childWorkflowID(input slotChildIdentity) string {
	return fmt.Sprintf("%s:plan:%d:slot:%s:attempt:%d", scopedWorkflowID(input.Identity, input.RunID), input.PlanRevision, input.Slot.ID, input.Attempt)
}

// EffectRecoveryWorkflowIDForSlot builds an ID from structured fields. Empty
// action IDs retain the original deterministic ID for automatic handoff and
// historical replay compatibility.
func EffectRecoveryWorkflowIDForSlot(identity imageagent.ExecutionIdentity, planRevision int64, runID, slotID string, attempt int, actionID string) string {
	tenantID := strings.TrimSpace(identity.TenantID)
	userID := strings.TrimSpace(identity.UserID)
	runID = strings.TrimSpace(runID)
	slotID = strings.TrimSpace(slotID)
	base := ""
	if strings.ContainsAny(tenantID+userID+runID+slotID, ":") {
		encode := func(value string) string {
			return base64.RawURLEncoding.EncodeToString([]byte(value))
		}
		base = fmt.Sprintf("image-agent-effect-recovery:v2:%s:%s:%s:%d:%s:%d", encode(tenantID), encode(userID), encode(runID), planRevision, encode(slotID), attempt)
	} else {
		base = fmt.Sprintf("image-agent-effect-recovery:%s:%s:%s:%d:%s:%d", tenantID, userID, runID, planRevision, slotID, attempt)
	}
	base = "organization-v1:" + base
	if strings.TrimSpace(actionID) == "" {
		return base
	}
	return base + ":action:" + base64.RawURLEncoding.EncodeToString([]byte(actionID))
}
