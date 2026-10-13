package storecenter

import (
	"context"
	"errors"
	"time"
)

var (
	ErrOfficialConnectionUnavailable = errors.New("official store connection is not configured or available")
	ErrOfficialExchangeUnknown       = errors.New("official credential exchange outcome is unknown")
	ErrOfficialAuthorizationRejected = errors.New("official authorization was rejected")
	ErrOfficialCredentialExpired     = errors.New("official merchant credential is no longer valid")
)

type OfficialApplication struct {
	AppID, Version, CallbackURL string
	Platform                    Platform
	IdentityPartition           OfficialIdentityPartition
	AuthorizationMethod         OfficialAuthorizationMethod
	SupportedModes              OfficialModeSet
}
type OfficialApplicationType string
type OfficialApplicationChoice struct {
	Platform            Platform                    `json:"platform"`
	IdentityPartition   OfficialIdentityPartition   `json:"identityPartition,omitempty"`
	AuthorizationMethod OfficialAuthorizationMethod `json:"authorizationMethod"`
	SupportedModes      OfficialModeSet             `json:"supportedModes,omitempty"`
	AppID               string                      `json:"appId"`
	Revision            string                      `json:"revision"`
	Type                OfficialApplicationType     `json:"type,omitempty"`
}

const (
	ApplicationSelfOperated OfficialApplicationType = "self_operated"
	ApplicationSemiManaged  OfficialApplicationType = "semi_managed"
	ApplicationFullyManaged OfficialApplicationType = "fully_managed"
)

func (t OfficialApplicationType) Valid() bool {
	return t == ApplicationSelfOperated || t == ApplicationSemiManaged || t == ApplicationFullyManaged
}

type OfficialMerchantCredential struct {
	AppID      string
	OpenKeyID  string
	SecretKey  string `json:"-"`
	SupplierID string
	Temu       *TemuMerchantCredential `json:"-"`
}
type OfficialStoreInformation struct {
	SupplierID   *string
	StoreName    *string
	StoreStatus  *int
	Verification *OfficialVerificationFacts
}
type OfficialConnectionProvider interface {
	Application() OfficialApplication
	AuthorizationURL(string) (string, error)
	Exchange(context.Context, string, string) (OfficialMerchantCredential, error)
	QueryStore(context.Context, OfficialMerchantCredential) (OfficialStoreInformation, error)
}

// Protection is provider-specific encryption at rest. Its associated data must
// bind the immutable attempt; provider wire decryption belongs to the adapter.
type OfficialCredentialProtection interface {
	Seal(OfficialConnectionAttempt, OfficialMerchantCredential) (keyID, ciphertext string, err error)
	Open(OfficialConnectionAttempt, string, string) (OfficialMerchantCredential, error)
}
type OfficialConnectionAttempt struct {
	OrganizationID, StoreID, AttemptID, ActorID, MemberID string
	AppID, AppVersion, StateHash                          string
	StoreVersion, ConnectionVersion                       int64
	ExpiresAt                                             time.Time
	State, KeyID, Ciphertext                              string
	Platform                                              Platform
	IdentityPartition                                     OfficialIdentityPartition
	AuthorizationMethod                                   OfficialAuthorizationMethod
	PrivateInputDigest, VerificationJSON                  string
}
type OfficialConnectionCommand struct {
	OrganizationID, StoreID, AttemptID string
	ExpectedStoreVersion               int64
	ApplicationID                      string
	IdentityPartition                  OfficialIdentityPartition
}
type InstallOfficialToken struct {
	OrganizationID, StoreID, AttemptID string
	AccessToken                        string `json:"-"`
}
type OfficialConnectionBegin struct {
	AttemptID           string                      `json:"attemptId"`
	AuthorizationURL    string                      `json:"authorizationUrl"`
	ExpiresAt           time.Time                   `json:"expiresAt"`
	AuthorizationMethod OfficialAuthorizationMethod `json:"authorizationMethod"`
}
type CompleteOfficialConnection struct{ OrganizationID, StoreID, AttemptID, AppID, State, TempToken string }
type OfficialConnectionView struct {
	Platform             Platform                    `json:"platform"`
	IdentityPartition    OfficialIdentityPartition   `json:"identityPartition,omitempty"`
	AuthorizationMethod  OfficialAuthorizationMethod `json:"authorizationMethod,omitempty"`
	Mode                 OfficialMerchantMode        `json:"mode,omitempty"`
	MerchantIdentityHash string                      `json:"merchantIdentityHash,omitempty"`
	RegionID             int64                       `json:"regionId,omitempty"`
	CredentialExpiresAt  *time.Time                  `json:"credentialExpiresAt,omitempty"`
	AppID                string                      `json:"appId,omitempty"`
	AppRevision          string                      `json:"appRevision,omitempty"`
	AttemptID            string                      `json:"attemptId"`
	State                string                      `json:"state"`
	Status               ConnectionStatus            `json:"connectionStatus"`
	Version              int64                       `json:"version"`
	ObservedAt           *time.Time                  `json:"observedAt"`
}
type OfficialConnectionStore interface {
	ReadOfficialInstallAttempt(context.Context, string, string, string) (OfficialConnectionAttempt, error)
	InstallOfficialPrivateCredential(context.Context, OfficialConnectionAttempt, string, string, string, time.Time) (OfficialConnectionAttempt, error)
	ReadOfficialAttemptBinding(context.Context, string, string, string) (OfficialAttemptBinding, error)
	BeginOfficialConnection(context.Context, OfficialConnectionCommand, OfficialApplication, string, time.Time) (OfficialConnectionAttempt, error)
	ClaimOfficialExchange(context.Context, string, string, string, string, string, string, time.Time) (OfficialConnectionAttempt, bool, error)
	ReadOfficialQueryAttempt(context.Context, string, string, string) (OfficialConnectionAttempt, error)
	SaveOfficialCredential(context.Context, OfficialConnectionAttempt, string, string, string) error
	CompleteOfficialConnection(context.Context, OfficialConnectionAttempt, ConnectionStatus, OfficialStoreInformation, time.Time) (OfficialConnectionView, error)
	ReadOfficialConnection(context.Context, string, string) (OfficialConnectionView, error)
	ReadOfficialCredential(context.Context, ConnectionStatusInput) (OfficialConnectionAttempt, OfficialConnectionView, error)
	ObserveOfficialConnection(context.Context, OfficialConnectionAttempt, ConnectionStatus, OfficialStoreInformation, time.Time) error
	DisconnectOfficialConnection(context.Context, OfficialConnectionCommand, time.Time) (OfficialConnectionView, error)
}

// The member-authorized binding contains no credential and permits no dispatch.
type OfficialAttemptBinding struct {
	AppID, AppVersion string
	Platform          Platform
	IdentityPartition OfficialIdentityPartition
}
