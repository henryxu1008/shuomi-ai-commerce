package store

import (
	"context"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/imageagent"
	resourceadapter "task-processor/internal/integration/orgresource"
	"testing"
)

func TestGenerationRejectionSettlesOriginalReservationOnceAndRetainsFence(t *testing.T) {
	owner, commercial, _, intent := generationPointDatabases(t, true)
	// Use the exact admitted route and keep original economic identity.
	intent.Provider = "bailian"
	intent.Model = "qwen-image-3.0-pro"
	intent.Protocol = "dashscope-image-sync-v1"
	fact, err := owner.PrepareGenerationIntent(context.Background(), intent)
	require.NoError(t, err)
	originalTerminal, err := fact.RecordNoGeneration()
	require.NoError(t, err)
	resources, err := resourceadapter.NewGormImageGenerationRepository(commercial, resourceadapter.TransactionConfig{}, owner, generationPointTestAuth{})
	require.NoError(t, err)
	receipt, err := resources.ReserveImageGeneration(context.Background(), intent.Identity)
	require.NoError(t, err)
	_, err = owner.BindGenerationReservation(context.Background(), intent, receipt)
	require.NoError(t, err)
	_, _, err = owner.BeginGenerationDispatch(context.Background(), intent)
	require.NoError(t, err)
	_, err = owner.MarkGenerationUnknown(context.Background(), intent)
	require.NoError(t, err)
	proof := imageagent.GenerationRejection{Kind: "bailian_free_tier", Digest: strings.Repeat("a", 64), RequestID: "070f62f5-244a-9703-85c5-a5f0c0d039e0", HTTPStatus: 403, ProviderCode: "AllocationQuota.FreeTierOnly"}
	// Losing the evidence write must roll the terminal fact back too.
	require.NoError(t, owner.db.Exec(`CREATE TRIGGER reject_evidence BEFORE INSERT ON image_agent_v2_events WHEN NEW.type='generation.no_generation' BEGIN SELECT RAISE(ABORT,'evidence fault'); END`).Error)
	_, err = owner.RecordGenerationRejection(context.Background(), intent, proof)
	require.Error(t, err)
	var count int64
	require.NoError(t, owner.db.Model(&eventRecord{}).Where("type = ?", "generation.no_generation").Count(&count).Error)
	require.Zero(t, count)
	fact, err = owner.ReadGenerationFact(context.Background(), intent.Identity)
	require.NoError(t, err)
	require.Equal(t, imageagent.GenerationUnknown, fact.State)
	require.NoError(t, owner.db.Exec(`DROP TRIGGER reject_evidence`).Error)
	fact, err = owner.RecordGenerationRejection(context.Background(), intent, proof)
	require.NoError(t, err)
	require.Equal(t, originalTerminal.TerminalProofDigest(), fact.TerminalProofDigest())
	recovery, err := imageagent.NewGenerationRecovery(owner, resources)
	require.NoError(t, err)
	for n := 0; n < 2; n++ {
		_, err = owner.RecordGenerationRejection(context.Background(), intent, proof)
		require.NoError(t, err)
		state, err := recovery.ReconcileExistingGeneration(context.Background(), intent.Identity)
		require.NoError(t, err)
		require.Equal(t, imageagent.GenerationNoEffect, state)
	}
	fact, err = owner.ReadGenerationFact(context.Background(), intent.Identity)
	require.NoError(t, err)
	require.Equal(t, "released", fact.Settlement.State)
	require.Equal(t, receipt.ReservationID, fact.Settlement.ReservationID)
	var bucket struct{ Available, Reserved, Consumed int64 }
	require.NoError(t, commercial.Table("saas_organization_resource_buckets").Where("organization_id = ?", intent.Identity.TenantID).Take(&bucket).Error)
	require.EqualValues(t, 100, bucket.Available)
	require.Zero(t, bucket.Reserved)
	require.Zero(t, bucket.Consumed)
	require.NoError(t, owner.db.Model(&eventRecord{}).Where("type = ?", "generation.no_generation").Count(&count).Error)
	require.EqualValues(t, 1, count)
	// A retained parent conversation must be able to save after the evidence
	// event, without rewriting the blocked historical snapshot or its receipt.
	current, err := owner.GetProjection(context.Background(), intent.Identity.RunScope)
	require.NoError(t, err)
	var evidence eventRecord
	require.NoError(t, owner.db.Where("type = ?", "generation.no_generation").First(&evidence).Error)
	commit := imageagent.ProjectionCommit{Scope: intent.Identity.RunScope, CommitID: "parent:after-rejection", ExpectedProjectionVersion: current.ProjectionVersion, Snapshot: current, EventType: "command.receipt.updated", EventPayload: []byte(`{}`)}
	next, err := owner.CommitProjection(context.Background(), commit)
	require.NoError(t, err)
	require.Equal(t, evidence.Cursor+1, next.ProjectionVersion)
	replayed, err := owner.CommitProjection(context.Background(), commit)
	require.NoError(t, err)
	require.Equal(t, next, replayed)
	changed := proof
	changed.Digest = strings.Repeat("b", 64)
	_, err = owner.RecordGenerationRejection(context.Background(), intent, changed)
	require.ErrorIs(t, err, imageagent.ErrRevisionConflict)
	_, err = owner.RecordGenerationSuccess(context.Background(), intent, imageagent.GenerationSuccess{ResponseID: "late", ResultDigest: strings.Repeat("c", 64), ResultUnavailable: "invalid_result"})
	require.Error(t, err)
	effect, err := owner.GetSlotExternalEffectV3(context.Background(), intent.Identity)
	require.NoError(t, err)
	require.Equal(t, imageagent.SlotEffectV3ProviderClaimed, effect.Phase)
}
