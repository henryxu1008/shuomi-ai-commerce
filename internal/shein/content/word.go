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
