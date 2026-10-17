package publishing

import (
	"fmt"

	"task-processor/internal/pkg/skugen"
)

// buildUniqueSKUSNs retains source order and delegates code generation to its existing owner.
func buildUniqueSKUSNs(mainASIN string, variantASINs []string, prefix, suffix string) []string {
	var codes []string
	seen := make(map[string]bool)
	appendASIN := func(asin string) {
		if asin == "" {
			return
		}
		code := skugen.Generate(asin, skugen.StrategyASINOnly, prefix, suffix)
		if seen[code] {
			return
		}
		seen[code] = true
		codes = append(codes, code)
	}
	appendASIN(mainASIN)
	for _, asin := range variantASINs {
		appendASIN(asin)
	}
	return codes
}

// skuSNFailure carries diagnostic values, without task, client or persistence ownership.
type skuSNFailure struct {
	outSKUSN    string
	usedGoodsID string
	usedSKUID   string
	failReason  string
}

func validateSKUSNFailures(failures []skuSNFailure) error {
	if len(failures) == 0 {
		return nil
	}
	details := make([]string, 0, len(failures))
	for _, failure := range failures {
		details = append(details, fmt.Sprintf("SKU编码 '%s' 已被商品 %s 使用 (SKU ID: %s): %s",
			failure.outSKUSN, failure.usedGoodsID, failure.usedSKUID, failure.failReason))
	}
	return fmt.Errorf("发现SKU编码重复，请检查并修改重复的编码: %v", details)
}
