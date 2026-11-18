package setup

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

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

// manifestFieldName matches the case folding used by encoding/json for struct
// fields. A map has no input order to resolve multiple spellings of one field.
func manifestFieldName(document map[string]any, field string) (string, error) {
	key, found := field, false
	for name := range document {
		if !strings.EqualFold(name, field) {
			continue
		}
		if found {
			return "", setupcontrol.ErrInvalid
		}
		key, found = name, true
	}
	return key, nil
}
