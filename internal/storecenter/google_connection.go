package storecenter

import (
	"context"
	"time"

	auth "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/googlemerchant"
	market "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/googlemerchant"
)

type GoogleAttempt struct {
	OrganizationID, StoreID, AttemptID, ActorID, MemberID string
	AppID, AppRevision, StateHash, State, KeyID, GrantID  string
	Ciphertext                                            string `json:"-"`
	StoreVersion, Generation                              int64
	ExpiresAt, CreatedAt, UpdatedAt                       time.Time
}

func (GoogleAttempt) TableName() string { return "workbench_store_google_oauth_attempts" }

func (a GoogleAttempt) CredentialBinding() auth.CredentialBinding {
	return auth.CredentialBinding{OrganizationID: a.OrganizationID, StoreID: a.StoreID,
		ApplicationID: a.AppID, ApplicationRevision: a.AppRevision, ID: a.AttemptID, Purpose: auth.AttemptCredentials}
}

type GoogleGrant struct {
	OrganizationID, StoreID, ID, AttemptID, AppID, AppRevision string
	Generation                                                 int64
	Target                                                     market.Target `gorm:"embedded"`
	KeyID                                                      string
	Ciphertext                                                 string `json:"-"`
	Revoked                                                    bool
	RefreshUnavailable                                         bool `gorm:"-" json:"-"`
	AccessExpiresAt, CreatedAt, UpdatedAt                      time.Time
}

func (GoogleGrant) TableName() string { return "workbench_store_google_oauth_grants" }

func (g GoogleGrant) CredentialBinding() auth.CredentialBinding {
	return auth.CredentialBinding{OrganizationID: g.OrganizationID, StoreID: g.StoreID,
		ApplicationID: g.AppID, ApplicationRevision: g.AppRevision, ID: g.ID, Purpose: auth.GrantCredentials, Generation: g.Generation}
}

type GoogleConnectionRepository interface {
	BeginGoogleConnection(context.Context, OfficialConnectionCommand, auth.Application, string, time.Time) (GoogleAttempt, error)
	ClaimGoogleExchange(context.Context, string, string, string, string, auth.Application, time.Time) (GoogleAttempt, error)
	SaveGoogleCandidate(context.Context, GoogleAttempt, string, string, time.Time) error
	ReadGoogleCandidate(context.Context, string, string, string, time.Time) (GoogleAttempt, error)
	BindGoogleConnection(context.Context, GoogleAttempt, GoogleGrant, time.Time) (GoogleGrant, error)
	ReadGoogleGrant(context.Context, string, string, bool) (GoogleGrant, error)
	DisconnectGoogleConnection(context.Context, OfficialConnectionCommand, time.Time) error
}

type GoogleConnectionView struct {
	Status          ConnectionStatus `json:"status"`
	AccountID       string           `json:"accountId,omitempty"`
	DataSourceID    string           `json:"dataSourceId,omitempty"`
	ContentLanguage string           `json:"contentLanguage,omitempty"`
	FeedLabel       string           `json:"feedLabel,omitempty"`
}

type GoogleAccountSelection struct {
	AccountID, DataSourceID, ContentLanguage, FeedLabel string
	DedicatedSourceConfirmed                            bool
}
