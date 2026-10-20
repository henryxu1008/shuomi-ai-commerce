package materials

import (
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/product/collection"
)

// VerifyCommit compares the Target owner's actual transient facts to the
// disclosed proposal. The caller must separately recheck current identity,
// exact run/hash, configuration, mode and explicit human confirmation.
func VerifyCommit(i Inspection, p Proposal, facts record.TargetCommitFacts) error {
	preview, err := Preview(i, p)
	if err != nil {
		return err
	}
	r := facts.Record
	if collection.Digest(r.Input) != collection.Digest(preview.Input) || r.TargetID != i.Base.TargetID || r.Revision != i.Base.Revision+1 ||
		r.Source != i.Base.Source || r.EffectiveVersion != i.Base.EffectiveVersion || r.ApplyReceiptID != i.Base.ApplyReceiptID || r.ProductHash != i.Base.ProductHash ||
		collection.Digest(r.Merchant) != collection.Digest(i.Base.Merchant) || r.InventoryHash != i.Base.InventoryHash || collection.Digest(facts.Inventory) != i.Base.InventoryHash || r.RulesHash != collection.Digest(facts.Rules) {
		return ErrInspectionChanged
	}
	if preview.Input.Draft.Product.CategoryID == i.Base.Input.Draft.Product.CategoryID {
		if r.RulesHash != i.Base.RulesHash {
			return ErrInspectionChanged
		}
	} else {
		// Category-specific fill/attribute/linked rules are freshly loaded by
		// Target. The disclosed category catalog and merchant facts stay exact.
		before, current := i.Rules, facts.Rules
		before.Fill, before.Attributes, before.Linked = current.Fill, current.Attributes, current.Linked
		if collection.Digest(before) != collection.Digest(current) {
			return ErrInspectionChanged
		}
	}
	return nil
}
