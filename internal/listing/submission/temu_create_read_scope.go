package submission

import (
	"task-processor/internal/authidentity"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

type TemuCreateReadScope struct {
	StoreID, ApplicationID, MerchantIdentityHash, Site string
	IdentityPartition                                  storecenter.OfficialIdentityPartition
	MerchantMode                                       storecenter.OfficialMerchantMode
	RegionID                                           int64
}

func (s TemuCreateReadScope) Valid() bool {
	if !collection.ValidID(s.StoreID) || !authidentity.IsBoundedIdentifier(s.ApplicationID) || !isExecutionDigest(s.MerchantIdentityHash) {
		return false
	}
	if s.IdentityPartition == storecenter.PartitionCN {
		return s.Site == "temu-supplier" && s.RegionID == 0 &&
			(s.MerchantMode == storecenter.MerchantFullyManaged || s.MerchantMode == storecenter.MerchantSemiManaged)
	}
	return s.Site == "temu-local" && s.MerchantMode == storecenter.MerchantLocal && s.RegionID > 0 &&
		(s.IdentityPartition == storecenter.PartitionUS || s.IdentityPartition == storecenter.PartitionEU || s.IdentityPartition == storecenter.PartitionGlobal)
}

func (s TemuCreateReadScope) MatchesBinding(b storecenter.ProductMerchantBinding) bool {
	return b.ValidTemuApplication() && s.StoreID == b.StoreID && s.ApplicationID == b.ApplicationID &&
		s.MerchantIdentityHash == b.SupplierIdentityHash && s.Site == b.Site && s.IdentityPartition == b.IdentityPartition &&
		s.MerchantMode == b.MerchantMode && s.RegionID == b.RegionID
}
