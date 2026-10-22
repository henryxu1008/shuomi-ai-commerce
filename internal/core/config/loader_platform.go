package config

import "github.com/spf13/viper"

func BuildPlatformConfig(v *viper.Viper, prefix string) PlatformConfig {
	cfg := PlatformConfig{
		HTTPClient: BuildHTTPClientConfig(v, prefix+".httpClient"),
	}

	return cfg
}
