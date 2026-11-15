package aiworkbench

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type ScheduledPlanCopySource struct {
	ID       string
	Revision int64
	Digest   string
}

// A replacement is prepared by the current application owners, never bound
// from HTTP. The consumed observation fences a claim between quote and commit.
type ScheduledPlanReplacement struct {
	Draft               ScheduledPlanDraft
	ConsumedOccurrences uint64
}

func ScheduledPlanSourceDigest(p ScheduledPlanDraft) string {
	wire, _ := json.Marshal(struct {
		Scope     Scope
		Member    string
		Selection TaskSelection
		Report    *DailyReportSpec
		Goal      string
		Rule      ScheduleRule
		Budget    ScheduleBudget
		Execution json.RawMessage
	}{p.Scope, p.MemberID, p.Selection, p.Report, p.GoalSummary, p.Rule, p.Budget, p.ExecutionSpec})
	sum := sha256.Sum256(wire)
	return hex.EncodeToString(sum[:])
}
