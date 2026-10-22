package config

import "strings"

// PlatformsConfig 平台配置
type PlatformsConfig struct {
	Temu        PlatformConfig    `mapstructure:"temu" yaml:"temu"`               // TEMU平台配置
	Shein       PlatformConfig    `mapstructure:"shein" yaml:"shein"`             // SHEIN平台配置
	SDS         SDSPlatformConfig `mapstructure:"sds" yaml:"sds"`                 // SDS平台配置
	Alibaba1688 Alibaba1688Config `mapstructure:"alibaba1688" yaml:"alibaba1688"` // 1688平台配置
}

func (c *Config) EffectiveSDSAuthRedis() RedisConfig {
	if c == nil {
		return RedisConfig{}
	}
	authRedis := c.Platforms.SDS.AuthRedis
	if strings.TrimSpace(authRedis.Host) == "" && c.Redis != nil {
		authRedis = *c.Redis
	}
	return authRedis
}

// PlatformConfig 单个平台的完整配置
type PlatformConfig struct {
	HTTPClient HTTPClientConfig `yaml:"httpClient"`
}

type SDSPlatformConfig struct {
	LoginService  SDSLoginServiceConfig  `mapstructure:"loginService" yaml:"loginService"`   // SDS 统一登录服务配置
	AuthRedis     RedisConfig            `mapstructure:"authRedis" yaml:"authRedis"`         // SDS 登录态共享 Redis 配置
	AuthBootstrap SDSAuthBootstrapConfig `mapstructure:"authBootstrap" yaml:"authBootstrap"` // SDS 静态认证引导配置
}

type SDSAuthBootstrapConfig struct {
	StaticAccessToken       string `mapstructure:"staticAccessToken" yaml:"staticAccessToken"`
	StaticOutToken          string `mapstructure:"staticOutToken" yaml:"staticOutToken"`
	StaticMerchantID        int64  `mapstructure:"staticMerchantID" yaml:"staticMerchantID"`
	StaticCookie            string `mapstructure:"staticCookie" yaml:"staticCookie"`
	LoginDomainName         string `mapstructure:"loginDomainName" yaml:"loginDomainName"`
	LoginVerifyCaptchaParam string `mapstructure:"loginVerifyCaptchaParam" yaml:"loginVerifyCaptchaParam"`
	LoginExtraInfo          string `mapstructure:"loginExtraInfo" yaml:"loginExtraInfo"`
}

type LoginServiceConfig struct {
	BaseURL             string `mapstructure:"baseURL" yaml:"baseURL"`           // login 服务地址，例如 http://login:8000
	SharedKey           string `mapstructure:"sharedKey" yaml:"sharedKey"`       // 与 login 服务 LOGIN_INTERNAL_SHARED_KEY 对齐
	TenantID            string `mapstructure:"tenantID" yaml:"tenantID"`         // 可选兼容回退；运行时按目标租户和店铺解析
	Identifier          string `mapstructure:"identifier" yaml:"identifier"`     // 可选兼容回退；运行时按目标店铺解析，通常对应 listing_store.id
	MerchantName        string `mapstructure:"merchantName" yaml:"merchantName"` // SDS 商户号
	Username            string `mapstructure:"username" yaml:"username"`         // SDS 登录用户名
	Password            string `mapstructure:"password" yaml:"password"`         // SDS 登录密码
	MaxConcurrentLogins int    `mapstructure:"maxConcurrentLogins" yaml:"maxConcurrentLogins"`
	ProfileRootDir      string `mapstructure:"profileRootDir" yaml:"profileRootDir"`
	ArtifactDir         string `mapstructure:"artifactDir" yaml:"artifactDir"`
	DefaultHeadless     bool   `mapstructure:"defaultHeadless" yaml:"defaultHeadless"`
	ForceHeadless       bool   `mapstructure:"forceHeadless" yaml:"forceHeadless"`
	// ExecutionMode selects where browser login runs. "inline" preserves the
	// historical HTTP-process execution; "worker" dispatches to the dedicated
	// SHEIN login worker.
	ExecutionMode       string `mapstructure:"executionMode" yaml:"executionMode"`
	CloakBrowserEnabled bool   `mapstructure:"cloakBrowserEnabled" yaml:"cloakBrowserEnabled"`
	CloakBrowserPath    string `mapstructure:"cloakBrowserPath" yaml:"cloakBrowserPath"`
}

type SDSLoginServiceConfig = LoginServiceConfig

// Alibaba1688Config 1688平台配置
type Alibaba1688Config struct {
	Enabled        bool   `yaml:"enabled"`  // 是否启用1688处理器
	Timeout        int    `yaml:"timeout"`  // 处理超时时间（秒）
	PoolSize       int    `yaml:"poolSize"` // 浏览器池大小
	ProfileRootDir string `mapstructure:"profileRootDir" yaml:"profileRootDir"`
}
