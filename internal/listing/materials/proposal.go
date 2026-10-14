package materials

import (
	"encoding/json"
	"errors"
	"task-processor/internal/agent"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/shein/goods"
	model "task-processor/internal/marketplace/shein/model"

	sigjson "sigs.k8s.io/json"
)

var ErrInvalidProposal = errors.New("invalid platform material proposal")

// Proposal is an explicit whitelist, not a full official product DTO. SKU
// identities/commercial data and canonical text have no writable field here.
type Proposal struct {
	CategoryID  *int64                     `json:"categoryId,omitempty"`
	BrandCode   *string                    `json:"brandCode,omitempty"`
	Attributes  *[]model.AttributeValue    `json:"attributes,omitempty"`
	SKCs        []SKCProposal              `json:"skcs,omitempty"`
	Images      *[]goods.OfficialImageSlot `json:"images,omitempty"`
	EvidenceIDs []string                   `json:"evidenceIds"`
}

type SKCProposal struct {
	Index         *int                  `json:"index"`
	SaleAttribute *model.AttributeValue `json:"saleAttribute,omitempty"`
	SKUs          []SKUProposal         `json:"skus,omitempty"`
}

type SKUProposal struct {
	Index          *int                    `json:"index"`
	SaleAttributes *[]model.AttributeValue `json:"saleAttributes,omitempty"`
	Attributes     *[]model.AttributeValue `json:"attributes,omitempty"`
}

func ParseProposal(raw json.RawMessage) (Proposal, error) {
	var p Proposal
	if len(raw) == 0 || len(raw) > agent.MaxModelOutputBytes {
		return p, ErrInvalidProposal
	}
	strict, err := sigjson.UnmarshalStrict(raw, &p)
	if err != nil || len(strict) != 0 || validateProposal(p) != nil {
		return Proposal{}, ErrInvalidProposal
	}
	return p, nil
}

func validateProposal(p Proposal) error {
	if p.CategoryID == nil && p.BrandCode == nil && p.Attributes == nil && p.Images == nil && len(p.SKCs) == 0 ||
		p.CategoryID != nil && *p.CategoryID <= 0 || p.BrandCode != nil && !agent.ValidID(*p.BrandCode) || len(p.EvidenceIDs) == 0 || len(p.EvidenceIDs) > 64 {
		return ErrInvalidProposal
	}
	evidence := make(map[string]bool, len(p.EvidenceIDs))
	for _, id := range p.EvidenceIDs {
		if !agent.ValidID(id) || evidence[id] {
			return ErrInvalidProposal
		}
		evidence[id] = true
	}
	indices := make(map[int]bool, len(p.SKCs))
	for _, skc := range p.SKCs {
		if skc.Index == nil || *skc.Index < 0 || indices[*skc.Index] || skc.SaleAttribute == nil && len(skc.SKUs) == 0 {
			return ErrInvalidProposal
		}
		indices[*skc.Index] = true
		skus := make(map[int]bool, len(skc.SKUs))
		for _, sku := range skc.SKUs {
			if sku.Index == nil || *sku.Index < 0 || skus[*sku.Index] || sku.SaleAttributes == nil && sku.Attributes == nil {
				return ErrInvalidProposal
			}
			skus[*sku.Index] = true
		}
	}
	return nil
}

// ApplyProposal returns a new Target input for the original owner's CAS. It
// neither validates official readiness nor saves any Product/Target fact.
func ApplyProposal(base record.TargetRecord, p Proposal) (record.TargetInput, error) {
	if validateProposal(p) != nil {
		return record.TargetInput{}, ErrInvalidProposal
	}
	if p.CategoryID != nil && *p.CategoryID != base.Input.Draft.Product.CategoryID && (p.BrandCode != nil || p.Attributes != nil || len(p.SKCs) != 0 || p.Images != nil) {
		// A new category needs its actual attribute rules on the next exact
		// record; never guess fields against another category's schema.
		return record.TargetInput{}, ErrInvalidProposal
	}
	raw, err := json.Marshal(base.Input)
	if err != nil {
		return record.TargetInput{}, ErrInvalidProposal
	}
	var result record.TargetInput
	if json.Unmarshal(raw, &result) != nil {
		return record.TargetInput{}, ErrInvalidProposal
	}
	// Clone the proposal as well so caller-owned slices cannot alter the
	// approved input after validation/adoption preparation.
	raw, err = json.Marshal(p)
	if err != nil {
		return record.TargetInput{}, ErrInvalidProposal
	}
	var copied Proposal
	if json.Unmarshal(raw, &copied) != nil {
		return record.TargetInput{}, ErrInvalidProposal
	}
	p = copied
	result.ExpectedRevision = base.Revision
	if p.CategoryID != nil {
		result.Draft.Product.CategoryID = *p.CategoryID
	}
	if p.BrandCode != nil {
		result.Draft.Product.BrandCode = *p.BrandCode
	}
	if p.Attributes != nil {
		result.Draft.Product.Attributes = *p.Attributes
	}
	if p.Images != nil {
		result.Draft.Images = *p.Images
	}
	for _, change := range p.SKCs {
		if *change.Index >= len(result.Draft.Product.SKCs) {
			return record.TargetInput{}, ErrInvalidProposal
		}
		skc := &result.Draft.Product.SKCs[*change.Index]
		if change.SaleAttribute != nil {
			skc.SaleAttribute = *change.SaleAttribute
		}
		for _, changeSKU := range change.SKUs {
			if *changeSKU.Index >= len(skc.SKUs) {
				return record.TargetInput{}, ErrInvalidProposal
			}
			sku := &skc.SKUs[*changeSKU.Index]
			if changeSKU.SaleAttributes != nil {
				sku.SaleAttributes = *changeSKU.SaleAttributes
			}
			if changeSKU.Attributes != nil {
				sku.Attributes = *changeSKU.Attributes
			}
		}
	}
	return result, nil
}
