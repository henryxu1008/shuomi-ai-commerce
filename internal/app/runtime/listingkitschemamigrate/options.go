package listingkitschemamigrate

import "strings"

type Options struct {
	Config    string
	LogLevel  string
	Scope     string
	Version   string
	BuildTime string
}

func (o Options) ConfigPath() string {
	return ResolveConfigPath(o.Config)
}

func ResolveConfigPath(configPath string) string {
	if strings.TrimSpace(configPath) == "" {
		return ""
	}
	return configPath
}
