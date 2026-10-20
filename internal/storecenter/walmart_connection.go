package storecenter

import (
	"context"
	auth "task-processor/internal/storecenter/walmart"
	"time"
)

type WalmartConnectionAttempt struct {
	OrganizationID, AttemptID, StoreID, ActorID, MemberID          string
	AppID, AppRevision, Environment, CallbackURL, StateHash, Nonce string
	StoreVersion, ConnectionVersion, GrantGeneration, GrantVersion int64
	GrantID, SellerID, State, KeyID                                string
	Ciphertext                                                     string `json:"-"`
	ExpiresAt, CreatedAt, UpdatedAt                                time.Time
}

func (WalmartConnectionAttempt) TableName() string { return "workbench_store_walmart_oauth_attempts" }
func (a WalmartConnectionAttempt) CredentialBinding() auth.CredentialBinding {
	return auth.CredentialBinding{OrganizationID: a.OrganizationID, AppID: a.AppID, AppRevision: a.AppRevision, Environment: a.Environment, ID: a.AttemptID, Purpose: "attempt"}
}

type WalmartGrant struct {
	OrganizationID, ID, StoreID, AppID, AppRevision, Environment, MerchantIdentityHash, SellerID, MerchantName string
	Generation, Version                                                                                        int64
	Status, KeyID, ScopesHash                                                                                  string
	Ciphertext                                                                                                 string `json:"-"`
	AccessExpiresAt, RefreshExpiresAt                                                                          time.Time
	RefreshOperationID, RefreshInputHash, RefreshActorID, RefreshMemberID                                      string
	RefreshGeneration                                                                                          int64
	ClaimExpiresAt, CreatedAt, UpdatedAt                                                                       time.Time
}

func (WalmartGrant) TableName() string { return "workbench_store_walmart_oauth_grants" }
func (g WalmartGrant) CredentialBinding() auth.CredentialBinding {
	return auth.CredentialBinding{OrganizationID: g.OrganizationID, AppID: g.AppID, AppRevision: g.AppRevision, Environment: g.Environment, ID: g.ID, Purpose: "grant", Generation: g.Generation}
}
func (g WalmartGrant) CanRead(now time.Time) bool {
	return g.Status == "active" && g.Generation > 0 && now.Before(g.AccessExpiresAt) && now.Before(g.RefreshExpiresAt)
}

type WalmartConnectionDetails struct {
	Environment  string `json:"environment,omitempty"`
	Market       string `json:"market"`
	PartnerID    string `json:"partnerId,omitempty"`
	MerchantName string `json:"merchantName,omitempty"`
	GrantID      string `json:"grantId,omitempty"`
	Generation   int64  `json:"generation,omitempty"`
	GrantVersion int64  `json:"grantVersion,omitempty"`
}
type CompleteWalmartConnection struct{ OrganizationID, StoreID, AttemptID, AppID, State, Code, SellerID, CallbackType, MerchantConfirmation string }
type WalmartBindingIntent struct {
	Attempt WalmartConnectionAttempt
	Grant   WalmartGrant
}
type WalmartRefreshClaim struct {
	Grant    WalmartGrant
	Dispatch bool
}
type WalmartConnectionStore interface {
	Get(context.Context, string, string) (*Store, error)
	BeginWalmartConnection(context.Context, OfficialConnectionCommand, auth.Application, string, string, time.Time) (WalmartConnectionAttempt, error)
	ReadWalmartAttempt(context.Context, string, string, string) (WalmartConnectionAttempt, error)
	ClaimWalmartExchange(context.Context, CompleteWalmartConnection, string, time.Time) (WalmartConnectionAttempt, bool, error)
	SaveWalmartCandidate(context.Context, WalmartConnectionAttempt, string, string, time.Time) error
	FailWalmartExchange(context.Context, WalmartConnectionAttempt, bool, time.Time) error
	BindWalmartConnection(context.Context, WalmartBindingIntent, time.Time) (OfficialConnectionView, error)
	ReadWalmartGrant(context.Context, string, string) (WalmartGrant, error)
	ReadWalmartConnection(context.Context, string, string) (OfficialConnectionView, error)
	ClaimWalmartRefresh(context.Context, string, string, string, int64, string, time.Time) (WalmartRefreshClaim, error)
	CompleteWalmartRefresh(context.Context, WalmartRefreshClaim, WalmartGrant, time.Time) error
	UnknownWalmartRefresh(context.Context, WalmartRefreshClaim, time.Time) error
	DisconnectWalmartConnection(context.Context, OfficialConnectionCommand, time.Time) (OfficialConnectionView, error)
}
