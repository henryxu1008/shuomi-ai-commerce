package tiktokshop

import "testing"

func TestCreatedGlobalProductMustMatchEveryOriginalSellerSKU(t *testing.T) {
	raw := []byte(`{"code":0,"data":{"global_product_id":"global-product","global_skus":[{"id":"sku-b","seller_sku":"B"},{"id":"sku-a","seller_sku":"A"}]}}`)
	product, err := DecodeCreatedGlobal(raw, []string{"A", "B"})
	if err != nil || product.ID != "global-product" || len(product.SKUs) != 2 {
		t.Fatalf("valid creation rejected: %v", err)
	}
	for _, expected := range [][]string{{"A"}, {"A", "C"}, {"A", "A"}} {
		if _, err := DecodeCreatedGlobal(raw, expected); err == nil {
			t.Fatal("partial or wrong SKU set accepted")
		}
	}
	if _, err := DecodeCreatedGlobal([]byte(`{"code":0,"data":{"global_product_id":"product","global_skus":[{"id":"sku","seller_sku":"A"},{"id":"sku","seller_sku":"B"}]}}`), []string{"A", "B"}); err == nil {
		t.Fatal("duplicate global ID accepted")
	}
}
func TestUploadedImageMustMatchUseCaseAndActualReturnedDimensions(t *testing.T) {
	raw := []byte(`{"code":0,"data":{"uri":"tos/image","url":"https://image.example.org/image.png","width":600,"height":600,"use_case":"MAIN_IMAGE"}}`)
	image, err := DecodeUploadedImage(raw, "MAIN_IMAGE")
	if err != nil || image.URI != "tos/image" {
		t.Fatal(err)
	}
	if _, err := DecodeUploadedImage(raw, "CERTIFICATION_IMAGE"); err == nil {
		t.Fatal("wrong image use case accepted")
	}
	if _, err := DecodeUploadedImage([]byte(`{"code":0,"data":{"uri":"image","url":"https://image.example.org/image.png","width":100,"height":100,"use_case":"MAIN_IMAGE"}}`), "MAIN_IMAGE"); err == nil {
		t.Fatal("invalid returned main image dimensions accepted")
	}
}
