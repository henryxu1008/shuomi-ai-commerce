package aiworkbench

import (
	"encoding/json"
	"fmt"

	strict "sigs.k8s.io/json"
)

const maxTitleSkillSnapshotBytes = 64 << 10

var ErrTitleSkillSnapshotTooLarge = fmt.Errorf("title skill snapshot exceeds the byte limit: %w", ErrInvalid)

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
	if err != nil {
		return nil, ErrInvalid
	}
	if len(canonical) > maxTitleSkillSnapshotBytes {
		return nil, ErrTitleSkillSnapshotTooLarge
	}
	return canonical, nil
}
