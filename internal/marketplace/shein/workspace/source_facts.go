package workspace

import "github.com/henryxu1008/shuomi-ai-commerce/internal/listing/sourcefacts"

// SourceFactsReady evaluates whether source-derived SHEIN submit facts are ready for submit.
func SourceFactsReady(metadata map[string]string) (bool, string) {
	return sourcefacts.Ready(metadata)
}
