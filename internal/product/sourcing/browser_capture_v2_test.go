package sourcing

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func browserV2Fixture(t *testing.T) []byte {
	t.Helper()
	var payload map[string]any
	require.NoError(t, json.Unmarshal(browserFixture(t), &payload))
	payload["captureVersion"] = 2
	e := payload["evidence"].(map[string]any)
	e["schemaVersion"], e["parserVersion"] = 2, "1688-browser-dom/v3"
	v := e["variants"].([]any)[0].(map[string]any)
	v["images"] = []any{map[string]any{"url": "https://cbu01.alicdn.com/blue.jpg", "role": "source"}}
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	return raw
}

func TestBrowserCaptureV2StrictWire(t *testing.T) {
	body := browserV2Fixture(t)
	capture, err := ParseBrowserCapture(body)
	require.NoError(t, err)
	require.Contains(t, string(capture.CanonicalPayload), `"images":[{"url":"https://cbu01.alicdn.com/blue.jpg","role":"source"}]`)
	for _, mutate := range []func(string) string{
		func(s string) string { return strings.Replace(s, `"captureVersion":2`, `"captureVersion":1`, 1) },
		func(s string) string { return strings.Replace(s, `"schemaVersion":2`, `"schemaVersion":1`, 1) },
		func(s string) string { return strings.Replace(s, `1688-browser-dom/v3`, `1688-browser-dom/v2`, 1) },
		func(s string) string {
			return strings.Replace(s, `"role":"source"`, `"role":"source","approved":true`, 1)
		},
		func(s string) string {
			return strings.Replace(s, `"captureVersion":2`, `"captureVersion":2,"captureVersion":2`, 1)
		},
	} {
		_, err := ParseBrowserCapture([]byte(mutate(string(body))))
		require.Error(t, err)
	}
}

func TestBrowserCaptureV2ImagesStayWithTheirVariant(t *testing.T) {
	capture, err := ParseBrowserCapture(browserV2Fixture(t))
	require.NoError(t, err)
	blue := capture.Evidence.Variants[0]
	red := blue
	id := "red"
	red.SourceID = &id
	red.Attributes = []AcquisitionAttribute{{Name: "颜色", Value: "红色"}}
	red.Images = []AcquisitionImage{{URL: "https://cbu01.alicdn.com/red.jpg", Role: "source"}, blue.Images[0]}
	capture.Evidence.Variants = append(capture.Evidence.Variants, red)
	envelope, err := MapBrowserCaptureEvidence(capture.Source, capture.Evidence, "v2-operation")
	require.NoError(t, err)
	snapshot, err := ToSnapshot(envelope)
	require.NoError(t, err)
	require.Len(t, snapshot.Variants, 2)
	require.Equal(t, "https://cbu01.alicdn.com/blue.jpg", snapshot.Variants[0].Images[0].URL)
	require.Equal(t, "https://cbu01.alicdn.com/red.jpg", snapshot.Variants[1].Images[0].URL)
	require.Equal(t, "https://cbu01.alicdn.com/blue.jpg", snapshot.Variants[1].Images[1].URL)
	require.Equal(t, BrowserCaptureV2Contract, envelope.RawReference.Metadata["contract_version"])
	_, err = MapAcquisitionEvidence(capture.Source, capture.Evidence, AcquisitionChannelBrowser, "v1-operation")
	require.Error(t, err)
	capture.Evidence.Variants[1].Attributes = blue.Attributes
	_, err = MapBrowserCaptureEvidence(capture.Source, capture.Evidence, "ambiguous")
	require.Error(t, err)
}

func TestBrowserCaptureV2NestedImageBudgetAndDigest(t *testing.T) {
	body := browserV2Fixture(t)
	capture, err := ParseBrowserCapture(body)
	require.NoError(t, err)
	changed, err := ParseBrowserCapture([]byte(strings.Replace(string(body), "blue.jpg", "red.jpg", 1)))
	require.NoError(t, err)
	require.NotEqual(t, capture.PayloadSHA256, changed.PayloadSHA256)
	var p map[string]any
	require.NoError(t, json.Unmarshal(body, &p))
	e := p["evidence"].(map[string]any)
	v := e["variants"].([]any)[0].(map[string]any)
	images := make([]any, 256)
	for i := range images {
		images[i] = map[string]any{"url": "https://cbu01.alicdn.com/blue.jpg", "role": "source"}
	}
	v["images"] = images
	e["variants"] = []any{v, v, v, v}
	raw, err := json.Marshal(p)
	require.NoError(t, err)
	_, err = ParseBrowserCapture(raw)
	require.Error(t, err, "each reference counts even when the URL repeats")
	capture.Evidence.Variants[0].SourceID = nil
	capture.Evidence.Variants[0].Attributes = nil
	_, err = MapBrowserCaptureEvidence(capture.Source, capture.Evidence, "unidentified")
	require.Error(t, err)
}
