package agent

import (
	"encoding/hex"
	"strconv"
)

const CandidateRulesContextKind = "listing-candidate-rules-v1"

// CandidateRulesRef is minted by the authorized Listing reader. It freezes a
// read-only selection, not a category change or permission to access a Store.
type CandidateRulesRef struct {
	BaseRevision, BaseDigest, StoreID, Site, MerchantDigest string
	CandidateCategoryID, SelectorDigest                     string
}

func (r CandidateRulesRef) Valid() bool {
	decimal := func(s string) bool {
		n, err := strconv.ParseUint(s, 10, 63)
		return err == nil && n > 0 && strconv.FormatUint(n, 10) == s
	}
	digest := func(s string) bool {
		v, err := hex.DecodeString(s)
		return err == nil && len(v) == 32 && hex.EncodeToString(v) == s
	}
	return decimal(r.BaseRevision) && decimal(r.CandidateCategoryID) && selectionUUID(r.StoreID) && ValidID(r.Site) && digest(r.BaseDigest) && digest(r.MerchantDigest) && digest(r.SelectorDigest)
}

func (b Binding) validCandidateRules() bool {
	if !selectionUUID(b.ContextID) || b.TargetPlatform != "shein" || !b.CandidateRules.Valid() {
		return false
	}
	base := b
	base.ContextKind, base.CandidateRules = "listing-target-record", CandidateRulesRef{}
	return base.Valid()
}
