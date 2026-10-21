package config

import "github.com/spf13/viper"

func BuildPlatformConfig(v *viper.Viper, prefix string) PlatformConfig {
	cfg := PlatformConfig{
		FetchMode:  v.GetString(prefix + ".fetchMode"),
		HTTPClient: BuildHTTPClientConfig(v, prefix+".httpClient"),
	}

	return cfg
}

func BuildLoginServiceConfig(v *viper.Viper, prefix string) LoginServiceConfig {
	return LoginServiceConfig{
		BaseURL:             v.GetString(prefix + ".baseURL"),
		SharedKey:           v.GetString(prefix + ".sharedKey"),
		TenantID:            v.GetString(prefix + ".tenantID"),
		Identifier:          v.GetString(prefix + ".identifier"),
		MerchantName:        v.GetString(prefix + ".merchantName"),
		Username:            v.GetString(prefix + ".username"),
		Password:            v.GetString(prefix + ".password"),
		MaxConcurrentLogins: v.GetInt(prefix + ".maxConcurrentLogins"),
		ProfileRootDir:      v.GetString(prefix + ".profileRootDir"),
		ArtifactDir:         v.GetString(prefix + ".artifactDir"),
		DefaultHeadless:     v.GetBool(prefix + ".defaultHeadless"),
		ForceHeadless:       v.GetBool(prefix + ".forceHeadless"),
		ExecutionMode:       v.GetString(prefix + ".executionMode"),
	}
}

func BuildSDSAuthBootstrapConfig(v *viper.Viper, prefix string) SDSAuthBootstrapConfig {
	return SDSAuthBootstrapConfig{
		StaticAccessToken:       v.GetString(prefix + ".staticAccessToken"),
		StaticOutToken:          v.GetString(prefix + ".staticOutToken"),
		StaticMerchantID:        v.GetInt64(prefix + ".staticMerchantID"),
		StaticCookie:            v.GetString(prefix + ".staticCookie"),
		LoginDomainName:         v.GetString(prefix + ".loginDomainName"),
		LoginVerifyCaptchaParam: v.GetString(prefix + ".loginVerifyCaptchaParam"),
		LoginExtraInfo:          v.GetString(prefix + ".loginExtraInfo"),
	}
}
