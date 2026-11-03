package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"task-processor/internal/authz"
	"task-processor/internal/imageagent"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/integration/openai"
	assetstore "task-processor/internal/integration/persistence/product/asset"
	"task-processor/internal/product/asset"
	"testing"
)

type analysisFixtureTransport func(*http.Request) (*http.Response, error)

type analysisReadbackFailureLedger struct {
	ProductAgentInvocationLedger
	reads atomic.Int32
}

func (l *analysisReadbackFailureLedger) ReadModelInvocation(ctx context.Context, org, id string) (aicapability.InvocationRecord, error) {
	// First read is the actual domain validator; the second is HTTP readback
	// after the Run/provider/settlement has already completed.
	if l.reads.Add(1) == 2 {
		return aicapability.InvocationRecord{}, agent.ErrUnavailable
	}
	return l.ProductAgentInvocationLedger.ReadModelInvocation(ctx, org, id)
}

func (f analysisFixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func configureImageAnalysisFixture(s *ProductAgentDependencies, row openai.AIClientCredential, calls *atomic.Int32) {
	p := s.TextPolicies["B"]
	p.AdmittedCredentialVersion = governed.CredentialVersion(row)
	p.AdmittedEndpointIdentityDigest = governed.EndpointIdentityDigest(row.BaseURL)
	p.Profile.ProviderID = "openai"
	p.Profile.ModelID = aicapability.SingleImageMiniModel
	p.Profile.AdapterPolicyVersion = aicapability.SingleImageMiniAdapterPolicy
	p.Profile.UsageMappingVersion = "openai-text-v1"
	p.Profile.PromptVersion = agentconfig.ImageAnalysisPromptVersion
	p.Profile.OutputSchemaVersion = agentconfig.ImageAnalysisResultVersion
	p.Profile.MaximumPromptTokens = 16000
	p.Profile.MaximumCompletionTokens = 1024
	p.Profile.OfficialTariff.Model = p.Profile.ModelID
	p.Profile.OfficialTariff.Channel = "openai-direct"
	p.Profile.OfficialTariff.Rates.CachedInputText = 1000
	s.ImageAnalysisPolicies = map[string]governed.RoutePolicy{"B": p}
	s.TextTransport = analysisFixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "data:image/png;base64,") {
			return nil, fmt.Errorf("expected original typed single-image wire")
		}
		opinion := `{"observations":[],"suggestions":[],"unknowns":["Synthetic fixture only; visual quality not evaluated."]}`
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"id":"fixture","object":"chat.completion","created":1,"model":%q,"choices":[{"index":0,"message":{"role":"assistant","content":%q},"finish_reason":"stop"}],"usage":{"prompt_tokens":30,"completion_tokens":5,"total_tokens":35,"prompt_tokens_details":{"cached_tokens":10}}}`, aicapability.SingleImageMiniModel, opinion)))}, nil
	})
}

func TestImageAnalysisHTTPConfirmationAndOriginalReadonlyHistory(t *testing.T) {
	testProductAgentOwners(t, "image analysis")
}

func testImageAnalysisHTTPThroughOwners(t *testing.T, f *acquisitionHTTPFixture, parent *productAgentApplication, deps routeAuthDependencies, auth *authz.ListingKitAuthorizer, operation string, calls *atomic.Int32) {
	t.Helper()
	a, err := buildImageAnalysisApplication(parent, nil, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, a)
	var pngBytes bytes.Buffer
	require.NoError(t, png.Encode(&pngBytes, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	reads := 0
	a.materials.ReadPublic = func(context.Context, imageagent.AuthorizedAsset, int64) ([]byte, error) {
		reads++
		return pngBytes.Bytes(), nil
	}
	// Reuse the existing API route/auth/PG fixture; no retained instance or provider.
	_, err = parent.configuration.Execute(context.Background(), agentconfig.Command{Scope: agent.Scope{OrganizationID: "B", ActorID: "admin"}, AgentID: agentconfig.ImageAgentID, Operation: "enable", Key: uuid.NewString(), Absent: true})
	require.NoError(t, err)
	app := buildCurrentApplicationHTTPServer(imageAnalysisRoutes(a), deps)
	server := httptest.NewServer(app.Handler)
	defer server.Close()
	base := productAcquisitionBase + "/" + operation + "/images/analysis"
	call := func(method, path, key string, body any, status int) []byte {
		t.Helper()
		raw := ""
		if body != nil {
			b, e := json.Marshal(body)
			require.NoError(t, e)
			raw = string(b)
		}
		code, b, e := acquisitionHTTPRequest(server, method, path, "admin", "B", key, raw)
		require.NoError(t, e)
		require.Equal(t, status, code, string(b))
		return b
	}
	var source imageAnalysisSourceDTO
	require.NoError(t, json.Unmarshal(call("GET", base, "", nil, 200), &source))
	require.True(t, source.Ready, source.Reason)
	require.NotEmpty(t, source.Choices)
	key := uuid.NewString()
	intent := imageAnalysisPrepareBody{Kind: "source", ImageID: source.Choices[0].ID, CatalogVersion: source.CatalogVersion, PublicationID: source.PublicationID, ExpectedModeRevision: "0"}
	var prepared imageAnalysisResultDTO
	require.NoError(t, json.Unmarshal(call("POST", base+"/prepare", key, intent, 200), &prepared))
	require.Equal(t, "prepared", prepared.Status)
	require.Zero(t, calls.Load())
	before := reads
	call("GET", base+"/by-key/"+key, "", nil, 200)
	require.Equal(t, before, reads, "pre-claim receipt read does not download images")
	consent := imageAnalysisConfirmBody{PreparationID: prepared.PreparationID, ConfigurationDigest: prepared.ConfigurationDigest, QuoteDigest: prepared.QuoteDigest, SelectionDigest: prepared.SelectionDigest, ContentSHA256: prepared.ContentSHA256, ModelID: prepared.ModelID, MaximumMicropoints: prepared.MaximumMicropoints, Confirmed: false}
	call("POST", base+"/by-key/"+key+"/confirm", "", consent, 400)
	require.Zero(t, calls.Load())
	consent.Confirmed = true
	parent.config.Ledger = &analysisReadbackFailureLedger{ProductAgentInvocationLedger: parent.config.Ledger}
	var uncertain struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(call("POST", base+"/by-key/"+key+"/confirm", "", consent, 503), &uncertain))
	require.Equal(t, "OUTCOME_UNKNOWN", uncertain.Code, "submitted execution cannot become an ordinary rejection when its readback fails")
	require.EqualValues(t, 1, calls.Load())
	var result imageAnalysisResultDTO
	require.NoError(t, json.Unmarshal(call("GET", base+"/by-key/"+key, "", nil, 200), &result))
	require.Equal(t, "completed", result.Status)
	require.NotNil(t, result.Opinion)
	require.Equal(t, "31", result.ActualMicropoints)
	require.EqualValues(t, 1, calls.Load())
	before = reads
	call("POST", base+"/by-key/"+key+"/confirm", "", consent, 200)
	require.Equal(t, before, reads)
	require.EqualValues(t, 1, calls.Load())
	// A separately identified approved source uses the actual inventory Tool
	// and original source provenance, never an untyped URL or generated ID.
	snapshot, found, err := parent.configuration.FindImageAnalysisPreparation(context.Background(), agent.Scope{OrganizationID: "B", ActorID: "admin"}, key)
	require.NoError(t, err)
	require.True(t, found)
	version, err := strconv.ParseUint(source.CatalogVersion, 10, 64)
	require.NoError(t, err)
	repository, err := assetstore.NewRepository(f.owner)
	require.NoError(t, err)
	selected := source.Choices[0]
	_, err = repository.CommitApproval(context.Background(), asset.ApprovalCommit{TenantID: "B", ProductKey: source.ProductKey, TargetPlatform: "product", SourceSnapshotVersion: version, ActionID: uuid.NewString(), Assets: []asset.ApprovedAsset{{ID: "approved-native-source", Role: asset.RoleMain, URL: selected.URL, SourceAssetID: selected.ID, SourceApproval: &asset.SourceApprovalProvenance{ActorID: "admin", MemberID: snapshot.MemberID, OriginalPublicationID: source.PublicationID, OriginalSnapshotVersion: version, ReferenceHash: asset.ReferenceHash(selected.ID, selected.URL)}}}})
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(call("GET", base, "", nil, 200), &source))
	require.Equal(t, "READY", source.InventoryStatus)
	approvedKey := uuid.NewString()
	approvedIntent := imageAnalysisPrepareBody{Kind: "approved", ImageID: "approved-native-source", CatalogVersion: source.CatalogVersion, PublicationID: source.PublicationID, InventoryDigest: source.InventoryDigest, ExpectedModeRevision: "0"}
	var approved imageAnalysisResultDTO
	require.NoError(t, json.Unmarshal(call("POST", base+"/prepare", approvedKey, approvedIntent, 200), &approved))
	approvedConsent := imageAnalysisConfirmBody{PreparationID: approved.PreparationID, ConfigurationDigest: approved.ConfigurationDigest, QuoteDigest: approved.QuoteDigest, SelectionDigest: approved.SelectionDigest, ContentSHA256: approved.ContentSHA256, ModelID: approved.ModelID, MaximumMicropoints: approved.MaximumMicropoints, Confirmed: true}
	require.NoError(t, json.Unmarshal(call("POST", base+"/by-key/"+approvedKey+"/confirm", "", approvedConsent, 200), &approved))
	if approved.Status != "completed" {
		s, _, _ := parent.configuration.FindImageAnalysisPreparation(context.Background(), agent.Scope{OrganizationID: "B", ActorID: "admin"}, approvedKey)
		r, _, _ := parent.store.Lookup(context.Background(), s.Scope, s.Request.Binding, approvedKey)
		for _, h := range r.State.History {
			t.Logf("approved stop=%s tool=%s audit=%s", approved.StopReason, h.Tool.ID, h.AuditStatus)
		}
	}
	require.Equal(t, "completed", approved.Status)
	require.Equal(t, "31", approved.ActualMicropoints)
	require.EqualValues(t, 2, calls.Load())
	before = reads
	_, err = parent.configuration.Execute(context.Background(), agentconfig.Command{Scope: agent.Scope{OrganizationID: "B", ActorID: "admin"}, AgentID: agentconfig.ImageAgentID, Operation: "disable", Key: uuid.NewString(), Expected: 1})
	require.NoError(t, err)
	delete(parent.config.ImageAnalysisPolicies, "B")
	call("GET", base+"/by-key/"+key, "", nil, 200)
	call("POST", base+"/prepare", key, intent, 200)
	var recent struct {
		Items []struct {
			RequestKey string `json:"requestKey"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(call("GET", base+"/runs", "", nil, 200), &recent))
	require.Len(t, recent.Items, 2)
	require.Equal(t, before, reads, "disabled/current model removal does not refresh original history or preparation")
	require.EqualValues(t, 2, calls.Load())
}
