package agentconfigpersistence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	aistore "task-processor/internal/aicapability/store"
	imageapp "task-processor/internal/app/imageagent"
	"task-processor/internal/authidentity"
	"task-processor/internal/commercetool"
	einoruntime "task-processor/internal/integration/agent/eino"
	agentmodel "task-processor/internal/integration/agent/einomodel"
	governed "task-processor/internal/integration/aicapability/einomodel"
	resources "task-processor/internal/integration/orgresource"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/catalog/tools/canonicalinspect"
)

type nativeAnalysisFacts struct {
	scope     agent.Scope
	snapshot  catalog.PublishedSnapshot
	content   []byte
	canonical *canonicalinspect.Invoker
}

func (f *nativeAnalysisFacts) Authorize(context.Context, agent.Binding) (agent.Scope, error) {
	return f.scope, nil
}
func (f *nativeAnalysisFacts) AuthorizeModelInvocation(context.Context, aicapability.InvocationRecord) error {
	return nil
}
func (f *nativeAnalysisFacts) GetSnapshot(context.Context, catalog.SnapshotIdentity, uint64) (catalog.PublishedSnapshot, error) {
	return f.snapshot, nil
}
func (f *nativeAnalysisFacts) GetApprovedInventory(context.Context, asset.InventoryScope) (asset.ApprovedAssetInventory, error) {
	return asset.ApprovedAssetInventory{}, asset.ErrApprovedAssetsNotReady
}
func (f *nativeAnalysisFacts) ReadAnalysisSource(context.Context, agent.Binding, catalog.IdentifiedImage, int64) ([]byte, error) {
	return f.content, nil
}
func (f *nativeAnalysisFacts) ReadAnalysisApproved(context.Context, agent.Binding, asset.ApprovedAsset, int64) ([]byte, error) {
	return nil, agent.ErrInvalid
}
func (f *nativeAnalysisFacts) Invoke(ctx context.Context, ref commercetool.ToolRef, meta commercetool.CallMetadata, b agent.Binding) (commercetool.Result, error) {
	if ref != canonicalinspect.Definition().Ref {
		return commercetool.Result{}, agent.ErrInvalid
	}
	return f.canonical.Invoke(ctx, meta, canonicalinspect.Input{ProductKey: b.ProductKey, CatalogVersion: b.CatalogVersion})
}

type nativeAnalysisPrincipal struct{ scope agent.Scope }

func (f nativeAnalysisPrincipal) ResolvePrincipal(context.Context) (commercetool.Principal, error) {
	return commercetool.Principal{TenantID: f.scope.OrganizationID, UserID: f.scope.ActorID, Roles: []string{"fixture"}}, nil
}
func (f nativeAnalysisPrincipal) Authorize(context.Context, commercetool.Principal, commercetool.PermissionRequirement) error {
	return nil
}

type nativeAnalysisTransport func(*http.Request) (*http.Response, error)

