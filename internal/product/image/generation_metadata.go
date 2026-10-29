package image

import (
	"sort"
	"strings"
)

func NormalizeGenerationMetadata(metadata GenerationMetadata) (GenerationMetadata, error) {
	if err := preflightGenerationMetadata(metadata); err != nil {
		return GenerationMetadata{}, ErrInputInvalid
	}
	result := GenerationMetadata{
		Capability:      strings.TrimSpace(metadata.Capability),
		ModelFamily:     strings.TrimSpace(metadata.ModelFamily),
		InvocationID:    strings.TrimSpace(metadata.InvocationID),
		PromptReference: strings.TrimSpace(metadata.PromptReference),
		PromptVersion:   strings.TrimSpace(metadata.PromptVersion),
	}
	if len(metadata.Values) > 0 {
		result.Values = make(map[string]string, len(metadata.Values))
		keys := make([]string, 0, len(metadata.Values))
		for rawKey := range metadata.Values {
			keys = append(keys, rawKey)
		}
		sort.Strings(keys)
		for _, rawKey := range keys {
			rawValue := metadata.Values[rawKey]
			key, value := strings.TrimSpace(rawKey), strings.TrimSpace(rawValue)
			if key == "" || value == "" {
				return GenerationMetadata{}, ErrInputInvalid
			}
			if _, exists := result.Values[key]; exists {
				return GenerationMetadata{}, ErrInputInvalid
			}
			result.Values[key] = value
		}
		if len(result.Values) == 0 {
			result.Values = nil
		}
	}
	return result, nil
}

func preflightGenerationMetadata(metadata GenerationMetadata) error {
	if len(metadata.Values) > maxMetadataValues {
		return ErrInputInvalid
	}
	used := 0
	fixed := [...]string{
		metadata.Capability, metadata.ModelFamily, metadata.InvocationID,
		metadata.PromptReference, metadata.PromptVersion,
	}
	for _, value := range fixed {
		if !addImageStringBytes(&used, value) {
			return ErrInputInvalid
		}
	}
	for key, value := range metadata.Values {
		if !addImageStringBytes(&used, key) || !addImageStringBytes(&used, value) {
			return ErrInputInvalid
		}
	}
	return nil
}
