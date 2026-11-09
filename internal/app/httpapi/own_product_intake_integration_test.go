package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	channel "task-processor/internal/aicapability/channelcredential"
	d "task-processor/internal/aicapability/routeconfig"
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/authz"
	qualified "task-processor/internal/integration/aicapability/routeconfig"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	routestore "task-processor/internal/integration/persistence/routeconfig"
	s3 "task-processor/internal/integration/s3"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/product/collection"
	collectionhttp "task-processor/internal/product/collection/httpapi"
	"task-processor/internal/product/sourcing"
)

const ownIntakeFixtureNotes = "黑 M，SKU BLK-M，CNY 29，库存明确为 0\n黑 L，SKU BLK-L，CNY 29，库存未填写\n白 M，SKU WHT-M，CNY 29，图片2"
const ownIntakeFixtureCandidate = `{"title":"隔离验收 三规格上衣","description":"三种在售规格。","rows":[{"attributes":{"颜色":"黑","尺码":"M"},"evidence":"黑 M，SKU BLK-M，CNY 29，库存明确为 0","sku":"BLK-M","priceSuggestion":{"amount":29,"currency":"CNY","evidence":"CNY 29"},"stockSuggestion":{"quantity":0,"evidence":"库存明确为 0"}},{"attributes":{"颜色":"黑","尺码":"L"},"evidence":"黑 L，SKU BLK-L，CNY 29，库存未填写","sku":"BLK-L","priceSuggestion":{"amount":29,"currency":"CNY","evidence":"CNY 29"}},{"attributes":{"颜色":"白","尺码":"M"},"evidence":"白 M，SKU WHT-M，CNY 29，图片2","sku":"WHT-M","priceSuggestion":{"amount":29,"currency":"CNY","evidence":"CNY 29"},"image":2}]}`

func TestOwnIntakeHTTPAndIsolatedBrowser(t *testing.T) {
	testProductAgentOwners(t, "own intake")
}

