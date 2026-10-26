package offlinepackage

import (
	"strings"
	"testing"

	sheinpolicy "task-processor/internal/marketplace/shein/publishing"
	"task-processor/internal/publishing/common"
	sheinproduct "task-processor/internal/shein/api/product"
)

func TestSecondarySaleAttributeRequiredRequiresMultiSKUSecondarySourceAndTemplate(t *testing.T) {
	t.Parallel()

	pkg := &Package{
		SaleAttributeResolution: &SaleAttributeResolution{
			PrimaryAttributeID:       1001,
			SecondarySourceDimension: "尺码",
			SourceDimensions: []SourceVariantDimension{{
				Name:          "size",
				DistinctCount: 2,
			}},
			TemplateOptions: []SaleAttributeTemplateOption{{
				AttributeID: 2002,
				Name:        "Size",
			}},
		},
		DraftPayload: &RequestDraft{SKCList: []SKCRequestDraft{{
			SKUList: []SKUDraft{{SupplierSKU: "SKU-1"}, {SupplierSKU: "SKU-2"}},
		}}},
	}

	if !secondarySaleAttributeRequired(pkg) {
		t.Fatal("SecondarySaleAttributeRequired = false, want true")
	}

	pkg.SaleAttributeResolution.TemplateOptions[0].AttributeID = 1001
	if secondarySaleAttributeRequired(pkg) {
		t.Fatal("SecondarySaleAttributeRequired = true without secondary template candidate")
	}
}

func TestSaleAttributesReadinessFailureReasonsReportsSKUAttributeBlockers(t *testing.T) {
	t.Parallel()

	valueID := 10
	pkg := &Package{
		SaleAttributeResolution: &SaleAttributeResolution{
			Status:                   "resolved",
			PrimaryAttributeID:       1001,
			SecondaryAttributeID:     2002,
			SecondarySourceDimension: "size",
		},
		DraftPayload: &RequestDraft{SKCList: []SKCRequestDraft{{
			SupplierCode:  "SKC-1",
			SaleAttribute: &ResolvedSaleAttribute{AttributeID: 1001, AttributeValueID: &valueID},
			SKUList: []SKUDraft{{
				SupplierSKU: "SKU-1",
			}},
		}}},
	}

	reasons := saleAttributesReadinessFailureReasons(pkg)

	if len(reasons) != 1 || reasons[0] != `sku "SKU-1" is missing sale_attributes` {
		t.Fatalf("reasons = %#v, want missing sku sale attributes", reasons)
	}
}

func TestSaleAttributesReadyForSubmitAcceptsResolvedSKCAndSKUAttributes(t *testing.T) {
	t.Parallel()

	skcValueID := 10
	skuValueID := 20
	pkg := &Package{
		SaleAttributeResolution: &SaleAttributeResolution{
			Status:               "resolved",
			PrimaryAttributeID:   1001,
			SecondaryAttributeID: 2002,
		},
		DraftPayload: &RequestDraft{SKCList: []SKCRequestDraft{{
			SupplierCode:  "SKC-1",
			SaleAttribute: &ResolvedSaleAttribute{AttributeID: 1001, AttributeValueID: &skcValueID},
			SKUList: []SKUDraft{{
				SupplierSKU: "SKU-1",
				SaleAttributes: []ResolvedSaleAttribute{{
					AttributeID:      2002,
					AttributeValueID: &skuValueID,
				}},
			}},
		}}},
	}

	if !SaleAttributesReadyForSubmit(pkg) {
		t.Fatalf("SaleAttributesReadyForSubmit = false; reasons=%#v", saleAttributesReadinessFailureReasons(pkg))
	}
}

func TestHasBlockingPendingAttributesReportsRequiredAndManualPendingAttributes(t *testing.T) {
	t.Parallel()

	if !HasBlockingPendingAttributes(&Package{}) {
		t.Fatal("HasBlockingPendingAttributes without resolution = false, want true")
	}
	if !HasBlockingPendingAttributes(&Package{AttributeResolution: &AttributeResolution{
		PendingAttributeCandidates: []PendingAttributeCandidate{{Required: true}},
	}}) {
		t.Fatal("HasBlockingPendingAttributes required candidate = false, want true")
	}
	if !HasBlockingPendingAttributes(&Package{AttributeResolution: &AttributeResolution{
		PendingAttributes: []common.Attribute{{Name: "Material"}},
	}}) {
		t.Fatal("HasBlockingPendingAttributes manual pending attribute = false, want true")
	}
	if HasBlockingPendingAttributes(&Package{AttributeResolution: &AttributeResolution{}}) {
		t.Fatal("HasBlockingPendingAttributes empty resolution = true, want false")
	}
}

