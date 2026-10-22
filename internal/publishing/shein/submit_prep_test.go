package shein

import (
	"context"
	"strings"
	"testing"

	openaiclient "task-processor/internal/integration/openai"
	"task-processor/internal/product/catalog/canonical"
	common "task-processor/internal/publishing/common"
	sharedtenantctx "task-processor/internal/shared/tenantctx"
	"task-processor/internal/shein/authorizedbrand"
)

type recordingChatCompleter struct {
	response   *openaiclient.ChatCompletionResponse
	lastPrompt string
}

func (s *recordingChatCompleter) Generate(_ context.Context, prompt string) (string, error) {
	s.lastPrompt = prompt
	if s.response == nil || len(s.response.Choices) == 0 {
		return "", nil
	}
	return s.response.Choices[0].Message.Content, nil
}

func TestPromptEntryPointsUseBasePromptWithoutLegacyPolicy(t *testing.T) {
	ai := &recordingChatCompleter{
		response: &openaiclient.ChatCompletionResponse{
			Choices: []openaiclient.ChatCompletionChoice{{
				Message: openaiclient.ChatCompletionMessage{
					Content: `{"title":"Door Curtain","description":"A door curtain."}`,
				},
			}},
		},
	}

	title := extractPromptTitleWithLLM(context.Background(), "Please design a curtain print with text and graphics, 3000px", nil, "Door curtain", ai)
	if title != "Door Curtain" {
		t.Fatalf("title = %q, want base-prompt result", title)
	}
	if ai.lastPrompt == "" {
		t.Fatal("system prompt is empty")
	}
	if strings.Contains(ai.lastPrompt, "Additional tenant content restrictions:") {
		t.Fatalf("system prompt = %q, want base prompt without legacy policy block", ai.lastPrompt)
	}
	ai.response.Choices[0].Message.Content = `{"addition":"Rock Typography Graphic Print"}`
	ai.lastPrompt = ""
	addition := extractListingTitleAdditionWithLLM(
		sharedtenantctx.WithTenantID(context.Background(), "101"), "Door curtain",
		&canonical.Product{Attributes: map[string]canonical.Attribute{
			"picture_request": {Value: "Please design a rock style door curtain"},
		}}, "Door curtain", ai,
	)
	if addition != "Rock Typography Graphic Print" {
		t.Fatalf("addition = %q, want base-prompt result", addition)
	}
	if ai.lastPrompt == "" || strings.Contains(ai.lastPrompt, "Additional tenant content restrictions:") {
		t.Fatalf("addition prompt = %q, want base prompt without legacy policy block", ai.lastPrompt)
	}
}

func TestSanitizeDraftPayloadSensitiveContent_CleansDraftTextFields(t *testing.T) {
	pkg := &Package{
		DraftPayload: &RequestDraft{
			MultiLanguageNameList: []LocalizedText{{Language: "en", Name: "Amazon Trademark Vase"}},
			MultiLanguageDescList: []LocalizedText{{Language: "en", Name: "Amazon Trademark vase for home decor."}},
			ProductAttributeList: []common.Attribute{
				{Name: "Material Detail", Value: "Amazon Trademark acrylic"},
				{Name: "Length", Value: "12"},
			},
			SKCList: []SKCRequestDraft{{
				SkcName:               "Amazon Trademark Blue",
				MultiLanguageNameList: []LocalizedText{{Language: "en", Name: "Amazon Trademark Blue"}},
			}},
		},
	}

	changed := SanitizeDraftPayloadSensitiveContent(pkg, sharedtenantctx.WithTenantID(context.Background(), "101"), nil, nil)
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
	ctx := sharedtenantctx.WithTenantID(authorizedbrand.WithResolved(context.Background(), &authorizedbrand.Resolved{
		Enabled: true,
		Name:    "Amazon Basics",
		NameEn:  "Amazon Basics",
	}), "101")
	pkg := &Package{DraftPayload: &RequestDraft{
		ProductAttributeList: []common.Attribute{{
			Value: "Amazon Basics Trademark acrylic",
		}},
		SKCList: []SKCRequestDraft{{
			SkcName: "Amazon Basics Trademark Blue",
			MultiLanguageNameList: []LocalizedText{{
				Language: "en",
				Name:     "Amazon Basics Trademark Blue",
			}},
		}},
	}}

	if !SanitizeDraftPayloadSensitiveContent(pkg, ctx, nil, nil) {
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
	pkg := &Package{DraftPayload: &RequestDraft{
		MultiLanguageNameList: []LocalizedText{{
			Language: "en",
			Name:     "Amazon Trademark Vase",
		}},
		MultiLanguageDescList: []LocalizedText{{
			Language: "en",
			Name:     "Amazon Trademark vase for home decor.",
		}},
		ProductAttributeList: []common.Attribute{
			{Value: "Amazon Trademark acrylic"},
			{Value: "12"},
		},
		SKCList: []SKCRequestDraft{{
			SkcName: "Amazon Trademark Blue",
			MultiLanguageNameList: []LocalizedText{{
				Language: "en",
				Name:     "Amazon Trademark Blue",
			}},
		}},
	}}

	if !SanitizeDraftPayloadSensitiveContent(pkg, sharedtenantctx.WithTenantID(context.Background(), "101"), nil, nil) {
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

func findLocalizedText(items []LocalizedText, language string) string {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Language), language) {
			return strings.TrimSpace(item.Name)
		}
	}
	return ""
}