func configureOwnIntakeFixture(t *testing.T, settings *ProductAgentDependencies, calls *atomic.Int32) func(int64) error {
	t.Helper()
	ctx, now := context.Background(), time.Now().UTC()
	db := settings.RunDB
	require.NoError(t, routestore.InstallSchema(ctx, db))
	repo, err := routestore.New(db)
	require.NoError(t, err)
	conn, err := newHTTPChannelFixture(db).owner.Publish(ctx, channel.Command{
		OrganizationID: "B", ActorID: "synthetic-operator", Key: uuid.NewString(),
		ConnectionID: "openai-mini", Provider: "openai", Channel: "openai-direct", Region: "global",
		Account: "fixture", BaseURL: "https://api.openai.com/v1", Secret: "fixture-key", Enabled: true,
	})
	require.NoError(t, err)
	prices := &d.PlatformPricesService{Repository: repo}
	price := d.PublishedModelPrice{
		ID: "fixture-mini", Provider: "openai", Model: aicapability.SingleImageMiniModel,
		Channel: conn.Channel, Region: conn.Region, ServiceType: "standard-sync", Currency: "USD",
		UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 750_000, CachedInputText: 75_000, OutputText: 4_500_000},
		SourceURL:  "https://developers.openai.com/api/docs/models/gpt-5.4-mini",
		VerifiedAt: now, EffectiveFrom: now.Add(-time.Minute), EffectiveUntil: now.Add(time.Hour),
	}
	revision := d.NoRevision
	publishFX := func(rate int64) error {
		fx := d.PublishedFXRate{Currency: "USD", CNYMicrosPerCurrencyUnit: rate,
			SourceURL: "https://www.chinamoney.com.cn/chinese/bkccpr/", RateDate: now.Format(time.DateOnly),
			VerifiedAt: now, EffectiveFrom: price.EffectiveFrom, EffectiveUntil: price.EffectiveUntil}
		next, err := prices.Publish(ctx, "synthetic-platform", uuid.NewString(), d.PlatformPricesInput{
			ExpectedRevision: revision, Prices: []d.PublishedModelPrice{price}, FX: []d.PublishedFXRate{fx},
		})
		if err != nil {
			return err
		}
		revision = next.ID
		return nil
	}
	require.NoError(t, publishFX(8_000_000))
	settings.TextLogger = logrus.NewEntry(logrus.New())
	p := settings.ImageAnalysisPolicies["B"]
	p.Profile.PromptVersion, p.Profile.OutputSchemaVersion = agentconfig.OwnIntakePromptVersion, agentconfig.OwnIntakeActionVersion
	settings.ImageAnalysisPolicies = nil
	p.Profile.ClientName, p.Profile.ConnectionID = conn.ConnectionID, conn.ConnectionID
	p.AdmittedCredentialVersion = conn.Reference
	p.Profile.MaximumPromptTokens, p.Profile.MaximumCompletionTokens = 24_000, 4_000
	p.Profile.MaximumInputBytes = 1 << 20
	p.Profile.InputMicrosPerMillion, p.Profile.OutputMicrosPerMillion = 6_000_000, 36_000_000
	settings.TextPolicies = nil
	settings.Limits.CostMicros = 300_000
	ref := d.ModelBindingReference{ID: conn.ConnectionID, Version: "fixture-mini-v1"}
	catalog, err := qualified.NewCatalog([]qualified.Definition{{
		Scope:        d.Scope{OrganizationID: "B", Capability: string(aicapability.OperationOwnProductIntake)},
		Manufacturer: conn.Provider, Channel: conn.Channel, Region: conn.Region, ServiceType: price.ServiceType,
		PriceID: price.ID, ReferenceVersion: ref.Version, BoundEvidenceVersion: "fixture-mini-bound",
		ConnectionID: conn.ConnectionID, ConnectionProvider: conn.Provider, ConnectionEndpoint: conn.BaseURL,
		Text: &p, MaximumCostMicros: settings.Limits.CostMicros, MaximumMicropoints: 30_000_000,
	}})
	require.NoError(t, err)
	service := &d.SharedModelService{Repository: repo, Catalog: &qualified.SharedCatalog{Catalog: catalog, Credentials: repo, Prices: prices}}
	_, err = service.Publish(ctx, "B", "admin", uuid.NewString(), d.SharedModelInput{
		ExpectedRevision: d.NoRevision,
		Selection: d.SharedModelSelection{SchemaVersion: 1, UnitVersion: aicapability.AIPointMicrounitV1, DefaultChannel: conn.Channel,
			Text:  d.ModelBindingReference{ID: "openai-mini", Version: "approved-default-v1"},
			Image: d.ModelBindingReference{ID: "openai-flare", Version: "approved-default-v1"}, Multimodal: ref},
	})
	require.NoError(t, err)
	settings.CurrentTextPolicies = qualified.SharedTextReader{Service: service}
	settings.TextTransport = analysisFixtureTransport(func(r *http.Request) (*http.Response, error) {
		dispatch := calls.Add(1)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		wantImages := 1
		if dispatch > 1 {
			wantImages = 0
		}
		var wire struct {
			Model               string `json:"model"`
			MaxCompletionTokens int    `json:"max_completion_tokens"`
		}
		if err := json.Unmarshal(body, &wire); err != nil {
			return nil, err
		}
		if r.URL.String() != "https://api.openai.com/v1/chat/completions" || wire.Model != aicapability.SingleImageMiniModel || wire.MaxCompletionTokens != 4_000 ||
			strings.Count(string(body), "data:image/png;base64,") != wantImages || !strings.Contains(string(body), "WHT-M") {
			return nil, fmt.Errorf("expected %d PNG parts and original SKU facts on the governed model wire", wantImages)
		}
		candidate := ownIntakeFixtureCandidate
		if os.Getenv("OWN_INTAKE_BROWSER") == "1" && os.Getenv("OWN_INTAKE_BROWSER_PRESAVE") == "1" && dispatch == 2 {
			candidate = strings.Replace(candidate, `"sku":"BLK-L","priceSuggestion"`, `"sku":"BLK-L","stockSuggestion":{"quantity":0,"evidence":"库存未填写"},"priceSuggestion"`, 1)
		}
		return ownIntakeControlledResponse(candidate), nil
	})
	return publishFX
}

