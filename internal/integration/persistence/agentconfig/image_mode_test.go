package agentconfigpersistence

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
	"testing"
)

func TestReadonlyImageModeBlocksGenerationAdmission(t *testing.T) {
	_, repo, _ := fixture(t)
	ctx := context.Background()
	start := imageStartFixture(t, repo)
	identity := agent.ExecutionIdentity{OrganizationID: start.Scope.OrganizationID, ActorID: start.Scope.ActorID, MemberID: start.MemberID}
	subject := agentconfig.ResolvedModeSubject{Subject: agentconfig.ModeSubject{Kind: agentconfig.RegisteredAgentSubject, ID: agentconfig.ImageAgentID}, Definition: commercetool.AgentRef{ID: agentconfig.ImageAgentID, Version: agentconfig.ImageAgentVersion}}
	mode, err := repo.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: identity, Subject: subject, Key: uuid.NewString(), Mode: agent.ModeReadOnly})
	require.NoError(t, err)
	start.ExpectedModeRevision = mode.Revision
	snapshot, err := repo.PrepareImageConfiguration(ctx, start)
	require.NoError(t, err)
	require.True(t, snapshot.ModeSnapshotRef.Valid())
	admission := agentconfig.ImageRunAdmissionCommand{Scope: start.Scope, Snapshot: snapshot.Ref(), MemberID: start.MemberID, RunID: start.RunID, ConfirmActionID: uuid.NewString(), SourceDigest: start.SourceDigest, InputDigest: start.InputDigest, PlanDigest: digest([]byte("plan")), QuoteDigest: digest([]byte("quote")), Limits: agentconfig.ImageRunLimits{Images: 2, Points: 30, ElapsedSeconds: 180}}
	_, err = repo.AdmitImageRun(ctx, admission, start.HardLimits)
	require.ErrorIs(t, err, agentconfig.ErrForbidden)
	_, err = repo.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: identity, Subject: subject, Key: uuid.NewString(), ExpectedRevision: 1, Mode: agent.ModeFullAccess})
	require.NoError(t, err)
	_, err = repo.AdmitImageRun(ctx, admission, start.HardLimits)
	require.ErrorIs(t, err, agentconfig.ErrChanged, "widening mode must not upgrade the prepared original plan")
}
