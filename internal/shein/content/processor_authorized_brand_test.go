package content_test

import (
	"strings"
	"testing"

	"task-processor/internal/shein/authorizedbrand"
	"task-processor/internal/shein/content"
)

func TestSanitizeDisplayTextWithContext_PreservesAuthorizedAmazonBrand(t *testing.T) {
	t.Parallel()

	service := content.NewSensitiveWordServiceInMemory(nil)
	cleanupCtx := &content.CleanupContext{
		AuthorizedBrand: &authorizedbrand.Resolved{
			Enabled: true,
			Name:    "Amazon Basics",
			NameEn:  "Amazon Basics",
		},
	}

	got := service.SanitizeDisplayTextWithContext(cleanupCtx, "Amazon Basics wireless mouse")
	if !strings.Contains(got, "Amazon Basics") {
		t.Fatalf("sanitized text = %q, want authorized brand preserved", got)
	}
}

func TestSanitizeDisplayTextWithContext_RemovesContextBrandButKeepsAuthorizedBrand(t *testing.T) {
	t.Parallel()

	service := content.NewSensitiveWordServiceInMemory(nil)
	cleanupCtx := &content.CleanupContext{
		AuthorizedBrand: &authorizedbrand.Resolved{
			Enabled: true,
			Name:    "Logitech",
			NameEn:  "Logitech",
		},
		SourceBrand: "Sony",
	}

	got := service.SanitizeDisplayTextWithContext(cleanupCtx, "Logitech mouse by Sony for office")
	if !strings.Contains(got, "Logitech") {
		t.Fatalf("sanitized text = %q, want authorized brand preserved", got)
	}
	if strings.Contains(strings.ToLower(got), "sony") {
		t.Fatalf("sanitized text = %q, want context brand removed", got)
	}
}
