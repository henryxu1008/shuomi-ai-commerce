package zenrows

import (
	"context"
	"math"
	"time"

	"task-processor/internal/product/dataacquisition"
)

// UsageRecord represents a per-request entry from trusted account evidence.
// AcceptedAt is provider acceptance time, normalized to the approval clock.
type UsageRecord struct {
	AccountRef, KeyRef, URL, RequestID string
	AcceptedAt                         time.Time
	Final                              bool
	Credits                            *int
}

type UsageEvidence struct {
	Binding string
	// AcceptanceWindowVerified attests that Records covers the dedicated key's exclusive
	// approval window, including verified acceptance timestamps.
	AcceptanceWindowVerified bool
	Records                  []UsageRecord
}

// ReadUsageEvidence is a trusted, read-only evidence reader. Its caller has
// already checked reconciliation authority. It does not submit Fetch requests.
type ReadUsageEvidence func(context.Context, dataacquisition.SHEINUSTrialApproval, dataacquisition.SHEINUSTrialAttempt) (UsageEvidence, error)

func NewLookup(read ReadUsageEvidence) (dataacquisition.SHEINUSTrialLookup, error) {
	if read == nil {
		return nil, dataacquisition.ErrUnavailable
	}
	return func(ctx context.Context, a dataacquisition.SHEINUSTrialApproval, req dataacquisition.SHEINUSTrialRequest, attempt dataacquisition.SHEINUSTrialAttempt) (dataacquisition.SHEINUSTrialUsage, error) {
		binding := a.Binding()
		if !req.Valid() || attempt.ApprovalID != a.ID || attempt.Binding != binding ||
			attempt.ClaimedAt.IsZero() || !attempt.ClaimedAt.Before(a.ExpiresAt) || ctx.Err() != nil {
			return dataacquisition.SHEINUSTrialUsage{}, dataacquisition.ErrUnknown
		}
		evidence, err := read(ctx, a, attempt)
		if err != nil || ctx.Err() != nil || evidence.Binding != binding ||
			(attempt.RequestID == "" && !evidence.AcceptanceWindowVerified) {
			return dataacquisition.SHEINUSTrialUsage{}, dataacquisition.ErrUnknown
		}
		record, err := originalRecord(evidence.Records, a, req, attempt)
		if err != nil {
			return dataacquisition.SHEINUSTrialUsage{}, err
		}
		actual := *record.Credits
		return dataacquisition.SHEINUSTrialUsage{ApprovalID: a.ID, Binding: binding, RequestID: record.RequestID, Credits: &actual}, nil
	}, nil
}

func originalRecord(records []UsageRecord, a dataacquisition.SHEINUSTrialApproval, req dataacquisition.SHEINUSTrialRequest, attempt dataacquisition.SHEINUSTrialAttempt) (UsageRecord, error) {
	var match *UsageRecord
	for i := range records {
		r := &records[i]
		if r.AccountRef != a.AccountRef || !matchesOriginal(*r, a, req, attempt) {
			continue
		}
		if match != nil {
			return UsageRecord{}, dataacquisition.ErrUnknown
		}
		match = r
	}
	if match == nil || !matchesOriginalDetails(*match, a, req, attempt) || !match.Final || !dataacquisition.ValidSHEINUSTrialRequestID(match.RequestID) ||
		match.Credits == nil || *match.Credits < 0 || *match.Credits > math.MaxInt32 {
		return UsageRecord{}, dataacquisition.ErrUnknown
	}
	return *match, nil
}

func matchesOriginal(r UsageRecord, a dataacquisition.SHEINUSTrialApproval, req dataacquisition.SHEINUSTrialRequest, attempt dataacquisition.SHEINUSTrialAttempt) bool {
	if attempt.RequestID != "" {
		return r.RequestID == attempt.RequestID
	}
	return r.KeyRef == a.KeyRef && r.URL == req.URL &&
		(r.AcceptedAt.IsZero() || (!r.AcceptedAt.Before(attempt.ClaimedAt) && r.AcceptedAt.Before(a.ExpiresAt)))
}

func matchesOriginalDetails(r UsageRecord, a dataacquisition.SHEINUSTrialApproval, req dataacquisition.SHEINUSTrialRequest, attempt dataacquisition.SHEINUSTrialAttempt) bool {
	if attempt.RequestID == "" {
		return !r.AcceptedAt.IsZero()
	}
	return (r.KeyRef == "" || r.KeyRef == a.KeyRef) && (r.URL == "" || r.URL == req.URL)
}
