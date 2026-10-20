package shein

import (
	"strings"
	"testing"

	sheinproduct "task-processor/internal/shein/api/product"
)

func TestPrepareProductForValidationPreservesIdentityAndNormalizesTransport(t *testing.T) {
	makeProduct := func() *sheinproduct.Product {
		return &sheinproduct.Product{SPUName: "title", SKCList: []sheinproduct.SKC{{SKUS: []sheinproduct.SKU{{SupplierSKU: "sku"}}}}}
	}
	for _, identity := range []string{"", "given-identity"} {
		product := makeProduct()
		product.PointKey = identity
		PrepareProductForValidation(product)
		if product.PointKey != identity {
			t.Fatal("validation changed submission identity")
		}
		if product.SPUName != "" || product.SourceSystem != "listingkit" || product.SupplierCode == "" {
			t.Fatalf("normalization = %+v", product)
		}
		sku := product.SKCList[0].SKUS[0]
		if sku.SupplierSKU != "sku" || sku.WeightUnit != "g" || sku.LengthUnit != "Inch" {
			t.Fatalf("SKU transport defaults = %+v", sku)
		}
		if strings.TrimSpace(product.SupplierCode) != product.SupplierCode {
			t.Fatal("supplier code is not normalized")
		}
	}
	PrepareProductForValidation(nil)
}
