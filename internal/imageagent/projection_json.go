package imageagent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SlotProjection owns the persisted durable-candidate JSON representation,
// separately from the executor's in-process result.
func (value SlotProjection) MarshalJSON() ([]byte, error) {
	var candidates []slotProjectionJSONCandidate
	if value.Candidates != nil {
		candidates = make([]slotProjectionJSONCandidate, len(value.Candidates))
	}
	for index, candidate := range value.Candidates {
		candidates[index] = slotProjectionJSONCandidate{
			GenerationProof: candidate.GenerationProof,
			Artwork:         cloneArtworkProvenance(candidate.Artwork),
			AssetID:         candidate.AssetID,
			SourceAssetID:   candidate.SourceAssetID,
			Width:           candidate.Width,
			Height:          candidate.Height,
			Operations:      append([]string(nil), candidate.Operations...),
		}
		identity := candidate.DurableAsset
		candidates[index].DurableAsset = &identity
	}
	return json.Marshal(slotProjectionJSON{
		Closure:    value.Closure,
		Slot:       value.Slot,
		Attempt:    value.Attempt,
		Candidates: candidates,
		ErrorCode:  value.ErrorCode,
	})
}

func (value *SlotProjection) UnmarshalJSON(raw []byte) error {
	var decoded slotProjectionJSON
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}
	var candidates []AssetCandidate
	if decoded.Candidates != nil {
		candidates = make([]AssetCandidate, len(decoded.Candidates))
	}
	for index, candidate := range decoded.Candidates {
		candidates[index] = AssetCandidate{
			GenerationProof: candidate.GenerationProof,
			Artwork:         cloneArtworkProvenance(candidate.Artwork),
			AssetID:         candidate.AssetID,
			SourceAssetID:   candidate.SourceAssetID,
			Width:           candidate.Width,
			Height:          candidate.Height,
			Operations:      append([]string(nil), candidate.Operations...),
		}
		if candidate.DurableAsset != nil {
			candidates[index].DurableAsset = *candidate.DurableAsset
		}
	}
	*value = SlotProjection{Closure: decoded.Closure, Slot: decoded.Slot, Attempt: decoded.Attempt, Candidates: candidates, ErrorCode: decoded.ErrorCode}
	return nil
}

type slotProjectionJSON struct {
	Closure    *ImageSlotClosure `json:",omitempty"`
	Slot       Slot
	Attempt    int
	Candidates []slotProjectionJSONCandidate
	ErrorCode  string
}

type slotProjectionJSONCandidate struct {
	Artwork         *ArtworkProvenance    `json:",omitempty"`
	GenerationProof *ImageGenerationProof `json:",omitempty"`
	AssetID         string
	SourceAssetID   string
	DurableAsset    *DurableAssetIdentity `json:"DurableAsset,omitempty"`
	Width           int                   `json:"Width,omitempty"`
	Height          int                   `json:"Height,omitempty"`
	Operations      []string              `json:"Operations,omitempty"`
}

func (value *slotProjectionJSONCandidate) UnmarshalJSON(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	for name := range fields {
		if strings.EqualFold(name, "URL") || strings.EqualFold(name, "Metadata") {
			return fmt.Errorf("%w: retired projection candidate field %q", ErrValidation, name)
		}
	}
	type durableCandidate slotProjectionJSONCandidate
	var decoded durableCandidate
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}
	*value = slotProjectionJSONCandidate(decoded)
	return nil
}
