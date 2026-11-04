package asset

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	strict "sigs.k8s.io/json"
)

const ImageVisualAnalysisSchema = "image-analysis-result-v1"

var ErrImageVisualAnalysis = errors.New("invalid bounded readonly image opinion")

type ImageVisualObservation struct {
	Area   string `json:"area"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type ImageVisualSuggestion struct {
	Area   string `json:"area"`
	Action string `json:"action"`
	Reason string `json:"reason"`
}

// ImageVisualOpinion is model advice, never Catalog/ApprovedAsset facts or an
// executable image command. The model cannot supply the result's authority.
type ImageVisualOpinion struct {
	Observations []ImageVisualObservation `json:"observations"`
	Suggestions  []ImageVisualSuggestion  `json:"suggestions"`
	Unknowns     []string                 `json:"unknowns"`
}

// ImageVisualAnalysisResult metadata is attached only by the governed consumer
// from the frozen preparation and actual invocation. It is not model output.
type ImageVisualAnalysisResult struct {
	Schema              string `json:"schema"`
	SelectionDigest     string `json:"selectionDigest"`
	ContentSHA256       string `json:"contentSha256"`
	InvocationID        string `json:"invocationId"`
	ModelID             string `json:"modelId"`
	ConfigurationDigest string `json:"configurationDigest"`
	ModelOutputSHA256   string `json:"modelOutputSha256"`
	// Exact bounded raw response binds the typed view to the native invocation
	// hash, including whitespace. No image bytes or provider credentials.
	ModelOutput string             `json:"modelOutput"`
	QuoteDigest string             `json:"quoteDigest"`
	Opinion     ImageVisualOpinion `json:"opinion"`
}

func visualArea(s string) bool {
	return s == "subject" || s == "framing" || s == "background" || s == "clarity"
}

func visualReason(s string) bool {
	return s != "" && len(s) <= 512 && utf8.ValidString(s) && strings.TrimSpace(s) == s &&
		strings.IndexFunc(s, func(r rune) bool { return unicode.IsControl(r) && r != '\n' && r != '\t' }) < 0
}

func (o ImageVisualOpinion) Valid() bool {
	if o.Observations == nil || o.Suggestions == nil || o.Unknowns == nil || len(o.Observations) > 8 || len(o.Suggestions) > 8 || len(o.Unknowns) > 8 || len(o.Observations)+len(o.Unknowns) == 0 {
		return false
	}
	for _, v := range o.Observations {
		if !visualArea(v.Area) || !visualReason(v.Reason) || v.Status != "possible_issue" && v.Status != "no_obvious_issue" && v.Status != "unknown" {
			return false
		}
	}
	for _, v := range o.Suggestions {
		if !visualArea(v.Area) || !visualReason(v.Reason) || v.Action != "check" && v.Action != "change_image" && v.Action != "open_image_workflow" {
			return false
		}
	}
	for _, v := range o.Unknowns {
		if !visualReason(v) {
			return false
		}
	}
	return true
}

func DecodeImageVisualOpinion(raw []byte) (ImageVisualOpinion, error) {
	var o ImageVisualOpinion
	if len(raw) == 0 || len(raw) > 16<<10 {
		return o, ErrImageVisualAnalysis
	}
	violations, err := strict.UnmarshalStrict(raw, &o, strict.DisallowDuplicateFields, strict.DisallowUnknownFields)
	if err != nil || len(violations) != 0 || !o.Valid() {
		return ImageVisualOpinion{}, ErrImageVisualAnalysis
	}
	return o, nil
}

func visualDigest(s string) bool {
	if len(s) != 64 || strings.ToLower(s) != s {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func (r ImageVisualAnalysisResult) Valid() bool {
	opinion, err := DecodeImageVisualOpinion([]byte(r.ModelOutput))
	h := sha256.Sum256([]byte(r.ModelOutput))
	return err == nil && reflect.DeepEqual(opinion, r.Opinion) && hex.EncodeToString(h[:]) == r.ModelOutputSHA256 &&
		r.Schema == ImageVisualAnalysisSchema && r.Opinion.Valid() && validAdviceID(r.InvocationID) && validAdviceID(r.ModelID) &&
		visualDigest(r.SelectionDigest) && visualDigest(r.ContentSHA256) && visualDigest(r.ConfigurationDigest) && visualDigest(r.ModelOutputSHA256) && visualDigest(r.QuoteDigest)
}

func DecodeImageVisualAnalysisResult(raw []byte) (ImageVisualAnalysisResult, error) {
	var r ImageVisualAnalysisResult
	if len(raw) == 0 || len(raw) > 64<<10 {
		return r, ErrImageVisualAnalysis
	}
	violations, err := strict.UnmarshalStrict(raw, &r, strict.DisallowDuplicateFields, strict.DisallowUnknownFields)
	if err != nil || len(violations) != 0 || !r.Valid() {
		return ImageVisualAnalysisResult{}, ErrImageVisualAnalysis
	}
	return r, nil
}
