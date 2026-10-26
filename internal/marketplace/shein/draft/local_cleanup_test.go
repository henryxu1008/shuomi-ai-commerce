package draft

import (
	"context"
	"strings"
	"testing"

	"task-processor/internal/marketplace/shein/offlinepackage"
	"task-processor/internal/publishing/common"
	"task-processor/internal/shein/authorizedbrand"
)

func TestSanitizeSheinListingCopyPreservesAuthorizedBrandFromRuntimeContext(t *testing.T) {
	t.Parallel()

	copy := &localCopy{
		Title:        "Amazon Basics Wireless Mouse",
		Description:  "Amazon Basics office mouse for home use",
		SKCTitleBase: "Amazon Basics Mouse",
	}

	changed := sanitizeSheinListingCopy(copy, authorizedbrand.WithResolved(context.Background(), &authorizedbrand.Resolved{
		Enabled: true,
		Name:    "Amazon Basics",
		NameEn:  "Amazon Basics",
	}), nil, nil)
	if changed {
		t.Fatalf("changed = true, want authorized brand preserved without cleanup")
	}
	if copy.Title != "Amazon Basics Wireless Mouse" || copy.Description != "Amazon Basics office mouse for home use" || copy.SKCTitleBase != "Amazon Basics Mouse" {
		t.Fatalf("copy after sanitize = %+v", copy)
	}
}

func TestSanitizeDraftPayloadSensitiveContentPreservesAuthorizedBrandFromRuntimeContext(t *testing.T) {
	t.Parallel()

	pkg := &offlinepackage.Package{
		ProductAttributes: []common.Attribute{
			{Name: "Material Detail", Value: "Amazon Basics acrylic"},
		},
		DraftPayload: &offlinepackage.RequestDraft{
			MultiLanguageNameList: []offlinepackage.LocalizedText{{Language: "en", Name: "Amazon Basics Wireless Mouse"}},
			MultiLanguageDescList: []offlinepackage.LocalizedText{{Language: "en", Name: "Amazon Basics office mouse"}},
			ProductAttributeList:  []common.Attribute{{Name: "Material Detail", Value: "Amazon Basics acrylic"}},
			SKCList: []offlinepackage.SKCRequestDraft{{
				SkcName:               "Amazon Basics Blue",
				MultiLanguageNameList: []offlinepackage.LocalizedText{{Language: "en", Name: "Amazon Basics Blue"}},
			}},
		},
	}

	runtimeCtx := authorizedbrand.WithResolved(context.Background(), &authorizedbrand.Resolved{
		Enabled: true,
		Name:    "Amazon Basics",
		NameEn:  "Amazon Basics",
	})

	changed := sanitizeLocalDraft(pkg, runtimeCtx, nil, nil)
	if changed {
		t.Fatalf("changed = true, want authorized brand preserved without cleanup")
	}
	if pkg.DraftPayload.SKCList[0].SkcName != "Amazon Basics Blue" {
		t.Fatalf("SkcName = %q, want authorized brand preserved", pkg.DraftPayload.SKCList[0].SkcName)
	}
	if pkg.DraftPayload.ProductAttributeList[0].Value != "Amazon Basics acrylic" {
		t.Fatalf("ProductAttributeList[0].Value = %q, want authorized brand preserved", pkg.DraftPayload.ProductAttributeList[0].Value)
	}
}

func TestSanitizeDraftPayloadSensitiveContent_CleansDraftTextFields(t *testing.T) {
	pkg := &offlinepackage.Package{
		DraftPayload: &offlinepackage.RequestDraft{
			MultiLanguageNameList: []offlinepackage.LocalizedText{{Language: "en", Name: "Amazon Trademark Vase"}},
			MultiLanguageDescList: []offlinepackage.LocalizedText{{Language: "en", Name: "Amazon Trademark vase for home decor."}},
			ProductAttributeList: []common.Attribute{
				{Name: "Material Detail", Value: "Amazon Trademark acrylic"},
				{Name: "Length", Value: "12"},
			},
			SKCList: []offlinepackage.SKCRequestDraft{{
				SkcName:               "Amazon Trademark Blue",
				MultiLanguageNameList: []offlinepackage.LocalizedText{{Language: "en", Name: "Amazon Trademark Blue"}},
			}},
		},
	}

	changed := sanitizeLocalDraft(pkg, context.Background(), nil, nil)
	if !changed {
		t.Fatal("changed = false, want true")
	}

	assertNoSensitivePhrase(t, findLocalizedText(pkg.DraftPayload.MultiLanguageNameList, "en"), "draft title")
	assertNoSensitivePhrase(t, findLocalizedText(pkg.DraftPayload.MultiLanguageDescList, "en"), "draft description")
	assertNoSensitivePhrase(t, pkg.DraftPayload.SKCList[0].SkcName, "draft skc name")
	assertNoSensitivePhrase(t, pkg.DraftPayload.SKCList[0].MultiLanguageNameList[0].Name, "draft localized skc name")
	assertNoSensitivePhrase(t, pkg.DraftPayload.ProductAttributeList[0].Value, "draft free-text attribute")
	if got := pkg.DraftPayload.ProductAttributeList[1].Value; got != "12" {
		t.Fatalf("structured attribute value = %q, want unchanged", got)
	}
}

