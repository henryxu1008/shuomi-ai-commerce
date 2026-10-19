package shein

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"

	openaiclient "task-processor/internal/integration/openai"
	"task-processor/internal/listingadmin"
	"task-processor/internal/product/catalog/canonical"
	sharedtenantctx "task-processor/internal/shared/tenantctx"
	"task-processor/internal/shein/submitprep"
)

func TestAdminGenerationTopicPolicyPersistenceFlowsIntoPromptAndPreviewSanitizer(t *testing.T) {
	repo := newGenerationTopicPolicyRepo(t)
	createGenerationTopicPolicy(t, repo, "101", "children")
	createGenerationTopicPolicy(t, repo, "202", "meals")

	restorePromptRepo := SetGenerationTopicPolicyRepository(repo)
	defer restorePromptRepo()
	restoreSanitizerRepoWords := submitprep.SetSensitiveWordRepository(nil)
	defer restoreSanitizerRepoWords()
	restoreSanitizerRepoTopics := submitprep.SetGenerationTopicPolicyRepository(repo)
	defer restoreSanitizerRepoTopics()
	restoreSanitizerRepoOverrides := submitprep.SetGenerationTopicOverrideRepository(nil)
	defer restoreSanitizerRepoOverrides()

	ai := &recordingChatCompleter{
		response: &openaiclient.ChatCompletionResponse{
			Choices: []openaiclient.ChatCompletionChoice{{
				Message: openaiclient.ChatCompletionMessage{
					Content: `{"title":"Door Curtain","description":"A door curtain."}`,
				},
			}},
		},
	}

	_ = extractPromptTitleWithLLM(sharedtenantctx.WithTenantID(context.Background(), "101"), "Please design a curtain print with text and graphics, 3000px", nil, "Door curtain", ai)
	if ai.lastPrompt == "" {
		t.Fatal("system prompt is empty")
	}
	systemPrompt := ai.lastPrompt
	if !strings.Contains(systemPrompt, "Do not mention children, babies, or age-specific users.") {
		t.Fatalf("system prompt = %q, want children directive from repository-created policy", systemPrompt)
	}
	if strings.Contains(systemPrompt, "breakfast") || strings.Contains(systemPrompt, "meal") {
		t.Fatalf("system prompt = %q, want tenant 101 prompt to exclude tenant 202 directives", systemPrompt)
	}

	copyA := buildSheinListingCopy(sharedtenantctx.WithTenantID(context.Background(), "101"), &canonical.Product{
		Title:       "Kids Room Curtain",
		Description: "Decor for children bedroom",
		Attributes: map[string]canonical.Attribute{
			"product_english_name": {Value: "Kids Room Curtain"},
		},
	}, "Kids Room Curtain", nil)
	copyB := buildSheinListingCopy(sharedtenantctx.WithTenantID(context.Background(), "202"), &canonical.Product{
		Title:       "Breakfast Table Curtain",
		Description: "Meal-themed decor",
		Attributes: map[string]canonical.Attribute{
			"product_english_name": {Value: "Breakfast Table Curtain"},
		},
	}, "Breakfast Table Curtain", nil)

	if strings.Contains(strings.ToLower(copyA.Title), "kids") || strings.Contains(strings.ToLower(copyA.Description), "children") {
		t.Fatalf("tenant 101 copy = %+v, want children terms removed", copyA)
	}
	if strings.Contains(strings.ToLower(copyB.Title), "breakfast") || strings.Contains(strings.ToLower(copyB.Description), "meal") {
		t.Fatalf("tenant 202 copy = %+v, want meal terms removed", copyB)
	}
}

