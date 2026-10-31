package orgresourceadapter

import (
	"encoding/json"
	"testing"
	"time"

	"task-processor/internal/aicapability"
)

func modelPointHashFixture() aicapability.InvocationRecord {
	start := time.Date(2026, 10, 31, 17, 23, 41, 123456789, time.FixedZone("fixture", 19800))
	return aicapability.InvocationRecord{
		PointTariff:         aicapability.ModelPointTariff{PriceVersion: "fixture-price", InputPointsPerMillionTokens: 1000000, OutputPointsPerMillionTokens: 2000000},
		MaximumPromptTokens: 20, MaximumCompletionTokens: 30,
		InvocationID: "fixture-invocation", ParentInvocationID: "fixture-parent", AgentRunID: "fixture-run",
		TenantID: "fixture-org", UserID: "fixture-actor", MemberID: "fixture-member", BusinessTaskID: "fixture-task", TraceID: "fixture-trace",
		Capability: aicapability.CapabilityProductEnrichText, Operation: aicapability.OperationProductAgentDecision,
		RouteMode: "fixture-route-mode", RouteOutcome: aicapability.RouteOutcomeActive, CacheStatus: "fixture-cache",
		ProviderID: "fixture-provider", ModelID: "fixture-model", RequestedRoutingKey: "fixture-requested", RoutingKey: "fixture-selected",
		CredentialReference: "fixture-credential-ref", PolicyVersion: "fixture-policy", ConfigurationVersion: "fixture-config",
		PromptKey: "fixture-prompt", PromptVersion: "fixture-version", PromptScope: "fixture-scope", PromptHash: "fixture-prompt-hash",
		StartedAt: start, FinishedAt: start.Add(1234567890 * time.Nanosecond), LatencyMilliseconds: 1234, Attempt: 1, FallbackIndex: 0,
		PromptTokens: 7, CompletionTokens: 5, TotalTokens: 12, ImageCount: 0, EstimatedCostMicros: 19,
		EstimatedCostKnown: true, UsageKnown: true, Currency: "fixture-currency", Outcome: aicapability.InvocationUsageObservedFailed,
		ErrorCategory: "fixture-output", RouteErrorCategory: "fixture-route-error", ErrorCode: "fixture-code",
		ProviderRequestID: "fixture-request", UpstreamJobID: "fixture-job", InputHash: "fixture-input", OutputHash: "fixture-output",
	}
}

func modelPointDispatchFixture(fact aicapability.InvocationRecord) aicapability.InvocationRecord {
	fact.StartedAt = fact.StartedAt.UTC().Truncate(time.Microsecond)
	fact.FinishedAt = time.Time{}
	fact.LatencyMilliseconds = 0
	fact.PromptTokens, fact.CompletionTokens, fact.TotalTokens, fact.ImageCount = 0, 0, 0, 0
	fact.EstimatedCostMicros, fact.EstimatedCostKnown, fact.UsageKnown = 0, false, false
	fact.Outcome = aicapability.InvocationDispatched
	fact.ErrorCategory, fact.ErrorCode, fact.RouteErrorCategory = "", "", ""
	fact.ProviderRequestID, fact.UpstreamJobID, fact.OutputHash = "", "", ""
	return fact
}

