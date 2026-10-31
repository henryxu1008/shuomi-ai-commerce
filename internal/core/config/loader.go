package config

func NewDefaultConfig() *Config {
	return &Config{
		OpenAI: OpenAIConfig{
			APIKey:  "",
			Model:   "gemini-2.0-flash",
			BaseURL: "https://ai.linkcloudai.com/v1",
			Timeout: 120,
		},
		ImageAgent: ImageAgentConfig{},
		ListingKit: ListingKitConfig{
			Zitadel: ListingKitZitadelConfig{},
		},
	}
}
