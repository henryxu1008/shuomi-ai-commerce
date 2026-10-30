package config

import (
	"strings"

	"github.com/spf13/viper"
)

func getStringSlice(v *viper.Viper, key string) []string {
	stringSlice := v.GetStringSlice(key)
	if len(stringSlice) > 0 {
		result := make([]string, 0, len(stringSlice))
		for _, item := range stringSlice {
			for _, part := range strings.Split(item, ",") {
				part = strings.TrimSpace(strings.Trim(part, ","))
				if part == "" {
					continue
				}
				result = append(result, part)
			}
		}
		return result
	}

	raw := strings.TrimSpace(v.GetString(key))
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		result = append(result, part)
	}
	return result
}
