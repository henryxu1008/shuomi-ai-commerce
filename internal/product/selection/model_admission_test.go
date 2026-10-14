package selection

import (
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
		{"maximum count", strings.Fields(strings.Repeat("missing ", 4096)), true},
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
