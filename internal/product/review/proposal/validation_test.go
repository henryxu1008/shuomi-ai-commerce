package proposal

import (
	"context"
	"math"
	"reflect"
	"task-processor/internal/product/sourcing"
	"testing"
	"time"
)

func TestCandidateBuildsOneCanonicalEvidenceBackedProposal(t *testing.T) {
	t.Parallel()

	candidate := Candidate{Changes: []FieldChange{{
		Field:       "description",
		Value:       "Steel bottle",
		EvidenceIDs: []string{"raw-1"},
	}}}

	capturedAt := time.Date(2026, time.August, 31, 9, 8, 7, 0, time.UTC)
	request := validRequest()
	request.Source.RawReference = sourcing.RawSourceReference{
		ReferenceType: "crawler_snapshot",
		ReferenceID:   "raw-1",
		SnapshotID:    "snapshot-7",
		Checksum:      "sha256:source",
		URL:           "https://source.example/products/B001",
		CapturedAt:    capturedAt,
		Metadata:      map[string]string{"etag": "v7"},
	}

	proposal, err := ValidateCandidate(context.Background(), request, candidate)
	if err != nil {
		t.Fatalf("ValidateCandidate() error = %v", err)
	}
	wantEvidence := []Evidence{{
		ReferenceType: "crawler_snapshot",
		ID:            "raw-1",
		ReferenceID:   "raw-1",
		SnapshotID:    "snapshot-7",
		Checksum:      "sha256:source",
		URL:           "https://source.example/products/B001",
		CapturedAt:    capturedAt,
		Metadata:      map[string]string{"etag": "v7"},
	}}
	if !reflect.DeepEqual(proposal.Evidence, wantEvidence) {
		t.Fatalf("ValidateCandidate() evidence = %#v, want %#v", proposal.Evidence, wantEvidence)
	}
	if got := proposal.Changes; !reflect.DeepEqual(got, []FieldChange{{Field: "description", Value: "Steel bottle", EvidenceIDs: []string{"raw-1"}}}) {
		t.Fatalf("ValidateCandidate() changes = %#v", got)
	}
	if proposal.Quality.Overall != 100 || proposal.Quality.EvidenceCoverage != 100 || proposal.Quality.RequiredFieldCoverage != 100 {
		t.Fatalf("ValidateCandidate() quality = %#v, want all scores 100", proposal.Quality)
	}
	if !proposal.Validation.Valid || proposal.Validation.EvaluatedChanges != 1 {
		t.Fatalf("ValidateCandidate() validation = %#v, want valid one-change proposal", proposal.Validation)
	}
}

func TestCandidateUsesOneCanonicalRawEvidenceIdentity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		reference   sourcing.RawSourceReference
		referenceID string
		wantErr     error
	}{
		{
			name: "reference id wins over snapshot and checksum",
			reference: sourcing.RawSourceReference{
				ReferenceID: "reference-1",
				SnapshotID:  "snapshot-1",
				Checksum:    "sha256:one",
			},
			referenceID: "reference-1",
		},
		{
			name:        "snapshot id is the first fallback",
			reference:   sourcing.RawSourceReference{SnapshotID: "snapshot-1", Checksum: "sha256:one"},
			referenceID: "snapshot-1",
		},
		{
			name:        "checksum is the final fallback",
			reference:   sourcing.RawSourceReference{Checksum: "sha256:one"},
			referenceID: "sha256:one",
		},
		{
			name:        "source identity is not raw evidence",
			reference:   sourcing.RawSourceReference{},
			referenceID: "B001",
			wantErr:     ErrEvidenceInsufficient,
		},
		{
			name:        "unknown raw evidence id is rejected",
			reference:   sourcing.RawSourceReference{ReferenceID: "raw-1"},
			referenceID: "unknown",
			wantErr:     ErrEvidenceInsufficient,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := validRequest()
			request.Source.RawReference = tt.reference
			candidate := Candidate{Changes: []FieldChange{{
				Field:       "description",
				Value:       "Steel bottle",
				EvidenceIDs: []string{tt.referenceID, tt.referenceID},
			}}}

			proposal, err := ValidateCandidate(context.Background(), request, candidate)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("ValidateCandidate() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateCandidate() error = %v", err)
			}
			if len(proposal.Evidence) != 1 || proposal.Evidence[0].ID != tt.referenceID {
				t.Fatalf("ValidateCandidate() evidence = %#v, want one evidence with ID %q", proposal.Evidence, tt.referenceID)
			}
			if got := proposal.Changes[0].EvidenceIDs; !reflect.DeepEqual(got, []string{tt.referenceID}) {
				t.Fatalf("ValidateCandidate() change evidence IDs = %v, want one canonical ID", got)
			}
		})
	}
}

