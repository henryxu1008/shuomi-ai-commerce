package imageagentworker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/require"
	"image"
	"image/png"
	"io"
	"net/http"
	"strconv"
	"strings"
	"task-processor/internal/ai"
	"task-processor/internal/aicapability"
	"task-processor/internal/core/config"
	"task-processor/internal/imageagent"
	"task-processor/internal/imageagent/objectstore"
	temporal "task-processor/internal/imageagent/temporal"
	"task-processor/internal/integration/openai"
	"testing"
	"time"
)

func flareWorkerBilling() imageagent.ImageBilling {
	return imageagent.ImageBilling{OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "reference-v1", FXVersion: "identity-v1", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: aicapability.FlareImageModel, Channel: "openai-direct", UsageVersion: aicapability.ReferenceImagesTotalsV1, Rates: aicapability.OfficialModelRates{InputText: 10000, InputImage: 20000, OutputImage: 10000}}, ReferenceEstimate: aicapability.ReferenceImageEstimate{InputTokens: 100, OutputTokens: 100, OperationScope: aicapability.ReferenceImageOperationScope, Size: "1024x1024", Quality: "auto", AllowAutoSize: true, MaximumPromptBytes: 65536, MaximumReferenceImages: 3, MaximumReferenceBytes: 16 << 20, MaximumReferencePixels: 64000000}}
}

type flareMemoryObjects struct {
	objectstore.ImmutableObjectStore
	objects map[string]objectstore.ImmutableObjectPut
}

func (m *flareMemoryObjects) PublicURL(key string) string { return "https://artifact.example/" + key }
func (m *flareMemoryObjects) InspectObject(_ context.Context, key string) (objectstore.ObjectInspection, error) {
	p, ok := m.objects[key]
	return objectstore.ObjectInspection{Exists: ok, ContentLength: p.SizeBytes, ContentType: p.ContentType, Metadata: map[string]string{"sha256": p.SHA256, "size-bytes": strconv.FormatInt(p.SizeBytes, 10)}}, nil
}
func (m *flareMemoryObjects) ReadObject(ctx context.Context, key string, max int64) ([]byte, objectstore.ObjectInspection, error) {
	i, _ := m.InspectObject(ctx, key)
	if !i.Exists {
		return nil, i, objectstore.ErrArtifactUnavailable
	}
	p := m.objects[key]
	if int64(len(p.Data)) > max {
		return nil, i, objectstore.ErrArtifactUnavailable
	}
	return append([]byte(nil), p.Data...), i, nil
}
func (m *flareMemoryObjects) PutImmutable(_ context.Context, p objectstore.ImmutableObjectPut) error {
	if old, ok := m.objects[p.Key]; ok && !bytes.Equal(old.Data, p.Data) {
		return objectstore.ErrObjectConflict
	}
	p.Data = append([]byte(nil), p.Data...)
	m.objects[p.Key] = p
	return nil
}

type flareArtifactFault struct {
	temporal.DurableArtifactStore
	mode string
}

func (s flareArtifactFault) PreserveSlotArtifacts(ctx context.Context, id imageagent.SlotExternalEffectIdentity, p objectstore.PreparedSlotArtifacts) error {
	if s.mode == "put failed" {
		return errors.New("controlled PUT failure")
	}
	if err := s.DurableArtifactStore.PreserveSlotArtifacts(ctx, id, p); err != nil {
		return err
	}
	if s.mode == "put ACK lost" {
		return errors.New("controlled PUT ACK loss")
	}
	return nil
}

type flareReadFact struct {
	imageagent.GenerationFactRepository
	fact imageagent.GenerationFact
}

func (f flareReadFact) ReadGenerationFact(context.Context, imageagent.SlotExternalEffectIdentity) (imageagent.GenerationFact, error) {
	return f.fact, nil
}

