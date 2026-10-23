// Package content 提供SHEIN平台的敏感词服务核心功能
package content

import "github.com/sirupsen/logrus"

// NewSensitiveWordServiceInMemory 创建纯内存敏感词服务，不依赖本地配置文件。
func NewSensitiveWordServiceInMemory(logs func(string) *logrus.Entry) *SensitiveWordService {
	service := &SensitiveWordService{logger: componentLogger(logs, "shein.sensitive_word")}
	service.initDefaultConfig()
	return service
}

// initDefaultConfig 初始化默认配置
func (s *SensitiveWordService) initDefaultConfig() {
	s.config = s.createDefaultConfig()
	s.logger.Debug("✅ 已初始化默认敏感词配置")
}

// ProcessProductData 处理产品数据中的敏感词
func (s *SensitiveWordService) ProcessProductData(ctx *CleanupContext) error {
	s.logger.Info("🔍 开始敏感词处理（删除模式）...")

	if ctx == nil || ctx.ProductData == nil {
		s.logger.Warn("⚠️ 上下文或产品数据为空，跳过敏感词处理")
		return nil
	}

	processedCount := 0

	if ctx.ProductData.MultiLanguageNameList != nil {
		processedCount += s.processMultiLanguageNames(ctx.ProductData.MultiLanguageNameList)
	}
	if ctx.ProductData.MultiLanguageDescList != nil {
		processedCount += s.processMultiLanguageDescs(ctx.ProductData.MultiLanguageDescList)
	}
	if ctx.ProductData.ProductAttributeList != nil {
		processedCount += s.processProductAttributes(ctx, ctx.ProductData.ProductAttributeList)
	}
	if ctx.ProductData.SKCList != nil {
		processedCount += s.processSKCData(ctx, ctx.ProductData.SKCList)
	}

	s.logSensitiveWordStats()
	s.logger.Infof("✅ 敏感词处理完成，共处理了 %d 个字段", processedCount)
	return nil
}

// HandleValidationErrors 处理验证错误中的敏感词
func (s *SensitiveWordService) HandleValidationErrors(ctx *CleanupContext, validationResults []ValidationResult) bool {
	s.logger.Info("🔍 开始处理验证错误中的敏感词（按语言分类模式）...")

	extractedWords := s.extractSensitiveWordsFromValidation(validationResults)
	if len(extractedWords) == 0 {
		s.logger.Info("未发现敏感词错误，无需重试")
		return false
	}

	s.logger.Infof("从验证错误中提取到敏感词: %v", extractedWords)
	s.AddDynamicSensitiveWords(extractedWords)

	s.mutex.Lock()
	s.markConfigUpdatedLocked()
	s.mutex.Unlock()

	if err := s.ProcessProductData(ctx); err != nil {
		s.logger.Errorf("重新处理产品数据失败: %v", err)
		return false
	}

	s.logger.Infof("✅ 敏感词处理完成，发现 %d 个新敏感词", len(extractedWords))
	return true
}

// getAllSensitiveWords 获取所有敏感词列表（静态 + 动态）
func (s *SensitiveWordService) getAllSensitiveWords() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var allWords []string
	for _, words := range s.config.StaticWords {
		allWords = append(allWords, words...)
	}
	for _, words := range s.config.DynamicWords {
		allWords = append(allWords, words...)
	}
	return s.deduplicateWords(allWords)
}

// GetStaticSensitiveWords 获取当前静态敏感词列表（所有语言）
func (s *SensitiveWordService) GetStaticSensitiveWords() []string {
	return s.getWordsFromConfig(func() map[string][]string { return s.config.StaticWords })
}

// GetDynamicSensitiveWords 获取当前动态敏感词列表（所有语言）
func (s *SensitiveWordService) GetDynamicSensitiveWords() []string {
	return s.getWordsFromConfig(func() map[string][]string { return s.config.DynamicWords })
}

// getWordsFromConfig 通用：从配置中收集所有语言的词并去重
func (s *SensitiveWordService) getWordsFromConfig(getMap func() map[string][]string) []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var words []string
	for _, langWords := range getMap() {
		words = append(words, langWords...)
	}
	return s.deduplicateWords(words)
}

// GetSensitiveWordsByLanguage 按语言获取敏感词列表
func (s *SensitiveWordService) GetSensitiveWordsByLanguage(language string) (static []string, dynamic []string) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	static = make([]string, len(s.config.StaticWords[language]))
	copy(static, s.config.StaticWords[language])
	dynamic = make([]string, len(s.config.DynamicWords[language]))
	copy(dynamic, s.config.DynamicWords[language])
	return static, dynamic
}

// AddDynamicSensitiveWords 添加动态敏感词（自动检测语言）
func (s *SensitiveWordService) AddDynamicSensitiveWords(words []string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.addSensitiveWordsLocked(words, s.config.DynamicWords, "动态")
}

// AddStaticSensitiveWords 添加静态敏感词（自动检测语言）
func (s *SensitiveWordService) AddStaticSensitiveWords(words []string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.addSensitiveWordsLocked(words, s.config.StaticWords, "静态")
}

// AddStaticSensitiveWordsByLanguage 按指定语言添加静态敏感词
func (s *SensitiveWordService) AddStaticSensitiveWordsByLanguage(language string, words []string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.addWordsByLanguageLocked(s.config.StaticWords, language, words, "静态")
}

// addSensitiveWordsLocked 按语言分类后添加；调用者持有原 mutex 写锁。
func (s *SensitiveWordService) addSensitiveWordsLocked(words []string, target map[string][]string, kind string) {
	if len(words) == 0 {
		return
	}
	wordsByLang := s.classifyWordsByLanguage(words)
	if totalAdded := s.addWordsToConfig(target, wordsByLang, kind); totalAdded > 0 {
		s.logger.Infof("✅ 成功添加 %d 个%s敏感词", totalAdded, kind)
		s.markConfigUpdatedLocked()
	}
}

// AddDynamicSensitiveWordsByLanguage 按指定语言添加动态敏感词
func (s *SensitiveWordService) AddDynamicSensitiveWordsByLanguage(language string, words []string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.addWordsByLanguageLocked(s.config.DynamicWords, language, words, "动态")
}

// ClearDynamicSensitiveWords 清空动态敏感词列表
func (s *SensitiveWordService) ClearDynamicSensitiveWords() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	oldCount := s.countWordsInConfig(s.config.DynamicWords)
	s.config.DynamicWords = make(map[string][]string)
	s.logger.Infof("✅ 已清空 %d 个动态敏感词", oldCount)
	s.markConfigUpdatedLocked()
}