func TestCandidateUsesFixedValidationAndScoringOrder(t *testing.T) {
	t.Parallel()
	t.Run("input validation precedes evidence validation", func(t *testing.T) {
		request := validRequest()
		request.Policy.Version = ""
		_, err := ValidateCandidate(context.Background(), request, Candidate{Changes: []FieldChange{{Field: "", Value: "", EvidenceIDs: []string{"unknown"}}}})
		if err != ErrInputInvalid {
			t.Fatalf("ValidateCandidate() error = %v, want ErrInputInvalid", err)
		}
	})
	t.Run("evidence validation precedes output validation", func(t *testing.T) {
		_, err := ValidateCandidate(context.Background(), validRequest(), Candidate{Changes: []FieldChange{{Field: "", Value: "", EvidenceIDs: []string{"unknown"}}}})
		if err != ErrEvidenceInsufficient {
			t.Fatalf("ValidateCandidate() error = %v, want ErrEvidenceInsufficient", err)
		}
	})
	t.Run("scoring precedes output validation", func(t *testing.T) {
		proposal, err := ValidateCandidate(context.Background(), validRequest(), Candidate{Changes: []FieldChange{{Field: "", Value: "", EvidenceIDs: []string{"raw-1"}}}})
		if err != ErrOutputValidation {
			t.Fatalf("ValidateCandidate() error = %v, want ErrOutputValidation", err)
		}
		if proposal.Quality.EvidenceCoverage != 100 {
			t.Fatalf("ValidateCandidate() quality = %#v, want scoring result retained before validation error", proposal.Quality)
		}
	})
}

func TestCandidateRejectsMalformedWarningsAndRejections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		candidate Candidate
	}{
		{
			name: "warning without code",
			candidate: Candidate{
				Changes:  []FieldChange{{Field: "description", Value: "Steel bottle", EvidenceIDs: []string{"raw-1"}}},
				Warnings: []Warning{{Code: "  ", Message: "source is limited"}},
			},
		},
		{
			name: "rejection without code",
			candidate: Candidate{
				Changes:    []FieldChange{{Field: "description", Value: "Steel bottle", EvidenceIDs: []string{"raw-1"}}},
				Rejections: []Rejection{{Code: "", Message: "cannot support claim"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateCandidate(context.Background(), validRequest(), tt.candidate)
			if err != ErrOutputValidation {
				t.Fatalf("ValidateCandidate() error = %v, want ErrOutputValidation", err)
			}
		})
	}
}

func TestCandidateValidatesContextAndInputBeforeValidation(t *testing.T) {
	t.Parallel()

	validCandidate := Candidate{Changes: []FieldChange{{Field: "description", Value: "Steel bottle", EvidenceIDs: []string{"raw-1"}}}}
	t.Run("nil context", func(t *testing.T) {
		_, err := ValidateCandidate(nil, validRequest(), validCandidate) //nolint:staticcheck
		if err != ErrInputInvalid {
			t.Fatalf("ValidateCandidate() error = %v, want ErrInputInvalid", err)
		}
	})

	t.Run("canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := ValidateCandidate(ctx, validRequest(), validCandidate)
		if err != context.Canceled {
			t.Fatalf("ValidateCandidate() error = %v, want context.Canceled", err)
		}
	})

	tests := []struct {
		name   string
		mutate func(*Request)
	}{
		{
			name: "source identity is incomplete",
			mutate: func(request *Request) {
				request.Source.Identity.SourcePlatform = ""
			},
		},
		{
			name: "policy version is not canonical",
			mutate: func(request *Request) {
				request.Policy.Version = " v1 "
			},
		},
		{
			name: "policy field is duplicated",
			mutate: func(request *Request) {
				request.Policy.AllowedFields = []string{"description", "description"}
			},
		},
		{
			name: "required field is not allowed",
			mutate: func(request *Request) {
				request.Policy.RequiredFields = []string{"title"}
			},
		},
		{
			name: "minimum score is NaN",
			mutate: func(request *Request) {
				request.Policy.MinimumQualityScore = math.NaN()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := validRequest()
			tt.mutate(&request)
			_, err := ValidateCandidate(context.Background(), request, validCandidate)
			if err != ErrInputInvalid {
				t.Fatalf("ValidateCandidate() error = %v, want ErrInputInvalid", err)
			}
		})
	}
}

