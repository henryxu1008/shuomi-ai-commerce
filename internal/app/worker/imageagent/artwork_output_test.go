package imageagentworker

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/require"
	"image"
	"image/png"
	"strings"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/imageagent"
	"testing"
	"time"
)

func TestArtworkOutputRecoveryUsesOriginalSuccessWithoutAnOriginalOrNewPost(t *testing.T) {
	var pixels bytes.Buffer
	require.NoError(t, png.Encode(&pixels, image.NewNRGBA(image.Rect(0, 0, 1024, 1024))))
	quote := imageagent.ImageGenerationQuote{Provider: "grsai", Model: "gpt-image-2.5", Protocol: "grsai-json-sync-v1", Resolution: "1024x1024", Quality: "auto", Points: 12, PriceVersion: "price", RouteReference: "route", CredentialReference: "credential", ConfigurationVersion: "config"}
	a := &imageagent.ArtworkPlan{Schema: imageagent.ArtworkSchema, Theme: "原创动物", PromptVersion: imageagent.ArtworkSchema, Prompt: "Create original artwork", Configuration: agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: "00000000-0000-4000-8000-000000000001", Digest: strings.Repeat("a", 64)}, ModeSnapshotRef: agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: "00000000-0000-4000-8000-000000000002", Digest: strings.Repeat("b", 64)}, ConfigurationEpoch: "1", ParametersDigest: strings.Repeat("c", 64), InputDigest: strings.Repeat("d", 64), Quote: quote, MaxPoints: quote.Points, QuoteDigest: imageagent.ArtworkQuoteDigest(quote)}
	catalog, err := imageagent.NormalizeAssetCatalog(imageagent.AssetCatalog{})
	require.NoError(t, err)
	in := imageagent.SlotExecutionInput{Artwork: a, RunID: "artwork-run", TenantID: "org", UserID: "actor", PlanRevision: 1, Attempt: 1, IdempotencyKey: "artwork-attempt", TargetPlatform: "sds", AssetCatalog: catalog, OrganizationIdentity: imageagent.ExecutionIdentity{MemberID: "member"}, Slot: imageagent.Slot{ID: "artwork", Role: imageagent.SlotRoleMain, Brief: a.Theme, IdempotencyKey: "artwork"}}
	digest, err := imageagent.ArtworkGenerationInputDigest(in)
	require.NoError(t, err)
	intent := imageagent.GenerationIntent{Identity: imageagent.SlotExternalEffectIdentity{RunScope: imageagent.RunScope{TenantID: in.TenantID, OwnerUserID: in.UserID, RunID: in.RunID}, PlanRevision: 1, SlotID: "artwork", Attempt: 1}, MemberID: "member", CatalogHash: catalog.Manifest.Hash, SourceDigest: digest, InputDigest: digest, InputProtocol: imageagent.ArtworkSchema, PromptVersion: imageagent.ArtworkSchema, Provider: quote.Provider, Model: quote.Model, Protocol: quote.Protocol, Resolution: quote.Resolution, Quality: quote.Quality, Points: quote.Points, PriceVersion: quote.PriceVersion, RouteReference: quote.RouteReference, CredentialReference: quote.CredentialReference, ConfigurationVersion: quote.ConfigurationVersion, LimitVersion: 1, MonthStart: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)}
	fact, err := imageagent.NewGenerationFact(intent)
	require.NoError(t, err)
	fact, err = fact.BindReservation(imageagent.GenerationReservationReceipt{IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, OrganizationID: in.TenantID, MemberID: "member", OperationID: "image-reserve:" + fact.IntentID, ReservationID: "reserve", ResourceType: "ai_point", Points: quote.Points, PriceVersion: quote.PriceVersion, LimitVersion: 1, MonthStart: intent.MonthStart})
	require.NoError(t, err)
	fact, _, err = fact.BeginDispatch()
	require.NoError(t, err)
	fact, err = fact.RecordSuccess(imageagent.GenerationSuccess{ResponseID: "response", ResultDigest: strings.Repeat("e", 64)}.WithResultLocator("https://output.example/artwork.png", ""))
	require.NoError(t, err)
	for _, test := range []string{"valid", "changed", "foreign", "truncated"} {
		t.Run(test, func(t *testing.T) {
			input := in
			input.Artwork = imageagent.CloneArtworkPlan(in.Artwork)
			if test == "changed" {
				input.Artwork.Prompt = "changed"
			}
			if test == "foreign" {
				input.UserID = "other"
			}
			gets := 0
			recover := generationOutputRecovery(func(context.Context, string) ([]byte, error) {
				gets++
				if test == "truncated" {
					return pixels.Bytes()[:pixels.Len()-16], nil
				}
				return pixels.Bytes(), nil
			}, nil)
			out, err := recover(context.Background(), input, fact)
			if test != "valid" {
				require.Error(t, err)
				if test == "changed" || test == "foreign" {
					require.Zero(t, gets)
				}
				return
			}
			require.NoError(t, err)
			require.Equal(t, 1, gets)
			require.Empty(t, out.SourceAssetID)
			require.Empty(t, out.Assets[0].SourceURL)
			require.NoError(t, out.Assets[0].Artwork.ValidateForInput(input))
			require.Equal(t, pixels.Bytes(), out.Assets[0].Bytes)
		})
	}
}
