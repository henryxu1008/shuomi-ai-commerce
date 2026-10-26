package httpapi

import (
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/httproute"
)

// Retains the original #717 probe whose full and title-only inventories both
// failed on #715 dbfea. The validator is the real global startup admission.
func TestContentTitleOnlyAdmissionUsesExactTrustedProfile(t *testing.T) {
	validate := func(routes []httproute.Descriptor, titleOnly bool) error {
		return validateCurrentApplicationRoutesInternal(routes, false, false, false, false, false, false, false, currentApplicationOptionalRoutes{ProductAgent: true, ContentTitleOnly: titleOnly})
	}
	for _, titleOnly := range []bool{false, true} {
		name := "full"
		if titleOnly {
			name = "title-only"
		}
		t.Run(name, func(t *testing.T) {
			routes := agentAndPointRoutes(false, false)
			if !titleOnly {
				routes = append(routes, productAgentRoutes(nil)...)
			}
			routes = append(routes, registeredProductAgentRoutes(nil, true)...)
			routes = append(routes, productReviewRoutes(nil, nil)...)
			require.NoError(t, validate(routes, titleOnly))
			require.Error(t, validate(routes, !titleOnly), "the other profile must not be admitted")
			missing := append([]httproute.Descriptor(nil), routes[:len(routes)-1]...)
			require.Error(t, validate(missing, titleOnly), "the exact profile still requires every route")
		})
	}
	require.Error(t, validateCurrentApplicationRoutesInternal(agentAndPointRoutes(false, false), false, false, false, false, false, false, false, currentApplicationOptionalRoutes{ContentTitleOnly: true}))
}
