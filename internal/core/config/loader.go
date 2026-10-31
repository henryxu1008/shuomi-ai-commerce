package config

func NewDefaultConfig() *Config {
	return &Config{
		ImageAgent: ImageAgentConfig{},
		ListingKit: ListingKitConfig{
			Zitadel: ListingKitZitadelConfig{},
		},
	}
}
