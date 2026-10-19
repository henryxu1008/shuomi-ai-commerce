package shein

import (
	"context"
	"errors"
	"strings"
	"testing"

	openaiclient "task-processor/internal/integration/openai"
	"task-processor/internal/listingadmin"
	"task-processor/internal/product/catalog/canonical"
	common "task-processor/internal/publishing/common"
	sharedtenantctx "task-processor/internal/shared/tenantctx"
	"task-processor/internal/shein/authorizedbrand"
	"task-processor/internal/shein/submitprep"
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

func promptPolicySection(prompt string) string {
	for _, marker := range []string{"\n\nBase title:", "\n\nFallback product title:"} {
		if head, _, ok := strings.Cut(prompt, marker); ok {
			return head
		}
	}
	return prompt
}

type stubSensitiveWordRepository struct {
	pages   map[int64][]listingadmin.SensitiveWord
	created []listingadmin.SensitiveWord
	updated []listingadmin.SensitiveWord
}

func (s *stubSensitiveWordRepository) ListSensitiveWords(ctx context.Context, query listingadmin.SensitiveWordQuery) (*listingadmin.SensitiveWordPage, error) {
	items := append([]listingadmin.SensitiveWord(nil), s.pages[query.TenantID]...)
	if query.Word != "" {
		filtered := items[:0]
		for _, item := range items {
			if strings.Contains(strings.ToLower(item.Word), strings.ToLower(query.Word)) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if query.Status != nil {
		filtered := items[:0]
		for _, item := range items {
			if item.Status == *query.Status {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = len(items)
		if pageSize == 0 {
			pageSize = 1
		}
	}
	start := (page - 1) * pageSize
	if start >= len(items) {
		return &listingadmin.SensitiveWordPage{Items: nil, Total: int64(len(items)), Page: page, PageSize: pageSize}, nil
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return &listingadmin.SensitiveWordPage{
		Items:    append([]listingadmin.SensitiveWord(nil), items[start:end]...),
		Total:    int64(len(items)),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *stubSensitiveWordRepository) GetSensitiveWord(context.Context, int64, int64) (*listingadmin.SensitiveWord, error) {
	return nil, errors.New("not implemented")
}

func (s *stubSensitiveWordRepository) CreateSensitiveWord(_ context.Context, word *listingadmin.SensitiveWord) (*listingadmin.SensitiveWord, error) {
	if word == nil {
		return nil, errors.New("word is nil")
	}
	created := *word
	created.ID = int64(len(s.created) + len(s.pages[created.TenantID]) + 1)
	s.created = append(s.created, created)
	s.pages[created.TenantID] = append(s.pages[created.TenantID], created)
	return &created, nil
}

func (s *stubSensitiveWordRepository) UpdateSensitiveWord(_ context.Context, word *listingadmin.SensitiveWord) (*listingadmin.SensitiveWord, error) {
	if word == nil {
		return nil, errors.New("word is nil")
	}
	updated := *word
	s.updated = append(s.updated, updated)
	items := s.pages[updated.TenantID]
	for i := range items {
		if items[i].ID == updated.ID {
			items[i] = updated
			s.pages[updated.TenantID] = items
			return &updated, nil
		}
	}
	s.pages[updated.TenantID] = append(s.pages[updated.TenantID], updated)
	return &updated, nil
}

func (s *stubSensitiveWordRepository) UpdateSensitiveWordStatus(context.Context, int64, int64, int16, string) (*listingadmin.SensitiveWord, error) {
	return nil, errors.New("not implemented")
}

func (s *stubSensitiveWordRepository) DeleteSensitiveWord(context.Context, int64, int64) error {
	return errors.New("not implemented")
}

func TestExtractListingTitleAdditionWithLLM_IncludesTenantGenerationPolicyText(t *testing.T) {
	restoreRepo := SetGenerationTopicPolicyRepository(&stubGenerationTopicPolicyRepository{
		keys: map[int64][]string{
			101: []string{"children", "rock", "baby"},
		},
	})
	defer restoreRepo()

	ai := &recordingChatCompleter{
		response: &openaiclient.ChatCompletionResponse{
			Choices: []openaiclient.ChatCompletionChoice{{
				Message: openaiclient.ChatCompletionMessage{
					Content: `{"addition":"Rock Typography Graphic Print"}`,
				},
			}},
		},
	}

	addition := extractListingTitleAdditionWithLLM(
		sharedtenantctx.WithTenantID(context.Background(), "101"),
		"Door curtain",
		&canonical.Product{
			Attributes: map[string]canonical.Attribute{
				"ai_style":        {Value: "Please design a rock style door curtain"},
				"picture_request": {Value: "Please design a rock print"},
			},
		},
		"Door curtain",
		ai,
	)
	if addition != "Rock Typography Graphic Print" {
		t.Fatalf("addition = %q, want extracted addition", addition)
	}
	if ai.lastPrompt == "" {
		t.Fatal("ai prompt is empty, want system prompt")
	}
	systemPrompt := promptPolicySection(ai.lastPrompt)
	if !strings.Contains(systemPrompt, "Additional tenant content restrictions:") {
		t.Fatalf("system prompt = %q, want tenant policy header", systemPrompt)
	}
	if !strings.Contains(systemPrompt, "Do not mention children, babies, or age-specific users.") {
		t.Fatalf("system prompt = %q, want children directive", systemPrompt)
	}
	if !strings.Contains(systemPrompt, "Do not mention babies, newborns, or infant-specific usage.") {
		t.Fatalf("system prompt = %q, want baby directive", systemPrompt)
	}
	if strings.Contains(systemPrompt, "rock") {
		t.Fatalf("system prompt = %q, want unknown topic keys omitted", systemPrompt)
	}
}

func TestExtractPromptTitleWithLLM_IncludesTenantGenerationPolicyText(t *testing.T) {
	restoreRepo := SetGenerationTopicPolicyRepository(&stubGenerationTopicPolicyRepository{
		keys: map[int64][]string{
			101: []string{"children", "rock", "baby"},
		},
	})
	defer restoreRepo()

	ai := &recordingChatCompleter{
		response: &openaiclient.ChatCompletionResponse{
			Choices: []openaiclient.ChatCompletionChoice{{
				Message: openaiclient.ChatCompletionMessage{
					Content: `{"title":"Floral Door Curtain"}`,
				},
			}},
		},
	}

	title := extractPromptTitleWithLLM(
		sharedtenantctx.WithTenantID(context.Background(), "101"),
		"Please design a floral door curtain print with dramatic text and graphics, 3000px",
		nil,
		"Door curtain",
		ai,
	)
	if title != "Floral Door Curtain" {
		t.Fatalf("title = %q, want extracted title", title)
	}
	if ai.lastPrompt == "" {
		t.Fatal("ai prompt is empty, want system prompt")
	}
	systemPrompt := promptPolicySection(ai.lastPrompt)
	if !strings.Contains(systemPrompt, "Additional tenant content restrictions:") {
		t.Fatalf("system prompt = %q, want tenant policy header", systemPrompt)
	}
	if !strings.Contains(systemPrompt, "Do not mention children, babies, or age-specific users.") {
		t.Fatalf("system prompt = %q, want children directive", systemPrompt)
	}
	if !strings.Contains(systemPrompt, "Do not mention babies, newborns, or infant-specific usage.") {
		t.Fatalf("system prompt = %q, want baby directive", systemPrompt)
	}
	if strings.Contains(systemPrompt, "rock") {
		t.Fatalf("system prompt = %q, want unknown topic keys omitted", systemPrompt)
	}
}

func TestPromptEntryPointsOmitTenantPolicyWithoutTenantContext(t *testing.T) {
	restoreRepo := SetGenerationTopicPolicyRepository(&stubGenerationTopicPolicyRepository{
		keys: map[int64][]string{
			101: []string{"children"},
		},
	})
	defer restoreRepo()

	ai := &recordingChatCompleter{
		response: &openaiclient.ChatCompletionResponse{
			Choices: []openaiclient.ChatCompletionChoice{{
				Message: openaiclient.ChatCompletionMessage{
					Content: `{"title":"Door Curtain","description":"A door curtain."}`,
				},
			}},
		},
	}

	_ = extractPromptTitleWithLLM(context.Background(), "Please design a curtain print with text and graphics, 3000px", nil, "Door curtain", ai)
	if ai.lastPrompt == "" {
		t.Fatal("system prompt is empty")
	}
	if strings.Contains(ai.lastPrompt, "Additional tenant content restrictions:") {
		t.Fatalf("system prompt = %q, want no tenant policy block without tenant context", ai.lastPrompt)
	}
}

func TestSanitizeDraftPayloadSensitiveContent_CleansDraftTextFields(t *testing.T) {
	restoreRepo := submitprep.SetSensitiveWordRepository(&stubSensitiveWordRepository{
		pages: map[int64][]listingadmin.SensitiveWord{
			101: {
				{TenantID: 101, Language: "en", Word: "amazon", Status: 1},
				{TenantID: 101, Language: "en", Word: "bpa free", Status: 1},
			},
		},
	})
	defer restoreRepo()

	pkg := &Package{
		DraftPayload: &RequestDraft{
			MultiLanguageNameList: []LocalizedText{{Language: "en", Name: "Amazon BPA Free Vase"}},
			MultiLanguageDescList: []LocalizedText{{Language: "en", Name: "Amazon BPA Free vase for home decor."}},
			ProductAttributeList: []common.Attribute{
				{Name: "Material Detail", Value: "Amazon BPA Free acrylic"},
				{Name: "Length", Value: "12"},
			},
			SKCList: []SKCRequestDraft{{
				SkcName:               "Amazon BPA Free Blue",
				MultiLanguageNameList: []LocalizedText{{Language: "en", Name: "Amazon BPA Free Blue"}},
			}},
		},
	}

	changed := SanitizeDraftPayloadSensitiveContent(pkg, sharedtenantctx.WithTenantID(context.Background(), "101"), nil)
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

func TestSanitizeDraftPayloadSensitiveContent_LoadsTenantSensitiveWordsFromRepository(t *testing.T) {
	restoreRepo := submitprep.SetSensitiveWordRepository(&stubSensitiveWordRepository{
		pages: map[int64][]listingadmin.SensitiveWord{
			101: {{
				TenantID: 101,
				Language: "en",
				Word:     "whimsy",
				Status:   1,
			}},
		},
	})
	defer restoreRepo()

	ctx := sharedtenantctx.WithTenantID(context.Background(), "101")
	pkg := &Package{DraftPayload: &RequestDraft{
		MultiLanguageNameList: []LocalizedText{{
			Language: "en",
			Name:     "Whimsy Door Curtain",
		}},
		MultiLanguageDescList: []LocalizedText{{
			Language: "en",
			Name:     "Whimsy curtain for home decor.",
		}},
		ProductAttributeList: []common.Attribute{{
			Value: "Whimsy fabric finish",
		}},
		SKCList: []SKCRequestDraft{{
			SkcName: "Whimsy White",
			MultiLanguageNameList: []LocalizedText{{
				Language: "en",
				Name:     "Whimsy White",
			}},
		}},
	}}

	if !SanitizeDraftPayloadSensitiveContent(pkg, ctx, nil) {
		t.Fatal("sensitive content was not sanitized")
	}

	if strings.Contains(strings.ToLower(findLocalizedText(pkg.DraftPayload.MultiLanguageNameList, "en")), "whimsy") {
		t.Fatalf("english title still contains tenant sensitive word: %+v", pkg.DraftPayload.MultiLanguageNameList)
	}
	if strings.Contains(strings.ToLower(findLocalizedText(pkg.DraftPayload.MultiLanguageDescList, "en")), "whimsy") {
		t.Fatalf("english description still contains tenant sensitive word: %+v", pkg.DraftPayload.MultiLanguageDescList)
	}
	if strings.Contains(strings.ToLower(pkg.DraftPayload.ProductAttributeList[0].Value), "whimsy") {
		t.Fatalf("attribute still contains tenant sensitive word: %+v", pkg.DraftPayload.ProductAttributeList)
	}
	if strings.Contains(strings.ToLower(findLocalizedText(pkg.DraftPayload.SKCList[0].MultiLanguageNameList, "en")), "whimsy") {
		t.Fatalf("skc still contains tenant sensitive word: %+v", pkg.DraftPayload.SKCList[0].MultiLanguageNameList)
	}
}

func TestSanitizeDraftPayloadSensitiveContent_PreservesAuthorizedBrandFromRuntimeContext(t *testing.T) {
	restoreRepo := submitprep.SetSensitiveWordRepository(&stubSensitiveWordRepository{
		pages: map[int64][]listingadmin.SensitiveWord{
			101: {
				{TenantID: 101, Language: "en", Word: "bpa free", Status: 1},
			},
		},
	})
	defer restoreRepo()

	ctx := sharedtenantctx.WithTenantID(authorizedbrand.WithResolved(context.Background(), &authorizedbrand.Resolved{
		Enabled: true,
		Name:    "Amazon Basics",
		NameEn:  "Amazon Basics",
	}), "101")
	pkg := &Package{DraftPayload: &RequestDraft{
		ProductAttributeList: []common.Attribute{{
			Value: "Amazon Basics BPA Free acrylic",
		}},
		SKCList: []SKCRequestDraft{{
			SkcName: "Amazon Basics BPA Free Blue",
			MultiLanguageNameList: []LocalizedText{{
				Language: "en",
				Name:     "Amazon Basics BPA Free Blue",
			}},
		}},
	}}

	if !SanitizeDraftPayloadSensitiveContent(pkg, ctx, nil) {
		t.Fatal("sensitive content was not sanitized")
	}

	if strings.Contains(strings.ToLower(pkg.DraftPayload.ProductAttributeList[0].Value), "bpa free") {
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

func TestSanitizeDraftPayloadSensitiveContent_LoadsTenantGenerationTopicOverrideLexicon(t *testing.T) {
	restoreReposWords := submitprep.SetSensitiveWordRepository(nil)
	defer restoreReposWords()
	restoreReposTopics := submitprep.SetGenerationTopicPolicyRepository(&stubGenerationTopicPolicyRepository{
		keys: map[int64][]string{
			101: {"children"},
		},
	})
	defer restoreReposTopics()
	restoreReposOverrides := submitprep.SetGenerationTopicOverrideRepository(&stubGenerationTopicOverrideRepository{
		items: map[string]listingadmin.GenerationTopicOverride{
			overrideRepoKey(101, "shein", "children"): {
				TenantID: 101,
				Platform: "shein",
				TopicKey: "children",
				AdditionalLexiconByLanguage: map[string][]string{
					"en": {"toddler"},
				},
				Status: 1,
			},
		},
	})
	defer restoreReposOverrides()

	ctx := sharedtenantctx.WithTenantID(context.Background(), "101")
	pkg := &Package{DraftPayload: &RequestDraft{
		MultiLanguageNameList: []LocalizedText{{
			Language: "en",
			Name:     "Toddler Door Curtain",
		}},
		MultiLanguageDescList: []LocalizedText{{
			Language: "en",
			Name:     "Toddler decor for bedrooms.",
		}},
		SKCList: []SKCRequestDraft{{
			SkcName: "Toddler White",
			MultiLanguageNameList: []LocalizedText{{
				Language: "en",
				Name:     "Toddler White",
			}},
		}},
	}}

	if !SanitizeDraftPayloadSensitiveContent(pkg, ctx, nil) {
		t.Fatal("sensitive content was not sanitized")
	}
	if strings.Contains(strings.ToLower(findLocalizedText(pkg.DraftPayload.MultiLanguageNameList, "en")), "toddler") {
		t.Fatalf("english title still contains override lexicon: %+v", pkg.DraftPayload.MultiLanguageNameList)
	}
	if strings.Contains(strings.ToLower(findLocalizedText(pkg.DraftPayload.MultiLanguageDescList, "en")), "toddler") {
		t.Fatalf("english description still contains override lexicon: %+v", pkg.DraftPayload.MultiLanguageDescList)
	}
	if strings.Contains(strings.ToLower(findLocalizedText(pkg.DraftPayload.SKCList[0].MultiLanguageNameList, "en")), "toddler") {
		t.Fatalf("skc still contains override lexicon: %+v", pkg.DraftPayload.SKCList[0].MultiLanguageNameList)
	}
}

func TestSanitizeDraftPayloadSensitiveContent_CleansFreeTextAttributesAndSKCNames(t *testing.T) {
	restoreRepo := submitprep.SetSensitiveWordRepository(&stubSensitiveWordRepository{
		pages: map[int64][]listingadmin.SensitiveWord{
			101: {
				{TenantID: 101, Language: "en", Word: "amazon", Status: 1},
				{TenantID: 101, Language: "en", Word: "bpa free", Status: 1},
			},
		},
	})
	defer restoreRepo()

	pkg := &Package{DraftPayload: &RequestDraft{
		MultiLanguageNameList: []LocalizedText{{
			Language: "en",
			Name:     "Amazon BPA Free Vase",
		}},
		MultiLanguageDescList: []LocalizedText{{
			Language: "en",
			Name:     "Amazon BPA Free vase for home decor.",
		}},
		ProductAttributeList: []common.Attribute{
			{Value: "Amazon BPA Free acrylic"},
			{Value: "12"},
		},
		SKCList: []SKCRequestDraft{{
			SkcName: "Amazon BPA Free Blue",
			MultiLanguageNameList: []LocalizedText{{
				Language: "en",
				Name:     "Amazon BPA Free Blue",
			}},
		}},
	}}

	if !SanitizeDraftPayloadSensitiveContent(pkg, sharedtenantctx.WithTenantID(context.Background(), "101"), nil) {
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
