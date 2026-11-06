package collection

// ListingTransferKind is the current Preparation source admission, not a grant.
func ListingTransferKind(kind string) bool {
	switch kind {
	case "acquisition", "own", "market", "sds_finished", "amazon_data":
		return true
	}
	return false
}