func (f nativeAnalysisTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Compose the actual Prepare/Guard, Registry/audit, Eino graph, guarded SDK,
// Invocation/Resource owner and typed result validator. Only facts, identity
// authorization and wire response are synthetic; this is NOT product acceptance.
func TestImageAnalysisNativeRunAuditsThenSendsOnceAndSettlesOriginalMicropoints(t *testing.T) {
	for _, scenario := range []string{"success", "invalid output", "usage unknown"} {
		t.Run(scenario, func(t *testing.T) { nativeAnalysisRun(t, scenario) })
	}
}

func nativeAnalysisRun(t *testing.T, scenario string) {
	db, configuration, runs := fixture(t)
	c := analysisStart(t, configuration)
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	facts := &nativeAnalysisFacts{scope: c.Scope, content: b.Bytes(), snapshot: catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: c.Scope.OrganizationID, ProductKey: c.Request.Binding.ProductKey}, PublicationID: c.Request.Binding.PublicationID, Version: 1, Snapshot: catalog.ProductSnapshot{Images: []catalog.Image{{URL: "https://source.example/synthetic.png"}}}}}
	endpoint := "https://api.openai.com/v1"
	endpointHash := sha256.Sum256([]byte(endpoint))
	c.ExecutionModelProfile.EndpointIdentityDigest = hex.EncodeToString(endpointHash[:])
	c.ExecutionModelProfile.MaximumInputBytes = 32 << 10
	c.ExecutionModelProfile.MaximumPromptTokens = 16000
	c.Request.Limits.Tokens, c.Request.Limits.CostMicros = 18000, 20000
	id := authidentity.AuthenticatedIdentity{TenantID: c.Scope.OrganizationID, EffectiveOrganizationID: c.Scope.OrganizationID, UserID: c.Scope.ActorID, EffectiveMemberID: c.MemberID, TokenExpiresAt: time.Now().Add(time.Hour)}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), id)
	source, err := imageapp.NewAnalysisSources(facts, facts, facts, facts)
	require.NoError(t, err)
	c, err = source.FreezeImageAnalysis(ctx, c, "source", "catalog-image-1", "", "")
	require.NoError(t, err)
	snap, err := configuration.Prepare(context.Background(), c)
	require.NoError(t, err)
	r := snap.Request
	r.ConfigurationSnapshotRef = agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: snap.ID, Digest: snap.Digest}
	require.NoError(t, aistore.AutoMigrateInvocationLedger(db))
	require.NoError(t, resources.AutoMigrate(db))
	ledger := aistore.NewGormInvocationRecorder(db)
	points, err := resources.NewGormModelInvocationRepository(db, resources.TransactionConfig{}, ledger, facts)
	require.NoError(t, err)
	ledger.SetUsageSettler(points)
	limits, err := resources.NewGormMemberLimitRepository(db, resources.TransactionConfig{})
	require.NoError(t, err)
	_, err = limits.SetMonthlyLimit(ctx, orgresource.SetMemberLimitExecution{UnitVersion: aicapability.AIPointMicrounitV1, OrganizationID: id.TenantID, MemberID: id.EffectiveMemberID, ActorID: "fixture-admin", OperationID: "fixture-limit", Target: 1000000})
	require.NoError(t, err)
	now := time.Now().UTC()
	require.NoError(t, db.Exec("INSERT INTO saas_organization_resource_buckets (quantity_unit,organization_id,resource_type,available,allocated,reserved,consumed,created_at,updated_at) VALUES ('ai-micropoint-v1',?,'ai_point',1000000,0,0,0,?,?)", id.TenantID, now, now).Error)
	definition, tools := imageapp.ImageAnalysisRegistration(false)
	principal := nativeAnalysisPrincipal{scope: c.Scope}
	facts.canonical, err = canonicalinspect.NewInvoker(facts, definition, commercetool.InvocationDependencies{PrincipalResolver: principal, Authorizer: principal, Recorder: runs, Tracer: otel.Tracer("image-analysis-fixture"), Now: time.Now, AuditTimeout: time.Second})
	require.NoError(t, err)
	validator, err := imageapp.NewAnalysisResultValidator(configuration, runs, ledger, source)
	require.NoError(t, err)
	var sends atomic.Int32
	opinion := `{ "observations":[],"suggestions":[],"unknowns":["Synthetic image; actual quality not evaluated."] }`
	if scenario == "invalid output" {
		opinion = `{ "approval":true }`
	}
	executor := &governed.Executor{Ledger: ledger, Admission: governed.NewBoundedAdmission(), Authorize: func(context.Context, aicapability.TextInputIdentity) error { return nil }, Resolve: func(context.Context, aicapability.TextInputIdentity) (governed.QualifiedRoute, error) {
		return governed.QualifiedRoute{Profile: c.ExecutionModelProfile, Endpoint: endpoint, APIKey: "synthetic-only"}, nil
	}, BaseTransport: nativeAnalysisTransport(func(request *http.Request) (*http.Response, error) {
		sends.Add(1)
		var auditCount int64
		require.NoError(t, db.Table("product_agent_tool_calls").Count(&auditCount).Error)
		require.EqualValues(t, 1, auditCount, "actual audit must exist before the only model handoff")
		raw, e := io.ReadAll(request.Body)
		require.NoError(t, e)
		require.Contains(t, string(raw), "data:image/png;base64,")
		usage := `"prompt_tokens_details":{"cached_tokens":10}`
		if scenario == "usage unknown" {
			usage = `"prompt_tokens_details":{}`
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"id":"fixture","object":"chat.completion","created":1,"model":%q,"choices":[{"index":0,"message":{"role":"assistant","content":%q},"finish_reason":"stop"}],"usage":{"prompt_tokens":30,"completion_tokens":5,"total_tokens":35,%s}}`, aicapability.SingleImageMiniModel, opinion, usage)))}, nil
	})}
	model, err := agentmodel.NewImageAnalysisModel(executor, configuration, func(context.Context, string) (aicapability.ModelProfile, error) { return c.ExecutionModelProfile, nil }, definition.AllowedTools, func(context.Context) (authidentity.AuthenticatedIdentity, error) { return id, nil }, source)
	require.NoError(t, err)
	guard, err := NewGuard(db, configuration, runs, func(string, string) (agent.Limits, error) { return c.Request.Limits, nil }, func(context.Context) (agent.ExecutionIdentity, error) {
		return agent.ExecutionIdentity{OrganizationID: id.TenantID, ActorID: id.UserID, MemberID: id.EffectiveMemberID}, nil
	})
	require.NoError(t, err)
	runtime, err := einoruntime.New(einoruntime.Config{SingleCallReadOnly: true, PreReadTools: definition.AllowedTools, Definition: definition, Tools: tools, Model: model, Gateway: facts, DomainProposalKind: asset.ImageVisualAnalysisSchema, DomainValidator: validator, Authorizer: facts, Store: guard})
	require.NoError(t, err)
	record, err := runtime.Start(ctx, r)
	require.NoError(t, err)
	if scenario == "success" {
		require.Equal(t, agent.HumanReviewRequired, record.State.Phase, "phase alone does not prove an admitted result")
		require.Empty(t, record.State.StopReason)
		require.NotNil(t, record.State.Validation)
		require.True(t, record.State.Validation.Valid)
	} else {
		require.Equal(t, agent.Stopped, record.State.Phase)
		require.Nil(t, record.State.DomainProposal)
		if scenario == "invalid output" {
			require.Equal(t, agent.StopInvalidOutput, record.State.StopReason)
		} else {
			require.Equal(t, agent.StopModelUnknown, record.State.StopReason)
		}
	}
	require.Zero(t, record.State.Repairs)
	require.EqualValues(t, 1, sends.Load())
	var wg sync.WaitGroup
	type replayResult struct {
		record agent.Record
		err    error
	}
	results := make(chan replayResult, 6)
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			replay, e := runtime.Start(ctx, r)
			results <- replayResult{replay, e}
		}()
	}
	wg.Wait()
	close(results)
	for result := range results {
		require.NoError(t, result.err)
		require.Equal(t, record.State.RunID, result.record.State.RunID)
	}
	require.EqualValues(t, 1, sends.Load())
	var bucket struct{ Available, Consumed, Reserved int64 }
	require.NoError(t, db.Table("saas_organization_resource_buckets").Select("available,consumed,reserved").Take(&bucket).Error)
	if scenario == "usage unknown" {
		require.Zero(t, bucket.Consumed)
		require.Positive(t, bucket.Reserved, "missing complete usage retains the original reservation")
		require.EqualValues(t, 1000000, bucket.Available+bucket.Reserved)
	} else {
		require.EqualValues(t, 31, bucket.Consumed, "cached P already includes the image: charge once even for invalid model output")
		require.EqualValues(t, 999969, bucket.Available)
		require.Zero(t, bucket.Reserved)
	}
	_, err = runtime.Resume(ctx, r, record.State.Revision, "repeat")
	require.ErrorIs(t, err, agent.ErrInvalid)
}
