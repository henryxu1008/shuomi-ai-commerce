package shein

import (
	"testing"

	sheinproduct "task-processor/internal/shein/api/product"
)

func TestCloneProductForSubmitDeepCopiesProduct(t *testing.T) {
	t.Parallel()

	product := &sheinproduct.Product{
		SupplierCode: "SUP-1",
		ImageInfo: &sheinproduct.ImageInfo{
			ImageInfoList: []sheinproduct.ImageDetail{{ImageURL: "https://cdn.example.com/1.jpg"}},
		},
	}

	cloned, err := CloneProductForSubmit(product)
	if err != nil {
		t.Fatalf("CloneProductForSubmit() error = %v", err)
	}
	if cloned == nil || cloned.SupplierCode != product.SupplierCode {
		t.Fatalf("cloned product = %+v, want supplier code copied", cloned)
	}
	cloned.ImageInfo.ImageInfoList[0].ImageURL = "https://cdn.example.com/changed.jpg"
	if product.ImageInfo.ImageInfoList[0].ImageURL == cloned.ImageInfo.ImageInfoList[0].ImageURL {
		t.Fatal("CloneProductForSubmit() did not deep copy image list")
	}
}
