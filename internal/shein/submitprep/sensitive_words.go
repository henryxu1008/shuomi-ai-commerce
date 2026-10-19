package submitprep

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"task-processor/internal/listingadmin"
	"task-processor/internal/shared/tenantctx"
	sheincontent "task-processor/internal/shein/content"
	generationtopics "task-processor/internal/shein/generationtopics"
)

var (
	sensitiveWordRepoMu           sync.RWMutex
	sensitiveWordRepo             listingadmin.SensitiveWordRepository
	generationTopicRepoMu         sync.RWMutex
	generationTopicRepo           listingadmin.GenerationTopicPolicyRepository
	generationTopicOverrideRepoMu sync.RWMutex
	generationTopicOverrideRepo   listingadmin.GenerationTopicOverrideRepository
)

func NewSensitiveWordService() *sheincontent.SensitiveWordService {
	return NewSensitiveWordServiceForContext(context.Background())
}

func NewSensitiveWordServiceForContext(ctx context.Context) *sheincontent.SensitiveWordService {
	service := sheincontent.NewSensitiveWordServiceInMemory()
	overlaySensitiveWordsFromRepository(ctx, service)
	overlayGenerationTopicLexicons(ctx, service)
	return service
}

func SetSensitiveWordRepository(repo listingadmin.SensitiveWordRepository) func() {
	sensitiveWordRepoMu.Lock()
	previous := sensitiveWordRepo
	sensitiveWordRepo = repo
	sensitiveWordRepoMu.Unlock()
	return func() {
		sensitiveWordRepoMu.Lock()
		sensitiveWordRepo = previous
		sensitiveWordRepoMu.Unlock()
	}
}

func currentSensitiveWordRepository() listingadmin.SensitiveWordRepository {
	sensitiveWordRepoMu.RLock()
	repo := sensitiveWordRepo
	sensitiveWordRepoMu.RUnlock()
	return repo
}

func SetGenerationTopicPolicyRepository(repo listingadmin.GenerationTopicPolicyRepository) func() {
	generationTopicRepoMu.Lock()
	previous := generationTopicRepo
	generationTopicRepo = repo
	generationTopicRepoMu.Unlock()
	return func() {
		generationTopicRepoMu.Lock()
		generationTopicRepo = previous
		generationTopicRepoMu.Unlock()
	}
}

func currentGenerationTopicPolicyRepository() listingadmin.GenerationTopicPolicyRepository {
	generationTopicRepoMu.RLock()
	repo := generationTopicRepo
	generationTopicRepoMu.RUnlock()
	return repo
}

func SetGenerationTopicOverrideRepository(repo listingadmin.GenerationTopicOverrideRepository) func() {
	generationTopicOverrideRepoMu.Lock()
	previous := generationTopicOverrideRepo
	generationTopicOverrideRepo = repo
	generationTopicOverrideRepoMu.Unlock()
	return func() {
		generationTopicOverrideRepoMu.Lock()
		generationTopicOverrideRepo = previous
		generationTopicOverrideRepoMu.Unlock()
	}
}

func currentGenerationTopicOverrideRepository() listingadmin.GenerationTopicOverrideRepository {
	generationTopicOverrideRepoMu.RLock()
	repo := generationTopicOverrideRepo
	generationTopicOverrideRepoMu.RUnlock()
	return repo
}

func overlaySensitiveWordsFromRepository(ctx context.Context, service *sheincontent.SensitiveWordService) {
	if ctx == nil || service == nil {
		return
	}
	repo := currentSensitiveWordRepository()
	if repo == nil {
		return
	}
	tenantID, ok := tenantIDFromContext(ctx)
	if !ok {
		return
	}
	wordsByLanguage, err := loadTenantSensitiveWords(ctx, repo, tenantID)
	if err != nil || len(wordsByLanguage) == 0 {
		return
	}
	for language, words := range wordsByLanguage {
		service.AddStaticSensitiveWordsByLanguage(language, words)
	}
}

