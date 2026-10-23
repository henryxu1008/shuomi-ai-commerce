// Package imagex provides current image decoding and metadata inspection.
package imagex

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

// ImageInfo describes basic image metadata derived from decoded bytes.
type ImageInfo struct {
	Width  int
	Height int
	Format string
	Size   int
}

// FromBytesWithFormat 将字节数组转换为图片，同时返回格式
func FromBytesWithFormat(data []byte) (image.Image, string, error) {
	if len(data) == 0 {
		return nil, "", fmt.Errorf("data is empty")
	}

	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("decode image failed: %w", err)
	}

	return img, format, nil
}

// Inspect decodes image bytes and returns their basic metadata.
func Inspect(data []byte) (*ImageInfo, error) {
	img, format, err := FromBytesWithFormat(data)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	return &ImageInfo{
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
		Format: format,
		Size:   len(data),
	}, nil
}
