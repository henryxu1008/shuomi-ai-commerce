package publishing

import "fmt"

// validateSPUStructure checks values without owning a legacy task or API payload.
func validateSPUStructure(name string, skcs [][][]submissionSpec) error {
	if name == "" {
		return fmt.Errorf("商品名称不能为空")
	}
	if len(skcs) == 0 {
		return fmt.Errorf("SKC列表不能为空")
	}
	for i, skc := range skcs {
		if len(skc) == 0 {
			return fmt.Errorf("SKC[%d]的SKU列表不能为空", i)
		}
		for j, sku := range skc {
			if len(sku) == 0 {
				return fmt.Errorf("SKC[%d]的SKU[%d]的规格列表不能为空，TEMU要求每个SKU必须有规格", i, j)
			}
			for k, spec := range sku {
				if spec.specID == "" {
					return fmt.Errorf("SKC[%d]的SKU[%d]的规格[%d]的SpecID不能为空", i, j, k)
				}
				if spec.parentSpecID == "" {
					return fmt.Errorf("SKC[%d]的SKU[%d]的规格[%d]的ParentSpecID不能为空", i, j, k)
				}
				if spec.name == "" {
					return fmt.Errorf("SKC[%d]的SKU[%d]的规格[%d]的SpecName不能为空", i, j, k)
				}
			}
		}
	}
	return nil
}
