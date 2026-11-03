package einomodel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"task-processor/internal/authidentity"
	"task-processor/internal/commercetool"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/catalog/tools/canonicalinspect"
)

const visualOpinionFixture = `{"observations":[{"area":"framing","status":"possible_issue","reason":"The visible subject may be cropped."}],"suggestions":[{"area":"framing","action":"check","reason":"Review the original image."}],"unknowns":["Actual material is not established by this image."]}`

type analysisSourceFixture struct {
	content []byte
	err     error
	checks  int
}

func (s *analysisSourceFixture) ReadImageAnalysis(context.Context, agent.ModelInput, agentconfig.ImageAnalysisConfiguration) ([]byte, json.RawMessage, error) {
	return s.content, json.RawMessage(`{"source":"catalog-image-1","scope":"exact saved version"}`), s.err
}
func (s *analysisSourceFixture) CheckImageAnalysis(context.Context, agent.ModelInput, agentconfig.ImageAnalysisConfiguration) error {
	s.checks++
	return s.err
}

type analysisExecutorFixture struct {
	calls               int
	raw                 string
	input               aicapability.TextInputIdentity
	beforeGate          func()
	terminalWriteFailed bool
}

func (e *analysisExecutorFixture) GenerateWithGate(ctx context.Context, input aicapability.TextInputIdentity, q aicapability.TextQuote, validate func(string) error, gate func(context.Context) (func(), error)) (governed.TextOutput, error) {
	e.calls++
	e.input = input
	if e.beforeGate != nil {
		e.beforeGate()
	}
	release, err := gate(ctx)
	if release != nil {
		defer release()
	}
	if err != nil {
		return governed.TextOutput{}, governed.ErrNotDispatched
	}
	out := governed.TextOutput{Content: e.raw, Usage: governed.ObservedUsage{PromptTokens: 200, CompletionTokens: 50, TotalTokens: 250, Known: true}}
	if validate(e.raw) != nil {
		if e.terminalWriteFailed {
			return out, errors.Join(governed.ErrInvalid, governed.ErrOutcomeUnknown)
		}
		return out, errors.Join(governed.ErrInvalid, governed.ErrObservedInvalidSettled)
	}
	return out, nil
}
func analysisModelFixture(t *testing.T) (*ImageAnalysisModel, context.Context, agent.ModelInput, *analysisExecutorFixture, *analysisSourceFixture) {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	content := b.Bytes()
	p := aicapability.ModelProfile{ClientName: "mini", ProviderID: "openai", AdapterKind: "openai-compatible", ModelID: aicapability.SingleImageMiniModel, EndpointIdentityDigest: strings.Repeat("a", 64), CredentialVersion: "fixture", RoutingPolicyVersion: "shared", AdapterPolicyVersion: aicapability.SingleImageMiniAdapterPolicy, PromptVersion: agentconfig.ImageAnalysisPromptVersion, OutputSchemaVersion: agentconfig.ImageAnalysisResultVersion, UsageMappingVersion: "openai-text-v1", CostPricingVersion: "budget", Currency: "CNY", InputMicrosPerMillion: 1000000, OutputMicrosPerMillion: 1000000, MaximumPromptTokens: 16000, MaximumCompletionTokens: 1024, MaximumInputBytes: 32 << 10, MaximumOutputBytes: 16384, DeadlineBound: time.Minute, OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "fixture", FXVersion: "fixture", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: aicapability.SingleImageMiniModel, Channel: "openai-direct", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 10000, CachedInputText: 1000, OutputText: 20000}}}
	i := authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", UserID: "actor", EffectiveMemberID: "member", TokenExpiresAt: time.Now().Add(time.Hour)}
	c := agentconfig.StartCommand{Scope: agent.Scope{OrganizationID: "org", ActorID: "actor"}, MemberID: "member", ExpectedModeRevision: "1", AgentID: agentconfig.ImageAgentID, AgentVersion: agentconfig.ImageAgentVersion, ExecutionModelProfile: p, Request: agent.Request{Key: uuid.NewString(), Binding: agent.Binding{ContextKind: "own_product", ContextID: uuid.NewString(), ProductKey: "product", CatalogVersion: "1", PublicationID: "publication", TargetPlatform: "product"}, PolicyVersion: agentconfig.ImageAnalysisPolicyVersion, PromptVersion: agentconfig.ImageAnalysisPromptVersion, Limits: agent.Limits{Steps: 6, ModelCalls: 1, Tokens: 18000, CostMicros: 20000, Currency: "CNY", Runtime: time.Minute}}}
	a, err := agentconfig.FreezeImageAnalysis(c, "source", "catalog-image-1", "", "", aicapability.SingleImageInput{ContentSHA256: agentTextHash(content), MediaType: "image/png", ByteLength: len(content), Width: 2, Height: 2, Detail: "high", MeteringVersion: aicapability.SingleImageMiniMetering})
	require.NoError(t, err)
	ref := agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: uuid.NewString(), Digest: strings.Repeat("c", 64)}
	c.Request.ModeSnapshotRef = agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: uuid.NewString(), Digest: strings.Repeat("d", 64)}
	s := agentconfig.Snapshot{Scope: c.Scope, MemberID: c.MemberID, ExpectedModeRevision: c.ExpectedModeRevision, AgentID: c.AgentID, AgentVersion: c.AgentVersion, Request: c.Request, ExecutionModelProfile: p, ImageAnalysis: &a}
	source := &analysisSourceFixture{content: content}
	executor := &analysisExecutorFixture{raw: visualOpinionFixture}
	model, err := NewImageAnalysisModel(executor, testSnapshotReader{s}, func(context.Context, string) (aicapability.ModelProfile, error) { return p, nil }, []commercetool.ToolRef{canonicalinspect.Definition().Ref}, func(context.Context) (authidentity.AuthenticatedIdentity, error) { return i, nil }, source)
	require.NoError(t, err)
	in := agent.ModelInput{ConfigurationSnapshotRef: ref, ModeSnapshotRef: c.Request.ModeSnapshotRef, Binding: c.Request.Binding, PolicyVersion: c.Request.PolicyVersion, PromptVersion: c.Request.PromptVersion, AgentRunID: uuid.NewString(), AgentID: agentconfig.ImageAnalysisAgentID, AgentVersion: agentconfig.ImageAnalysisVersion, History: []agent.Observation{{Step: 1, Tool: canonicalinspect.Definition().Ref, CallID: uuid.NewString(), AuditStatus: commercetool.AuditStatusRecorded, Output: json.RawMessage(`{"snapshot":"exact version"}`)}}}
	return model, authidentity.WithAuthenticatedIdentity(context.Background(), i), in, executor, source
}

