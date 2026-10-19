package shein

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPackageSemanticFieldNamesRemainUsable(t *testing.T) {
	pkg := &Package{
		DraftPayload: &RequestDraft{
			SpuName: "Semantic Field Product",
		},
	}

	pkg.PreviewPayload = BuildPreviewProduct(pkg)
	pkg.FinalSubmissionDraft = &FinalDraft{Confirmed: true}

	if pkg.DraftPayload == nil || pkg.DraftPayload.SpuName != "Semantic Field Product" {
		t.Fatalf("draft payload = %+v", pkg.DraftPayload)
	}
	if pkg.PreviewPayload == nil || pkg.PreviewPayload.SPUName != "Semantic Field Product" {
		t.Fatalf("preview payload = %+v", pkg.PreviewPayload)
	}
	if pkg.FinalSubmissionDraft == nil || !pkg.FinalSubmissionDraft.Confirmed {
		t.Fatalf("final submission draft = %+v", pkg.FinalSubmissionDraft)
	}
}

func TestPackageJSONIncludesLegacyAndSemanticFieldNames(t *testing.T) {
	pkg := NormalizePackageSemanticFields(&Package{
		DraftPayload:         &RequestDraft{SpuName: "Semantic Field Product"},
		FinalSubmissionDraft: &FinalDraft{Confirmed: true},
	})
	SetPreviewPayload(pkg, BuildPreviewProduct(pkg))

	data, err := json.Marshal(pkg)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	text := string(data)
	for _, key := range []string{
		`"request_draft"`,
		`"draft_payload"`,
		`"preview_product"`,
		`"preview_payload"`,
		`"final_draft"`,
		`"final_submission_draft"`,
	} {
		if !strings.Contains(text, key) {
			t.Fatalf("json output missing %s: %s", key, text)
		}
	}
}

func TestPackageRejectsRetiredExecutionFields(t *testing.T) {
	for _, key := range []string{"inspection", "submission", "submission_state", "submission_events"} {
		t.Run(key, func(t *testing.T) {
			if _, err := DecodePersistedPackageStrict([]byte(`{"` + key + `":{}}`)); err == nil {
				t.Fatal("retired execution field accepted")
			}
		})
	}
}
