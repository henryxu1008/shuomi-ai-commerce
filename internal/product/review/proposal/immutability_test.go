package proposal

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"task-processor/internal/product/catalog"
)

func TestValidateCandidateIsolatesInputsAndCandidateOutput(t *testing.T) {
	t.Parallel()

	request := validRequest()
	request.Snapshot.Attributes = []catalog.Attribute{{Name: "material", Value: "steel"}}
	request.Snapshot.Specifications = &catalog.Specifications{Technical: map[string]string{"grade": "304"}}
	request.Source.RawReference.ReferenceType = "crawler_snapshot"
	request.Source.RawReference.Metadata = map[string]string{"etag": "source-v1"}
	request.Source.ProductCandidate.Attributes = map[string]string{"finish": "matte"}
	request.Policy.AllowedFields = []string{"description"}
	request.Policy.RequiredFields = []string{"description"}
	before, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request before ValidateCandidate: %v", err)
	}

	returned := Candidate{
		Changes: []FieldChange{{Field: "description", Value: "Steel bottle", EvidenceIDs: []string{"raw-1"}}},
		Warnings: []Warning{{
			Code:     "SOURCE_LIMITED",
			Field:    "description",
			Message:  "Only one source",
			Metadata: map[string]string{"source": "raw-1"},
		}},
	}
	isolated := cloneRequest(request)
	isolated.Snapshot.Attributes[0].Value = "plastic"
	isolated.Snapshot.Specifications.Technical["grade"] = "unknown"
	isolated.Source.RawReference.Metadata["etag"] = "copy-mutated"
	isolated.Source.RawReference.ReferenceType = "copy-mutated"
	isolated.Source.ProductCandidate.Attributes["finish"] = "gloss"
	isolated.Policy.AllowedFields[0] = "title"
	isolated.Policy.RequiredFields[0] = "title"
	proposal, err := ValidateCandidate(context.Background(), request, returned)
	if err != nil {
		t.Fatalf("ValidateCandidate() error = %v", err)
	}
	after, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request after ValidateCandidate: %v", err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("ValidateCandidate() mutated request\nbefore: %s\nafter:  %s", before, after)
	}
	if got := proposal.Evidence[0].Metadata["etag"]; got != "source-v1" {
		t.Fatalf("proposal evidence metadata = %q, want source-v1", got)
	}
	if got := proposal.Evidence[0].ReferenceType; got != "crawler_snapshot" {
		t.Fatalf("proposal evidence reference type = %q, want crawler_snapshot", got)
	}

	returned.Changes[0].EvidenceIDs[0] = "candidate-mutated"
	returned.Warnings[0].Metadata["source"] = "candidate-mutated"
	if got := proposal.Changes[0].EvidenceIDs[0]; got != "raw-1" {
		t.Fatalf("proposal change aliases candidate: evidence ID = %q", got)
	}
	if got := proposal.Warnings[0].Metadata["source"]; got != "raw-1" {
		t.Fatalf("proposal warning aliases candidate: metadata = %q", got)
	}

	proposal.Changes[0].EvidenceIDs[0] = "proposal-mutated"
	proposal.Warnings[0].Metadata["source"] = "proposal-mutated"
	proposal.Evidence[0].Metadata["etag"] = "proposal-mutated"
	if got := returned.Changes[0].EvidenceIDs[0]; got != "candidate-mutated" {
		t.Fatalf("candidate aliases proposal: evidence ID = %q", got)
	}
	if got := returned.Warnings[0].Metadata["source"]; got != "candidate-mutated" {
		t.Fatalf("candidate warning aliases proposal: metadata = %q", got)
	}
	if got := request.Source.RawReference.Metadata["etag"]; got != "source-v1" {
		t.Fatalf("source metadata aliases proposal evidence: etag = %q", got)
	}
}
