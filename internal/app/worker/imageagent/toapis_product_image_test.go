package imageagentworker

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent"
	"github.com/stretchr/testify/require"
)

func TestToAPIsSquareOutputNormalization(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 960, 960))
	source.Set(480, 480, color.NRGBA{R: 255, A: 255})
	var pixels bytes.Buffer
	require.NoError(t, png.Encode(&pixels, source))
	normalized, err := normalizeToAPIsProductImage(pixels.Bytes(), "image/png", 960, 960)
	require.NoError(t, err)
	result, err := png.Decode(bytes.NewReader(normalized))
	require.NoError(t, err)
	require.Equal(t, image.Rect(0, 0, 1024, 1024), result.Bounds())

	for _, shape := range [][2]int{{960, 959}, {2049, 2049}, {1024, 1024}} {
		_, err := normalizeToAPIsProductImage(pixels.Bytes(), "image/png", shape[0], shape[1])
		require.ErrorIs(t, err, imageagent.ErrInvalidGeneratedOutput)
	}
}
