package publishing

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestValidateSubmissionSpecs(t *testing.T) {
	blue := submissionSpec{parentSpecID: "color", specID: "blue", name: "Blue"}
	small := submissionSpec{parentSpecID: "size", specID: "s", name: "Small"}
	tests := []struct {
		name       string
		used       []submissionSpec
		configured []configuredSubmissionSpec
		wantErrors []string
	}{
		{name: "empty usage ignores unused incomplete configuration", configured: []configuredSubmissionSpec{{parentSpecID: "color", specID: "blue"}}},
		{name: "all used dimensions configured", used: []submissionSpec{blue, small}, configured: []configuredSubmissionSpec{
			{parentSpecID: "color", specID: "blue", vid: 10}, {parentSpecID: "size", specID: "s", vid: 20},
		}},
		{name: "missing parent", used: []submissionSpec{blue}, configured: []configuredSubmissionSpec{{parentSpecID: "size", specID: "blue", vid: 10}}, wantErrors: []string{"missing=[Blue(blue)]"}},
		{name: "missing spec within parent", used: []submissionSpec{blue}, configured: []configuredSubmissionSpec{{parentSpecID: "color", specID: "red", vid: 10}}, wantErrors: []string{"Blue(blue)"}},
		{name: "used zero VID", used: []submissionSpec{blue}, configured: []configuredSubmissionSpec{{parentSpecID: "color", specID: "blue"}}, wantErrors: []string{"Blue(blue)-missing_vid"}},
		{name: "nonzero VID keeps original constraint", used: []submissionSpec{blue}, configured: []configuredSubmissionSpec{{parentSpecID: "color", specID: "blue", vid: -1}}},
		{name: "unused incomplete spec ignored", used: []submissionSpec{blue}, configured: []configuredSubmissionSpec{
			{parentSpecID: "color", specID: "blue", vid: 10}, {parentSpecID: "color", specID: "red"},
		}},
		{name: "duplicate configuration does not erase missing VID", used: []submissionSpec{blue}, configured: []configuredSubmissionSpec{
			{parentSpecID: "color", specID: "blue"}, {parentSpecID: "color", specID: "blue", vid: 10},
		}, wantErrors: []string{"Blue(blue)-missing_vid"}},
		{name: "duplicate used and configured specs", used: []submissionSpec{blue, blue}, configured: []configuredSubmissionSpec{
			{parentSpecID: "color", specID: "blue", vid: 10}, {parentSpecID: "color", specID: "blue", vid: 20},
		}},
		{name: "missing and incomplete across SKUs", used: []submissionSpec{blue, small, blue}, configured: []configuredSubmissionSpec{
			{parentSpecID: "size", specID: "s"},
		}, wantErrors: []string{"Blue(blue)", "Small(s)-missing_vid"}},
		{name: "first explicit SKU name used for diagnostics", used: []submissionSpec{
			blue, {parentSpecID: "color", specID: "blue", name: "Other name"},
		}, wantErrors: []string{"missing=[Blue(blue)]"}},
		{name: "empty explicit name retains ID", used: []submissionSpec{{parentSpecID: "color", specID: "blue"}}, wantErrors: []string{"missing=[(blue)]"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usedBefore := slices.Clone(tt.used)
			configuredBefore := slices.Clone(tt.configured)
			_, err := validateSubmissionSpecs(tt.used, tt.configured)
			if len(tt.wantErrors) == 0 && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(tt.wantErrors) != 0 {
				if err == nil {
					t.Fatal("expected missing/incomplete specification error")
				}
				for _, want := range tt.wantErrors {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("error %q does not contain %q", err, want)
					}
				}
			}
			if !reflect.DeepEqual(tt.used, usedBefore) || !reflect.DeepEqual(tt.configured, configuredBefore) {
				t.Fatal("validation modified input specifications")
			}
		})
	}
}

func TestValidateSubmissionSpecsReturnsUsedDimensionsOnFailure(t *testing.T) {
	used := []submissionSpec{
		{parentSpecID: "color", specID: "blue", name: "Blue"},
		{parentSpecID: "color", specID: "blue", name: "Blue"},
		{parentSpecID: "color", specID: "red", name: "Red"},
		{parentSpecID: "size", specID: "s", name: "Small"},
	}
	got, err := validateSubmissionSpecs(used, nil)
	if err == nil {
		t.Fatal("expected missing specifications")
	}
	want := map[string]map[string]bool{
		"color": {"blue": true, "red": true},
		"size":  {"s": true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("used dimensions = %v, want %v", got, want)
	}
}

func TestValidateSubmissionPackaging(t *testing.T) {
	tests := []struct {
		name        string
		properties  []string
		pieces      []int
		wantInvalid bool
		wantIndex   int
	}{
		{name: "empty input", wantIndex: -1},
		{name: "no multiple sets does not constrain pieces", properties: []string{"Single", "Multiple items"}, pieces: []int{0, 1}, wantIndex: -1},
		{name: "case insensitive substring", properties: []string{"Other", "Includes MULTIPLE SETS package"}, pieces: []int{2, 3}, wantIndex: 1},
		{name: "zero pieces invalid", properties: []string{"Multiple Sets"}, pieces: []int{0}, wantInvalid: true, wantIndex: 0},
		{name: "one piece in later SKU invalid", properties: []string{"Other", "multiple sets"}, pieces: []int{3, 2, 1}, wantInvalid: true, wantIndex: 1},
		{name: "negative pieces invalid", properties: []string{"Multiple Sets"}, pieces: []int{-1}, wantInvalid: true, wantIndex: 0},
		{name: "all SKUs above one", properties: []string{"Multiple Sets"}, pieces: []int{2, 5}, wantIndex: 0},
		{name: "first matching property retained on failure", properties: []string{"Other", "multiple sets", "MULTIPLE SETS"}, pieces: []int{1}, wantInvalid: true, wantIndex: 1},
		{name: "first matching property retained on success", properties: []string{"multiple sets", "multiple sets"}, pieces: []int{2}, wantIndex: 0},
		{name: "multiple sets with no SKU keeps original result", properties: []string{"Other", "Multiple Sets"}, wantIndex: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			propertiesBefore := slices.Clone(tt.properties)
			piecesBefore := slices.Clone(tt.pieces)
			invalid, index := validateSubmissionPackaging(tt.properties, tt.pieces)
			if invalid != tt.wantInvalid || index != tt.wantIndex {
				t.Fatalf("packaging = (%v, %d), want (%v, %d)", invalid, index, tt.wantInvalid, tt.wantIndex)
			}
			if !reflect.DeepEqual(tt.properties, propertiesBefore) || !reflect.DeepEqual(tt.pieces, piecesBefore) {
				t.Fatal("validation modified packaging inputs")
			}
		})
	}
}
