package einomodel

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
)

func TestBoundedAdmissionSharesCredentialBurstAcrossOperations(t *testing.T) {
	admission := NewBoundedAdmission()
	input := aicapability.TextInputIdentity{OrganizationID: "org", Operation: aicapability.OperationAIWorkbenchChatPlan,
		Profile: aicapability.ModelProfile{ClientName: "shared-model", ConnectionID: "shared-credential"}}
	for range 15 {
		release, err := admission.Acquire(context.Background(), input)
		require.NoError(t, err)
		release()
	}
	input.Operation = aicapability.OperationProductAgentDecision
	bounded, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	_, err := admission.Acquire(bounded, input)
	require.ErrorIs(t, err, ErrNotDispatched, "Chat and title use one credential burst budget")
	input.OrganizationID = "other-org"
	release, err := admission.Acquire(context.Background(), input)
	require.NoError(t, err, "one organization's budget must not consume another's")
	release()
}

func TestEinoSharedConnectionDoesNotMultiplyConcurrencyByModel(t *testing.T) {
	a := NewBoundedAdmission()
	in := aicapability.TextInputIdentity{OrganizationID: "org", Profile: aicapability.ModelProfile{ClientName: "model-one", ConnectionID: "shared-connection"}}
	var releases []func()
	defer func() {
		for _, release := range releases {
			release()
		}
	}()
	for range 10 {
		release, err := a.Acquire(context.Background(), in)
		require.NoError(t, err)
		releases = append(releases, release)
	}
	in.Profile.ClientName = "model-two"
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	release, err := a.Acquire(ctx, in)
	if release != nil {
		release()
	}
	require.ErrorIs(t, err, ErrNotDispatched, "another model using the same connection must not get another ten provider slots")
	in.OrganizationID = "other-org"
	release, err = a.Acquire(context.Background(), in)
	require.NoError(t, err)
	release()
}