func TestCandidateReturnsStableWarningsRejectionsAndQuality(t *testing.T) {
	t.Parallel()

	request := validRequest()
	request.Policy.AllowedFields = []string{"description", "title"}
	request.Policy.RequiredFields = []string{"description", "title"}
	request.Policy.MinimumQualityScore = 90
	candidate := Candidate{
		Changes: []FieldChange{{Field: "description", Value: "Steel bottle", EvidenceIDs: []string{"raw-1"}}},
		Warnings: []Warning{
			{Code: " ZETA ", Field: " title ", Message: " Last warning "},
			{Code: "ALPHA", Field: " description ", Message: " First warning "},
		},
		Rejections: []Rejection{
			{Code: " ZETA ", Field: " title ", Message: " Last rejection "},
			{Code: "ALPHA", Field: " description ", Message: " First rejection "},
		},
	}

	proposal, err := ValidateCandidate(context.Background(), request, candidate)
	if err != ErrPolicyRejected {
		t.Fatalf("ValidateCandidate() error = %v, want ErrPolicyRejected", err)
	}
	if proposal.Quality.EvidenceCoverage != 100 || proposal.Quality.RequiredFieldCoverage != 50 || proposal.Quality.Overall != 75 {
		t.Fatalf("ValidateCandidate() quality = %#v, want evidence=100 required=50 overall=75", proposal.Quality)
	}
	if proposal.Validation.Valid {
		t.Fatalf("ValidateCandidate() validation = %#v, want rejected", proposal.Validation)
	}
	wantWarnings := []Warning{
		{Code: "alpha", Field: "description", Message: "First warning"},
		{Code: "zeta", Field: "title", Message: "Last warning"},
	}
	if !reflect.DeepEqual(proposal.Warnings, wantWarnings) {
		t.Fatalf("ValidateCandidate() warnings = %#v, want %#v", proposal.Warnings, wantWarnings)
	}
	wantRejections := []Rejection{
		{Code: "alpha", Field: "description", Message: "First rejection"},
		{Code: "quality_below_minimum", Message: "proposal quality is below policy minimum"},
		{Code: "required_field_missing", Field: "title", Message: "required field change is missing"},
		{Code: "zeta", Field: "title", Message: "Last rejection"},
	}
	if !reflect.DeepEqual(proposal.Rejections, wantRejections) {
		t.Fatalf("ValidateCandidate() rejections = %#v, want %#v", proposal.Rejections, wantRejections)
	}
}

