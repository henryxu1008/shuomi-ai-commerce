package imageagentapp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"task-processor/internal/product/asset"
)

type analysisResultFacts struct {
	snapshot agentconfig.Snapshot
	run      agent.Record
	fact     aicapability.InvocationRecord
}

func (f *analysisResultFacts) LoadSnapshot(context.Context, agent.Scope, agent.ConfigurationSnapshotRef) (agentconfig.Snapshot, error) {
	return f.snapshot, nil
}
func (f *analysisResultFacts) Lookup(context.Context, agent.Scope, agent.Binding, string) (agent.Record, bool, error) {
	return f.run, true, nil
}
func (f *analysisResultFacts) ReadModelInvocation(context.Context, string, string) (aicapability.InvocationRecord, error) {
	return f.fact, nil
}

func analysisValidationFixture(t *testing.T) (*AnalysisResultValidator, *analysisFacts, *analysisResultFacts, context.Context, agent.Request, agent.DomainProposal, []agent.Observation) {
	t.Helper()
	source, f, ctx, in, a := analysisSourceFixture(t, false)
	raw := `{ "observations":[],"suggestions":[],"unknowns":["Only a read-only opinion."] }`
	opinion, err := asset.DecodeImageVisualOpinion([]byte(raw))
	require.NoError(t, err)
	a.QuoteDigest = strings.Repeat("b", 64)
	ref := agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: uuid.NewString(), Digest: strings.Repeat("c", 64)}
	r := agent.Request{Key: uuid.NewString(), Binding: in.Binding, ConfigurationSnapshotRef: ref, PolicyVersion: agentconfig.ImageAnalysisPolicyVersion, PromptVersion: agentconfig.ImageAnalysisPromptVersion}
	id := uuid.NewString()
	result := asset.ImageVisualAnalysisResult{Schema: asset.ImageVisualAnalysisSchema, SelectionDigest: a.Image.SelectionDigest, ContentSHA256: a.Image.ContentSHA256, InvocationID: id, ModelID: aicapability.SingleImageMiniModel, ConfigurationDigest: ref.Digest, ModelOutputSHA256: analysisEvidenceHash([]byte(raw)), ModelOutput: raw, QuoteDigest: a.QuoteDigest, Opinion: opinion}
	payload, err := json.Marshal(result)
	require.NoError(t, err)
	facts := &analysisResultFacts{snapshot: agentconfig.Snapshot{Scope: agent.Scope{OrganizationID: "org", ActorID: "actor"}, MemberID: "member", AgentID: agentconfig.ImageAgentID, AgentVersion: agentconfig.ImageAgentVersion, Request: agent.Request{Key: r.Key, Binding: r.Binding, PolicyVersion: r.PolicyVersion, PromptVersion: r.PromptVersion}, ImageAnalysis: &a, ExecutionModelProfile: aicapability.ModelProfile{ModelID: result.ModelID}}, run: agent.Record{State: agent.State{RunID: uuid.NewString(), Scope: agent.Scope{OrganizationID: "org", ActorID: "actor"}, Request: r}}, fact: aicapability.InvocationRecord{InvocationID: id, Operation: aicapability.OperationProductImageAnalysis, TenantID: "org", UserID: "actor", MemberID: "member", ModelID: result.ModelID, Outcome: aicapability.InvocationSucceeded, OutputHash: result.ModelOutputSHA256, UsageKnown: true, EstimatedCostKnown: true, PromptTokens: 2, CompletionTokens: 3, TotalTokens: 5, Currency: "CNY", EstimatedCostMicros: 5}}
	facts.fact.AgentRunID = facts.run.State.RunID
	facts.snapshot.ExecutionModelProfile.Currency = "CNY"
	history := append(in.History, agent.Observation{Step: 2, CallID: id, InvocationID: id, ObservedUsage: &agent.ObservedUsage{Known: true, Tokens: 5, CostMicros: 5, Currency: "CNY"}})
	v, err := NewAnalysisResultValidator(facts, facts, facts, source)
	require.NoError(t, err)
	return v, f, facts, ctx, r, agent.DomainProposal{Kind: asset.ImageVisualAnalysisSchema, Payload: payload}, history
}

func TestAnalysisValidatorBindsOriginalInvocationRunAndExactRawOpinion(t *testing.T) {
	v, source, _, ctx, r, p, h := analysisValidationFixture(t)
	valid, err := v.ValidateDomain(ctx, r, p, h)
	require.NoError(t, err)
	require.True(t, valid.Valid)
	require.Equal(t, r.PolicyVersion, valid.PolicyVersion)
	require.Zero(t, source.reads, "result validation never downloads an image or calls a model")
}
func TestAnalysisValidatorRejectsInventedMetadataOtherRunOrAlteredOpinion(t *testing.T) {
	for _, scenario := range []string{"run", "member", "outcome", "output hash", "scope", "usage", "typed opinion", "raw opinion", "permission"} {
		t.Run(scenario, func(t *testing.T) {
			v, source, f, ctx, r, p, h := analysisValidationFixture(t)
			switch scenario {
			case "run":
				f.fact.AgentRunID = uuid.NewString()
			case "member":
				f.fact.MemberID = "other"
			case "outcome":
				f.fact.Outcome = aicapability.InvocationDispatched
			case "output hash":
				f.fact.OutputHash = strings.Repeat("d", 64)
			case "scope":
				f.snapshot.Scope.OrganizationID = "other"
			case "usage":
				h[len(h)-1].ObservedUsage.Known = false
			case "typed opinion", "raw opinion":
				var result asset.ImageVisualAnalysisResult
				require.NoError(t, json.Unmarshal(p.Payload, &result))
				if scenario == "typed opinion" {
					result.Opinion.Unknowns = []string{"forged view"}
				} else {
					result.ModelOutput = "{}"
				}
				p.Payload, _ = json.Marshal(result)
			case "permission":
				source.deny = true
			}
			valid, err := v.ValidateDomain(ctx, r, p, h)
			require.True(t, err != nil || !valid.Valid)
			require.Zero(t, source.reads)
		})
	}
}
