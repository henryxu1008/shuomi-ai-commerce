package submission

import (
	"strings"
	marketplace "task-processor/internal/marketplace/tiktokshop"
	"task-processor/internal/storecenter"
	"testing"
)

func TestTikTokStageSubjectsDoNotChangeWithNewCredentialsOrProductData(t *testing.T) {
	b := storecenter.ProductMerchantBinding{StoreID: "00000000-0000-4000-8000-000000000001", SupplierIdentityHash: strings.Repeat("a", 64), ApplicationID: "app"}
	first, err := TikTokProductSubjectID("product-a", TikTokPublishStage)
	if err != nil {
		t.Fatal(err)
	}
	again, err := TikTokProductSubjectID("product-a", TikTokPublishStage)
	if err != nil || first != again {
		t.Fatal("new intent resets publication identity")
	}
	global, err := TikTokProductSubjectID("product-a", TikTokGlobalCreateStage)
	if err != nil || global == first {
		t.Fatal("global create and US publication collapsed")
	}
	image := marketplace.ImageObservation{ContentHash: strings.Repeat("b", 64), UseCase: "MAIN_IMAGE"}
	old, err := TikTokImageSubjectID(image, b)
	if err != nil {
		t.Fatal(err)
	}
	b.TikTok.GrantGeneration = 9
	next, err := TikTokImageSubjectID(image, b)
	if err != nil || old != next {
		t.Fatal("refresh recreates an image effect")
	}
	image.UseCase = "CERTIFICATION_IMAGE"
	other, err := TikTokImageSubjectID(image, b)
	if err != nil || other == old {
		t.Fatal("distinct image use cases share an effect")
	}
}