// Captured before the retirement from approved design HEAD df8973a99.
const originalDispatchJson = `{"PointTariff":{"priceVersion":"fixture-price","inputPointsPerMillionTokens":1000000,"outputPointsPerMillionTokens":2000000},"MaximumPromptTokens":20,"MaximumCompletionTokens":30,"InvocationID":"fixture-invocation","ParentInvocationID":"fixture-parent","AgentRunID":"fixture-run","TenantID":"fixture-org","UserID":"fixture-actor","MemberID":"fixture-member","BusinessTaskID":"fixture-task","TraceID":"fixture-trace","Capability":"productenrich.text_understanding","Operation":"product_agent_decision","RouteMode":"fixture-route-mode","RouteOutcome":"active","CacheStatus":"fixture-cache","ProviderID":"fixture-provider","ModelID":"fixture-model","RequestedRoutingKey":"fixture-requested","RoutingKey":"fixture-selected","CredentialReference":"fixture-credential-ref","PolicyVersion":"fixture-policy","ConfigurationVersion":"fixture-config","PromptKey":"fixture-prompt","PromptVersion":"fixture-version","PromptScope":"fixture-scope","PromptHash":"fixture-prompt-hash","StartedAt":"2026-10-31T11:53:41.123456Z","FinishedAt":"0001-01-01T00:00:00Z","LatencyMilliseconds":0,"Attempt":1,"FallbackIndex":0,"PromptTokens":0,"CompletionTokens":0,"TotalTokens":0,"ImageCount":0,"EstimatedCostMicros":0,"EstimatedCostKnown":false,"UsageKnown":false,"Currency":"fixture-currency","Outcome":"dispatched","ErrorCategory":"","RouteErrorCategory":"","ErrorCode":"","ProviderRequestID":"","UpstreamJobID":"","InputHash":"fixture-input","OutputHash":"","ReviewScore":0,"ReviewNeedsHumanReview":false,"ReviewReasons":null}`
const originalDispatchHash = `b87b1fb8875fbcd39437d562a99b8f76aaf1aa879a6d78d9e11b8ddb0c5294cc`
const originalTerminalJson = `{"PointTariff":{"priceVersion":"fixture-price","inputPointsPerMillionTokens":1000000,"outputPointsPerMillionTokens":2000000},"MaximumPromptTokens":20,"MaximumCompletionTokens":30,"InvocationID":"fixture-invocation","ParentInvocationID":"fixture-parent","AgentRunID":"fixture-run","TenantID":"fixture-org","UserID":"fixture-actor","MemberID":"fixture-member","BusinessTaskID":"fixture-task","TraceID":"fixture-trace","Capability":"productenrich.text_understanding","Operation":"product_agent_decision","RouteMode":"fixture-route-mode","RouteOutcome":"active","CacheStatus":"fixture-cache","ProviderID":"fixture-provider","ModelID":"fixture-model","RequestedRoutingKey":"fixture-requested","RoutingKey":"fixture-selected","CredentialReference":"fixture-credential-ref","PolicyVersion":"fixture-policy","ConfigurationVersion":"fixture-config","PromptKey":"fixture-prompt","PromptVersion":"fixture-version","PromptScope":"fixture-scope","PromptHash":"fixture-prompt-hash","StartedAt":"2026-10-31T11:53:41.123456Z","FinishedAt":"2026-10-31T11:53:42.358024Z","LatencyMilliseconds":1234,"Attempt":1,"FallbackIndex":0,"PromptTokens":7,"CompletionTokens":5,"TotalTokens":12,"ImageCount":0,"EstimatedCostMicros":19,"EstimatedCostKnown":true,"UsageKnown":true,"Currency":"fixture-currency","Outcome":"usage_observed_failed","ErrorCategory":"fixture-output","RouteErrorCategory":"fixture-route-error","ErrorCode":"fixture-code","ProviderRequestID":"fixture-request","UpstreamJobID":"fixture-job","InputHash":"fixture-input","OutputHash":"fixture-output","ReviewScore":0,"ReviewNeedsHumanReview":false,"ReviewReasons":null}`
const originalTerminalHash = `5dc849c746c475d0846ab5043751651ff2bf304bcc2bf0eb3b5560d7db1bc1ba`

func TestModelPointFactFingerprintsPreserveOriginalJSONAndHashes(t *testing.T) {
	fact := modelPointHashFixture()
	dispatch, err := json.Marshal(modelPointFactEncoding{InvocationRecord: modelPointDispatchFixture(fact)})
	if err != nil {
		t.Fatal(err)
	}
	if string(dispatch) != originalDispatchJson || modelPointFingerprint(fact) != originalDispatchHash {
		t.Fatalf("dispatch identity changed: %s / %s", dispatch, modelPointFingerprint(fact))
	}
	fact.StartedAt = fact.StartedAt.UTC().Truncate(time.Microsecond)
	fact.FinishedAt = fact.FinishedAt.UTC().Truncate(time.Microsecond)
	terminal, err := json.Marshal(modelPointFactEncoding{InvocationRecord: fact})
	if err != nil {
		t.Fatal(err)
	}
	if string(terminal) != originalTerminalJson || modelPointFactHash(fact) != originalTerminalHash {
		t.Fatalf("terminal proof changed: %s / %s", terminal, modelPointFactHash(fact))
	}
	if modelPointFactHash(fact) == modelPointFingerprint(fact) {
		t.Fatal("terminal proof was replaced by reserve fingerprint")
	}
	fact.OutputHash = "different-output"
	if modelPointFactHash(fact) == originalTerminalHash {
		t.Fatal("terminal proof lost observed output identity")
	}
	if modelPointFingerprint(fact) != originalDispatchHash {
		t.Fatal("terminal output polluted the dispatch identity")
	}
}
