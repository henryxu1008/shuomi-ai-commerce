package aiworkbench

import (
	"encoding/json"

	strict "sigs.k8s.io/json"
)

type TitleSkillDraft struct {
	Name         string            `json:"name"`
	DisplayName  string            `json:"displayName"`
	Description  string            `json:"description"`
	Instructions string            `json:"instructions"`
	Script       *TitleSkillScript `json:"script,omitempty"`
}

type TitleSkillScript struct {
	Description string `json:"description"`
	Source      string `json:"source"`
}

// CanonicalTitleSkillSnapshot returns the bytes used when the request was
// accepted. PostgreSQL JSONB can reorder object fields, including script fields.
func CanonicalTitleSkillSnapshot(snapshot json.RawMessage) (json.RawMessage, error) {
	var draft TitleSkillDraft
	issues, err := strict.UnmarshalStrict(snapshot, &draft, strict.DisallowDuplicateFields, strict.DisallowUnknownFields)
	if err != nil || len(issues) != 0 {
		return nil, ErrInvalid
	}
	canonical, err := json.Marshal(draft)
	if err != nil || len(canonical) > 65536 {
		return nil, ErrInvalid
	}
	return canonical, nil
}
