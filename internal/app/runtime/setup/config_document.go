package setup

import (
	"bytes"
	"encoding/json"
	"io"
	"task-processor/internal/app/setupcontrol"
)

// A module toggle must preserve unrelated int64 revisions and exact decimal
// quotes. Decoding deployment documents through float64 silently rounds them.
func documentMap(raw []byte) (map[string]any, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var result map[string]any
	if d.Decode(&result) != nil || result == nil || d.Decode(new(any)) != io.EOF {
		return nil, setupcontrol.ErrInvalid
	}
	return result, nil
}
