package collection

// SupplyTransferKind is the current Preparation source admission, not a grant.
func SupplyTransferKind(kind string) bool {
	switch kind {
	case "acquisition", "own", "market", "sds_finished", "amazon_data":
		return true
	}
	return false
}
