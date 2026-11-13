package httpapi

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
)

func TestSettledRunPointsRejectsIncompleteOrUnknownCalls(t *testing.T) {
	s := agent.State{RunID: "run", Scope: agent.Scope{OrganizationID: "org"}, Phase: agent.HumanReviewRequired, Usage: agent.Usage{ModelCalls: 2}, History: []agent.Observation{{InvocationID: "one"}, {InvocationID: "two"}}}
	read := func(_ context.Context, _, id, _ string) (int64, error) {
		if id == "one" {
			return 3, nil
		}
		return 4, nil
	}
	points, err := settledRunPoints(context.Background(), s, read)
	require.NoError(t, err)
	require.EqualValues(t, 7, points)
	dup := s
	dup.History = []agent.Observation{{InvocationID: "one"}, {InvocationID: "one"}}
	_, err = settledRunPoints(context.Background(), dup, read)
	require.Error(t, err)
	unknown := s
	unknown.PendingInvocationID = "pending"
	_, err = settledRunPoints(context.Background(), unknown, read)
	require.Error(t, err)
	_, err = settledRunPoints(context.Background(), s, func(context.Context, string, string, string) (int64, error) { return 0, errors.New("unsettled") })
	require.Error(t, err)
}
