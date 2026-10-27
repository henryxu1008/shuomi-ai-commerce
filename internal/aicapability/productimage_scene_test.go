package aicapability

import "testing"

func TestProductImageSceneCapabilityContract(t *testing.T) {
	if CapabilityProductImageScene != "productimage.scene_generation" {
		t.Fatalf("capability = %q", CapabilityProductImageScene)
	}
}

func TestProductImageModelOperationContracts(t *testing.T) {
	if OperationProductImageReview != "productimage_review" {
		t.Fatalf("unexpected product image operation contracts")
	}
}