func TestFinalSubmitImagesReadyRequiresPublishSKCImageAndSwatch(t *testing.T) {
	t.Parallel()

	pkg := &Package{
		FinalSubmissionDraft: &FinalDraft{MainImageURL: "https://cdn.example/main.jpg"},
		DraftPayload: &RequestDraft{
			ImageInfo: &ImageDraft{
				MainImage: "https://cdn.example/main.jpg",
				Gallery:   []string{"https://cdn.example/gallery.jpg"},
			},
			SKCList: []SKCRequestDraft{{}, {}},
		},
	}

	ready, message := FinalSubmitImagesReady(pkg, "publish")
	if ready || !strings.Contains(message, "SKC") {
		t.Fatalf("FinalSubmitImagesReady(no skc) = (%v, %q), want SKC blocker", ready, message)
	}

	pkg.DraftPayload.SKCList[0].ImageInfo = &ImageDraft{MainImage: "https://cdn.example/skc.jpg"}
	ready, message = FinalSubmitImagesReady(pkg, "publish")
	if !ready {
		t.Fatalf("FinalSubmitImagesReady(with skc) = (%v, %q), want ready", ready, message)
	}
}

func TestFinalSubmitImagesReadyAllowsSaveDraftWithoutSKCImage(t *testing.T) {
	t.Parallel()

	pkg := &Package{
		FinalSubmissionDraft: &FinalDraft{MainImageURL: "https://cdn.example/main.jpg"},
		DraftPayload: &RequestDraft{
			ImageInfo: &ImageDraft{
				MainImage: "https://cdn.example/main.jpg",
				Gallery:   []string{"https://cdn.example/gallery.jpg"},
			},
			SKCList: []SKCRequestDraft{{}},
		},
	}

	ready, message := FinalSubmitImagesReady(pkg, "save_draft")
	if !ready {
		t.Fatalf("FinalSubmitImagesReady(save_draft) = (%v, %q), want ready", ready, message)
	}
}

func TestSubmitPricingReadyRequiresPositiveSKUAndSitePrices(t *testing.T) {
	t.Parallel()

	pkg := &Package{DraftPayload: &RequestDraft{SKCList: []SKCRequestDraft{{
		SKUList: []SKUDraft{{
			BasePrice:     "12.34",
			SitePriceList: []SitePrice{{SubSite: "US", BasePrice: "15.99"}},
		}},
	}}}}

	if !SubmitPricingReady(pkg) {
		t.Fatal("SubmitPricingReady() = false, want true")
	}
	pkg.DraftPayload.SKCList[0].SKUList[0].SitePriceList[0].BasePrice = "0"
	if SubmitPricingReady(pkg) {
		t.Fatal("SubmitPricingReady(zero site price) = true, want false")
	}
}

func TestFinalReviewReadyRequiresConfirmationForPublish(t *testing.T) {
	t.Parallel()

	pkg := &Package{FinalSubmissionDraft: &FinalDraft{}}

	if FinalReviewReady(pkg, "publish") {
		t.Fatal("FinalReviewReady(publish, unconfirmed) = true, want false")
	}
	pkg.FinalSubmissionDraft.Confirmed = true
	if !FinalReviewReady(pkg, "publish") {
		t.Fatal("FinalReviewReady(publish, confirmed) = false, want true")
	}
}

func TestFinalReviewReadyAllowsSaveDraft(t *testing.T) {
	t.Parallel()

	pkg := &Package{FinalSubmissionDraft: &FinalDraft{}}

	if !FinalReviewReady(pkg, " save_draft ") {
		t.Fatal("FinalReviewReady(save_draft) = false, want true")
	}
}

func TestHasSubmitImageChecksPackageDraftAndPreviewImages(t *testing.T) {
	t.Parallel()

	if HasSubmitImage(&Package{Images: &common.ImageSet{WhiteBgImage: " https://cdn.example/white.jpg "}}) != true {
		t.Fatal("HasSubmitImage(package images) = false, want true")
	}
	if HasSubmitImage(&Package{DraftPayload: &RequestDraft{SKCList: []SKCRequestDraft{{
		SKUList: []SKUDraft{{MainImage: "https://cdn.example/sku.jpg"}},
	}}}}) != true {
		t.Fatal("HasSubmitImage(draft sku image) = false, want true")
	}
	if HasSubmitImage(&Package{PreviewPayload: &sheinproduct.Product{SKCList: []sheinproduct.SKC{{
		ImageInfo: sheinproduct.ImageInfo{ImageInfoList: []sheinproduct.ImageDetail{{ImageURL: "https://cdn.example/skc.jpg"}}},
	}}}}) != true {
		t.Fatal("HasSubmitImage(preview skc image) = false, want true")
	}
}

func TestHasAnySubmitSKURejectsSummaryWithoutDraft(t *testing.T) {
	pkg := &Package{SkcList: []SKCPackage{{SKUs: []common.Variant{{SKU: "summary-only"}}}}}
	if HasAnySubmitSKU(pkg) {
		t.Fatal("summary-only SKU must not be ready without canonical draft")
	}
}

func TestVariantImageCoverageReadsCanonicalMetadata(t *testing.T) {
	t.Parallel()

	pkg := &Package{}
	pkg.Metadata = map[string]string{sheinpolicy.VariantImageCoverageMessageKey: "needs images", sheinpolicy.VariantImageCoverageStatusKey: "blocked"}
	message, blocked := VariantImageCoverageStatus(pkg)
	if !blocked || message != "needs images" {
		t.Fatalf("VariantImageCoverageStatus() = %q, %v; want blocked message", message, blocked)
	}

	pkg.Metadata = nil
	if message, blocked := VariantImageCoverageStatus(pkg); message != "" || blocked {
		t.Fatalf("cleared metadata status = %q, %v", message, blocked)
	}
}
