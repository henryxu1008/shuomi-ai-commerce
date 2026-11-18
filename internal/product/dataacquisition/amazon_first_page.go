package dataacquisition

import (
	"context"
	"encoding/hex"
	"net/url"
	"strings"
	"time"

	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/product/collection"
)

const (
	AmazonZenRowsSearchVersion    = "amazon-zenrows-search-v2"
	amazonZenRowsSearchV1         = "amazon-zenrows-search-v1"
	AmazonFirstPageLimit          = 5
	AmazonFirstPageRequestCredits = 25
	AmazonFirstPageFetchTimeout   = 180 * time.Second
)

type AmazonFirstPageApproval struct {
	ID               string                      `json:"id"`
	Scope            collection.Scope            `json:"scope"`
	ProjectID        string                      `json:"projectId"`
	Keyword          string                      `json:"keyword"`
	AccountRef       string                      `json:"accountRef"`
	KeyDigest        string                      `json:"keyDigest"`
	BudgetRef        string                      `json:"budgetRef"`
	BudgetExpiresAt  time.Time                   `json:"budgetExpiresAt"`
	ExpiresAt        time.Time                   `json:"expiresAt"`
	MaximumCredits   int                         `json:"maximumCredits"`
	AvailableCredits int                         `json:"availableCredits"`
	NoOverageCharges bool                        `json:"noOverageCharges"`
	Funding          orgresource.ResourceFunding `json:"funding"`
	Price            PriceConfirmation           `json:"price"`
}

func (a AmazonFirstPageApproval) Query() Query {
	return Query{Site: "us", Mode: "keyword", Keyword: a.Keyword, Limit: AmazonFirstPageLimit}
}

func (a AmazonFirstPageApproval) TargetURL() string {
	return "https://www.amazon.com/s?" + url.Values{"k": {a.Keyword}, "page": {"1"}}.Encode()
}

func (a AmazonFirstPageApproval) Binding() string { return collection.Digest(a) }

func (a AmazonFirstPageApproval) Validate() error {
	query, err := NormalizeQuery(a.Query())
	hash, hashErr := hex.DecodeString(a.KeyDigest)
	if err != nil || query.Keyword != a.Keyword || !collection.ValidID(a.ID) ||
		a.Scope.Validate() != nil || !collection.ValidID(a.ProjectID) ||
		!collection.ValidID(a.AccountRef) || !collection.ValidID(a.BudgetRef) ||
		hashErr != nil || len(hash) != 32 || a.KeyDigest != strings.ToLower(a.KeyDigest) ||
		a.ExpiresAt.IsZero() || a.BudgetExpiresAt.IsZero() ||
		a.MaximumCredits < AmazonFirstPageRequestCredits || a.AvailableCredits < AmazonFirstPageRequestCredits ||
		!a.NoOverageCharges || a.Price.Validate() != nil ||
		(a.Funding != orgresource.FundingEnterprise && a.Funding != orgresource.FundingMember) {
		return ErrInvalid
	}
	return nil
}

func (a AmazonFirstPageApproval) SendBefore() time.Time {
	if a.BudgetExpiresAt.Before(a.ExpiresAt) {
		return a.BudgetExpiresAt
	}
	return a.ExpiresAt
}

type AmazonFirstPageState string

const (
	AmazonFirstPageUnknown      AmazonFirstPageState = "UNKNOWN"
	AmazonFirstPageRejected     AmazonFirstPageState = "REJECTED"
	AmazonFirstPageOverLimit    AmazonFirstPageState = "OVER_LIMIT"
	AmazonFirstPageInsufficient AmazonFirstPageState = "INSUFFICIENT"
	AmazonFirstPagePrepared     AmazonFirstPageState = "PREPARED"
)

type AmazonFirstPageReceipt struct {
	ApprovalID string               `json:"approvalId"`
	State      AmazonFirstPageState `json:"state"`
	RequestID  string               `json:"requestId,omitempty"`
	Credits    *int                 `json:"credits"`
	JobID      string               `json:"jobId,omitempty"`
}

func (r AmazonFirstPageReceipt) Validate() error {
	switch r.State {
	case AmazonFirstPageUnknown, AmazonFirstPageRejected, AmazonFirstPageOverLimit, AmazonFirstPageInsufficient, AmazonFirstPagePrepared:
	default:
		return ErrInvalid
	}
	if !collection.ValidID(r.ApprovalID) || (r.State == AmazonFirstPagePrepared) != collection.ValidID(r.JobID) ||
		(r.State != AmazonFirstPagePrepared && r.JobID != "") ||
		(r.RequestID != "" && !ValidSHEINUSTrialRequestID(r.RequestID)) ||
		(r.Credits != nil && (r.RequestID == "" || *r.Credits < 0 || *r.Credits > 1<<31-1)) ||
		(r.State == AmazonFirstPageOverLimit && (r.Credits == nil || *r.Credits <= AmazonFirstPageRequestCredits)) ||
		(r.State == AmazonFirstPagePrepared && (r.RequestID == "" || (r.Credits != nil && *r.Credits > AmazonFirstPageRequestCredits))) {
		return ErrInvalid
	}
	return nil
}

type AmazonFirstPageSubmission struct {
	Receipt  AmazonFirstPageReceipt
	Evidence []Evidence
}

type AmazonFirstPageSender func(context.Context, AmazonFirstPageApproval) AmazonFirstPageSubmission