func TestCandidateDeduplicatesCanonicalCandidateDiagnostics(t *testing.T) {
	t.Parallel()

	metadata := map[string]string{"source": "raw-1"}
	candidate := Candidate{
		Changes: []FieldChange{{Field: "description", Value: "Steel bottle", EvidenceIDs: []string{"raw-1"}}},
		Warnings: []Warning{
			{Code: " SOURCE_LIMITED ", Field: " description ", Message: " Only one source ", Metadata: metadata},
			{Code: "source_limited", Field: "description", Message: "Only one source", Metadata: map[string]string{"source": "raw-1"}},
		},
		Rejections: []Rejection{
			{Code: " CLAIM_UNSUPPORTED ", Field: " title ", Message: " Claim lacks evidence ", Metadata: metadata},
			{Code: "claim_unsupported", Field: "title", Message: "Claim lacks evidence", Metadata: map[string]string{"source": "raw-1"}},
		},
	}

	proposal, err := ValidateCandidate(context.Background(), validRequest(), candidate)
	if err != ErrPolicyRejected {
		t.Fatalf("ValidateCandidate() error = %v, want ErrPolicyRejected", err)
	}
	if len(proposal.Warnings) != 1 {
		t.Fatalf("ValidateCandidate() warnings = %#v, want one canonical warning", proposal.Warnings)
	}
	if len(proposal.Rejections) != 1 {
		t.Fatalf("ValidateCandidate() rejections = %#v, want one canonical rejection", proposal.Rejections)
	}
}

func TestCandidateOrdersDistinctDiagnosticMetadataDeterministically(t *testing.T) {
	t.Parallel()

	metadataOne := map[string]string{"a": "b", "c": "d"}
	metadataTwo := map[string]string{"a": "b\x00c\x00d"}
	propose := func(reverse bool) Proposal {
		t.Helper()

		warnings := []Warning{
			{Code: "same", Field: "description", Message: "same warning", Metadata: metadataOne},
			{Code: "same", Field: "description", Message: "same warning", Metadata: metadataTwo},
		}
		rejections := []Rejection{
			{Code: "same", Field: "title", Message: "same rejection", Metadata: metadataOne},
			{Code: "same", Field: "title", Message: "same rejection", Metadata: metadataTwo},
		}
		if reverse {
			warnings[0], warnings[1] = warnings[1], warnings[0]
			rejections[0], rejections[1] = rejections[1], rejections[0]
		}
		candidate := Candidate{
			Changes:    []FieldChange{{Field: "description", Value: "Steel bottle", EvidenceIDs: []string{"raw-1"}}},
			Warnings:   warnings,
			Rejections: rejections,
		}
		proposal, err := ValidateCandidate(context.Background(), validRequest(), candidate)
		if err != ErrPolicyRejected {
			t.Fatalf("ValidateCandidate() error = %v, want ErrPolicyRejected", err)
		}
		return proposal
	}

	forward := propose(false)
	reversed := propose(true)
	if !reflect.DeepEqual(forward, reversed) {
		t.Fatalf("diagnostic order depends on candidate order\nforward: %#v\nreverse: %#v", forward, reversed)
	}
	if len(forward.Warnings) != 2 || len(forward.Rejections) != 2 {
		t.Fatalf("distinct metadata was collapsed: warnings=%#v rejections=%#v", forward.Warnings, forward.Rejections)
	}
}

func TestCandidateDeduplicatesCandidateAndPolicyRejection(t *testing.T) {
	t.Parallel()

	request := validRequest()
	request.Policy.AllowedFields = []string{"description", "title"}
	request.Policy.RequiredFields = []string{"description", "title"}
	request.Policy.MinimumQualityScore = 0
	candidate := Candidate{
		Changes: []FieldChange{{Field: "description", Value: "Steel bottle", EvidenceIDs: []string{"raw-1"}}},
		Rejections: []Rejection{{
			Code:    " REQUIRED_FIELD_MISSING ",
			Field:   " title ",
			Message: " required field change is missing ",
		}},
	}

	proposal, err := ValidateCandidate(context.Background(), request, candidate)
	if err != ErrPolicyRejected {
		t.Fatalf("ValidateCandidate() error = %v, want ErrPolicyRejected", err)
	}
	want := []Rejection{{
		Code:    "required_field_missing",
		Field:   "title",
		Message: "required field change is missing",
	}}
	if !reflect.DeepEqual(proposal.Rejections, want) {
		t.Fatalf("ValidateCandidate() rejections = %#v, want %#v", proposal.Rejections, want)
	}
}
