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

// AcceptanceWindow attests complete coverage of the exclusive key for this
// exact claim, with provider acceptance times normalized to the claim clock.
// SendBefore is the original effective send deadline, including caller limits.
type AcceptanceWindow struct {
	ClaimedAt, SendBefore time.Time
}

type UsageEvidence struct {
	Binding string
	Window  *AcceptanceWindow // required only when the request ID was lost
	Records []UsageRecord
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
		if err != nil || ctx.Err() != nil || evidence.Binding != binding {
			return dataacquisition.SHEINUSTrialUsage{}, dataacquisition.ErrUnknown
		}
		var sendBefore time.Time
		if attempt.RequestID == "" {
			window := evidence.Window
			if window == nil || !window.ClaimedAt.Equal(attempt.ClaimedAt) ||
				!window.SendBefore.After(attempt.ClaimedAt) || window.SendBefore.After(a.ExpiresAt) ||
				window.SendBefore.After(attempt.ClaimedAt.Add(dataacquisition.SHEINUSTrialTimeout)) {
				return dataacquisition.SHEINUSTrialUsage{}, dataacquisition.ErrUnknown
			}
			sendBefore = window.SendBefore
		}
		record, err := originalRecord(evidence.Records, a, req, attempt, sendBefore)
		if err != nil {
			return dataacquisition.SHEINUSTrialUsage{}, err
		}
		actual := *record.Credits
		return dataacquisition.SHEINUSTrialUsage{ApprovalID: a.ID, Binding: binding, RequestID: record.RequestID, Credits: &actual}, nil
	}, nil
}

func originalRecord(records []UsageRecord, a dataacquisition.SHEINUSTrialApproval, req dataacquisition.SHEINUSTrialRequest, attempt dataacquisition.SHEINUSTrialAttempt, sendBefore time.Time) (UsageRecord, error) {
	var match *UsageRecord
	for i := range records {
		r := &records[i]
		if (r.AccountRef != "" && r.AccountRef != a.AccountRef) || !couldBeOriginal(*r, a, req, attempt, sendBefore) {
			continue
		}
		if match != nil {
			return UsageRecord{}, dataacquisition.ErrUnknown
		}
		match = r
	}
	if match == nil || match.AccountRef != a.AccountRef || !hasOriginalDetails(*match, a, req, attempt) || !match.Final ||
		!dataacquisition.ValidSHEINUSTrialRequestID(match.RequestID) || match.Credits == nil || *match.Credits < 0 || *match.Credits > math.MaxInt32 {
		return UsageRecord{}, dataacquisition.ErrUnknown
	}
	return *match, nil
}

func couldBeOriginal(r UsageRecord, a dataacquisition.SHEINUSTrialApproval, req dataacquisition.SHEINUSTrialRequest, attempt dataacquisition.SHEINUSTrialAttempt, sendBefore time.Time) bool {
	if attempt.RequestID != "" {
		return r.RequestID == attempt.RequestID
	}
	// Missing details cannot rule out a second possible original.
	return (r.KeyRef == "" || r.KeyRef == a.KeyRef) && (r.URL == "" || r.URL == req.URL) &&
		(r.AcceptedAt.IsZero() || (!r.AcceptedAt.Before(attempt.ClaimedAt) && r.AcceptedAt.Before(sendBefore)))
}

func hasOriginalDetails(r UsageRecord, a dataacquisition.SHEINUSTrialApproval, req dataacquisition.SHEINUSTrialRequest, attempt dataacquisition.SHEINUSTrialAttempt) bool {
	if attempt.RequestID == "" {
		return r.KeyRef == a.KeyRef && r.URL == req.URL && !r.AcceptedAt.IsZero()
	}
	return (r.KeyRef == "" || r.KeyRef == a.KeyRef) && (r.URL == "" || r.URL == req.URL)
}
