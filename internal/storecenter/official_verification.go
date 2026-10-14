package storecenter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type OfficialIdentityPartition string
type OfficialAuthorizationMethod string
type OfficialMerchantMode string
type OfficialMerchantNamespace string
type OfficialModeSet uint8

const (
	PartitionCN     OfficialIdentityPartition = "CN"
	PartitionUS     OfficialIdentityPartition = "US"
	PartitionEU     OfficialIdentityPartition = "EU"
	PartitionGlobal OfficialIdentityPartition = "GLOBAL"

	AuthorizationCallback    OfficialAuthorizationMethod = "callback"
	AuthorizationManualToken OfficialAuthorizationMethod = "manual_token"

	MerchantLocal        OfficialMerchantMode = "local"
	MerchantSemiManaged  OfficialMerchantMode = "semi_managed"
	MerchantFullyManaged OfficialMerchantMode = "fully_managed"

	MerchantSupplierNamespace OfficialMerchantNamespace = "supplier"
	MerchantLocalNamespace    OfficialMerchantNamespace = "local"

	ModesLocal        OfficialModeSet = 1
	ModesSemiManaged  OfficialModeSet = 2
	ModesFullyManaged OfficialModeSet = 4
)

func (m OfficialModeSet) Supports(mode OfficialMerchantMode) bool {
	switch mode {
	case MerchantLocal:
		return m&ModesLocal != 0
	case MerchantSemiManaged:
		return m&ModesSemiManaged != 0
	case MerchantFullyManaged:
		return m&ModesFullyManaged != 0
	default:
		return false
	}
}

// This variant is private material, never a Store/Listing/Tool projection.
type TemuMerchantCredential struct {
	AccessToken string `json:"-"`
}

// Facts belong to Store, not to the provider adapter or a client-selected mode.
// RegionID is an official mall fact; it is not a site or a routing instruction.
type OfficialVerificationFacts struct {
	Platform          Platform                  `json:"platform"`
	IdentityPartition OfficialIdentityPartition `json:"identityPartition"`
	Namespace         OfficialMerchantNamespace `json:"namespace"`
	MerchantID        string                    `json:"merchantId"`
	Mode              OfficialMerchantMode      `json:"mode"`
	RegionID          int64                     `json:"regionId,omitempty"`
	ExpiresAt         time.Time                 `json:"expiresAt"`
	Scopes            []string                  `json:"scopes"`
	ContractRevision  string                    `json:"contractRevision"`
	VerifiedAt        time.Time                 `json:"verifiedAt"`
}

// Token, application, mode, region and site may change without releasing an
// official merchant's permanent ownership or its create-effect fence.
func (f OfficialVerificationFacts) MerchantIdentityHash() (string, error) {
	if f.Platform != PlatformTemu || f.MerchantID == "" || len(f.MerchantID) > 200 || !utf8.ValidString(f.MerchantID) || strings.TrimSpace(f.MerchantID) != f.MerchantID || strings.IndexFunc(f.MerchantID, unicode.IsControl) >= 0 {
		return "", ErrOfficialConnectionUnavailable
	}
	var identity []string
	switch f.Namespace {
	case MerchantSupplierNamespace:
		if f.IdentityPartition != PartitionCN || f.RegionID != 0 || f.Mode != MerchantFullyManaged && f.Mode != MerchantSemiManaged {
			return "", ErrOfficialConnectionUnavailable
		}
		identity = []string{string(f.Platform), string(f.Namespace), f.MerchantID}
	case MerchantLocalNamespace:
		if f.Mode != MerchantLocal || f.RegionID < 1 || f.IdentityPartition != PartitionUS && f.IdentityPartition != PartitionEU && f.IdentityPartition != PartitionGlobal {
			return "", ErrOfficialConnectionUnavailable
		}
		identity = []string{string(f.Platform), string(f.IdentityPartition), string(f.Namespace), f.MerchantID}
	default:
		return "", ErrOfficialConnectionUnavailable
	}
	encoded, _ := json.Marshal(identity)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}
