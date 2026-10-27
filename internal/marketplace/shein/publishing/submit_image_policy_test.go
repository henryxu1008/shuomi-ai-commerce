package publishing

import (
	"task-processor/internal/marketplace/shein/offlinepackage/previewdata"
	"testing"
)

func TestFinalSubmitImagesRequireSKCSkipsSaveDraftOnly(t *testing.T) {
	t.Parallel()

	if FinalSubmitImagesRequireSKC(" save_draft ") {
		t.Fatal("FinalSubmitImagesRequireSKC(save_draft) = true, want false")
	}
	for _, action := range []string{"publish", "", "unknown"} {
		if !FinalSubmitImagesRequireSKC(action) {
			t.Fatalf("FinalSubmitImagesRequireSKC(%q) = false, want true", action)
		}
	}
}

func TestFinalSubmitImagesReadyRequiresFinalDraftAndImages(t *testing.T) {
	t.Parallel()

	for _, action := range []string{"publish", "save_draft"} {
		if ready, message := FinalSubmitImagesReady(action, FinalSubmitImageReadinessInput{}); ready || message != "缺少最终图片草稿，无法确认主图和图库" {
			t.Fatalf("%s without final draft = (%v, %q), want blocker", action, ready, message)
		}
	}
	if ready, message := FinalSubmitImagesReady("publish", FinalSubmitImageReadinessInput{HasFinalDraft: true}); ready || message == "" {
		t.Fatalf("missing main image = (%v, %q), want blocker", ready, message)
	}
	if ready, message := FinalSubmitImagesReady("publish", FinalSubmitImageReadinessInput{
		HasFinalDraft: true,
		HasMainImage:  true,
	}); ready || message == "" {
		t.Fatalf("missing gallery = (%v, %q), want blocker", ready, message)
	}
}

func TestFinalSubmitImagesReadyUsesActionSpecificSKCStrictness(t *testing.T) {
	t.Parallel()

	base := FinalSubmitImageReadinessInput{
		HasFinalDraft: true,
		HasMainImage:  true,
		HasGallery:    true,
	}
	if ready, message := FinalSubmitImagesReady("save_draft", base); !ready || message == "" {
		t.Fatalf("save draft readiness = (%v, %q), want ready", ready, message)
	}
	base.RequiresSKC = true
	if ready, message := FinalSubmitImagesReady("save_draft", base); ready || message == "" {
		t.Fatalf("save draft with explicit SKC requirement = (%v, %q), want blocker", ready, message)
	}
	base.RequiresSKC = false
	if ready, message := FinalSubmitImagesReady("publish", base); ready || message == "" {
		t.Fatalf("publish without SKC = (%v, %q), want blocker", ready, message)
	}
	base.HasSKCImage = true
	if ready, message := FinalSubmitImagesReady("publish", base); ready || message == "" {
		t.Fatalf("publish without swatch = (%v, %q), want blocker", ready, message)
	}
	base.HasSwatchRole = true
	if ready, message := FinalSubmitImagesReady("publish", base); !ready || message == "" {
		t.Fatalf("publish ready = (%v, %q), want ready", ready, message)
	}
}

func TestSubmitImageDraftHasImageChecksAllImageSources(t *testing.T) {
	t.Parallel()

	if SubmitImageDraftHasImage(SubmitImageDraftInput{}) {
		t.Fatal("empty draft = true, want false")
	}
	if !SubmitImageDraftHasImage(SubmitImageDraftInput{MainImage: " https://cdn.example/main.jpg "}) {
		t.Fatal("main image draft = false, want true")
	}
	if !SubmitImageDraftHasImage(SubmitImageDraftInput{Gallery: []string{"", "https://cdn.example/gallery.jpg"}}) {
		t.Fatal("gallery image draft = false, want true")
	}
	if !SubmitImageDraftHasImage(SubmitImageDraftInput{Source: []string{"https://cdn.example/source.jpg"}}) {
		t.Fatal("source image draft = false, want true")
	}
}

func TestUniqueNonEmptyImageURLsTrimsAndDedupes(t *testing.T) {
	t.Parallel()

	got := UniqueNonEmptyImageURLs([]string{" a.jpg ", "", "b.jpg", "a.jpg", " b.jpg "})
	want := []string{"a.jpg", "b.jpg"}
	if len(got) != len(want) {
		t.Fatalf("UniqueNonEmptyImageURLs() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("UniqueNonEmptyImageURLs()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestProductImageInfoHasImageChecksImageInfoList(t *testing.T) {
	t.Parallel()

	if ProductImageInfoHasImage(nil) {
		t.Fatal("ProductImageInfoHasImage(nil) = true, want false")
	}
	if ProductImageInfoHasImage(&previewdata.ImageInfo{ImageInfoList: []previewdata.ImageDetail{{ImageURL: " "}}}) {
		t.Fatal("ProductImageInfoHasImage(blank URL) = true, want false")
	}
	if !ProductImageInfoHasImage(&previewdata.ImageInfo{ImageInfoList: []previewdata.ImageDetail{{ImageURL: " https://cdn.example/image.jpg "}}}) {
		t.Fatal("ProductImageInfoHasImage(image URL) = false, want true")
	}
}
