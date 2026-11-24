package storecenter

import (
	"context"
	"encoding/json"
	"time"
)

// AmazonBrowserScope is supplied by the authenticated workbench server. The
// digest identifies the login session, not a value from browser input.
type AmazonBrowserScope struct {
	OrganizationID string
	SessionDigest  [32]byte
}

type AmazonAuthorizationApplication struct {
	ID, Revision, RedirectURI string
	Beta                      bool
}

type AmazonAuthorizationBegin struct {
	Scope                AmazonBrowserScope
	Application          AmazonAuthorizationApplication
	StoreID, AttemptID   string
	FlowDigest           string
	ExpectedStoreVersion int64
	Now, ExpiresAt       time.Time
}

type AmazonAuthorizationStep struct {
	Scope                             AmazonBrowserScope
	Application                       AmazonAuthorizationApplication
	FlowDigest, StateDigest, SellerID string
	Now                               time.Time
}

type AmazonAuthorizationClaimCommand struct {
	AmazonAuthorizationStep
	CodeDigest string
}

type AmazonAuthorizationClaim struct {
	Scope                       AmazonBrowserScope
	Application                 AmazonAuthorizationApplication
	StoreID, AttemptID          string
	ActorID, MemberID, SellerID string
	Generation                  int64
	Replay                      bool
}

func (c AmazonAuthorizationClaim) CredentialAAD() []byte {
	// AAD is reconstructed by Store when the bound credential is accessed.
	// No user or provider values are sent to the browser from this context.
	key, _ := json.Marshal([]any{
		"storecenter/amazon/refresh-token/v1", c.Scope.OrganizationID, c.StoreID, c.AttemptID,
		c.Application.ID, c.Application.Revision, c.Application.Beta, c.Generation, c.SellerID,
	})
	return key
}

type AmazonAuthorizationStatus string

const (
	AmazonAuthorizationStatusNotConnected AmazonAuthorizationStatus = "not_connected"
	AmazonAuthorizationStatusSaved        AmazonAuthorizationStatus = "authorization_saved"
	AmazonAuthorizationStatusReconnect    AmazonAuthorizationStatus = "reconnect_required"
)

type AmazonAuthorizationStore interface {
	ReadAmazonAuthorizationStatus(context.Context, string, string, AmazonAuthorizationApplication, time.Time) (AmazonAuthorizationStatus, error)
	// Each transition verifies the current member's Store write grant. Start
	// replaces the pending flow for the same browser session. Advance claims
	// the first callback and Claim consumes the second state in transactions.
	// Claim binds the code digest on first dispatch and matches it on replay.
	StartAmazonAuthorization(context.Context, AmazonAuthorizationBegin) error
	AdvanceAmazonAuthorization(context.Context, AmazonAuthorizationStep) error
	ClaimAmazonAuthorization(context.Context, AmazonAuthorizationClaimCommand) (AmazonAuthorizationClaim, error)
	// Bind rechecks the original claim's current version and the seller's
	// unique binding before committing the encrypted credential and receipt.
	BindAmazonAuthorization(context.Context, AmazonAuthorizationClaim, string, string, time.Time) error
}

type AmazonConnectionStart struct {
	AuthorizationURL string    `json:"authorizationUrl"`
	FlowCookie       string    `json:"-"`
	ExpiresAt        time.Time `json:"expiresAt"`
}

type AmazonLoginCallback struct {
	FlowCookie, AmazonCallbackURI, AmazonState, SellerID, Version string
}

type AmazonRedirectCallback struct {
	FlowCookie, State, SellerID, Code string
}

type AmazonConnectionResult struct {
	StoreID    string `json:"storeId"`
	Generation int64  `json:"generation"`
}
