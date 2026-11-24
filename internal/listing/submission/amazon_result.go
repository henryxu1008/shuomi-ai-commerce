package submission

import (
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
)

type AmazonIssue struct {
	Code       string   `json:"code"`
	Severity   string   `json:"severity"`
	Attributes []string `json:"attributes,omitempty"`
}

type AmazonReceipt struct {
	Accepted     bool          `json:"accepted"`
	SubmissionID string        `json:"submissionId"`
	Issues       []AmazonIssue `json:"issues,omitempty"`
}

type AmazonObservation struct {
	ASIN         string        `json:"asin"`
	Buyable      bool          `json:"buyable"`
	Discoverable bool          `json:"discoverable"`
	Issues       []AmazonIssue `json:"issues,omitempty"`
	ObservedAt   time.Time     `json:"observedAt"`
}

func (r AmazonReceipt) Valid() bool {
	return authidentity.IsBoundedIdentifier(r.SubmissionID) && validAmazonIssues(r.Issues)
}

func (o AmazonObservation) Valid() bool {
	return (o.ASIN == "" && !o.Buyable && !o.Discoverable || len(o.ASIN) == 10 && authidentity.IsBoundedIdentifier(o.ASIN)) &&
		!o.ObservedAt.IsZero() && validAmazonIssues(o.Issues)
}

func validAmazonIssues(issues []AmazonIssue) bool {
	if len(issues) > 50 {
		return false
	}
	for _, issue := range issues {
		if !authidentity.IsBoundedIdentifier(issue.Code) || len(issue.Attributes) > 20 ||
			(issue.Severity != "ERROR" && issue.Severity != "WARNING" && issue.Severity != "INFO") {
			return false
		}
		for _, attribute := range issue.Attributes {
			if !authidentity.IsBoundedIdentifier(attribute) {
				return false
			}
		}
	}
	return true
}

func AmazonReceiptEvidence(attempt ExecutionAttempt, accepted bool, at time.Time) ExecutionEvidence {
	outcome, reason := ExecutionSucceeded, ""
	if !accepted {
		outcome, reason = ExecutionFailedDefinitive, "AMAZON_LISTING_NOT_ACCEPTED"
	}
	return ExecutionEvidence{Kind: EvidenceProviderResponse, Outcome: outcome, Reference: attempt.AttemptID,
		Fingerprint: collection.Digest([]string{attempt.AttemptID, attempt.PayloadFingerprint, string(outcome)}),
		Reason:      reason, ObservedAt: at.UTC().Truncate(time.Microsecond)}
}
