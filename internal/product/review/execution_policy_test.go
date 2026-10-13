package review

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/agent"
	"testing"
)

func TestPolicyProvenanceIsDistinctFromHumanDecision(t *testing.T) {
	execution := ExecutionProvenance{RunID: uuid.NewString(), RequestKey: uuid.NewString(), Binding: agent.Binding{ContextKind: "acquisition", ContextID: uuid.NewString(), ProductKey: "p", CatalogVersion: "1", PublicationID: "pub", TargetPlatform: "shein"}, Configuration: agent.ConfigurationSnapshotRef{Kind: "agent-configuration-v1", ID: uuid.NewString(), Digest: strings.Repeat("a", 64)}, Mode: agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: uuid.NewString(), Digest: strings.Repeat("b", 64)}}
	proof := PolicyProvenance{Execution: execution, Identity: agent.ExecutionIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member"}, Mode: agent.ModeFullAccess, ModeRevision: "1", PolicyVersion: agent.ExecutionModePolicyVersion, ActionVersion: TitlePolicyAdoptionVersion, ProposalDigest: strings.Repeat("c", 64)}
	execution.Scope = agent.Scope{OrganizationID: "org", ActorID: "actor"}
	proof.Execution = execution
	require.True(t, proof.Valid())
	changed := proof
	changed.Mode = agent.ModeReadOnly
	require.False(t, changed.Valid())
	r := Record{Execution: &execution, PolicyAdoption: &proof}
	require.Empty(t, r.View().History, "policy authority cannot be recorded as a human click")
	require.Equal(t, &proof, r.View().PolicyAdoption)
}
