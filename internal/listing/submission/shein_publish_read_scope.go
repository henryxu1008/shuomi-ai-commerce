package submission

import (
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

type SheinPublishReadScope struct {
	StoreID, ApplicationID, MerchantIdentityHash string
	ApplicationType                              storecenter.OfficialApplicationType
}

func (s SheinPublishReadScope) Valid() bool {
	return collection.ValidID(s.StoreID) && authidentity.IsBoundedIdentifier(s.ApplicationID) &&
		isExecutionDigest(s.MerchantIdentityHash) && s.ApplicationType.Valid()
}

func (s SheinPublishReadScope) MatchesBinding(b storecenter.ProductMerchantBinding) bool {
	return (b.Platform == "" || b.Platform == storecenter.PlatformShein) && b.Site == "shein-us" && b.ValidApplication() &&
		s.StoreID == b.StoreID && s.ApplicationID == b.ApplicationID && s.ApplicationType == b.ApplicationType &&
		s.MerchantIdentityHash == b.SupplierIdentityHash
}
