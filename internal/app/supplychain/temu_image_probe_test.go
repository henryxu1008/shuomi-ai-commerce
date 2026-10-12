package supplychainapp

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/require"
	"image"
	"image/png"
	"io"
	"net/http"
	"task-processor/internal/product/asset"
	"testing"
)

func TestTemuImageEvidenceUsesActualBytesAndCannotProbePrivateSource(t *testing.T) {
	var content bytes.Buffer
	require.NoError(t, png.Encode(&content, image.NewRGBA(image.Rect(0, 0, 900, 900))))
	raw := content.Bytes()
	calls := 0
	probe := NewPublicImageProbe()
	probe.client = &http.Client{Transport: imageTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"image/jpeg"}}, Body: io.NopCloser(bytes.NewReader(raw))}, nil
	})}
	approved := asset.ApprovedAsset{ID: "a", URL: "https://images.example.org/source.png", Width: 1, Height: 1}
	observed, err := probe.ProbeTemu(context.Background(), approved, 34, 3, "local")
	require.NoError(t, err)
	require.Equal(t, "image/png", observed.MediaType)
	require.Equal(t, 900, observed.Width)
	require.EqualValues(t, len(raw), observed.Bytes)
	require.Empty(t, observed.RemoteURLs)
	raw = raw[:40]
	_, err = probe.ProbeTemu(context.Background(), approved, 34, 3, "local")
	require.Error(t, err)
	approved.URL = "https://127.0.0.1/private.png"
	_, err = probe.ProbeTemu(context.Background(), approved, 34, 3, "local")
	require.Error(t, err)
	require.Equal(t, 2, calls)
}