func ownIntakeControlledResponse(content string) *http.Response {
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"id":"intake-fixture","object":"chat.completion","created":1,"model":%q,"choices":[{"index":0,"message":{"role":"assistant","content":%q},"finish_reason":"stop"}],"usage":{"prompt_tokens":30,"completion_tokens":5,"total_tokens":35,"prompt_tokens_details":{"cached_tokens":10}}}`, aicapability.SingleImageMiniModel, content)))}
}

type ownIntakeFixtureMedia struct {
	mu      sync.Mutex
	objects map[string]s3.ImmutableObjectPut
}

func (m *ownIntakeFixtureMedia) PublicURL(key string) string {
	return "https://intake-media.example.com/" + key
}

func (m *ownIntakeFixtureMedia) PutImmutable(_ context.Context, obj s3.ImmutableObjectPut) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if old, ok := m.objects[obj.Key]; ok && !bytes.Equal(old.Data, obj.Data) {
		return collection.ErrConflict
	}
	obj.Data = bytes.Clone(obj.Data)
	m.objects[obj.Key] = obj
	return nil
}

func (m *ownIntakeFixtureMedia) ReadObject(_ context.Context, key string, max int64) ([]byte, s3.ObjectInspection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	obj, ok := m.objects[key]
	if !ok {
		return nil, s3.ObjectInspection{}, nil
	}
	if int64(len(obj.Data)) > max {
		return nil, s3.ObjectInspection{}, collection.ErrInvalid
	}
	return bytes.Clone(obj.Data), s3.ObjectInspection{Exists: true, ContentLength: obj.SizeBytes, ContentType: obj.ContentType}, nil
}

func testOwnIntakeThroughOwners(t *testing.T, f *acquisitionHTTPFixture, parent *productAgentApplication, deps routeAuthDependencies, permissions *authz.ListingKitAuthorizer, calls *atomic.Int32, publishFX func(int64) error) {
	t.Helper()
	ctx := context.Background()
	live := &productReviewLiveOrganizationAccess{resolver: deps.organizationResolver, now: time.Now}
	authority, err := collection.NewContextAuthorizer(live, permissions)
	require.NoError(t, err)
	require.NoError(t, collectionstore.InstallSchema(f.owner))
	repo, err := collectionstore.NewRepository(ctx, f.owner, func(tx *gorm.DB) (collectionstore.OwnPublisher, error) {
		return productsourcing.NewOwnProductWriter(tx, authority)
	})
	require.NoError(t, err)
	sources, err := productsourcing.NewPublishedAcquisitionReader(ctx, f.owner, live, permissions)
	require.NoError(t, err)
	service, err := collection.NewService(repo, authority, sources)
	require.NoError(t, err)
	snapshots, err := catalogstore.NewBoundedSnapshotReader(f.owner, sourcing.MaxEncodedSnapshotBytes)
	require.NoError(t, err)
	storage := &ownIntakeFixtureMedia{objects: map[string]s3.ImmutableObjectPut{}}
	media := productsourcing.SourceMedia{Storage: storage, Authorization: authority}
	service.WithSnapshots(snapshots).WithMedia(media)
	app, err := buildOwnIntakeApplication(parent, media.ReadOwnIntake)
	require.NoError(t, err)
	require.NotNil(t, app)
	parent.ownIntake = app
	_, err = parent.configuration.Execute(ctx, agentconfig.Command{Scope: agent.Scope{OrganizationID: "B", ActorID: "admin"}, AgentID: agentconfig.OwnIntakeAgentID, Operation: "enable", Key: uuid.NewString(), Absent: true})
	require.NoError(t, err)
	routes := append(ownIntakeRoutes(app), collectionhttp.Routes(service, (productReviewCapabilityBinder{now: time.Now}).Bind)...)
	configuration, err := buildAgentConfigurationModule(ctx, f.owner, deps.organizationResolver, permissions, nil, parent, nil, nil, nil, nil)
	require.NoError(t, err)
	registry := kernelmodule.NewRegistry()
	require.NoError(t, configuration.Register(registry))
	routes = append(routes, registry.Routes()...)
	router := buildCurrentApplicationHTTPServer(routes, deps).Handler
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if key, ok := strings.CutPrefix(r.URL.Path, "/fixture-media/"); ok {
			data, meta, err := storage.ReadObject(r.Context(), key, collection.MaxMediaBytes)
			if err != nil || !meta.Exists {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", meta.ContentType)
			_, _ = w.Write(data)
			return
		}
		router.ServeHTTP(w, r)
	}))
	defer server.Close()

	if os.Getenv("OWN_INTAKE_BROWSER") == "1" {
		runOwnIntakeBrowser(t, server.URL)
		wantCalls, wantProducts := 1, 1
		if os.Getenv("OWN_INTAKE_BROWSER_PRESAVE") == "1" {
			wantCalls, wantProducts = 2, 0
		}
		require.EqualValues(t, wantCalls, calls.Load(), "only explicitly confirmed controlled responses are dispatched")
		var count int64
		require.NoError(t, f.owner.Table("product_collection_items").Count(&count).Error)
		require.EqualValues(t, wantProducts, count, "pre-save mode must leave the own-product library empty")
		return
	}
	call := func(method, path, key string, body any, status int) []byte {
		t.Helper()
		raw := ""
		if body != nil {
			data, err := json.Marshal(body)
			require.NoError(t, err)
			raw = string(data)
		}
		got, b, err := acquisitionHTTPRequest(server, method, path, "admin", "B", key, raw)
		require.NoError(t, err)
		require.Equal(t, status, got, string(b))
		return b
	}
	upload := func(shade color.RGBA) string {
		t.Helper()
		pixels := image.NewRGBA(image.Rect(0, 0, 2, 2))
		pixels.SetRGBA(0, 0, shade)
		var raw bytes.Buffer
		require.NoError(t, png.Encode(&raw, pixels))
		req, err := http.NewRequest(http.MethodPost, server.URL+collectionhttp.BasePath+"/media", bytes.NewReader(raw.Bytes()))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("X-Content-SHA256", intakeDigest(raw.Bytes()))
		req.Header.Set("Authorization", "Bearer admin")
		req.Header.Set("X-Requested-Organization-ID", "B")
		response, err := server.Client().Do(req)
		require.NoError(t, err)
		defer response.Body.Close()
		var image collection.MediaImage
		result, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode, string(result))
		require.NoError(t, json.Unmarshal(result, &image))
		return image.URL
	}
	black, white := upload(color.RGBA{A: 255}), upload(color.RGBA{R: 255, G: 255, B: 255, A: 255})
	body := ownIntakeBody{Notes: ownIntakeFixtureNotes, Gallery: []string{intakeDigest([]byte(black)), intakeDigest([]byte(white))}, ImageNumber: 2, ImageURL: white}
	key := uuid.NewString()
	var prepared ownIntakePrepared
	require.NoError(t, json.Unmarshal(call("POST", ownIntakeBase+"/prepare", key, body, 200), &prepared))
	call("GET", ownIntakeBase+"/"+key, "", nil, 404)
	var recovered ownIntakePrepared
	require.NoError(t, json.Unmarshal(call("POST", ownIntakeBase+"/prepare", key, body, 200), &recovered))
	require.Equal(t, prepared, recovered, "missing Run recovery uses the same prepared snapshot and quote")
	require.Zero(t, calls.Load())
	require.Equal(t, "CNY", prepared.Quote.Currency)
	require.LessOrEqual(t, prepared.Quote.MaximumTokens, int64(28_000))
	require.Equal(t, int64(6*(prepared.Quote.MaximumTokens-4_000)+36*4_000), prepared.Quote.MaximumCostMicros)
	require.LessOrEqual(t, prepared.Quote.MaximumCostMicros, int64(288_000))
	confirmation := ownIntakeConfirm{ownIntakeBody: body, Ref: prepared.Ref, QuoteDigest: prepared.Quote.InputHash}
	confirmation.Notes += "\n变更"
	call("POST", ownIntakeBase+"/"+key+"/confirm", key, confirmation, 409)
	confirmation.Notes = body.Notes
	var result ownIntakeResult
	require.NoError(t, json.Unmarshal(call("POST", ownIntakeBase+"/"+key+"/confirm", key, confirmation, 200), &result))
	t.Logf("isolated intake phase=%s stop=%s model_dispatches=%d", result.Phase, result.StopReason, calls.Load())
	require.Equal(t, agent.HumanReviewRequired, result.Phase, result.StopReason)
	require.NotNil(t, result.Candidate)
	require.Len(t, result.Candidate.Rows, 3)
	require.Equal(t, new(int64(0)), result.Candidate.Rows[0].StockSuggestion.Quantity)
	require.Nil(t, result.Candidate.Rows[1].StockSuggestion)
	require.Equal(t, 2, result.Candidate.Rows[2].Image)
	original := call("GET", ownIntakeBase+"/"+key, "", nil, 200)
	require.JSONEq(t, string(original), string(call("POST", ownIntakeBase+"/"+key+"/confirm", key, confirmation, 200)))
	require.EqualValues(t, 1, calls.Load())
	other, _, err := acquisitionHTTPRequest(server, "GET", ownIntakeBase+"/"+key, "admin", "C", "", "")
	require.NoError(t, err)
	require.NotEqual(t, 200, other, "another organization cannot read the original result")
	var count int64
	require.NoError(t, f.owner.Table("product_collection_items").Count(&count).Error)
	require.Zero(t, count, "generation alone never saves a product")
	textBody := body
	textBody.ImageNumber, textBody.ImageURL = 0, ""
	textKey := uuid.NewString()
	require.NoError(t, json.Unmarshal(call("POST", ownIntakeBase+"/prepare", textKey, textBody, 200), &prepared))
	textResult := call("POST", ownIntakeBase+"/"+textKey+"/confirm", textKey, ownIntakeConfirm{ownIntakeBody: textBody, Ref: prepared.Ref, QuoteDigest: prepared.Quote.InputHash}, 200)
	var textOnly ownIntakeResult
	require.NoError(t, json.Unmarshal(textResult, &textOnly))
	require.NotNil(t, textOnly.Candidate, textOnly.StopReason)
	require.EqualValues(t, 2, calls.Load())

	// Change the published FX after the actual point reservation, while the
	// original request is about to enter the final guarded transport handoff.
	staleKey := uuid.NewString()
	require.NoError(t, json.Unmarshal(call("POST", ownIntakeBase+"/prepare", staleKey, body, 200), &prepared))
	stale := ownIntakeConfirm{ownIntakeBody: body, Ref: prepared.Ref, QuoteDigest: prepared.Quote.InputHash}
	ledger := app.executor.Ledger
	type fxChange struct {
		quantity int64
		err      error
	}
	changed := make(chan fxChange, 1)
	app.executor.Ledger = agentReserveHook{ProductAgentInvocationLedger: parent.config.Ledger, afterReserve: func() {
		var change fxChange
		change.err = f.owner.Table("saas_organization_resource_reservations").
			Where("organization_id = ? AND state = ?", "B", "reserved").
			Select("quantity").Take(&change.quantity).Error
		if change.err == nil {
			change.err = publishFX(8_100_000)
		}
		changed <- change
	}}
	var blocked ownIntakeResult
	require.NoError(t, json.Unmarshal(call("POST", ownIntakeBase+"/"+staleKey+"/confirm", staleKey, stale, 200), &blocked))
	select {
	case change := <-changed:
		require.NoError(t, change.err)
		require.Equal(t, prepared.Quote.MaximumCostMicros*100, change.quantity)
		require.LessOrEqual(t, change.quantity, int64(30_000_000))
	default:
		t.Fatal("expected an actual point reservation before changing FX")
	}
	require.Nil(t, blocked.Candidate)
	require.Equal(t, agent.StopDependency, blocked.StopReason)
	require.NoError(t, f.owner.Table("saas_organization_resource_reservations").
		Where("organization_id = ? AND state = ?", "B", "reserved").Count(&count).Error)
	require.Zero(t, count, "the no-send result releases the reservation")
	require.EqualValues(t, 2, calls.Load(), "no stale quote reaches the controlled provider")
	app.executor.Ledger = ledger
	call("POST", ownIntakeBase+"/prepare", uuid.NewString(), body, 503)
	require.NoError(t, publishFX(8_000_000))

	// Even an incorrect zero inferred from unknown stock remains only a suggestion.
	// The UI must require independent confirmation before applying it.
	invalidCandidate := strings.Replace(ownIntakeFixtureCandidate, `"sku":"BLK-L","priceSuggestion"`, `"sku":"BLK-L","stockSuggestion":{"quantity":0,"evidence":"库存未填写"},"priceSuggestion"`, 1)
	app.executor.BaseTransport = analysisFixtureTransport(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return ownIntakeControlledResponse(invalidCandidate), nil
	})
	invalidKey := uuid.NewString()
	require.NoError(t, json.Unmarshal(call("POST", ownIntakeBase+"/prepare", invalidKey, textBody, 200), &prepared))
	invalidConfirmation := ownIntakeConfirm{ownIntakeBody: textBody, Ref: prepared.Ref, QuoteDigest: prepared.Quote.InputHash}
	invalidResult := call("POST", ownIntakeBase+"/"+invalidKey+"/confirm", invalidKey, invalidConfirmation, 200)
	var rejected ownIntakeResult
	require.NoError(t, json.Unmarshal(invalidResult, &rejected))
	require.Equal(t, agent.HumanReviewRequired, rejected.Phase)
	require.NotNil(t, rejected.Candidate)
	require.Equal(t, new(int64(0)), rejected.Candidate.Rows[1].StockSuggestion.Quantity)
	require.Equal(t, "库存未填写", rejected.Candidate.Rows[1].StockSuggestion.Evidence)
	require.Equal(t, "BLK-L", rejected.Candidate.Rows[1].SKU)
	require.Nil(t, rejected.ValidationError)
	require.JSONEq(t, string(invalidResult), string(call("GET", ownIntakeBase+"/"+invalidKey, "", nil, 200)))
	require.JSONEq(t, string(invalidResult), string(call("POST", ownIntakeBase+"/"+invalidKey+"/confirm", invalidKey, invalidConfirmation, 200)))
	require.EqualValues(t, 3, calls.Load(), "pending stock suggestion readback must not call again")
	require.NoError(t, f.owner.Table("product_collection_items").Count(&count).Error)
	require.Zero(t, count, "all offline verification stops before product save")

	// Ambiguous prices remain suggestions for explicit human confirmation.
	// Other valid fields survive, and read/re-confirm never dispatches again.
	for _, source := range []string{"CNY 29-39", "CNY 29,999.00", "不是 CNY 29，价格待定", "CNY 29 或 CNY 39，尚未确认"} {
		t.Run("price "+source, func(t *testing.T) {
			notes := strings.ReplaceAll(ownIntakeFixtureNotes, "CNY 29", source)
			candidate := strings.ReplaceAll(ownIntakeFixtureCandidate, "CNY 29", source)
			app.executor.BaseTransport = analysisFixtureTransport(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return ownIntakeControlledResponse(candidate), nil
			})
			request := textBody
			request.Notes = notes
			key := uuid.NewString()
			require.NoError(t, json.Unmarshal(call("POST", ownIntakeBase+"/prepare", key, request, 200), &prepared))
			confirm := ownIntakeConfirm{ownIntakeBody: request, Ref: prepared.Ref, QuoteDigest: prepared.Quote.InputHash}
			before := calls.Load()
			response := call("POST", ownIntakeBase+"/"+key+"/confirm", key, confirm, 200)
			var result ownIntakeResult
			require.NoError(t, json.Unmarshal(response, &result))
			require.Equal(t, agent.HumanReviewRequired, result.Phase)
			require.NotNil(t, result.Candidate)
			require.Nil(t, result.ValidationError)
			require.Equal(t, "隔离验收 三规格上衣", result.Candidate.Title)
			require.Equal(t, source, result.Candidate.Rows[0].PriceSuggestion.Evidence)
			require.Equal(t, new(29.0), result.Candidate.Rows[0].PriceSuggestion.Amount)
			require.NotContains(t, string(response), `"price":`)
			require.NotContains(t, string(response), `"confirmed"`)
			require.Equal(t, new(int64(0)), result.Candidate.Rows[0].StockSuggestion.Quantity)
			require.JSONEq(t, string(response), string(call("GET", ownIntakeBase+"/"+key, "", nil, 200)))
			require.JSONEq(t, string(response), string(call("POST", ownIntakeBase+"/"+key+"/confirm", key, confirm, 200)))
			require.Equal(t, before+1, calls.Load())
			require.NoError(t, f.owner.Table("product_collection_items").Count(&count).Error)
			require.Zero(t, count, "pending price suggestions never save a product")
		})
	}

	beforeUnknown := calls.Load()
	app.executor.BaseTransport = analysisFixtureTransport(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, errors.New("isolated response lost after provider handoff")
	})
	unknownKey := uuid.NewString()
	require.NoError(t, json.Unmarshal(call("POST", ownIntakeBase+"/prepare", unknownKey, body, 200), &prepared))
	unknown := ownIntakeConfirm{ownIntakeBody: body, Ref: prepared.Ref, QuoteDigest: prepared.Quote.InputHash}
	unknownResult := call("POST", ownIntakeBase+"/"+unknownKey+"/confirm", unknownKey, unknown, 200)
	var uncertain ownIntakeResult
	require.NoError(t, json.Unmarshal(unknownResult, &uncertain))
	require.Equal(t, agent.StopModelUnknown, uncertain.StopReason)
	require.Nil(t, uncertain.Candidate)
	require.JSONEq(t, string(unknownResult), string(call("GET", ownIntakeBase+"/"+unknownKey, "", nil, 200)))
	require.JSONEq(t, string(unknownResult), string(call("POST", ownIntakeBase+"/"+unknownKey+"/confirm", unknownKey, unknown, 200)))
	require.Equal(t, beforeUnknown+1, calls.Load(), "an uncertain provider result must never be redispatched")

}

func runOwnIntakeBrowser(t *testing.T, backend string) {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())
	origin := fmt.Sprintf("http://127.0.0.1:%d", port)
	root, err := filepath.Abs("../../../web/listingkit-ui")
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "node", "node_modules/@playwright/test/cli.js", "test", "--config=playwright.own-intake.config.ts")
	command.Dir = root
	command.Env = append(os.Environ(),
		"OWN_INTAKE_BROWSER_ORIGIN="+origin, "OWN_INTAKE_BROWSER_BACKEND="+backend, "LISTINGKIT_SERVICE_API_BASE="+backend+"/api/v1",
		"AUTH_URL="+origin, "LISTINGKIT_PUBLIC_BASE_URL="+origin, "AUTH_SECRET=codex-own-intake-isolated-browser-only",
		"ZITADEL_ISSUER_URL=https://own-intake-ui.invalid", "ZITADEL_CLIENT_ID=own-intake-fixture", "NEXT_TELEMETRY_DISABLED=1",
		"LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED=true", "LISTINGKIT_AGENT_CONFIGURATION_ENABLED=true")
	output, err := command.CombinedOutput()
	t.Logf("%s", output)
	require.NoError(t, err)
}
