package einomodel

import (
	"bytes"
	"encoding/json"
	"fmt"

	"task-processor/internal/agent"
	"task-processor/internal/product/enrichment"
)

// ContentEvidenceForPrompt reuses the existing bounded, whole-field model view.
// Unlike title-only input, it refuses to silently omit any selected value.
func ContentEvidenceForPrompt(raw json.RawMessage, selection enrichment.ContentSelection) (json.RawMessage, error) {
	if !selection.Valid() {
		return nil, agent.ErrInvalid
	}
	projected, err := EvidenceForPrompt(raw)
	if err != nil || len(raw) == 0 || evidenceFits(raw) {
		return projected, err
	}
	var original map[string]json.RawMessage
	if json.Unmarshal(raw, &original) != nil {
		return nil, agent.ErrInvalid
	}
	var view titleEvidenceView
	if json.Unmarshal(projected, &view) != nil {
		return nil, agent.ErrInvalid
	}
	if original["snapshot"] != nil {
		var before, after map[string]json.RawMessage
		if json.Unmarshal(original["snapshot"], &before) != nil || json.Unmarshal(view.Evidence["snapshot"], &after) != nil {
			return nil, agent.ErrInvalid
		}
		for _, field := range append(selection.Fields(), "sources") {
			if before[field] != nil && !bytes.Equal(before[field], after[field]) {
				return nil, fmt.Errorf("%w: selected content evidence %s exceeds the prompt budget", agent.ErrInvalid, field)
			}
		}
	}
	view.Projection = "content-evidence-v1"
	encoded, err := json.Marshal(view)
	if err != nil || !evidenceFits(encoded) {
		return nil, agent.ErrInvalid
	}
	return encoded, nil
}
