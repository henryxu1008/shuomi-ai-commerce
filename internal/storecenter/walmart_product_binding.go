package storecenter

import "github.com/google/uuid"

type WalmartMerchantBinding struct {
	Environment  string `json:"environment"`
	Market       string `json:"market"`
	GrantID      string `json:"grantId"`
	Generation   int64  `json:"generation"`
	GrantVersion int64  `json:"grantVersion"`
}

func (b ProductMerchantBinding) ValidWalmartApplication() bool {
	return b.Platform == PlatformWalmart && b.Site == "walmart-us" && b.TikTok == (TikTokMerchantBinding{}) && b.ApplicationType == "" && b.IdentityPartition == "" && b.MerchantMode == "" && b.RegionID == 0 && b.ApplicationID != "" && b.ApplicationRevision != "" && b.Walmart.Market == "US" && (b.Walmart.Environment == "production" || b.Walmart.Environment == "sandbox") && uuid.Validate(b.Walmart.GrantID) == nil && b.Walmart.Generation > 0 && b.Walmart.GrantVersion >= b.Walmart.Generation && len(b.ScopesHash) == 64 && len(b.SupplierIdentityHash) == 64 && !b.CredentialExpiresAt.IsZero()
}
