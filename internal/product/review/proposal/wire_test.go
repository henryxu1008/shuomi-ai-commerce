package proposal

import (
	"encoding/json"
	"testing"
)

func TestProposalContractsPreserveCanonicalWire(t *testing.T) {
	for _, test := range []struct {
		name  string
		value any
		wire  string
	}{
		{"nil candidate", Candidate{}, `{"Changes":null,"Warnings":null,"Rejections":null}`},
		{"empty candidate", Candidate{Changes: []FieldChange{}, Warnings: []Warning{}, Rejections: []Rejection{}}, `{"Changes":[],"Warnings":[],"Rejections":[]}`},
		{"nil proposal", Proposal{}, `{"Changes":null,"Evidence":null,"Quality":{"Overall":0,"EvidenceCoverage":0,"RequiredFieldCoverage":0},"Validation":{"Valid":false,"EvaluatedChanges":0},"Warnings":null,"Rejections":null}`},
		{"validated proposal", Proposal{Changes: []FieldChange{{Field: "title", Value: "new", EvidenceIDs: []string{"raw"}}}, Quality: QualityScore{Overall: 100, EvidenceCoverage: 100, RequiredFieldCoverage: 100}, Validation: ValidationResult{Valid: true, EvaluatedChanges: 1}}, `{"Changes":[{"Field":"title","Value":"new","EvidenceIDs":["raw"]}],"Evidence":null,"Quality":{"Overall":100,"EvidenceCoverage":100,"RequiredFieldCoverage":100},"Validation":{"Valid":true,"EvaluatedChanges":1},"Warnings":null,"Rejections":null}`},
		{"zero selection", ContentSelection{}, `{"targetLanguage":""}`},
		{"content selection", ContentSelection{TargetLanguage: "en-US", Title: true, Description: true}, `{"targetLanguage":"en-US","title":true,"description":true}`},
		{"diagnostic metadata", Warning{Code: "source_limited", Field: "title", Message: "One source", Metadata: map[string]string{"z": "last", "a": "first"}}, `{"Code":"source_limited","Field":"title","Message":"One source","Metadata":{"a":"first","z":"last"}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			raw, err := json.Marshal(test.value)
			if err != nil {
				t.Fatal(err)
			}
			if string(raw) != test.wire {
				t.Fatalf("wire changed: %s, want %s", raw, test.wire)
			}
		})
	}
}
