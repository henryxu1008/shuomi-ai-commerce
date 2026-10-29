package imagex

import (
	"encoding/base64"
	"testing"
)

func TestFromBytesWithFormat_DecodesWebP(t *testing.T) {
	const tinyWebPBase64 = "UklGRjwAAABXRUJQVlA4IDAAAADQAQCdASoCAAEAAgA0JaACdLoB+AADsAD+8Oj3/yC5YXXI1/8gP+QH/ID/+PIAAAA="

	data, err := base64.StdEncoding.DecodeString(tinyWebPBase64)
	if err != nil {
		t.Fatalf("decode base64: %v", err)
	}

	img, format, err := FromBytesWithFormat(data)
	if err != nil {
		t.Fatalf("FromBytesWithFormat: %v", err)
	}
	if img == nil {
		t.Fatal("expected decoded image")
	}
	if format != "webp" {
		t.Fatalf("format = %q, want webp", format)
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		t.Fatalf("decoded size = %dx%d, want positive dimensions", width, height)
	}
}

func TestFromBytesWithFormatAndInspect_DecodesJPEGAndPNG(t *testing.T) {
	// Fixed synthetic 2x1 fixtures; do not import encoders here. The production
	// package must register each decoder without assistance from this test.
	const tinyJPEGBase64 = "/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/2wBDAQkJCQwLDBgNDRgyIRwhMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjL/wAARCAABAAIDASIAAhEBAxEB/8QAHwAAAQUBAQEBAQEAAAAAAAAAAAECAwQFBgcICQoL/8QAtRAAAgEDAwIEAwUFBAQAAAF9AQIDAAQRBRIhMUEGE1FhByJxFDKBkaEII0KxwRVS0fAkM2JyggkKFhcYGRolJicoKSo0NTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqDhIWGh4iJipKTlJWWl5iZmqKjpKWmp6ipqrKztLW2t7i5usLDxMXGx8jJytLT1NXW19jZ2uHi4+Tl5ufo6erx8vP09fb3+Pn6/8QAHwEAAwEBAQEBAQEBAQAAAAAAAAECAwQFBgcICQoL/8QAtREAAgECBAQDBAcFBAQAAQJ3AAECAxEEBSExBhJBUQdhcRMiMoEIFEKRobHBCSMzUvAVYnLRChYkNOEl8RcYGRomJygpKjU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6goOEhYaHiImKkpOUlZaXmJmaoqOkpaanqKmqsrO0tba3uLm6wsPExcbHyMnK0tPU1dbX2Nna4uPk5ebn6Onq8vP09fb3+Pn6/9oADAMBAAIRAxEAPwDyOiiiuw5D/9k="
	const tinyPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAIAAAABCAIAAAB7QOjdAAAAD0lEQVR4nGMU0bBhYGAAAAI+AHrgZlz7AAAAAElFTkSuQmCC"
	for _, fixture := range []struct {
		format string
		base64 string
	}{
		{format: "jpeg", base64: tinyJPEGBase64},
		{format: "png", base64: tinyPNGBase64},
	} {
		t.Run(fixture.format, func(t *testing.T) {
			data, err := base64.StdEncoding.DecodeString(fixture.base64)
			if err != nil {
				t.Fatalf("decode fixture: %v", err)
			}
			img, format, err := FromBytesWithFormat(data)
			if err != nil {
				t.Fatalf("FromBytesWithFormat: %v", err)
			}
			if img == nil || format != fixture.format {
				t.Fatalf("image = %v, format = %q, want non-nil %s", img, format, fixture.format)
			}
			bounds := img.Bounds()
			if bounds.Dx() != 2 || bounds.Dy() != 1 {
				t.Fatalf("decoded size = %dx%d, want 2x1", bounds.Dx(), bounds.Dy())
			}
			info, err := Inspect(data)
			if err != nil {
				t.Fatalf("Inspect: %v", err)
			}
			if info == nil || info.Width != 2 || info.Height != 1 || info.Format != fixture.format || info.Size != len(data) {
				t.Fatalf("metadata = %+v, want 2x1 %s with %d bytes", info, fixture.format, len(data))
			}
		})
	}
}
