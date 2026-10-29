package imageagent

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateProjectionSnapshotAcceptsOnlyValidatedDurableV3Candidates(t *testing.T) {
	const (
		tenantID = "tenant-a"
		runID    = "run-a"
	)
	sha := strings.Repeat("a", 64)
	plan := Plan{Revision: 1, IdempotencyKey: "plan-key", SourceAssetIDs: []string{"source-1"}, Slots: []Slot{{ID: "scene-1", Role: SlotRoleMain, IdempotencyKey: "scene-key", SourceAssetIDs: []string{"source-1"}, Status: SlotStatusPending}}}
	run := Run{ID: runID, TenantID: tenantID, UserID: "user-a", ActivePlanRevision: 1}
	snapshot := RunProjection{
		Run:               run,
		Plan:              plan,
		ProjectionVersion: 1,
		LastEventID:       1,
		Slots: []SlotProjection{{
			Slot:    Slot{ID: "scene-1", Role: SlotRoleMain, Status: SlotStatusAccepted},
			Attempt: 1,
			Candidates: []AssetCandidate{{
				AssetID:       "candidate-1",
				SourceAssetID: "source-1",
				DurableAsset:  DurableAssetIdentity{ObjectKey: "image-agent/public/tenant-a/fc95297aa4f56781f0decb7d4bf59b1447f09b3611039b80188b1c6beb03ee6a/run-a/1/scene-1/1/0-" + sha + ".png", SHA256: sha},
			}},
		}},
	}

	require.NoError(t, ValidateProjectionSnapshot(ScopeForRun(run), snapshot))

	snapshot.Slots[0].Candidates[0].DurableAsset = DurableAssetIdentity{}
	require.ErrorIs(t, ValidateProjectionSnapshot(ScopeForRun(run), snapshot), ErrValidation)
}

