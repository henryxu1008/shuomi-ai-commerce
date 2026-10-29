package config

// PlatformsConfig 平台配置
type PlatformsConfig struct {
	Alibaba1688 Alibaba1688Config `mapstructure:"alibaba1688" yaml:"alibaba1688"` // 1688平台配置
}

// Alibaba1688Config 1688平台配置
type Alibaba1688Config struct {
	Enabled        bool   `yaml:"enabled"`  // 是否启用1688处理器
	Timeout        int    `yaml:"timeout"`  // 处理超时时间（秒）
	PoolSize       int    `yaml:"poolSize"` // 浏览器池大小
	ProfileRootDir string `mapstructure:"profileRootDir" yaml:"profileRootDir"`
}
