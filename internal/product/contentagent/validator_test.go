package contentagent

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/product/enrichment"
	"task-processor/internal/product/review"
)

type contentAuthorization struct{}

func (contentAuthorization) Authorize(context.Context, agent.Binding) (agent.Scope, error) {
	return agent.Scope{OrganizationID: "org", ActorID: "actor"}, nil
}

type contentConfiguration struct {
	snapshot agentconfig.Snapshot
	revoked  bool
}

func (c *contentConfiguration) LoadSnapshot(context.Context, agent.Scope, agent.ConfigurationSnapshotRef) (agentconfig.Snapshot, error) {
	return c.snapshot, nil
}
func (c *contentConfiguration) CheckRegisteredTextExecutionMode(context.Context, agent.ExecutionIdentity, agent.ConfigurationSnapshotRef, agent.ModeSnapshotRef) (agentconfig.ModeSnapshot, error) {
	if c.revoked {
		return agentconfig.ModeSnapshot{}, agentconfig.ErrChanged
	}
	return agentconfig.ModeSnapshot{}, nil
}

func TestContentValidatorUsesFrozenConfigurationAndActualObservedEvidence(t *testing.T) {
	selection, err := enrichment.NewContentSelection("en-US", []string{"title", "description"})
	require.NoError(t, err)
	mode := agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: uuid.NewString(), Digest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	request := agent.Request{ContentSelection: selection, Key: uuid.NewString(), ModeSnapshotRef: mode, ConfigurationSnapshotRef: agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: uuid.NewString(), Digest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, Binding: agent.Binding{ContextKind: "acquisition", ContextID: "context", ProductKey: "product", CatalogVersion: "1", PublicationID: "publication", TargetPlatform: "product"}, PolicyVersion: enrichment.ContentPolicyVersion, PromptVersion: agentconfig.ContentPromptVersion, Limits: agent.Limits{Steps: 4, ModelCalls: 2, Tokens: 1000, CostMicros: 1000, Currency: "CNY", Runtime: time.Minute}}
	configuration := &contentConfiguration{snapshot: agentconfig.Snapshot{Scope: agent.Scope{OrganizationID: "org", ActorID: "actor"}, MemberID: "member", AgentID: agentconfig.ContentAgentID, AgentVersion: agentconfig.ContentAgentVersion, Request: request}}
	calls := 0
	observed := true
	validator := Validator{Authorization: contentAuthorization{}, Configuration: configuration, Identity: func(context.Context) (agent.ExecutionIdentity, error) {
		return agent.ExecutionIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member"}, nil
	}, Review: func(_ context.Context, input review.ContentValidationInput) (enrichment.Proposal, error) {
		calls++
		require.Equal(t, selection, input.Selection)
		require.Equal(t, "publication", input.PublicationID)
		return enrichment.Proposal{Validation: enrichment.ValidationResult{Valid: true}}, nil
	}, Observed: func(agent.Request, enrichment.Candidate, []agent.Observation) bool { return observed }}
	result, err := validator.Validate(context.Background(), request, enrichment.Candidate{}, nil)
	require.NoError(t, err)
	require.True(t, result.Valid)
	require.Equal(t, 1, calls)
	bad := request
	bad.ContentSelection.TargetLanguage = "de"
	_, err = validator.Validate(context.Background(), bad, enrichment.Candidate{}, nil)
	require.Error(t, err)
	require.Equal(t, 1, calls)
	observed = false
	result, err = validator.Validate(context.Background(), request, enrichment.Candidate{}, nil)
	require.NoError(t, err)
	require.False(t, result.Valid)
	require.Equal(t, 1, calls)
	observed = true
	configuration.revoked = true
	_, err = validator.Validate(context.Background(), request, enrichment.Candidate{}, nil)
	require.ErrorIs(t, err, agentconfig.ErrChanged)
	require.Equal(t, 1, calls)
}
