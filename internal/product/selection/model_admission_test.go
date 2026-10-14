package selection

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
)

func TestSelectionUnresolvedFitsReadableResponseContract(t *testing.T) {
	for _, tc := range []struct {
		name       string
		unresolved []string
		valid      bool
	}{
		{"empty list", nil, true},
		{"maximum model count", strings.Fields(strings.Repeat("missing ", 4095)), true},
		{"must reserve server diagnostic", strings.Fields(strings.Repeat("missing ", 4096)), false},
		{"excess count", strings.Fields(strings.Repeat("missing ", 4097)), false},
		{"maximum entry", []string{strings.Repeat("x", 8192)}, true},
		{"oversized entry", []string{strings.Repeat("x", 8193)}, false},
		{"empty entry", []string{""}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := (ModelAdmission{}).ValidateAction(agent.Action{Kind: "interrupt", Unresolved: tc.unresolved})
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, agent.ErrInvalid)
			}
		})
	}
}

func TestSelectionModelCapacityLeavesRoomForNativeValidationDiagnostic(t *testing.T) {
	action := agent.Action{Kind: "interrupt", Unresolved: strings.Fields(strings.Repeat("missing ", 4095))}
	require.NoError(t, (ModelAdmission{}).ValidateAction(action))
	validation, err := (Validator{}).ValidateDomain(context.Background(), agent.Request{}, agent.DomainProposal{}, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"SELECTION_EVIDENCE_INVALID"}, validation.Unresolved)
	combined := append(append([]string{}, action.Unresolved...), validation.Unresolved...)
	require.Len(t, combined, 4096, "native rejection must still fit the unchanged response contract")
}
