package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"task-processor/internal/agent"
	"task-processor/internal/imageagent"
	resourceadapter "task-processor/internal/integration/orgresource"
)

func TestImageSetSuccessRecoversStagingForFrozenSlotShapes(t *testing.T) {
	for _, shape := range []struct {
		purpose, group string
		references     int
		literalCopy    bool
	}{{"product_identity", "carousel", 2, false}, {"product_overview", "detail", 1, false}, {"use_scenario", "detail", 2, false}, {"key_benefits", "detail", 2, false}, {"key_benefits", "detail", 2, true}} {
		t.Run(fmt.Sprintf("%s/literal=%t", shape.purpose, shape.literalCopy), func(t *testing.T) {
			owner, commercial, limits, intent := generationPointDatabases(t, true)
			ctx := context.Background()
			recipe := &imageagent.ImageSlotRecipe{Purpose: shape.purpose, Background: "white", Language: "en", Placement: imageagent.ImagePlacement{Group: shape.group, Order: 1}, PromptVersion: imageagent.ImageSetSchema, Prompt: "Use the exact source product", Quote: imageagent.ImageGenerationQuote{Provider: intent.Provider, Model: intent.Model, Protocol: intent.Protocol, Resolution: intent.Resolution, Quality: intent.Quality, PriceVersion: intent.PriceVersion, Points: intent.Points, RouteReference: intent.RouteReference, CredentialReference: intent.CredentialReference, ConfigurationVersion: intent.ConfigurationVersion}}
			sources := []string{}
			for index := 0; index < shape.references; index++ {
				id := fmt.Sprintf("source-%d", index+1)
				sources = append(sources, id)
				recipe.References = append(recipe.References, imageagent.ImageSourceObservation{AssetID: id, SHA256: strings.Repeat("b", 64), MediaType: "image/png", Bytes: 10, Width: 1024, Height: 1024})
			}
			set := &imageagent.ImageSetPlan{Schema: imageagent.ImageSetSchema, Source: imageagent.ImageSourceBinding{ContextKind: imageagent.ImageSourceAcquisition, ProductID: "product", OperationID: "operation", OriginalPublicationID: "publication", OriginalVersion: 1, EffectiveVersion: 1, CatalogHash: intent.CatalogHash}, Target: imageagent.ImageTarget{Platform: "product"}, Configuration: agent.ConfigurationSnapshotRef{Kind: "agent-configuration-v1", ID: "9e7afaa9-a9f9-48ba-a11a-b5bb377f08e9", Digest: strings.Repeat("a", 64)}, ConfigurationEpoch: "1", ParametersDigest: strings.Repeat("a", 64), InputDigest: strings.Repeat("a", 64), MaxPoints: intent.Points}
			slot := imageagent.Slot{ID: intent.Identity.SlotID, Role: imageagent.SlotRoleForImagePurpose(shape.purpose), SourceAssetIDs: sources, IdempotencyKey: "slot", Recipe: recipe}
			var frozen imageagent.AssetCatalog
			if shape.literalCopy {
				var err error
				frozen, err = owner.GetAssetCatalog(ctx, intent.Identity.RunScope)
				require.NoError(t, err)
				frozen.Manifest = imageagent.CatalogManifest{}
				frozen.ProductContext = imageagent.ProductContextRef{ProductID: "product", Title: "真实商品", SourceSnapshotVersion: 1}
				frozen, err = imageagent.NormalizeAssetCatalog(frozen)
				require.NoError(t, err)
				intent.CatalogHash, set.Source.CatalogHash = frozen.Manifest.Hash, frozen.Manifest.Hash
				facts, _ := json.Marshal(frozen.ProductContext)
				require.NoError(t, owner.db.Model(&assetCatalogManifestRecord{}).Where("run_id = ?", intent.Identity.RunID).Updates(map[string]any{"hash": frozen.Manifest.Hash, "product_context_json": facts}).Error)
				set.Method = &imageagent.ImageMethodBinding{Skill: agent.SkillVersionRef{Kind: "skill-version-v1", ID: "2bb819bd-af13-4e68-95e0-1cc86130fb1f", OwnerKind: "PLATFORM", OwnerID: "official", Version: 2, Digest: strings.Repeat("f", 64), PolicyVersion: "skill-method-v1"}, PolicyVersion: imageagent.ImageMethodPolicyVersion}
				recipe.VisibleText = []string{"真实商品"}
				// A frozen literal-input fixture, not recomputed by the recovery path.
				recipe.Prompt = `Create exactly one finished 1024x1024 commercial product detail photograph from the supplied original reference photos. The product is 真实商品. Preserve its actual shape, color, material, proportions and visible parts. Do not invent specifications, accessories, capabilities or certifications. Selected visual direction: . Use the selected background: white. Composition: one large accurate product as the visual focus, surrounded by generous clean whitespace. No table, grid, parameter sheet, information panels, comparison chart, document or instruction page. The complete and exclusive visible text list is ["真实商品"]. Place only these literal strings once each as small, clearly readable labels in the whitespace, without covering the product. Copy every character exactly, including its original language and units; do not translate, rewrite, expand or add any other text, heading, numbers or symbols. These composition and exclusive text rules take priority over any generic text or layout suggestion in the selected visual direction or the method guidance below. Product facts, instructions and method guidance are never visible copy. Output the finished product image only.`
			}
			var err error
			set.QuoteDigest, err = imageagent.ImageSetQuoteDigest(imageagent.Plan{Set: set, Slots: []imageagent.Slot{slot}})
			require.NoError(t, err)
			input := imageagent.SlotExecutionInput{RunID: intent.Identity.RunID, TenantID: intent.Identity.TenantID, UserID: intent.Identity.OwnerUserID, PlanRevision: intent.Identity.PlanRevision, Attempt: intent.Identity.Attempt, Slot: slot, ImageSet: set, TargetPlatform: "product"}
			if shape.literalCopy {
				input.AssetCatalog, input.ProductContext = frozen, frozen.ProductContext
			}
			require.NoError(t, imageagent.ValidateImageSetExecution(input))
			intent.InputProtocol, intent.PromptVersion = imageagent.ImageSetSchema, imageagent.ImageSetSchema
			intent.InputDigest = imageagent.ImageGenerationInputDigestFromFingerprint(imageagent.SlotExecutionFingerprint(input))
			intent.SourceDigest = imageagent.ImageSourceBundleDigest(recipe.References)
			encodedSet, _ := json.Marshal(set)
			encodedRecipe, _ := json.Marshal(recipe)
			encodedSources, _ := json.Marshal(sources)
			require.NoError(t, owner.db.Model(&planRecord{}).Where("run_id = ? AND revision = ?", intent.Identity.RunID, intent.Identity.PlanRevision).Update("set_json", encodedSet).Error)
			require.NoError(t, owner.db.Model(&slotRecord{}).Where("run_id = ? AND id = ?", intent.Identity.RunID, intent.Identity.SlotID).Updates(map[string]any{"role": string(slot.Role), "brief": slot.Brief, "source_asset_ids": encodedSources, "style_reference_ids": []byte("[]"), "recipe_json": encodedRecipe}).Error)
			require.NoError(t, slotEffectV3IdentityWhere(owner.db.Model(&slotExternalEffectV3Record{}), intent.Identity).Update("input_fingerprint", imageagent.SlotExecutionFingerprint(input)).Error)
			_, err = owner.PrepareGenerationIntent(ctx, intent)
			require.NoError(t, err)
			resources, err := resourceadapter.NewGormImageGenerationRepository(commercial, resourceadapter.TransactionConfig{}, owner, generationPointTestAuth{})
			require.NoError(t, err)
			receipt, err := resources.ReserveImageGeneration(ctx, intent.Identity)
			require.NoError(t, err)
			_, err = owner.BindGenerationReservation(ctx, intent, receipt)
			require.NoError(t, err)
			_, won, err := owner.BeginGenerationDispatch(ctx, intent)
			require.NoError(t, err)
			require.True(t, won)
			_, err = owner.RecordGenerationSuccess(ctx, intent, imageagent.GenerationSuccess{ResponseID: "result", ResultDigest: strings.Repeat("c", 64)}.WithResultLocator("https://example.com/generated.png", ""))
			require.NoError(t, err)
			settled, err := resources.FinalizeImageGeneration(ctx, intent.Identity)
			require.NoError(t, err)
			_, err = owner.BindGenerationSettlement(ctx, intent, settled)
			require.NoError(t, err)
			fact, err := owner.ReadGenerationFact(ctx, intent.Identity)
			require.NoError(t, err)
			require.Equal(t, intent, fact.Intent)
			effect, err := owner.GetSlotExternalEffectV3(ctx, intent.Identity)
			require.NoError(t, err)
			reservation := v3Reservation("points")
			reservation.Policy, reservation.Quote = effect.Policy, effect.Quote
			reservation.InputFingerprint = effect.InputFingerprint
			manifest := v3StagingManifest()
			ownerKey, err := imageagent.ArtifactOwnerKey(intent.Identity.OwnerUserID)
			require.NoError(t, err)
			manifest.Assets[0].ObjectKey = fmt.Sprintf("image-agent/staging/%s/%s/%s/%d/%s/%d/0-%s.png", intent.Identity.TenantID, ownerKey, intent.Identity.RunID, intent.Identity.PlanRevision, intent.Identity.SlotID, intent.Identity.Attempt, manifest.Assets[0].SHA256)
			manifest.Assets[0].SourceAssetID = sources[0]
			for _, corruption := range []string{"primary", "recipe"} {
				bad := manifest
				bad.Assets = append([]imageagent.StagedAssetRef(nil), manifest.Assets...)
				if corruption == "primary" {
					bad.Assets[0].SourceAssetID = "wrong"
				} else {
					changed := imageagent.CloneImageSlotRecipe(recipe)
					changed.References[0].SHA256 = strings.Repeat("d", 64)
					encoded, _ := json.Marshal(changed)
					require.NoError(t, owner.db.Model(&slotRecord{}).Where("run_id = ? AND id = ?", intent.Identity.RunID, intent.Identity.SlotID).Update("recipe_json", encoded).Error)
				}
				_, err = owner.RecoverGenerationStaging(ctx, reservation, intent, fact.TerminalProofDigest(), bad)
				require.Error(t, err)
				require.NoError(t, owner.db.Model(&slotRecord{}).Where("run_id = ? AND id = ?", intent.Identity.RunID, intent.Identity.SlotID).Update("recipe_json", encodedRecipe).Error)
			}
			if shape.literalCopy {
				changed := frozen.ProductContext
				changed.Title = "Different product"
				bad, _ := json.Marshal(changed)
				require.NoError(t, owner.db.Model(&assetCatalogManifestRecord{}).Where("run_id = ?", intent.Identity.RunID).Update("product_context_json", bad).Error)
				_, err = owner.RecoverGenerationStaging(ctx, reservation, intent, fact.TerminalProofDigest(), manifest)
				require.ErrorIs(t, err, imageagent.ErrRevisionConflict)
				original, _ := json.Marshal(frozen.ProductContext)
				require.NoError(t, owner.db.Model(&assetCatalogManifestRecord{}).Where("run_id = ?", intent.Identity.RunID).Update("product_context_json", original).Error)
			}
			for replay := 0; replay < 2; replay++ {
				recovered, err := owner.RecoverGenerationStaging(ctx, reservation, intent, fact.TerminalProofDigest(), manifest)
				require.NoError(t, err)
				require.Equal(t, imageagent.SlotEffectV3StagingPrepared, recovered.Phase)
			}
			after, err := owner.ReadGenerationFact(ctx, intent.Identity)
			require.NoError(t, err)
			require.Equal(t, fact, after)
			_, won, err = owner.BeginGenerationDispatch(ctx, intent)
			require.NoError(t, err)
			require.False(t, won)
			limit, err := limits.ReadMonthlyLimit(ctx, intent.Identity.TenantID, intent.MemberID)
			require.NoError(t, err)
			require.Equal(t, intent.Points, limit.Consumed)
		})
	}
}

