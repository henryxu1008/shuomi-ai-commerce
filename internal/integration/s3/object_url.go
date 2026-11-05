package s3

import "strings"

func ResolveObjectURL(publicBase, key string) string {
	trimmedKey := strings.TrimLeft(strings.TrimSpace(key), "/")
	if base := strings.TrimRight(strings.TrimSpace(publicBase), "/"); base != "" && trimmedKey != "" {
		return base + "/" + trimmedKey
	}
	return ""
}