func TestSlotProjectionJSONPreservesV3DurableIdentityWithoutChangingV2CandidateWire(t *testing.T) {
	sha := strings.Repeat("a", 64)
	v3 := SlotProjection{Candidates: []AssetCandidate{{
		AssetID: "candidate-1", SourceAssetID: "source-1",
		Width: 1200, Height: 900, Operations: []string{"render_scene", "review"},
		GenerationProof: &ImageGenerationProof{IntentID: sha, Fingerprint: sha, SettlementProofDigest: sha, Points: 12},
		Artwork:         &ArtworkProvenance{Protocol: ArtworkSchema, TenantID: "tenant-a", OwnerUserID: "user-a", RunID: "run-a", SlotID: "scene-1", PlanRevision: 1, Attempt: 1, InputDigest: sha, IntentID: sha, Fingerprint: sha},
		DurableAsset:    DurableAssetIdentity{ObjectKey: "image-agent/public/tenant-a/fc95297aa4f56781f0decb7d4bf59b1447f09b3611039b80188b1c6beb03ee6a/run-a/1/scene-1/1/0-" + sha + ".png", SHA256: sha},
	}}}
	encoded, err := json.Marshal(v3)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"DurableAsset"`)
	require.NotContains(t, string(encoded), `"URL"`)
	require.NotContains(t, string(encoded), `"Metadata"`)

	var decoded SlotProjection
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	require.Equal(t, v3, decoded)

	for _, field := range []string{`"URL":"https://cdn.example.test/legacy.png"`, `"Metadata":{"legacy":"kept"}`, `"URL":""`, `"URL":null`, `"Metadata":{}`, `"Metadata":null`, `"url":null`, `"mEtAdAtA":null`, `"\u0055RL":null`, `"\u004detadata":null`, `"URL":"forged","URL":null`} {
		for _, durable := range []bool{false, true} {
			t.Run(field+map[bool]string{false: "/only", true: "/mixed"}[durable], func(t *testing.T) {
				candidate := `{"AssetID":"candidate-1",` + field
				if durable {
					candidate += `,"DurableAsset":{"ObjectKey":"` + v3.Candidates[0].DurableAsset.ObjectKey + `","SHA256":"` + sha + `"}`
				}
				var rejected SlotProjection
				require.Error(t, json.Unmarshal([]byte(`{"Candidates":[`+candidate+`}]}`), &rejected))
			})
		}
	}
	for _, candidates := range []string{"null", "[]"} {
		var empty SlotProjection
		require.NoError(t, json.Unmarshal([]byte(`{"Candidates":`+candidates+`}`), &empty))
		if candidates == "null" {
			require.Nil(t, empty.Candidates)
		} else {
			require.NotNil(t, empty.Candidates)
			require.Empty(t, empty.Candidates)
		}
	}
	snapshot := RunProjection{Run: Run{ID: "run-a", TenantID: "tenant-a", UserID: "user-a", ActivePlanRevision: 1}, Plan: Plan{Revision: 1}, Slots: []SlotProjection{v3}, ProjectionVersion: 1, LastEventID: 1}
	full, err := json.Marshal(snapshot)
	require.NoError(t, err)
	commit, err := json.Marshal(ProjectionCommit{Scope: ScopeForRun(snapshot.Run), CommitID: "fixture-commit", Snapshot: snapshot, EventType: "fixture.event", EventPayload: json.RawMessage(`{"fixture":true}`)})
	require.NoError(t, err)
	require.Equal(t, "2179812526fba20fcb42f44ca16c21e54632872907c9f2affbb83a2cd9ec514a", fmt.Sprintf("%x", sha256.Sum256(full)), "complete current projection JSON bytes must remain frozen")
	require.Equal(t, "cc1ef100e1c5d8e7ac7330b7b884620f217654ebadf67f345fdcff70d0554654", fmt.Sprintf("%x", sha256.Sum256(commit)), "complete current commit JSON/fingerprint bytes must remain frozen")
}

func TestNormalizeRecoverableEffectsDeduplicatesStableEntriesAndRejectsConflicts(t *testing.T) {
	normalized, err := NormalizeRecoverableEffects([]RecoverableEffect{
		{SlotID: "slot-1", Attempt: 1, Code: "recovery_requested"},
		{SlotID: "slot-1", Attempt: 1, Code: "recovery_requested"},
		{SlotID: "slot-2", Attempt: 2, Code: "recovery_start_failed"},
	})
	require.NoError(t, err)
	require.Equal(t, []RecoverableEffect{
		{SlotID: "slot-1", Attempt: 1, Code: "recovery_requested"},
		{SlotID: "slot-2", Attempt: 2, Code: "recovery_start_failed"},
	}, normalized)

	_, err = NormalizeRecoverableEffects([]RecoverableEffect{
		{SlotID: "slot-1", Attempt: 1, Code: "recovery_requested"},
		{SlotID: "slot-1", Attempt: 1, Code: "recovery_start_failed"},
	})
	require.ErrorIs(t, err, ErrRevisionConflict)
}

func TestNormalizeRecoverableEffectsAcceptsReviewerTransportBlocks(t *testing.T) {
	normalized, err := NormalizeRecoverableEffects([]RecoverableEffect{{
		SlotID: "slot-1", Attempt: 1, Code: SlotReviewTransportRequiredCode,
	}})

	require.NoError(t, err)
	require.Equal(t, []RecoverableEffect{{
		SlotID: "slot-1", Attempt: 1, Code: SlotReviewTransportRequiredCode,
	}}, normalized)
}

func TestValidateProjectionSnapshotRejectsRecoverableEffectWithoutMatchingBlockedSlot(t *testing.T) {
	run := Run{ID: "run-a", TenantID: "tenant-a", UserID: "user-a", ActivePlanRevision: 1, Status: RunStatusBlocked}
	plan := Plan{Revision: 1, IdempotencyKey: "plan-key", SourceAssetIDs: []string{"source-1"}, Slots: []Slot{{ID: "scene-1", Role: SlotRoleMain, IdempotencyKey: "scene-key", SourceAssetIDs: []string{"source-1"}, Status: SlotStatusPending}}}
	snapshot := RunProjection{
		Run:               run,
		Plan:              plan,
		ProjectionVersion: 1,
		LastEventID:       1,
		Slots:             []SlotProjection{{Slot: Slot{ID: "scene-1", Role: SlotRoleMain, Status: SlotStatusBlocked}, Attempt: 1, ErrorCode: "recovery_requested"}},
		RecoverableEffects: []RecoverableEffect{
			{SlotID: "scene-1", Attempt: 2, Code: "recovery_requested"},
		},
	}

	require.ErrorIs(t, ValidateProjectionSnapshot(ScopeForRun(run), snapshot), ErrRevisionConflict)
}