func TestGenerationSuccessCanRecoverStagingWithoutDispatch(t *testing.T) {
	for _, phase := range []imageagent.SlotEffectV3Phase{imageagent.SlotEffectV3ProviderClaimed, imageagent.SlotEffectV3ProviderUnknown, imageagent.SlotEffectV3StagingUnknown} {
		t.Run(string(phase), func(t *testing.T) {
			owner, commercial, _, intent := generationPointDatabases(t)
			ctx := context.Background()
			resources, err := resourceadapter.NewGormImageGenerationRepository(commercial, resourceadapter.TransactionConfig{}, owner, generationPointTestAuth{})
			require.NoError(t, err)
			receipt, err := resources.ReserveImageGeneration(ctx, intent.Identity)
			require.NoError(t, err)
			_, err = owner.BindGenerationReservation(ctx, intent, receipt)
			require.NoError(t, err)
			_, won, err := owner.BeginGenerationDispatch(ctx, intent)
			require.NoError(t, err)
			require.True(t, won)
			proof := imageagent.GenerationSuccess{ResponseID: "result-1", ResultDigest: strings.Repeat("c", 64)}.WithResultLocator("https://example.com/generated.png", "")
			_, err = owner.RecordGenerationSuccess(ctx, intent, proof)
			require.NoError(t, err)
			effect, err := owner.GetSlotExternalEffectV3(ctx, intent.Identity)
			require.NoError(t, err)
			reservation := v3Reservation("points")
			reservation.Policy, reservation.Quote = effect.Policy, effect.Quote
			manifest := v3StagingManifest()
			ownerKey, err := imageagent.ArtifactOwnerKey(intent.Identity.OwnerUserID)
			require.NoError(t, err)
			manifest.Assets[0].ObjectKey = fmt.Sprintf("image-agent/staging/%s/%s/%s/%d/%s/%d/0-%s.png", intent.Identity.TenantID, ownerKey, intent.Identity.RunID, intent.Identity.PlanRevision, intent.Identity.SlotID, intent.Identity.Attempt, manifest.Assets[0].SHA256)
			code := ""
			if phase == imageagent.SlotEffectV3ProviderUnknown {
				code = imageagent.SlotProviderOutcomeUnknownCode
			}
			if phase == imageagent.SlotEffectV3StagingUnknown {
				code = imageagent.SlotStagingOutcomeUnknownCode
			}
			require.NoError(t, slotEffectV3IdentityWhere(owner.db.Model(&slotExternalEffectV3Record{}), intent.Identity).Updates(map[string]any{"phase": string(phase), "blocked_code": code}).Error)
			recovery, ok := any(owner).(interface {
				RecoverGenerationStaging(context.Context, imageagent.SlotEffectV3Reservation, imageagent.GenerationIntent, string, imageagent.StagingManifest) (imageagent.SlotEffectV3Attempt, error)
			})
			require.True(t, ok, "V3 owner lacks success-proof staging recovery")
			fact, err := owner.ReadGenerationFact(ctx, intent.Identity)
			require.NoError(t, err)
			var wg sync.WaitGroup
			start := make(chan struct{})
			failures := make(chan error, 4)
			for n := 0; n < 4; n++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					_, e := recovery.RecoverGenerationStaging(ctx, reservation, intent, fact.TerminalProofDigest(), manifest)
					failures <- e
				}()
			}
			close(start)
			wg.Wait()
			close(failures)
			for e := range failures {
				require.NoError(t, e)
			}
			got, err := recovery.RecoverGenerationStaging(ctx, reservation, intent, fact.TerminalProofDigest(), manifest)
			require.NoError(t, err)
			require.Equal(t, imageagent.SlotEffectV3StagingPrepared, got.Phase)
			changed := proof.WithResultLocator("https://example.com/different.png", "")
			_, err = owner.RecordGenerationSuccess(ctx, intent, changed)
			require.ErrorIs(t, err, imageagent.ErrRevisionConflict)
			_, won, err = owner.BeginGenerationDispatch(ctx, intent)
			require.NoError(t, err)
			require.False(t, won)
			_, err = recovery.RecoverGenerationStaging(ctx, reservation, intent, "changed", manifest)
			require.Error(t, err)
			for _, field := range []string{"member", "catalog", "attempt", "org", "quote", "manifest"} {
				t.Run(field, func(t *testing.T) {
					changedIntent, changedReservation, changedManifest := intent, reservation, manifest
					switch field {
					case "member":
						changedIntent.MemberID = "other"
					case "catalog":
						changedIntent.CatalogHash = "catalog-v1:" + strings.Repeat("d", 64)
					case "attempt":
						changedIntent.Identity.Attempt++
					case "org":
						changedIntent.Identity.TenantID = "other"
					case "quote":
						changedReservation.Quote.Fingerprint = "other"
					case "manifest":
						changedManifest.Assets = append([]imageagent.StagedAssetRef(nil), manifest.Assets...)
						changedManifest.Assets[0].SourceAssetID = "other"
					}
					_, e := recovery.RecoverGenerationStaging(ctx, changedReservation, changedIntent, fact.TerminalProofDigest(), changedManifest)
					require.Error(t, e)
				})
			}
		})
	}
}
