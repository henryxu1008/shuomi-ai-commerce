package publishing

import (
	"reflect"
	"slices"
	"testing"
)

func TestDeduplicateSKUSpecs(t *testing.T) {
	color := submissionSpec{parentSpecID: "color", specID: "blue", name: "Blue"}
	size := submissionSpec{parentSpecID: "size", specID: "s", name: "Small"}
	for _, tt := range []struct {
		name string
		in   []submissionSpec
		want []submissionSpec
	}{
		{name: "nil"},
		{name: "empty", in: []submissionSpec{}, want: []submissionSpec{}},
		{name: "single", in: []submissionSpec{color}, want: []submissionSpec{color}},
		{name: "stable first parent", in: []submissionSpec{size, color, {parentSpecID: "size", specID: "l"}}, want: []submissionSpec{size, color}},
		{name: "empty parent is a key", in: []submissionSpec{{specID: "first"}, color, {specID: "second"}}, want: []submissionSpec{{specID: "first"}, color}},
		{name: "no normalization", in: []submissionSpec{color, {parentSpecID: " color"}, {parentSpecID: "Color"}}, want: []submissionSpec{color, {parentSpecID: " color"}, {parentSpecID: "Color"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			before := slices.Clone(tt.in)
			if got := deduplicateSKUSpecs(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
			if !reflect.DeepEqual(tt.in, before) {
				t.Fatal("changed input specifications")
			}
		})
	}
}

func TestHasUnresolvedSKUSpecs(t *testing.T) {
	for _, tt := range []struct {
		name  string
		specs []submissionSpec
		want  bool
	}{
		{name: "nil"},
		{name: "empty", specs: []submissionSpec{}},
		{name: "exact prefix", specs: []submissionSpec{{specID: "TEMP_blue"}}, want: true},
		{name: "bare prefix", specs: []submissionSpec{{specID: "TEMP_"}}, want: true},
		{name: "later temporary", specs: []submissionSpec{{specID: "1"}, {specID: "TEMP_size"}}, want: true},
		{name: "case remains exact", specs: []submissionSpec{{specID: "temp_blue"}}},
		{name: "prefix position remains exact", specs: []submissionSpec{{specID: " TEMP_blue"}, {specID: "xTEMP_blue"}}},
		{name: "parent is not checked", specs: []submissionSpec{{parentSpecID: "TEMP_color", specID: "blue"}}},
		{name: "empty is not a temporary ID", specs: []submissionSpec{{}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			before := slices.Clone(tt.specs)
			if got := hasUnresolvedSKUSpecs(tt.specs); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(tt.specs, before) {
				t.Fatal("changed input specifications")
			}
		})
	}
}

func TestSKUSpecCheckKeepsDeduplicationBeforeTemporaryIDDetection(t *testing.T) {
	for _, tt := range []struct {
		name        string
		first, next string
		want        bool
	}{
		{name: "discarded temporary does not reject", first: "blue", next: "TEMP_red"},
		{name: "retained temporary rejects", first: "TEMP_blue", next: "red", want: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			specs := []submissionSpec{{parentSpecID: "color", specID: tt.first}, {parentSpecID: "color", specID: tt.next}}
			if got := hasUnresolvedSKUSpecs(deduplicateSKUSpecs(specs)); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeSingleItemPackaging(t *testing.T) {
	for _, tt := range []struct {
		name                           string
		classification, pieces, packed int
		wantPieces, wantPacked         int
	}{
		{name: "single already valid", classification: 1, pieces: 1, packed: 1, wantPieces: 1, wantPacked: 1},
		{name: "single zero values", classification: 1, wantPieces: 1, wantPacked: 1},
		{name: "single conflicting values", classification: 1, pieces: -2, packed: 7, wantPieces: 1, wantPacked: 1},
		{name: "zero classification keeps zero values"},
		{name: "other classification unchanged", classification: 2, pieces: 3, packed: 0, wantPieces: 3, wantPacked: 0},
		{name: "negative values unchanged", classification: -1, pieces: -2, packed: -3, wantPieces: -2, wantPacked: -3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pieces, packed := normalizeSingleItemPackaging(tt.classification, tt.pieces, tt.packed)
			if pieces != tt.wantPieces || packed != tt.wantPacked {
				t.Fatalf("got (%d,%d), want (%d,%d)", pieces, packed, tt.wantPieces, tt.wantPacked)
			}
		})
	}
}