func overlayGenerationTopicLexicons(ctx context.Context, service *sheincontent.SensitiveWordService) {
	if ctx == nil || service == nil {
		return
	}
	repo := currentGenerationTopicPolicyRepository()
	if repo == nil {
		return
	}
	tenantID, ok := tenantIDFromContext(ctx)
	if !ok {
		return
	}
	keys, err := repo.ListEnabledTopicKeys(ctx, tenantID, "shein")
	if err != nil || len(keys) == 0 {
		return
	}
	overrideRepo := currentGenerationTopicOverrideRepository()
	var definitions []generationtopics.Definition
	if overrideRepo == nil {
		definitions, _ = generationtopics.ResolveSheinTopicKeys(keys)
	} else {
		definitions = generationtopics.ResolveSheinTopicDefinitionsWithOverlay(keys, func(topicKey string) (generationtopics.DefinitionOverlay, error) {
			override, err := overrideRepo.GetGenerationTopicOverrideByTopicKey(ctx, tenantID, "shein", topicKey)
			if err != nil || override == nil || override.Status != 1 {
				return generationtopics.DefinitionOverlay{}, err
			}
			return generationtopics.DefinitionOverlay{
				Enabled:           true,
				PromptDirectives:  override.AdditionalPromptDirectives,
				LexiconByLanguage: override.AdditionalLexiconByLanguage,
			}, nil
		})
	}
	if len(definitions) == 0 {
		return
	}
	for language, words := range generationtopics.CollectLexiconsFromDefinitions(definitions) {
		service.AddStaticSensitiveWordsByLanguage(language, words)
	}
}

func tenantIDFromContext(ctx context.Context) (int64, bool) {
	tenantID, ok := tenantctx.TenantScopeFromContext(ctx)
	if !ok {
		return 0, false
	}
	if tenantID = strings.TrimSpace(tenantID); tenantID == "" || tenantID == tenantctx.DefaultTenantID {
		return 0, false
	}
	parsed, err := strconv.ParseInt(tenantID, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

func loadTenantSensitiveWords(ctx context.Context, repo listingadmin.SensitiveWordRepository, tenantID int64) (map[string][]string, error) {
	records, err := loadTenantSensitiveWordRecords(ctx, repo, tenantID, int16Ptr(1))
	if err != nil {
		return nil, err
	}
	wordsByLanguage := make(map[string][]string)
	seenByLanguage := make(map[string]map[string]struct{})
	for _, item := range records {
		language := strings.TrimSpace(item.Language)
		word := strings.TrimSpace(item.Word)
		if language == "" || word == "" {
			continue
		}
		languageSeen := seenByLanguage[language]
		if languageSeen == nil {
			languageSeen = make(map[string]struct{})
			seenByLanguage[language] = languageSeen
		}
		key := strings.ToLower(word)
		if _, exists := languageSeen[key]; exists {
			continue
		}
		languageSeen[key] = struct{}{}
		wordsByLanguage[language] = append(wordsByLanguage[language], word)
	}
	return wordsByLanguage, nil
}

func loadTenantSensitiveWordRecords(ctx context.Context, repo listingadmin.SensitiveWordRepository, tenantID int64, status *int16) ([]listingadmin.SensitiveWord, error) {
	const pageSize = 200
	page := 1
	records := make([]listingadmin.SensitiveWord, 0, pageSize)
	for {
		result, err := repo.ListSensitiveWords(ctx, listingadmin.SensitiveWordQuery{
			TenantID: tenantID,
			Page:     page,
			PageSize: pageSize,
			Status:   status,
		})
		if err != nil {
			return nil, err
		}
		if result == nil || len(result.Items) == 0 {
			return records, nil
		}
		records = append(records, result.Items...)
		if len(result.Items) < result.PageSize || int64(result.Page*result.PageSize) >= result.Total {
			return records, nil
		}
		page++
	}
}

func int16Ptr(value int16) *int16 {
	return &value
}