func TestAdminGenerationTopicOverrideFlowsIntoPromptAndPreviewSanitizer(t *testing.T) {
	policyRepo, overrideRepo := newGenerationTopicRepos(t)
	createGenerationTopicPolicy(t, policyRepo, "101", "children")
	createGenerationTopicOverride(t, overrideRepo, "101", `{
		"platform":"shein",
		"topicKey":"children",
		"additionalPromptDirectives":["Avoid toddler-focused positioning."],
		"additionalLexiconByLanguage":{"en":["toddler"]},
		"status":1
	}`)

	restorePromptPolicyRepo := SetGenerationTopicPolicyRepository(policyRepo)
	defer restorePromptPolicyRepo()
	restorePromptOverrideRepo := SetGenerationTopicOverrideRepository(overrideRepo)
	defer restorePromptOverrideRepo()
	restoreSanitizerReposWords := submitprep.SetSensitiveWordRepository(nil)
	defer restoreSanitizerReposWords()
	restoreSanitizerReposTopics := submitprep.SetGenerationTopicPolicyRepository(policyRepo)
	defer restoreSanitizerReposTopics()
	restoreSanitizerReposOverrides := submitprep.SetGenerationTopicOverrideRepository(overrideRepo)
	defer restoreSanitizerReposOverrides()

	ai := &recordingChatCompleter{
		response: &openaiclient.ChatCompletionResponse{
			Choices: []openaiclient.ChatCompletionChoice{{
				Message: openaiclient.ChatCompletionMessage{
					Content: `{"title":"Door Curtain","description":"A door curtain."}`,
				},
			}},
		},
	}

	_ = extractPromptTitleWithLLM(sharedtenantctx.WithTenantID(context.Background(), "101"), "Please design a curtain print with text and graphics, 3000px", nil, "Door curtain", ai)
	if ai.lastPrompt == "" {
		t.Fatal("system prompt is empty")
	}
	systemPrompt := ai.lastPrompt
	if !strings.Contains(systemPrompt, "Do not mention children, babies, or age-specific users.") {
		t.Fatalf("system prompt = %q, want default directive retained", systemPrompt)
	}
	if !strings.Contains(systemPrompt, "Avoid toddler-focused positioning.") {
		t.Fatalf("system prompt = %q, want override directive included", systemPrompt)
	}

	copy := buildSheinListingCopy(sharedtenantctx.WithTenantID(context.Background(), "101"), &canonical.Product{
		Title:       "Toddler Room Curtain",
		Description: "Toddler decor for children bedroom",
		Attributes: map[string]canonical.Attribute{
			"product_english_name": {Value: "Toddler Room Curtain"},
		},
	}, "Toddler Room Curtain", nil)

	if strings.Contains(strings.ToLower(copy.Title), "toddler") {
		t.Fatalf("tenant copy title = %q, want override lexicon removed", copy.Title)
	}
	if strings.Contains(strings.ToLower(copy.Description), "toddler") {
		t.Fatalf("tenant copy description = %q, want override lexicon removed", copy.Description)
	}
}

func newGenerationTopicPolicyRepo(t *testing.T) listingadmin.GenerationTopicPolicyRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: ":memory:"}, &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := listingadmin.AutoMigrateGenerationTopicPolicyRepository(db); err != nil {
		t.Fatalf("migrate generation topic policy: %v", err)
	}
	return listingadmin.NewGormGenerationTopicPolicyRepository(db)
}

func newGenerationTopicRepos(t *testing.T) (listingadmin.GenerationTopicPolicyRepository, listingadmin.GenerationTopicOverrideRepository) {
	t.Helper()
	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: ":memory:"}, &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := listingadmin.AutoMigrateGenerationTopicPolicyRepository(db); err != nil {
		t.Fatalf("migrate generation topic policy: %v", err)
	}
	if err := listingadmin.AutoMigrateGenerationTopicOverrideRepository(db); err != nil {
		t.Fatalf("migrate generation topic override: %v", err)
	}
	return listingadmin.NewGormGenerationTopicPolicyRepository(db), listingadmin.NewGormGenerationTopicOverrideRepository(db)
}

func createGenerationTopicPolicy(t *testing.T, repo listingadmin.GenerationTopicPolicyRepository, tenantID string, topicKey string) {
	t.Helper()
	tenant, err := strconv.ParseInt(tenantID, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	ctx := listingadmin.WithOwnerUserID(context.Background(), "user-"+tenantID)
	created, err := repo.CreateGenerationTopicPolicy(ctx, &listingadmin.GenerationTopicPolicy{TenantID: tenant, Platform: "shein", TopicKey: topicKey, Status: 1})
	if err != nil {
		t.Fatalf("persist policy: %v", err)
	}
	if created.TopicKey != topicKey || created.Platform != "shein" {
		t.Fatalf("created policy = %+v", created)
	}
}

func createGenerationTopicOverride(t *testing.T, repo listingadmin.GenerationTopicOverrideRepository, tenantID string, payload string) {
	t.Helper()
	var item listingadmin.GenerationTopicOverride
	if err := json.Unmarshal([]byte(payload), &item); err != nil {
		t.Fatal(err)
	}
	tenant, err := strconv.ParseInt(tenantID, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	item.TenantID = tenant
	ctx := listingadmin.WithOwnerUserID(context.Background(), "user-"+tenantID)
	if _, err := repo.CreateGenerationTopicOverride(ctx, &item); err != nil {
		t.Fatalf("persist override: %v", err)
	}
}
