package httpimage

import (
	"bytes"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/jpeg"
	_ "image/png"
	productimage "task-processor/internal/product/image"
)

func InspectGeneratedArtifact(content []byte) (string, int, int, error) {
	config, format, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil || config.Width <= 0 || config.Height <= 0 {
		return "", 0, 0, productimage.ErrOutputValidation
	}
	mediaType := map[string]string{"jpeg": "image/jpeg", "png": "image/png", "webp": "image/webp"}[format]
	if mediaType == "" {
		return "", 0, 0, productimage.ErrOutputValidation
	}
	return mediaType, config.Width, config.Height, nil
}
