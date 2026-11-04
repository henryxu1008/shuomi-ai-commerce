package asset

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImageVisualOpinionIsBoundedReadonlyAdvice(t *testing.T) {
	raw := []byte(`{"observations":[{"area":"framing","status":"possible_issue","reason":"主体可能被裁切，请人工核对。"}],"suggestions":[{"area":"framing","action":"check","reason":"核对原图边缘。"}],"unknowns":["未评价商品属性与平台准入。"]}`)
	opinion, err := DecodeImageVisualOpinion(raw)
	require.NoError(t, err)
	require.Len(t, opinion.Observations, 1)
	require.Equal(t, "possible_issue", opinion.Observations[0].Status)
	for _, invalid := range []string{
		strings.Replace(string(raw), `"possible_issue"`, `"approved"`, 1),
		strings.Replace(string(raw), `"check"`, `"generate"`, 1),
		strings.Replace(string(raw), `"framing"`, `"material"`, 1),
		strings.Replace(string(raw), `"unknowns":`, `"approved":true,"unknowns":`, 1),
		strings.Replace(string(raw), `"status":"possible_issue"`, `"status":"possible_issue","status":"possible_issue"`, 1),
		`{"observations":[],"suggestions":[],"unknowns":[]}`,
		`{"observations":[],"suggestions":[],"unknowns":null}`,
		`{"observations":[],"suggestions":[],"unknowns":["缺少证据"],"invocationId":"model-invented"}`,
	} {
		_, err := DecodeImageVisualOpinion([]byte(invalid))
		require.Error(t, err, invalid)
	}
	_, err = DecodeImageVisualOpinion([]byte(`{"observations":[],"suggestions":[],"unknowns":["无法确定，请人工核对。"]}`))
	require.NoError(t, err, "uncertainty is not converted into a factual failure or approval")
}
