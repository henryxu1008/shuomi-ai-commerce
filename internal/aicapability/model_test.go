package aicapability

import (
	"errors"
	"testing"
)

func TestProductEnrichTextCapabilityContracts(t *testing.T) {
	if CapabilityProductEnrichText != "productenrich.text_understanding" {
		t.Fatalf("capability = %q", CapabilityProductEnrichText)
	}
}

func TestCategoryOf(t *testing.T) {
	if got := CategoryOf(nil); got != "" {
		t.Fatalf("CategoryOf(nil) = %q", got)
	}
	if got := CategoryOf(errors.New("plain")); got != ErrorUnknown {
		t.Fatalf("CategoryOf(plain) = %q, want %q", got, ErrorUnknown)
	}
	if got := CategoryOf(NewError(ErrorProviderTimeout, "image_generate", errors.New("timeout"))); got != ErrorProviderTimeout {
		t.Fatalf("CategoryOf(capability error) = %q", got)
	}
}
