package offlinepackage

import (
	"encoding/json"
	"strings"
	"task-processor/internal/marketplace/shein/offlinepackage/previewdata"
	"testing"
)

func TestPackageSemanticFieldNamesRemainUsable(t *testing.T) {
	pkg := &Package{
		DraftPayload: &RequestDraft{
			SpuName: "Semantic Field Product",
		},
	}

	pkg.PreviewPayload = &previewdata.Product{SPUName: pkg.DraftPayload.SpuName}
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

func TestPackageJSONIncludesOnlyCanonicalFieldNames(t *testing.T) {
	pkg := &Package{
		DraftPayload:         &RequestDraft{SpuName: "Semantic Field Product"},
		FinalSubmissionDraft: &FinalDraft{Confirmed: true},
	}
	pkg.PreviewPayload = &previewdata.Product{SPUName: pkg.DraftPayload.SpuName}

	data, err := json.Marshal(pkg)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	text := string(data)
	for _, key := range []string{
		`"draft_payload"`,
		`"preview_payload"`,
		`"final_submission_draft"`,
	} {
		if !strings.Contains(text, key) {
			t.Fatalf("json output missing %s: %s", key, text)
		}
	}
	for _, key := range []string{`"request_draft"`, `"preview_product"`, `"final_draft"`} {
		if strings.Contains(text, key) {
			t.Fatalf("retired JSON key %s emitted: %s", key, text)
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

func TestPackageRejectsRetiredJSONAliases(t *testing.T) {
	for _, key := range []string{"request_draft", "preview_product", "final_draft"} {
		for _, value := range []string{"null", "{}"} {
			t.Run(key+value, func(t *testing.T) {
				if _, err := DecodePersistedPackageStrict([]byte(`{"` + key + `":` + value + `}`)); err == nil {
					t.Fatal("retired alias accepted")
				}
			})
		}
	}
}

func TestCanonicalPackageAllowsMetadataWithHistoricalNames(t *testing.T) {
	pkg, err := DecodePersistedPackageStrict([]byte(`{"metadata":{"request_draft":"note","preview_product":"note","final_draft":"note"}}`))
	if err != nil || len(pkg.Metadata) != 3 {
		t.Fatalf("free metadata keys rejected: %v", err)
	}
	for _, pair := range [][2]string{{"request_draft", "draft_payload"}, {"preview_product", "preview_payload"}, {"final_draft", "final_submission_draft"}} {
		if _, err := DecodePersistedPackageStrict([]byte(`{"` + pair[0] + `":{},"` + pair[1] + `":{}}`)); err == nil {
			t.Fatalf("equal retired alias %s accepted", pair[0])
		}
	}
}
