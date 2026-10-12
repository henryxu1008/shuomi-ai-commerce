package agentcustomization

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"testing"
)

func TestPrivateRunCannotBypassModeOwner(t *testing.T) {
	command := RunCommand{Scope: Scope{OrganizationID: "org", ActorID: "actor"}, Key: uuid.NewString(), DeliveryID: uuid.NewString(), ExpectedModeRevision: "0", Input: DraftSelection{RecordID: uuid.NewString(), ExpectedRevision: 1}}
	command.ModeSnapshotRef = agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: uuid.NewString(), Digest: "forged"}
	_, err := (&Service{}).RunQuality(context.Background(), command)
	require.ErrorIs(t, err, ErrUnavailable, "a caller-supplied ref cannot replace the real mode owner")
}
