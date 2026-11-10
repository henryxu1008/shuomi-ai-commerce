package goods

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLocalTrialAuthorityNeverProducesSubmissionEvenWithCompleteEvidence(t *testing.T) {
	input, rules, inventory := officialFixture()
	// Decode as a client would: an unrecognized field must not silently become
	// official authority at the shared native builder boundary.
	raw, err := json.Marshal(rules)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, json.Unmarshal(raw, &body))
	for _, authority := range []string{"local_trial", "unknown"} {
		t.Run(authority, func(t *testing.T) {
			body["authority"] = authority
			raw, err := json.Marshal(body)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(raw, &rules))
			observations := []OfficialImageObservation{}
			for i, a := range inventory.Assets {
				observations = append(observations, OfficialImageObservation{AssetID: a.ID, SourceURL: a.URL, Width: a.Width, Height: a.Height, Type: input.Images[i].Type, RemoteURL: "https://img.shein.com/" + a.ID + ".jpg", ResponseHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ContentHash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Bytes: 1000, MediaType: "image/jpeg"})
			}
			result := AssessOfficial(input, rules, inventory, observations, nil).Native()
			require.False(t, result.ReadyForUpload)
			require.Empty(t, result.SubmissionPayload)
			require.Contains(t, issueFields(result), "rules.authority")
		})
	}
}
