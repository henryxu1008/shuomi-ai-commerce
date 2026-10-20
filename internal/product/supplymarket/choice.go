package supplymarket

import (
	"errors"

	"task-processor/internal/product/catalog"
)

const (
	ChoiceNotOptimized      = "not_optimized"
	ChoiceIncompleteProduct = "incomplete_product"
)

// ProjectChoice consumes an already verified exact selection. An unqualified
// saved product is a normal read result, never a disclosure or write authority.
func ProjectChoice(selection SelectionInput, snapshot catalog.ProductSnapshot) (ProductChoice, error) {
	choice := ProductChoice{Selection: selection, Optimized: selection.ApplyID != ""}
	if !choice.Optimized {
		choice.IneligibleReason = ChoiceNotOptimized
	}
	product, err := PublicProjection(snapshot)
	if err != nil {
		if !errors.Is(err, ErrInvalid) {
			return ProductChoice{}, err
		}
		if choice.Optimized {
			choice.IneligibleReason = ChoiceIncompleteProduct
		}
		return choice, nil
	}
	// Unoptimized but complete products remain usable by the existing official
	// draft flow, which intentionally does not require an Apply.
	choice.Product = &product
	return choice, nil
}

func (c ProductChoice) Eligible() bool {
	return c.Optimized && c.IneligibleReason == "" && c.Product != nil
}
