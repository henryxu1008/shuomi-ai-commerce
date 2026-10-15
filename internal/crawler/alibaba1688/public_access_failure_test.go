package alibaba1688

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPublicAccessErrorPreservesKindAndCause(t *testing.T) {
	for _, kind := range []PublicAccessFailureKind{
		PublicAccessFailureBrowser, PublicAccessFailureChallenge, PublicAccessFailureMissingFields,
		PublicAccessFailureValidation, PublicAccessFailureInvalidURL, PublicAccessFailureTransport,
	} {
		t.Run(string(kind), func(t *testing.T) {
			for _, cause := range []error{errors.New("source failed"), context.Canceled, context.DeadlineExceeded} {
				err := NewPublicAccessError(kind, cause)
				var classified *PublicAccessError
				require.ErrorAs(t, fmt.Errorf("acquire: %w", err), &classified)
				require.Equal(t, kind, classified.Kind)
				require.ErrorIs(t, err, cause)
				require.Equal(t, cause.Error(), err.Error())
			}
		})
	}
}
