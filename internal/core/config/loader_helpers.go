package config

import (
	"bufio"
	"os"
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

func readLineList(path string) []string {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var result []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		result = append(result, line)
	}
	return result
}

func uniqueStringList(items ...[]string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, group := range items {
		for _, item := range group {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

func getStringFromMap(m map[string]any, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
