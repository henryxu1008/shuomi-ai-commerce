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
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/authz"
	governed "task-processor/internal/integration/aicapability/einomodel"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	s3 "task-processor/internal/integration/s3"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/product/collection"
	collectionhttp "task-processor/internal/product/collection/httpapi"
	"task-processor/internal/product/sourcing"
)

const ownIntakeFixtureNotes = "黑 M，SKU BLK-M，CNY 29，库存 0\n黑 L，SKU BLK-L，CNY 29\n白 M，SKU WHT-M，CNY 29，图片2"
const ownIntakeFixtureCandidate = `{"title":"隔离验收 三规格上衣","description":"三种在售规格。","rows":[{"attributes":{"颜色":"黑","尺码":"M"},"evidence":"黑 M，SKU BLK-M，CNY 29，库存 0","sku":"BLK-M","currency":"CNY","price":29,"stock":0},{"attributes":{"颜色":"黑","尺码":"L"},"evidence":"黑 L，SKU BLK-L，CNY 29","sku":"BLK-L","currency":"CNY","price":29},{"attributes":{"颜色":"白","尺码":"M"},"evidence":"白 M，SKU WHT-M，CNY 29，图片2","sku":"WHT-M","currency":"CNY","price":29,"image":2}]}`

func TestOwnIntakeHTTPAndIsolatedBrowser(t *testing.T) {
	testProductAgentOwners(t, "own intake")
}

func configureOwnIntakeFixture(settings *ProductAgentDependencies, calls *atomic.Int32) {
	settings.TextLogger = logrus.NewEntry(logrus.New())
	p := settings.ImageAnalysisPolicies["B"]
	p.Profile.PromptVersion, p.Profile.OutputSchemaVersion = agentconfig.OwnIntakePromptVersion, agentconfig.OwnIntakeActionVersion
	settings.ImageAnalysisPolicies = nil
	settings.CurrentTextPolicies = syntheticCurrentTextPolicies{settings: settings, ownIntake: map[string]governed.RoutePolicy{"B": p}}
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
		if strings.Count(string(body), "data:image/png;base64,") != wantImages || !strings.Contains(string(body), "WHT-M") {
			return nil, fmt.Errorf("expected %d PNG parts and original SKU facts on the governed model wire", wantImages)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"id":"intake-fixture","object":"chat.completion","created":1,"model":%q,"choices":[{"index":0,"message":{"role":"assistant","content":%q},"finish_reason":"stop"}],"usage":{"prompt_tokens":30,"completion_tokens":5,"total_tokens":35,"prompt_tokens_details":{"cached_tokens":10}}}`, aicapability.SingleImageMiniModel, ownIntakeFixtureCandidate)))}, nil
	})
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

func testOwnIntakeThroughOwners(t *testing.T, f *acquisitionHTTPFixture, parent *productAgentApplication, deps routeAuthDependencies, permissions *authz.ListingKitAuthorizer, calls *atomic.Int32) {
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
		require.EqualValues(t, 1, calls.Load(), "browser confirmation dispatches once")
		var count int64
		require.NoError(t, f.owner.Table("product_collection_items").Count(&count).Error)
		require.EqualValues(t, 1, count, "browser saved exactly one product")
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
	require.Equal(t, new(0), result.Candidate.Rows[0].Stock)
	require.Nil(t, result.Candidate.Rows[1].Stock)
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
	require.EqualValues(t, 3, calls.Load(), "an uncertain provider result must never be redispatched")
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