func TestImageAnalysisModelUsesOriginalExecutorAndServerBoundReadonlyResult(t *testing.T) {
	m, ctx, in, e, source := analysisModelFixture(t)
	q, err := m.Quote(ctx, in)
	require.NoError(t, err)
	in.UpperBound, in.InvocationID = q, uuid.NewString()
	r, err := m.Decide(ctx, in)
	require.NoError(t, err)
	require.Equal(t, 1, e.calls)
	require.Equal(t, 1, source.checks)
	require.Equal(t, aicapability.OperationProductImageAnalysis, e.input.Operation)
	require.Equal(t, in.AgentRunID, e.input.AgentRunID)
	require.Equal(t, source.content, e.input.ImageContent)
	require.NotContains(t, e.input.Prompt, "data:image")
	require.Equal(t, "propose", r.Action.Kind)
	require.Empty(t, r.Action.Candidate.Changes)
	require.True(t, r.Usage.Known)
	var result asset.ImageVisualAnalysisResult
	require.NoError(t, json.Unmarshal(r.Action.DomainProposal.Payload, &result))
	require.True(t, result.Valid())
	require.Equal(t, in.InvocationID, result.InvocationID)
	require.Equal(t, in.ConfigurationSnapshotRef.Digest, result.ConfigurationDigest)
	require.Equal(t, agentTextHash([]byte(e.raw)), result.ModelOutputSHA256)
}

func TestImageAnalysisModelRequiresAuditedPrereadAndExactBytes(t *testing.T) {
	for _, scenario := range []string{"missing", "unrecorded", "duplicate", "wrong bytes", "anonymous"} {
		t.Run(scenario, func(t *testing.T) {
			m, ctx, in, e, s := analysisModelFixture(t)
			switch scenario {
			case "missing":
				in.History = nil
			case "unrecorded":
				in.History[0].AuditStatus = ""
			case "duplicate":
				in.History = append(in.History, in.History[0])
			case "wrong bytes":
				s.content = []byte("replacement")
			case "anonymous":
				ctx = context.Background()
			}
			_, err := m.Quote(ctx, in)
			require.Error(t, err)
			require.Zero(t, e.calls)
		})
	}
}

func TestImageAnalysisModelFreshRefusalIsKnownNoSendAndInvalidNeedsSettlement(t *testing.T) {
	for _, scenario := range []string{"before decide", "final handoff", "profile before decide", "profile final handoff", "invalid settled", "invalid unsettled"} {
		t.Run(scenario, func(t *testing.T) {
			m, ctx, in, e, s := analysisModelFixture(t)
			q, err := m.Quote(ctx, in)
			require.NoError(t, err)
			in.UpperBound, in.InvocationID = q, uuid.NewString()
			switch scenario {
			case "before decide":
				s.err = agent.ErrUnavailable
			case "final handoff":
				e.beforeGate = func() { s.err = agent.ErrUnavailable }
			case "profile before decide", "profile final handoff":
				change := func() {
					m.selectProfile = func(context.Context, string) (aicapability.ModelProfile, error) {
						return aicapability.ModelProfile{}, agent.ErrUnavailable
					}
				}
				if scenario == "profile before decide" {
					change()
				} else {
					e.beforeGate = change
				}
			case "invalid settled":
				e.raw = `{"invocationId":"fake","observations":[],"suggestions":[],"unknowns":["uncertain"]}`
			case "invalid unsettled":
				e.raw = "invalid"
				e.terminalWriteFailed = true
			}
			r, err := m.Decide(ctx, in)
			require.Error(t, err)
			require.Empty(t, r.Action.Kind)
			switch scenario {
			case "before decide", "final handoff", "profile before decide", "profile final handoff":
				require.ErrorIs(t, err, agent.ErrModelNotDispatched)
				require.True(t, r.Usage.Known)
				require.Zero(t, r.Usage.Tokens)
			case "invalid settled":
				require.ErrorIs(t, err, agent.ErrModelInvalidOutput)
				require.EqualValues(t, 250, r.Usage.Tokens)
			case "invalid unsettled":
				require.False(t, r.Usage.Known)
				require.NotErrorIs(t, err, agent.ErrModelInvalidOutput)
			}
		})
	}
}
