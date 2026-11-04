package imageagent

import (
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/aicapability"
	"testing"
)

func TestFlareReferenceRouteKeepsTheApprovedModelAndAutoQuality(t *testing.T) {
	require.Equal(t, "auto", GenerationRouteQuality("openai", aicapability.FlareImageModel, aicapability.FlareImageProtocol))
	require.Empty(t, GenerationRouteQuality("grsai", aicapability.FlareImageModel, aicapability.FlareImageProtocol))
	require.Empty(t, GenerationRouteQuality("openai", "gpt-image-2.5-flare", aicapability.FlareImageProtocol))
}

func TestFlareExplicitZeroUsageIsNotMissingOrAnotherPolicy(t *testing.T) {
	s := GenerationSuccess{ResponseID: "synthetic-response", ResultDigest: strings.Repeat("a", 64), ResultUnavailable: "invalid_result", UsageKnown: true,
		BillableUsage: aicapability.BillableModelUsage{MappingVersion: aicapability.ReferenceImagesTotalsV1, Complete: true, ReferenceImageTotals: aicapability.ReferenceImageTotals{Known: true}}}
	require.True(t, validGenerationSuccess(s))
	s.BillableUsage.ReferenceImageTotals.Known = false
	require.False(t, validGenerationSuccess(s))
	s.BillableUsage = aicapability.BillableModelUsage{MappingVersion: "openai-images-direct-v1", Complete: true}
	require.False(t, validGenerationSuccess(s), "the reference strategy must not relax another protocol")
}

func TestFlareProfileRefusesInputsOutsideFrozenEstimateBeforeReservation(t *testing.T) {
	f := flareFactFixture(t)
	m := GenerationProviderMetadata{Provider: f.Intent.Provider, Model: f.Intent.Model, Protocol: f.Intent.Protocol, Billing: f.Intent.Billing}
	tooLong := SlotExecutionInput{Artwork: &ArtworkPlan{Prompt: strings.Repeat("x", m.Billing.ReferenceEstimate.MaximumPromptBytes+1)}}
	require.Error(t, validateGenerationRouteInput(m, tooLong))
	in := SlotExecutionInput{Slot: Slot{Recipe: &ImageSlotRecipe{Prompt: "Keep product", References: []ImageSourceObservation{{Bytes: 100, Width: 10, Height: 10, MediaType: "image/png"}}}}}
	require.NoError(t, validateGenerationRouteInput(m, in))
	in.Slot.Recipe.References[0].Bytes = m.Billing.ReferenceEstimate.MaximumReferenceBytes + 1
	require.Error(t, validateGenerationRouteInput(m, in))
}
