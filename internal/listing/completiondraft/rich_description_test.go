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
