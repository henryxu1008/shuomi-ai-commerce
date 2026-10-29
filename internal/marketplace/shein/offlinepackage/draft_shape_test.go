package offlinepackage

import (
	"testing"
)

func TestDraftPayloadOfReadsCanonicalDraft(t *testing.T) {
	canonical := &RequestDraft{SpuName: "Canonical product"}
	pkg := &Package{DraftPayload: canonical}

	got := DraftPayloadOf(pkg)
	if got != canonical {
		t.Fatalf("DraftPayloadOf() = %p, want canonical draft %p", got, canonical)
	}
	if pkg.DraftPayload != canonical {
		t.Fatalf("DraftPayload = %p, want canonical draft %p", pkg.DraftPayload, canonical)
	}
}

func TestValidateDraftPayloadReportsStructuralIssues(t *testing.T) {
	issues := ValidateDraftPayload(&RequestDraft{
		SKCList: []SKCRequestDraft{{SupplierCode: "SKC-1"}},
	})

	if !hasDraftIssueCode(issues, "skc_sku_list_required") {
		t.Fatalf("issues = %+v, want skc_sku_list_required", issues)
	}
}

func TestValidateDraftPayloadAcceptsStructurallyCompleteDraft(t *testing.T) {
	issues := ValidateDraftPayload(&RequestDraft{
		SpuName: "Complete product",
		SKCList: []SKCRequestDraft{{
			SupplierCode: "SKC-1",
			SKUList:      []SKUDraft{{SupplierSKU: "SKU-1"}},
		}},
	})

	if len(issues) != 0 {
		t.Fatalf("issues = %+v, want none", issues)
	}
}

func TestHasAnySubmitSKUUsesDraftAsSourceWhenBothViewsExist(t *testing.T) {
	pkg := &Package{
		SkcList: []SKCPackage{{SKUs: []Variant{{SKU: "stale-package-sku"}}}},
		DraftPayload: &RequestDraft{SKCList: []SKCRequestDraft{{
			SKUList: nil,
		}}},
	}

	if HasAnySubmitSKU(pkg) {
		t.Fatal("HasAnySubmitSKU() = true, want false when semantic draft has no SKU")
	}
}

func hasDraftIssueCode(issues []DraftPayloadIssue, want string) bool {
	for _, issue := range issues {
		if issue.Code == want {
			return true
		}
	}
	return false
}
