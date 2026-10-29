package image

import "strings"

type SceneOptions struct {
	SceneCategory     string
	SceneStyle        string
	BackgroundTone    string
	Composition       string
	PropsLevel        string
	AudienceHint      string
	CustomSceneHint   string
	SlotRole          string
	SlotBrief         string
	StyleReferenceIDs []string
}

func MergeSceneOptions(base, override *SceneOptions) (*SceneOptions, error) {
	if base == nil && override == nil {
		return nil, nil
	}
	merged := SceneOptions{}
	if base != nil {
		var err error
		merged, err = normalizedSceneOptions(*base)
		if err != nil {
			return nil, err
		}
	}
	if override != nil {
		normalized, err := normalizedSceneOptions(*override)
		if err != nil {
			return nil, err
		}
		mergeSceneOptionValue(&merged.SceneCategory, normalized.SceneCategory)
		mergeSceneOptionValue(&merged.SceneStyle, normalized.SceneStyle)
		mergeSceneOptionValue(&merged.BackgroundTone, normalized.BackgroundTone)
		mergeSceneOptionValue(&merged.Composition, normalized.Composition)
		mergeSceneOptionValue(&merged.PropsLevel, normalized.PropsLevel)
		mergeSceneOptionValue(&merged.AudienceHint, normalized.AudienceHint)
		mergeSceneOptionValue(&merged.CustomSceneHint, normalized.CustomSceneHint)
		mergeSceneOptionValue(&merged.SlotRole, normalized.SlotRole)
		mergeSceneOptionValue(&merged.SlotBrief, normalized.SlotBrief)
		if len(normalized.StyleReferenceIDs) > 0 {
			merged.StyleReferenceIDs = normalized.StyleReferenceIDs
		}
	}
	if err := validateSceneOptions(merged); err != nil {
		return nil, err
	}
	return &merged, nil
}

func normalizedSceneOptions(options SceneOptions) (SceneOptions, error) {
	if err := preflightSceneOptions(options); err != nil {
		return SceneOptions{}, err
	}
	options.SceneCategory = strings.TrimSpace(options.SceneCategory)
	options.SceneStyle = strings.TrimSpace(options.SceneStyle)
	options.BackgroundTone = strings.TrimSpace(options.BackgroundTone)
	options.Composition = strings.TrimSpace(options.Composition)
	options.PropsLevel = strings.TrimSpace(options.PropsLevel)
	options.AudienceHint = strings.TrimSpace(options.AudienceHint)
	options.CustomSceneHint = strings.TrimSpace(options.CustomSceneHint)
	options.SlotRole = strings.TrimSpace(options.SlotRole)
	options.SlotBrief = strings.TrimSpace(options.SlotBrief)
	styleReferenceIDs, err := normalizedStrings(options.StyleReferenceIDs, maxStyleReferences)
	if err != nil {
		return SceneOptions{}, err
	}
	options.StyleReferenceIDs = styleReferenceIDs
	return options, nil
}

func preflightSceneOptions(options SceneOptions) error {
	if len(options.StyleReferenceIDs) > maxStyleReferences {
		return ErrInputInvalid
	}
	used := 0
	values := [...]string{
		options.SceneCategory, options.SceneStyle, options.BackgroundTone, options.Composition,
		options.PropsLevel, options.AudienceHint, options.CustomSceneHint, options.SlotRole, options.SlotBrief,
	}
	for _, value := range values {
		if !addImageStringBytes(&used, value) {
			return ErrInputInvalid
		}
	}
	for _, reference := range options.StyleReferenceIDs {
		if !addImageStringBytes(&used, reference) {
			return ErrInputInvalid
		}
	}
	return nil
}

func validateSceneOptions(options SceneOptions) error {
	return preflightSceneOptions(options)
}

func mergeSceneOptionValue(target *string, value string) {
	if value != "" {
		*target = value
	}
}
