package httpimage_test

import (
	"encoding/base64"
	"task-processor/internal/integration/httpimage"
	productimage "task-processor/internal/product/image"
	"testing"
)

func TestInspectGeneratedArtifactKeepsRegisteredFormatAndDimensions(t *testing.T) {
	// Fixed synthetic repository fixtures: no encoder/decoder imports may help registration.
	for _, fixture := range []struct{ name, encoded, mime string }{
		{"jpeg", "/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/2wBDAQkJCQwLDBgNDRgyIRwhMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjL/wAARCAABAAIDASIAAhEBAxEB/8QAHwAAAQUBAQEBAQEAAAAAAAAAAAECAwQFBgcICQoL/8QAtRAAAgEDAwIEAwUFBAQAAAF9AQIDAAQRBRIhMUEGE1FhByJxFDKBkaEII0KxwRVS0fAkM2JyggkKFhcYGRolJicoKSo0NTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqDhIWGh4iJipKTlJWWl5iZmqKjpKWmp6ipqrKztLW2t7i5usLDxMXGx8jJytLT1NXW19jZ2uHi4+Tl5ufo6erx8vP09fb3+Pn6/8QAHwEAAwEBAQEBAQEBAQAAAAAAAAECAwQFBgcICQoL/8QAtREAAgECBAQDBAcFBAQAAQJ3AAECAxEEBSExBhJBUQdhcRMiMoEIFEKRobHBCSMzUvAVYnLRChYkNOEl8RcYGRomJygpKjU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6goOEhYaHiImKkpOUlZaXmJmaoqOkpaanqKmqsrO0tba3uLm6wsPExcbHyMnK0tPU1dbX2Nna4uPk5ebn6Onq8vP09fb3+Pn6/9oADAMBAAIRAxEAPwDyOiiiuw5D/9k=", "image/jpeg"},
		{"png", "iVBORw0KGgoAAAANSUhEUgAAAAIAAAABCAIAAAB7QOjdAAAAD0lEQVR4nGMU0bBhYGAAAAI+AHrgZlz7AAAAAElFTkSuQmCC", "image/png"},
		{"webp", "UklGRjwAAABXRUJQVlA4IDAAAADQAQCdASoCAAEAAgA0JaACdLoB+AADsAD+8Oj3/yC5YXXI1/8gP+QH/ID/+PIAAAA=", "image/webp"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			data, err := base64.StdEncoding.DecodeString(fixture.encoded)
			if err != nil {
				t.Fatal(err)
			}
			mime, width, height, err := httpimage.InspectGeneratedArtifact(data)
			if err != nil || mime != fixture.mime || width != 2 || height != 1 {
				t.Fatalf("inspection = %q %dx%d %v, want %s 2x1", mime, width, height, err, fixture.mime)
			}
		})
	}
}

func TestInspectGeneratedArtifactRejectsInvalidOrUnsupportedHeaders(t *testing.T) {
	for name, data := range map[string][]byte{
		"nil": nil, "empty": {}, "invalid": []byte("not an image"),
		"truncated png":   {137, 80, 78, 71, 13, 10, 26, 10},
		"unsupported gif": []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00"),
	} {
		t.Run(name, func(t *testing.T) {
			mime, width, height, err := httpimage.InspectGeneratedArtifact(data)
			if err != productimage.ErrOutputValidation || mime != "" || width != 0 || height != 0 {
				t.Fatalf("invalid inspection = %q %dx%d %v; want original output-validation sentinel", mime, width, height, err)
			}
		})
	}
}
