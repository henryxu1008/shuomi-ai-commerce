package einomodel

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
)

func TestSkillAuthoringRecordsDefinitiveOpenAIParameterRejection(t *testing.T) {
	const rejection = `{"error":{"message":"The supplied parameter is unsupported.","type":"invalid_request_error","code":"unsupported_parameter","param":"max_tokens"}}`
	for _, tc := range []struct {
		name, endpoint, body, requestID string
		status                          int
		operation                       aicapability.Operation
		recognized                      bool
	}{
		{"definitive_rejection", TitlePilotOpenAIEndpoint, rejection, "req_fixture", 400, aicapability.OperationAIWorkbenchSkillAuthoring, true},
		{"gateway", "https://gateway.invalid/v1", rejection, "req_fixture", 400, aicapability.OperationAIWorkbenchSkillAuthoring, false},
		{"another_operation", TitlePilotOpenAIEndpoint, rejection, "req_fixture", 400, aicapability.OperationAIWorkbenchChatPlan, false},
		{"no_request_id", TitlePilotOpenAIEndpoint, rejection, "", 400, aicapability.OperationAIWorkbenchSkillAuthoring, false},
		{"rate_limited", TitlePilotOpenAIEndpoint, rejection, "req_fixture", 429, aicapability.OperationAIWorkbenchSkillAuthoring, false},
		{"unrecognized_parameter", TitlePilotOpenAIEndpoint, strings.Replace(rejection, "max_tokens", "temperature", 1), "req_fixture", 400, aicapability.OperationAIWorkbenchSkillAuthoring, false},
		{"usage_conflicts", TitlePilotOpenAIEndpoint, rejection[:len(rejection)-1] + `,"usage":{"total_tokens":12}}`, "req_fixture", 400, aicapability.OperationAIWorkbenchSkillAuthoring, false},
		{"duplicate_field", TitlePilotOpenAIEndpoint, strings.Replace(rejection, `"param":`, `"param":"temperature","param":`, 1), "req_fixture", 400, aicapability.OperationAIWorkbenchSkillAuthoring, false},
		{"incomplete_json", TitlePilotOpenAIEndpoint, rejection[:len(rejection)-1], "req_fixture", 400, aicapability.OperationAIWorkbenchSkillAuthoring, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			profile := testTextProfile(tc.endpoint)
			profile.ProviderID = "openai"
			profile.ModelID = aicapability.SingleImageMiniModel
			profile.OfficialTariff.Model = profile.ModelID
			profile.OfficialTariff.Channel = "openai-direct"
			input := aicapability.TextInputIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member", Operation: tc.operation, InvocationID: "original", System: "system", Prompt: "prompt", Profile: profile}
			quote, err := aicapability.QuoteText(input)
			require.NoError(t, err)
			ledger := &recordingLedger{}
			sends := 0
			executor := Executor{Ledger: ledger, Admission: NewBoundedAdmission(),
				Authorize: func(context.Context, aicapability.TextInputIdentity) error { return nil },
				Resolve: func(context.Context, aicapability.TextInputIdentity) (QualifiedRoute, error) {
					return QualifiedRoute{Profile: profile, Endpoint: tc.endpoint, APIKey: "fixture-only"}, nil
				},
				BaseTransport: imageFixtureTransport(func(r *http.Request) (*http.Response, error) {
					sends++
					wire, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					require.Contains(t, string(wire), `"max_tokens":64`)
					h := http.Header{}
					h.Set("Content-Type", "application/json; charset=utf-8")
					if tc.requestID != "" {
						h.Set("X-Request-Id", tc.requestID)
					}
					return &http.Response{StatusCode: tc.status, Header: h, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
				}),
			}
			_, err = executor.Generate(context.Background(), input, quote, nil)
			require.Error(t, err)
			require.True(t, ledger.reserved)
			require.Equal(t, 1, sends)
			if tc.recognized {
				require.NotErrorIs(t, err, ErrOutcomeUnknown)
				require.NotErrorIs(t, err, ErrNotDispatched)
				require.Len(t, ledger.records, 2)
				fact := ledger.records[1]
				require.Equal(t, aicapability.InvocationFailed, fact.Outcome)
				require.Equal(t, aicapability.ErrorProviderRejected, fact.ErrorCategory)
				require.False(t, fact.UsageKnown)
				require.True(t, fact.EstimatedCostKnown)
				require.Zero(t, fact.EstimatedCostMicros)
				require.NotContains(t, fact.ProviderRequestID, tc.requestID)
			} else {
				require.ErrorIs(t, err, ErrOutcomeUnknown)
				require.Len(t, ledger.records, 1)
			}
			_, err = executor.Generate(context.Background(), input, quote, nil)
			require.ErrorIs(t, err, ErrOutcomeUnknown)
			require.Equal(t, 1, sends)
		})
	}
}
