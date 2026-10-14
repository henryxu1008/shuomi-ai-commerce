package storecenter

import (
	"context"
	"time"

	tiktok "task-processor/internal/storecenter/tiktokshop"
)

// TikTok has a seller-wide OAuth grant. Its credential is never represented by
// the SHEIN/TEMU OfficialMerchantCredential or duplicated per Store attempt.
type TikTokConnectionAttempt struct {
	OrganizationID, AttemptID, StoreID, ActorID, MemberID string
	AppKey, AppRevision, StateHash                        string
	StoreVersion, ConnectionVersion                       int64
	GrantID                                               string
	GrantGeneration, GrantRevision                        int64
	SellerIdentityHash, State, KeyID                      string
	Ciphertext                                            string `json:"-"`
	ExpiresAt, CreatedAt, UpdatedAt                       time.Time
}

func (a TikTokConnectionAttempt) CredentialBinding() tiktok.CredentialBinding {
	return tiktok.CredentialBinding{OrganizationID: a.OrganizationID, AppKey: a.AppKey, AppRevision: a.AppRevision, ID: a.AttemptID, Purpose: "attempt"}
}

type TikTokShopBinding struct {
	OrganizationID, StoreID, GrantID, AppKey string
	Shop                                     tiktok.Shop
	Status                                   string
	Version                                  int64
	CreatedAt, UpdatedAt                     time.Time
}

type TikTokConnectionDetails struct {
	Market          string `json:"market"`
	SellerType      string `json:"sellerType"`
	ShopID          string `json:"shopId,omitempty"`
	GrantGeneration int64  `json:"grantGeneration,omitempty"`
	GrantRevision   int64  `json:"grantRevision,omitempty"`
	BindingRevision int64  `json:"bindingRevision,omitempty"`
}

// The binding intent is built after authenticated provider Shop verification.
// The CAS and ciphertext namespace refer to the exact canonical generation.
type TikTokBindingIntent struct {
	Attempt               TikTokConnectionAttempt
	Shop                  tiktok.Shop
	Grant                 tiktok.GrantSnapshot
	ExpectedGrantRevision int64
	ReplaceCredential     bool
}

type CompleteTikTokConnection struct {
	OrganizationID, StoreID, AttemptID, AppKey, State, AuthCode string
}

type TikTokRefreshClaim struct {
	Grant    tiktok.GrantSnapshot
	Fence    tiktok.GrantFence
	Binding  TikTokShopBinding
	Dispatch bool
}

// Every member-facing operation uses the current Store authorizer. Callback
// writes additionally retain the original actor/member and connection fence.
type TikTokConnectionStore interface {
	Get(context.Context, string, string) (*Store, error)
	BeginTikTokConnection(context.Context, OfficialConnectionCommand, tiktok.Application, string, time.Time) (TikTokConnectionAttempt, error)
	ClaimTikTokExchange(context.Context, CompleteTikTokConnection, string, time.Time) (TikTokConnectionAttempt, bool, error)
	ReadTikTokAttempt(context.Context, string, string, string) (TikTokConnectionAttempt, error)
	SaveTikTokCandidate(context.Context, TikTokConnectionAttempt, string, string, string, time.Time) error
	FailTikTokAttempt(context.Context, TikTokConnectionAttempt, bool, time.Time) error
	ReadTikTokSellerGrant(context.Context, TikTokConnectionAttempt, string) (tiktok.GrantSnapshot, bool, error)
	BindTikTokConnection(context.Context, TikTokBindingIntent, time.Time) (OfficialConnectionView, error)
	ReadTikTokConnection(context.Context, string, string) (OfficialConnectionView, error)
	ReadTikTokGrant(context.Context, string, string) (tiktok.GrantSnapshot, TikTokShopBinding, error)
	ClaimTikTokRefresh(context.Context, string, string, string, int64, string, time.Time) (TikTokRefreshClaim, error)
	CompleteTikTokRefresh(context.Context, TikTokRefreshClaim, tiktok.GrantSnapshot, time.Time) error
	UnknownTikTokRefresh(context.Context, TikTokRefreshClaim, time.Time) error
	DisconnectTikTokConnection(context.Context, OfficialConnectionCommand, time.Time) (OfficialConnectionView, error)
}

// Lifecycle events have provider signature authority, not user/org input.
type TikTokLifecycleStore interface {
	ApplyTikTokDeauthorization(context.Context, tiktok.Deauthorization, time.Time) error
}
