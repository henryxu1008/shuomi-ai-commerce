package sourcing

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// These are pure existing-v2 contract/mapping checks, without persistence or a
// running application. Reference media and observed weights remain source facts.
func TestListingMaterialsRemainInExistingCaptureSnapshots(t *testing.T) {
	for _, platform := range []string{"amazon", "1688"} {
		t.Run(platform, func(t *testing.T) {
			body := amazonFixture()
			image := "https://m.media-amazon.com/images/I/detail.jpg"
			if platform == "1688" {
				body = browserV2Fixture(t)
				image = "https://cbu01.alicdn.com/detail.jpg"
			}
			var payload map[string]any
			require.NoError(t, json.Unmarshal(body, &payload))
			e := payload["evidence"].(map[string]any)
			e["description"] = "Observed product description.\nObserved care instructions."
			e["attributes"] = []any{
				map[string]any{"name": "Material", "value": "Cotton"},
				map[string]any{"name": "来源包装重量（单位未提供）", "value": "0.123456789012345678901"},
			}
			e["images"] = []any{map[string]any{"url": image, "role": "detail"}}
			raw, err := json.Marshal(payload)
			require.NoError(t, err)
			var envelope SourceEnvelope
			if platform == "amazon" {
				capture, parseErr := ParseAmazonBrowserCapture(raw)
				require.NoError(t, parseErr)
				envelope, err = capture.Envelope("materials")
			} else {
				capture, parseErr := ParseBrowserCapture(raw)
				require.NoError(t, parseErr)
				envelope, err = MapBrowserCaptureEvidence(capture.Source, capture.Evidence, "materials")
			}
			require.NoError(t, err)
			snapshot, err := ToSnapshot(envelope)
			require.NoError(t, err)
			require.Equal(t, e["description"], snapshot.Description)
			facts := map[string]string{}
			for _, attribute := range snapshot.Attributes {
				facts[attribute.Name] = attribute.Value
			}
			require.Equal(t, "0.123456789012345678901", facts["来源包装重量（单位未提供）"])
			require.Equal(t, "Cotton", facts["Material"])
			require.Len(t, snapshot.Images, 1)
			require.Equal(t, image, snapshot.Images[0].URL)
			require.Equal(t, "detail", snapshot.Images[0].Role)
			require.NotEqual(t, image, snapshot.Variants[0].Images[0].URL)
			require.Zero(t, snapshot.Variants[0].Stock)
		})
	}
}
