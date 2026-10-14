package review

import (
	"encoding/json"
	"strings"
	"task-processor/internal/agent"
	"time"
)

// ValidateStorage reserves the exact remaining shape of a successful accept
// and Apply. A pending or accepted object must never exhaust storage only when
// its decision or receipt is appended. Oversized edits leave the old state intact.
func (r Record) ValidateStorage() error {
	future := r
	future.Content = r.Content.clone()
	at := time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC)
	actor := strings.Repeat("<", 128) // JSON's maximum escaping expansion per actor byte.
	if future.State == "pending" {
		future.Revision++
		if future.Content != nil {
			future.Content.Decisions = append(future.Content.Decisions, ContentDecision{"accept", actor, future.Revision, cloneContentValues(future.Content.After), cloneContentValues(future.Content.After), at})
		} else {
			future.History = append(append([]Decision(nil), r.History...), Decision{"accept", actor, future.Revision, r.Title, r.Title, at})
		}
	}
	if future.State == "pending" || future.State == "accepted" {
		future.State = "applied"
		future.Receipt = &Receipt{r.ID, future.Revision, 1<<63 - 1, "review:" + strings.Repeat("f", 64), actor, at}
	}
	values := []any{r, r.View(), future, future.View()}
	if r.Content != nil && r.Execution != nil && r.State == "pending" {
		// Native adoption appends provenance instead of a human click. Both
		// possible completion shapes must fit before a proposal can be saved.
		native := r
		native.State = "applied"
		native.PolicyAdoption = &PolicyProvenance{Execution: *r.Execution, Identity: agent.ExecutionIdentity{OrganizationID: r.Org, ActorID: r.Owner, MemberID: actor},
			Mode: agent.ModeFullAccess, ModeRevision: "9223372036854775807", PolicyVersion: agent.ExecutionModePolicyVersion,
			ActionVersion: ContentPolicyAdoptionVersion, ProposalDigest: strings.Repeat("f", 64), RequiresHuman: true}
		native.Receipt = &Receipt{r.ID, r.Revision, 1<<63 - 1, "review:" + strings.Repeat("f", 64), actor, at}
		values = append(values, native, native.View())
	}
	for _, value := range values {
		raw, err := json.Marshal(value)
		if err != nil {
			return ErrInvalid
		}
		if len(raw) > MaxRecordBytes {
			return ErrTooLarge
		}
	}
	return nil
}
