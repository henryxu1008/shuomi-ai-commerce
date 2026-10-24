package config

// AmazonConfig Amazon爬虫配置
type AmazonConfig struct {
	Enabled          bool                       `yaml:"enabled"`
	Zipcodes         map[string]string          `yaml:"zipcodes"`
	CrawlTimeout     int                        `yaml:"crawlTimeout"` // 爬虫超时时间（秒）
	FailureArtifacts FailureArtifactsConfig     `yaml:"failureArtifacts"`
	RiskControl      AmazonRiskControlConfig    `yaml:"riskControl"`
	QualityControl   AmazonQualityControlConfig `yaml:"qualityControl"`
	ProxyPool        AmazonProxyPoolConfig      `yaml:"proxyPool"`
}

type FailureArtifactsConfig struct {
	Enabled      bool   `yaml:"enabled"`
	Directory    string `yaml:"directory"`
	CaptureHTML  bool   `yaml:"captureHTML"`
	MaxHTMLBytes int    `yaml:"maxHTMLBytes"`
}

type AmazonRiskControlConfig struct {
	CaptchaRecreateThreshold        int `yaml:"captchaRecreateThreshold"`
	AuthenticationRecreateThreshold int `yaml:"authenticationRecreateThreshold"`
	BrowserCrashRecreateThreshold   int `yaml:"browserCrashRecreateThreshold"`
	TimeoutRecreateThreshold        int `yaml:"timeoutRecreateThreshold"`
	NetworkRecreateThreshold        int `yaml:"networkRecreateThreshold"`
	ServerErrorRecreateThreshold    int `yaml:"serverErrorRecreateThreshold"`
}

type AmazonQualityControlConfig struct {
	RetryOnValidationFailure   bool `yaml:"retryOnValidationFailure"`
	ValidationRetryMaxAttempts int  `yaml:"validationRetryMaxAttempts"`
}

type AmazonProxyPoolConfig struct {
	Enabled                bool     `yaml:"enabled"`
	Strategy               string   `yaml:"strategy"`
	FailureCooldownSeconds int      `yaml:"failureCooldownSeconds"`
	ProxyFile              string   `yaml:"proxyFile"`
	Proxies                []string `yaml:"proxies"`
}
