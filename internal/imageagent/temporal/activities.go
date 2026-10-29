package temporal

import (
	"context"
	"errors"
	"fmt"
	"task-processor/internal/imageagent"
	productasset "task-processor/internal/product/asset"
	"time"
)

const slotResultPersistedEventType = "slot.result.persisted"

var errPublicationOwnerRequiresActivity = errors.New("publication owner requires a Temporal activity context")

type RecoveryWorkflowStarter func(context.Context, EffectRecoveryWorkflowInput) error

type ActivityDependencies struct {
	ArtworkPublisher         imageagent.ArtworkPublisher
	ImageSetPolicyAdopter    imageagent.ImageSetPolicyAdopter
	ImageSetApprovals        productasset.ApprovalCommitReader
	ImageSetPublisher        imageagent.ApprovedImageSetPublisher
	GenerationRecovery       imageagent.GenerationRecovery
	GenerationOutputRecovery imageagent.GenerationOutputRecovery
	ExecutionAuthorizer      imageagent.ExecutionAuthorizer
	Repository               imageagent.Repository
	PublisherV3              imageagent.ApprovedAssetPublisherV3
	SlotEffectsV3            imageagent.SlotExternalEffectV3Repository
	StagedSlotExecutor       imageagent.StagedSlotExecutor
	ArtifactStore            DurableArtifactStore
	PublicationOwner         func(context.Context) (string, error)
	PublicationLeaseDuration time.Duration
	RecoveryWorkflowStarter  RecoveryWorkflowStarter
}

type Activities struct {
	artworkPublisher         imageagent.ArtworkPublisher
	imageSetPolicyAdopter    imageagent.ImageSetPolicyAdopter
	imageSetApprovals        productasset.ApprovalCommitReader
	imageSetPublisher        imageagent.ApprovedImageSetPublisher
	generationRecovery       imageagent.GenerationRecovery
	generationOutputRecovery imageagent.GenerationOutputRecovery
	executionAuthorizer      imageagent.ExecutionAuthorizer
	repository               imageagent.Repository
	publisherV3              imageagent.ApprovedAssetPublisherV3
	slotEffectsV3            imageagent.SlotExternalEffectV3Repository
	stagedSlotExecutor       imageagent.StagedSlotExecutor
	artifactStore            DurableArtifactStore
	publicationOwner         func(context.Context) (string, error)
	publicationLeaseDuration time.Duration
	recoveryWorkflowStarter  RecoveryWorkflowStarter
}

func NewActivities(dependencies ActivityDependencies) (*Activities, error) {
	if dependencies.ImageSetApprovals != nil && (dependencies.ExecutionAuthorizer == nil || dependencies.ImageSetPublisher == nil) {
		return nil, fmt.Errorf("image set approval recovery requires organization publication assembly")
	}
	if (dependencies.GenerationRecovery != nil || dependencies.GenerationOutputRecovery != nil) && dependencies.ExecutionAuthorizer == nil {
		return nil, fmt.Errorf("generation recovery requires organization execution assembly")
	}
	if dependencies.Repository == nil {
		return nil, fmt.Errorf("image agent repository is required")
	}
	if dependencies.ExecutionAuthorizer == nil {
		return nil, fmt.Errorf("organization image agent execution authorizer is required")
	}
	if dependencies.SlotEffectsV3 == nil {
		if slotEffects, ok := dependencies.Repository.(imageagent.SlotExternalEffectV3Repository); ok {
			dependencies.SlotEffectsV3 = slotEffects
		}
	}
	if dependencies.SlotEffectsV3 == nil {
		return nil, fmt.Errorf("image agent v3 slot external effect repository is required")
	}
	if dependencies.StagedSlotExecutor == nil {
		return nil, fmt.Errorf("image agent staged slot executor is required")
	}
	if dependencies.ArtifactStore == nil {
		return nil, fmt.Errorf("image agent durable artifact store is required")
	}
	if dependencies.PublisherV3 == nil && dependencies.ImageSetPublisher == nil {
		return nil, fmt.Errorf("image agent v3 approved asset publisher is required")
	}
	if dependencies.PublicationOwner == nil {
		dependencies.PublicationOwner = temporalPublicationOwner
	}
	if dependencies.PublicationLeaseDuration <= 0 {
		dependencies.PublicationLeaseDuration = 2 * time.Minute
	}
	return &Activities{
		artworkPublisher:         dependencies.ArtworkPublisher,
		imageSetApprovals:        dependencies.ImageSetApprovals,
		generationRecovery:       dependencies.GenerationRecovery,
		imageSetPublisher:        dependencies.ImageSetPublisher,
		generationOutputRecovery: dependencies.GenerationOutputRecovery,
		executionAuthorizer:      dependencies.ExecutionAuthorizer,
		imageSetPolicyAdopter:    dependencies.ImageSetPolicyAdopter,
		repository:               dependencies.Repository, publisherV3: dependencies.PublisherV3,
		slotEffectsV3: dependencies.SlotEffectsV3, stagedSlotExecutor: dependencies.StagedSlotExecutor, artifactStore: dependencies.ArtifactStore,
		publicationOwner: dependencies.PublicationOwner, publicationLeaseDuration: dependencies.PublicationLeaseDuration,
		recoveryWorkflowStarter: dependencies.RecoveryWorkflowStarter,
	}, nil
}
