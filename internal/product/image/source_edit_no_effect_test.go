package image

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSourceEditPreflightHasProofOnlyBeforeBackend(t *testing.T) {
	calls := 0
	cap, err := NewSourceEditCapability(sourceEditorFunc(func(context.Context, SourceEditRequest) (Candidate, error) {
		calls++
		return Candidate{}, ErrInputInvalid
	}))
	require.NoError(t, err)
	_, err = cap.EditSources(context.Background(), SourceEditRequest{})
	var proof interface{ NoExternalEffect() bool }
	require.ErrorAs(t, err, &proof)
	require.True(t, proof.NoExternalEffect())
	require.True(t, errors.Is(err, ErrInputInvalid))
	require.Zero(t, calls)
	_, err = cap.EditSources(context.Background(), validSourceEditRequest())
	proof = nil
	require.False(t, errors.As(err, &proof), "a generic backend validation error is not no-effect evidence")
	require.Equal(t, 1, calls)
}