func TestFlareFactoryKeepsBase64InOriginalBundleAndRecordsUsageOnStorageFailure(t *testing.T) {
	for _, mode := range []string{"success", "put failed", "put ACK lost", "observer ACK lost", "bad pixels"} {
		t.Run(mode, func(t *testing.T) {
			ctx := syntheticGenerationContext()
			var pixels bytes.Buffer
			require.NoError(t, png.Encode(&pixels, image.NewRGBA(image.Rect(0, 0, 1024, 1024))))
			encoded := base64.StdEncoding.EncodeToString(pixels.Bytes())
			if mode == "bad pixels" {
				encoded = base64.StdEncoding.EncodeToString([]byte("invalid image"))
			}
			objects := &flareMemoryObjects{objects: map[string]objectstore.ImmutableObjectPut{}}
			artifacts, err := objectstore.NewDurableArtifactStore(objects, objectstore.DurableArtifactStoreConfig{MaxArtifactBytes: 32 << 20, OperationTimeout: time.Second})
			require.NoError(t, err)
			resolver := &generationConfigTestResolver{resolved: &generationFixtureConfig{CacheKey: "credential-v1", Config: &openai.ClientConfig{APIKey: "synthetic-only", APIStyle: "openai", Model: aicapability.FlareImageModel, BaseURL: "https://api.openai.com/v1", Timeout: time.Second}}}
			current := syntheticFactoryCurrent(resolver, 300)
			current.q.Image.Provider = "openai"
			current.q.Image.Protocol = aicapability.FlareImageProtocol
			current.q.Image.Billing = flareWorkerBilling()
			current.q.Image.PriceVersion = "reference-v1"
			current.q.Price = current.q.Image.Billing.OfficialTariff
			calls, observed := 0, 0
			var proof imageagent.GenerationSuccess
			factory := generationProviderFactory{current: current, credentials: resolver, price: config.ImageAgentGenerationConfig{Enabled: true, UnitVersion: aicapability.AIPointMicrounitV1, MaximumMicropoints: 300}, profile: synchronousEditProfile{}, facts: unusedArtworkFacts{}, artifacts: flareArtifactFault{artifacts, mode}, providerHTTPClient: &http.Client{Transport: qwenFactoryTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, "/v1/images/edits", r.URL.Path)
				var body map[string]json.RawMessage
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.JSONEq(t, `"auto"`, string(body["size"]))
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"data":[{"b64_json":"` + encoded + `"}],"usage":{"input_tokens":100,"output_tokens":50,"total_tokens":150}}`))}, nil
			})}}
			sum := sha256.Sum256(pixels.Bytes())
			in := imageagent.SlotExecutionInput{RunID: "run-1", IdempotencyKey: "attempt", TenantID: "org-1", UserID: "actor", PlanRevision: 1, Attempt: 1, TargetPlatform: "product", ImagePolicyContext: &imageagent.ImagePolicyContext{Country: "zz", Family: "default", SceneCategory: "general"}, Slot: imageagent.Slot{ID: "main-1", Role: imageagent.SlotRoleMain, SourceAssetIDs: []string{"source-1"}, IdempotencyKey: "slot"}, AssetCatalog: imageagent.AssetCatalog{Assets: []imageagent.AuthorizedAsset{{ID: "source-1", Type: imageagent.AuthorizedAssetSource, URL: "https://source.example/image.png"}}}, ProductContext: imageagent.ProductContextRef{ProductID: "product-1", Title: "Product"}, SourceBytes: pixels.Bytes(), SourceDigest: hex.EncodeToString(sum[:])}
			prepared, err := factory.prepare(ctx, in, func(_ context.Context, p imageagent.GenerationSuccess) error {
				observed++
				proof = p
				if mode == "observer ACK lost" {
					return errors.New("controlled observed ACK loss")
				}
				return nil
			})
			require.NoError(t, err)
			quote, err := prepared.Executor.QuoteSlot(ctx, in, imageagent.BudgetPolicy{CostMicros: imageagent.Limit{Enabled: true, Value: 100}})
			require.NoError(t, err)
			require.EqualValues(t, 3, quote.Maximum.CostMicros)
			require.Zero(t, calls)
			out, err := prepared.Executor.GenerateQuotedSlot(ctx, in, quote)
			require.Equal(t, 1, calls)
			require.Equal(t, 1, observed)
			require.EqualValues(t, 100, proof.InputTokens)
			require.True(t, proof.BillableUsage.ReferenceImageTotals.Known)
			if mode == "success" {
				require.NoError(t, err)
				require.Equal(t, pixels.Bytes(), out.Assets[0].Bytes)
				require.Equal(t, proof.ResultDigest, out.Assets[0].ProviderReceiptID)
			} else {
				require.Error(t, err)
			}
			if mode == "bad pixels" {
				require.Equal(t, "invalid_result", proof.ResultUnavailable)
				require.Empty(t, proof.ResultLocatorKind)
				return
			}
			require.Equal(t, "artifact_bundle", proof.ResultLocatorKind)
			require.Empty(t, proof.ResultURL)
			id := imageagent.SlotExternalEffectIdentity{RunScope: imageagent.RunScope{TenantID: in.TenantID, OwnerUserID: in.UserID, RunID: in.RunID}, PlanRevision: 1, SlotID: in.Slot.ID, Attempt: 1}
			recovered, e := artifacts.RecoverSlotArtifacts(ctx, id, imageagent.StagingManifest{})
			if mode == "put failed" {
				require.Error(t, e)
				return
			}
			require.NoError(t, e)
			require.NoError(t, artifacts.EnsureStaged(ctx, recovered))
			require.Equal(t, proof.ResultDigest, recovered.Manifest.Assets[0].ProviderReceiptID)
			// Normal staging and recovery must converge on identical immutable metadata.
			if mode == "success" {
				normal, e := artifacts.PrepareSlotArtifacts(objectstore.PrepareSlotArtifactsInput{Identity: id, Assets: []objectstore.ArtifactInput{{Bytes: out.Assets[0].Bytes, ContentType: out.Assets[0].ContentType, Width: out.Assets[0].Width, Height: out.Assets[0].Height, SourceAssetID: out.SourceAssetID, Operations: out.Assets[0].Operations, ProviderReceiptID: out.Assets[0].ProviderReceiptID}}})
				require.NoError(t, e)
				require.Equal(t, recovered.Manifest, normal.Manifest)
			}
		})
	}
}

func TestFlareArtworkBase64UsesOriginalProvenanceAndNoURLDownload(t *testing.T) {
	_, in := admittedWorkerArtwork(t)
	billing := flareWorkerBilling()
	q := in.Artwork.Quote
	q.Billing = billing
	q.Provider = "openai"
	q.Model = aicapability.FlareImageModel
	q.Protocol = aicapability.FlareImageProtocol
	q.Points = 300
	q.PriceVersion = "reference-v1"
	in.Artwork.Quote = q
	in.Artwork.MaxPoints = 300
	in.Artwork.QuoteDigest = imageagent.ArtworkQuoteDigest(q)
	digest, err := imageagent.ArtworkGenerationInputDigest(in)
	require.NoError(t, err)
	fact, err := imageagent.NewGenerationFact(imageagent.GenerationIntent{Billing: billing, Identity: imageagent.SlotExternalEffectIdentity{RunScope: imageagent.RunScope{TenantID: in.TenantID, OwnerUserID: in.UserID, RunID: in.RunID}, PlanRevision: 1, SlotID: in.Slot.ID, Attempt: 1}, MemberID: "member", CatalogHash: in.AssetCatalog.Manifest.Hash, SourceDigest: digest, PromptVersion: imageagent.ArtworkSchema, RouteReference: q.RouteReference, CredentialReference: q.CredentialReference, ConfigurationVersion: q.ConfigurationVersion, Provider: q.Provider, Model: q.Model, Protocol: q.Protocol, Resolution: q.Resolution, Quality: q.Quality, PriceVersion: q.PriceVersion, Points: q.Points, LimitVersion: 1, MonthStart: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), InputProtocol: imageagent.ArtworkSchema, InputDigest: digest})
	require.NoError(t, err)
	f := fact
	receipt := imageagent.GenerationReservationReceipt{UnitVersion: aicapability.AIPointMicrounitV1, IntentID: f.IntentID, Fingerprint: f.Fingerprint, OrganizationID: f.Intent.Identity.TenantID, MemberID: f.Intent.MemberID, OperationID: "image-reserve:" + f.IntentID, ReservationID: "synthetic-reservation", ResourceType: "ai_point", Points: f.Intent.Points, PriceVersion: f.Intent.PriceVersion, LimitVersion: f.Intent.LimitVersion, MonthStart: f.Intent.MonthStart}
	fact, err = fact.BindReservation(receipt)
	require.NoError(t, err)
	fact, _, err = fact.BeginDispatch()
	require.NoError(t, err)
	// NewArtworkProvenance accepts the actual original fact; no fake success.
	var pixels bytes.Buffer
	require.NoError(t, png.Encode(&pixels, image.NewRGBA(image.Rect(0, 0, 1024, 1024))))
	objects := &flareMemoryObjects{objects: map[string]objectstore.ImmutableObjectPut{}}
	store, err := objectstore.NewDurableArtifactStore(objects, objectstore.DurableArtifactStoreConfig{MaxArtifactBytes: 32 << 20, OperationTimeout: time.Second})
	require.NoError(t, err)
	factory := generationProviderFactory{facts: flareReadFact{fact: fact}, artifacts: store}
	usage := openai.FlareImageUsage{}
	require.NoError(t, factory.observeFlare(context.Background(), in, openai.FlareImageObservation{Usage: usage, ResponseID: "artwork-result", ResultDigest: strings.Repeat("a", 64), ImageBytes: pixels.Bytes()}, func(_ context.Context, p imageagent.GenerationSuccess) error {
		require.Equal(t, "artifact_bundle", p.ResultLocatorKind)
		return nil
	}))
	recovered, err := store.RecoverSlotArtifacts(context.Background(), fact.Intent.Identity, imageagent.StagingManifest{})
	require.NoError(t, err)
	require.NotNil(t, recovered.Manifest.Assets[0].Artwork)
	require.Empty(t, recovered.Manifest.Assets[0].SourceAssetID)
	creator := artworkCreator{model: aicapability.FlareImageModel, quality: "auto", generate: func(context.Context, *ai.ImageGenerateRequest) (*ai.ImageResponse, error) {
		return &ai.ImageResponse{Data: []ai.ImageData{{B64JSON: base64.StdEncoding.EncodeToString(pixels.Bytes())}}, UpstreamJobID: "artwork-result"}, nil
	}, fetch: func(context.Context, string) ([]byte, error) {
		t.Fatal("base64 must not cause URL fetching")
		return nil, nil
	}}
	asset, err := creator.CreateArtwork(context.Background(), in.Artwork.Prompt)
	require.NoError(t, err)
	require.Equal(t, pixels.Bytes(), asset.Bytes)
}
