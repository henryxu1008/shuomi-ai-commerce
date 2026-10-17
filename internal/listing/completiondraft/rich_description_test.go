package completiondraft

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
	"testing"
)

func TestDescriptionFormatHashAndValidation(t *testing.T) {
	in := SaveInput{SourceID: uuid.NewString(), Content: Content{Title: "title", Description: "<p>literal</p>"}}
	oldJSON, _ := json.Marshal(in)
	require.NotContains(t, string(oldJSON), "descriptionFormat")
	require.Equal(t, collection.Digest(string(oldJSON)), InputHash(in))
	oldHash := InputHash(in)
	require.NoError(t, json.Unmarshal([]byte(`{"title":"title","description":"<p>literal</p>","descriptionFormat":"html-v1"}`), &in.Content))
	require.NoError(t, in.Validate())
	require.NotEqual(t, oldHash, InputHash(in))
	require.NoError(t, json.Unmarshal([]byte(`{"title":"title","description":"<script>x</script>","descriptionFormat":"html-v1"}`), &in.Content))
	require.ErrorIs(t, in.Validate(), preparation.ErrInvalid)
}
func TestSavedRichDescriptionRejectsImageOutsideFixedSource(t *testing.T) {
	s, _, r, ctx, source := fixture(t)
	var content Content
	require.NoError(t, json.Unmarshal([]byte(`{"title":"title","description":"<p><img src=\"https://images.example.org/foreign.png\"></p>","descriptionFormat":"html-v1"}`), &content))
	_, err := s.Save(ctx, uuid.NewString(), SaveInput{SourceID: source.ID, Content: content})
	require.ErrorIs(t, err, preparation.ErrInvalid)
	require.Zero(t, r.writes)
}

func TestCandidateDescriptionImagesAreVersionedWithoutChangingSource(t *testing.T) {
	s, _, r, ctx, source := fixture(t)
	var content Content
	require.NoError(t, json.Unmarshal([]byte(`{"title":"candidate","description":"<p><img src=\"https://images.example.org/new.png\"></p>","descriptionFormat":"html-v1","descriptionImages":["https://images.example.org/new.png"]}`), &content))
	input := SaveInput{SourceID: source.ID, Content: content}
	require.NoError(t, input.Validate())
	receipt, err := s.Save(ctx, uuid.NewString(), input)
	require.NoError(t, err)
	require.Equal(t, source, receipt.Record.Source)
	raw, err := json.Marshal(receipt.Record.Content)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"descriptionImages":["https://images.example.org/new.png"]`)
	require.Equal(t, 1, r.writes)
}

func TestCandidateDescriptionImageDeclarationsAreBoundedAndReferenced(t *testing.T) {
	for _, candidate := range []string{
		`{"title":"t","description":"<p>text</p>","descriptionFormat":"html-v1","descriptionImages":["https://images.example.org/new.png"]}`,
		`{"title":"t","description":"<img src=\"https://images.example.org/new.png\">","descriptionFormat":"html-v1","descriptionImages":["https://images.example.org/new.png","https://images.example.org/new.png"]}`,
		`{"title":"t","description":"plain","descriptionImages":["https://images.example.org/new.png"]}`,
	} {
		var in SaveInput
		in.SourceID = uuid.NewString()
		require.NoError(t, json.Unmarshal([]byte(candidate), &in.Content))
		require.ErrorIs(t, in.Validate(), preparation.ErrInvalid)
	}
}

func TestCandidateDescriptionImageHashPreservesAbsentEncoding(t *testing.T) {
	in := SaveInput{SourceID: uuid.NewString(), Content: Content{Title: "old", Description: "plain"}}
	before, err := json.Marshal(in)
	require.NoError(t, err)
	require.NotContains(t, string(before), "descriptionImages")
	beforeHash := InputHash(in)
	in.Content.DescriptionImages = []string{}
	require.Equal(t, beforeHash, InputHash(in))
	in.Content.Description = `<img src="https://images.example.org/new.png">`
	in.Content.DescriptionFormat = "html-v1"
	beforeHash = InputHash(in)
	in.Content.DescriptionImages = []string{"https://images.example.org/new.png"}
	require.NoError(t, in.Validate())
	require.NotEqual(t, beforeHash, InputHash(in))
}
