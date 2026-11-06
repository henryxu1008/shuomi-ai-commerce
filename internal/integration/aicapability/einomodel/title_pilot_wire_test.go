package einomodel

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
)

func TestTitlePilotWireRejectsUnpricedOrDifferentProtocol(t *testing.T) {
	wire := `{"model":"gpt-5.4-mini-2026-03-17","messages":[{"role":"system","content":"instructions"},{"role":"user","content":"facts"}],"max_completion_tokens":2048,"reasoning_effort":"none","service_tier":"default","n":1,"store":false}`
	require.True(t, validTitlePilotWire([]byte(wire), aicapability.SingleImageMiniModel))
	for _, changed := range []struct{ name, old, new string }{
		{"wrong snapshot", "gpt-5.4-mini-2026-03-17", "gpt-5.4-mini"},
		{"legacy output bound", "max_completion_tokens", "max_tokens"},
		{"different output bound", "2048", "2049"},
		{"paid tier", "default", "priority"},
		{"reasoning", "none", "high"},
		{"multiple outputs", `"n":1`, `"n":2`},
		{"store", `"store":false`, `"store":true`},
		{"provider tools", `"store":false`, `"store":false,"tools":[]`},
		{"duplicate cap", `"store":false`, `"store":false,"max_completion_tokens":2048`},
	} {
		t.Run(changed.name, func(t *testing.T) {
			require.False(t, validTitlePilotWire([]byte(strings.Replace(wire, changed.old, changed.new, 1)), aicapability.SingleImageMiniModel))
		})
	}
}