func TestSanitizeDraftPayloadSensitiveContent_PreservesAuthorizedBrandFromRuntimeContext(t *testing.T) {
	ctx := authorizedbrand.WithResolved(context.Background(), &authorizedbrand.Resolved{
		Enabled: true,
		Name:    "Amazon Basics",
		NameEn:  "Amazon Basics",
	})
	pkg := &offlinepackage.Package{DraftPayload: &offlinepackage.RequestDraft{
		ProductAttributeList: []common.Attribute{{
			Value: "Amazon Basics Trademark acrylic",
		}},
		SKCList: []offlinepackage.SKCRequestDraft{{
			SkcName: "Amazon Basics Trademark Blue",
			MultiLanguageNameList: []offlinepackage.LocalizedText{{
				Language: "en",
				Name:     "Amazon Basics Trademark Blue",
			}},
		}},
	}}

	if !sanitizeLocalDraft(pkg, ctx, nil, nil) {
		t.Fatal("sensitive content was not sanitized")
	}

	if strings.Contains(strings.ToLower(pkg.DraftPayload.ProductAttributeList[0].Value), "trademark") {
		t.Fatalf("Value = %q, want sensitive phrase removed", pkg.DraftPayload.ProductAttributeList[0].Value)
	}
	if !strings.Contains(strings.ToLower(pkg.DraftPayload.ProductAttributeList[0].Value), "amazon basics") {
		t.Fatalf("Value = %q, want authorized brand preserved", pkg.DraftPayload.ProductAttributeList[0].Value)
	}
	if !strings.Contains(strings.ToLower(pkg.DraftPayload.SKCList[0].SkcName), "amazon basics") {
		t.Fatalf("SKC primary name = %q, want authorized brand preserved", pkg.DraftPayload.SKCList[0].SkcName)
	}
	if !strings.Contains(strings.ToLower(pkg.DraftPayload.SKCList[0].MultiLanguageNameList[0].Name), "amazon basics") {
		t.Fatalf("SKC localized name = %q, want authorized brand preserved", pkg.DraftPayload.SKCList[0].MultiLanguageNameList[0].Name)
	}
}

func TestSanitizeDraftPayloadSensitiveContent_CleansFreeTextAttributesAndSKCNames(t *testing.T) {
	pkg := &offlinepackage.Package{DraftPayload: &offlinepackage.RequestDraft{
		MultiLanguageNameList: []offlinepackage.LocalizedText{{
			Language: "en",
			Name:     "Amazon Trademark Vase",
		}},
		MultiLanguageDescList: []offlinepackage.LocalizedText{{
			Language: "en",
			Name:     "Amazon Trademark vase for home decor.",
		}},
		ProductAttributeList: []common.Attribute{
			{Value: "Amazon Trademark acrylic"},
			{Value: "12"},
		},
		SKCList: []offlinepackage.SKCRequestDraft{{
			SkcName: "Amazon Trademark Blue",
			MultiLanguageNameList: []offlinepackage.LocalizedText{{
				Language: "en",
				Name:     "Amazon Trademark Blue",
			}},
		}},
	}}

	if !sanitizeLocalDraft(pkg, context.Background(), nil, nil) {
		t.Fatal("sensitive content was not sanitized")
	}

	assertNoSensitivePhrase(t, findLocalizedText(pkg.DraftPayload.MultiLanguageNameList, "en"), "submit title")
	assertNoSensitivePhrase(t, findLocalizedText(pkg.DraftPayload.MultiLanguageDescList, "en"), "submit description")
	assertNoSensitivePhrase(t, pkg.DraftPayload.ProductAttributeList[0].Value, "submit free-text attribute")
	if got := pkg.DraftPayload.ProductAttributeList[1].Value; got != "12" {
		t.Fatalf("structured submit attribute value = %q, want unchanged", got)
	}
	assertNoSensitivePhrase(t, pkg.DraftPayload.SKCList[0].SkcName, "submit skc name")
	assertNoSensitivePhrase(t, findLocalizedText(pkg.DraftPayload.SKCList[0].MultiLanguageNameList, "en"), "submit localized skc name")
}

func findLocalizedText(items []offlinepackage.LocalizedText, language string) string {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Language), language) {
			return strings.TrimSpace(item.Name)
		}
	}
	return ""
}
