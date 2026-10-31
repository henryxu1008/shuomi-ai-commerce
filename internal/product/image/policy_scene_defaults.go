package image

import "strings"

type PolicySceneDefaults struct {
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

func NormalizePolicySceneDefaults(options PolicySceneDefaults) (PolicySceneDefaults, error) {
	if err := preflightPolicySceneDefaults(options); err != nil {
		return PolicySceneDefaults{}, err
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
		return PolicySceneDefaults{}, err
	}
	options.StyleReferenceIDs = styleReferenceIDs
	return options, nil
}
func preflightPolicySceneDefaults(options PolicySceneDefaults) error {
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
