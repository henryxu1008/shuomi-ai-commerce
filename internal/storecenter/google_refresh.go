package storecenter

import (
	"context"
	"errors"
	"time"

	google "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/googlemerchant"
)

const (
	GoogleAccessHeadroom      = 15 * time.Second
	GoogleRefreshSaveReserve  = 5 * time.Second
	GoogleRefreshClaimWindow  = 4 * time.Second
	GoogleRefreshPermitWindow = 15 * time.Second
)

var (
	ErrGoogleRefreshUnavailable = errors.New("Google Merchant authorization temporarily unavailable")
	ErrGoogleReauthorization    = errors.New("Google Merchant authorization must be renewed by a store administrator")
)

type GoogleCredentialRequest struct {
	OrganizationID, StoreID string
	Application             google.Application
	Target                  google.Target
	// Zero authorizes a read of the original target using the current generation.
	Generation int64
}

type GoogleCredentialSnapshot struct {
	Binding                               google.CredentialBinding
	Target                                google.Target
	ActorID, MemberID, ConnectionRef      string
	Administrator                         bool
	MemberVersion, StoreVersion, Revision int64
}

func (s GoogleCredentialSnapshot) SameAuthorization(other GoogleCredentialSnapshot) bool {
	other.Revision = s.Revision
	return s == other
}

type GoogleRefreshPhase string

const (
	GoogleRefreshIdle        GoogleRefreshPhase = "idle"
	GoogleRefreshClaimed     GoogleRefreshPhase = "claimed"
	GoogleRefreshPossible    GoogleRefreshPhase = "possible"
	GoogleRefreshReauthorize GoogleRefreshPhase = "reauthorize"
)

type GoogleCredentialState struct {
	Snapshot GoogleCredentialSnapshot
	Grant    GoogleGrant
	Phase    GoogleRefreshPhase
	ClaimID  string
	Deadline time.Time
	Usable   bool
}

type GoogleRefreshOutcome struct {
	// A nonempty credential replaces the whole protected grant.
	KeyID, Ciphertext string
	AccessExpiresAt   time.Time
	// Reauthorize applies only to a confirmed rejection or invalid stored refresh token.
	Reauthorize bool
	// A confirmed no-exchange outcome returns the claim with this cooldown.
	Cooldown time.Duration
}

type GoogleRefreshRepository interface {
	GoogleCredential(ctx context.Context, request GoogleCredentialRequest, claimID string) (GoogleCredentialState, error)
	AuthorizeGoogleRefresh(ctx context.Context, request GoogleCredentialRequest, own GoogleCredentialState) (time.Time, error)
	FinishGoogleRefresh(ctx context.Context, request GoogleCredentialRequest, own GoogleCredentialState, result GoogleRefreshOutcome) error
}
