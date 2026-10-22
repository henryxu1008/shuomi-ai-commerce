package config

// PlatformsConfig 平台配置
type PlatformsConfig struct {
	Temu        PlatformConfig    `mapstructure:"temu" yaml:"temu"`               // TEMU平台配置
	Shein       PlatformConfig    `mapstructure:"shein" yaml:"shein"`             // SHEIN平台配置
	Alibaba1688 Alibaba1688Config `mapstructure:"alibaba1688" yaml:"alibaba1688"` // 1688平台配置
}

// PlatformConfig 单个平台的完整配置
type PlatformConfig struct {
	HTTPClient HTTPClientConfig `yaml:"httpClient"`
}

// Alibaba1688Config 1688平台配置
type Alibaba1688Config struct {
	Enabled        bool   `yaml:"enabled"`  // 是否启用1688处理器
	Timeout        int    `yaml:"timeout"`  // 处理超时时间（秒）
	PoolSize       int    `yaml:"poolSize"` // 浏览器池大小
	ProfileRootDir string `mapstructure:"profileRootDir" yaml:"profileRootDir"`
}
